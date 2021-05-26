variable "storage_pool" {
  description = "Storage Pool de libvirt où placer les images disque"
  default = "default"
}

# Les adresses IP des VMs doivent faire partie du réseau défini par la
# variable net_cidr
variable "vms" {
  description = "Liste des noms de vm avec leur adresse IP"
  default = {
    pg = {
      ip = "10.10.0.2"
      distrib = "centos7"
      vcpu = 1
      memory = 2048
      data_size = 5368709120 # 5G
      iface = "eth0"
    }
  }
}

variable "dns_domain" {
  description = "Le nom de domaine pour les VM"
  default = "carcass"
}

variable "net_cidr" {
  description = "Réseau des VM au format CIDR"
  default = "10.10.0.0/24"
}

variable "net_name" {
  description = "Name of the network inside libvirt"
  default = "carcass"
}

variable "user_name" {
  description = "Utilisateur dans chaque VM"
  default = "carcass"
}

variable "user_pubkey" {
  description = "Clé Publique SSH de l'utilisateur"
  default = ""
}

