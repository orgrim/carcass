# Réseau libvirt avec la configuration du DNS pour les VMs. Pour
# l'accès depuis la machine, on peut soit configurer un dnsmasq et
# modifier son /etc/resolv.conf, soit ajouter des entrées dans
# /etc/hosts
resource "libvirt_network" "network" {
  name = var.net_name
  mode = "nat"
  domain = var.dns_domain
  addresses = [ var.net_cidr ]
  dns {
    enabled = true
    dynamic "hosts" {
      for_each = var.vms
      content {
        hostname = "${hosts.key}.${var.dns_domain}"
        ip = hosts.value.ip
      }
    }
  }
  dhcp {
    enabled = false
  }
  autostart = true
}

# distrib-base.qcow2 must exist in the storage pool
resource "libvirt_volume" "os_volume" {
  name = "os_volume-${each.key}.${var.dns_domain}.qcow2"
  base_volume_name = "${each.value.distrib}-base.qcow2"
  base_volume_pool = var.storage_pool
  pool = var.storage_pool
  for_each = var.vms
}

resource "libvirt_volume" "data_volume" {
  name = "data_volume-${each.key}.${var.dns_domain}.qcow2"
  pool = var.storage_pool

  size  = each.value.data_size
  for_each = var.vms
}


# Cloud Init
data "template_file" "ci_user_data" {
  template = file("${path.module}/cloud_init_user_data")
  vars = {
    username = var.user_name
    ssh_pubkey = var.user_pubkey
  }
}

data "template_file"  "ci_meta_data" {
  template = file("${path.module}/cloud_init_meta_data")
  vars = {
    hostname = "${each.key}.${var.dns_domain}"
  }
  for_each = var.vms
}

data "template_file"  "ci_network_config" {
  template = file("${path.module}/cloud_init_network_config")
  vars = {
    ip = each.value.ip
    gw = cidrhost(var.net_cidr, 1)
    domain = var.dns_domain
    iface = each.value.iface
  }
  for_each = var.vms
}

resource "libvirt_cloudinit_disk" "ci_disk" {
  name = "cloud_init-${each.key}.${var.dns_domain}.iso"
  pool = var.storage_pool

  user_data = data.template_file.ci_user_data.rendered
  meta_data = data.template_file.ci_meta_data[each.key].rendered
  network_config = data.template_file.ci_network_config[each.key].rendered
  for_each = var.vms
}

resource "libvirt_domain" "kvm" {
  name = "${each.key}.${var.dns_domain}"
  memory = each.value.memory
  vcpu = each.value.vcpu

  network_interface {
    addresses = [ each.value.ip ]
    hostname = "${each.key}.${var.dns_domain}"
    network_name = var.net_name
  }

  disk {
    volume_id = libvirt_volume.os_volume[each.key].id
    scsi = true
  }

  disk {
    volume_id = libvirt_volume.data_volume[each.key].id
    scsi = true
  }

  cloudinit = libvirt_cloudinit_disk.ci_disk[each.key].id

  for_each = var.vms
}

terraform {
  required_version = ">= 0.13"
  required_providers {
    libvirt = {
      source = "dmacvicar/libvirt"
      version = "~> 0.6.2"
    }
    template = {
      source = "hashicorp/template"
      version = "~> 2.1.2"
    }
  }
}
