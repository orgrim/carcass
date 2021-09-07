// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hv

import (
	"encoding/xml"
	"fmt"
	libvirt "libvirt.org/go/libvirt"
	"log"
	"net"
)

type Hypervisor struct {
	Uri  string
	Conn *libvirt.Connect
}

type Domain struct {
	Type     string  `xml:"type,attr"`
	Name     string  `xml:"name"`
	Uuid     string  `xml:"uuid"`
	Memory   Memory  `xml:"memory"`
	Vcpu     int     `xml:"vcpu"`
	Emulator string  `xml:"device>emulator"`
	Disks    []Disk  `xml:"devices>disk"`
	Ifaces   []Iface `xml:"devices>interface"`
	Status   bool
}

type Memory struct {
	Size int    `xml:",chardata"`
	Unit string `xml:"unit,attr"`
}

type Disk struct {
	Type   string `xml:"type,attr"`
	Kind   string `xml:"device,attr"`
	Device Target `xml:"target"`
	Source Source `xml:"source"`
}

type Source struct {
	Pool    string `xml:"pool,attr"`
	Volume  string `xml:"volume,attr"`
	File    string `xml:"file,attr"`
	Network string `xml:"network,attr"`
	Bridge  string `xml:"bridge,attr"`
}

type Target struct {
	Dev string `xml:"dev,attr"`
	Bus string `xml:"bus,attr"`
}

type Iface struct {
	Type   string     `xml:"type,attr"`
	Mac    MacAddress `xml:"mac"`
	Source Source     `xml:"source"`
	Device Target     `xml:"target"`
}

type MacAddress struct {
	Address string `xml:"address,attr"`
}

type Network struct {
	XMLName xml.Name   `xml:"network"`
	Name    string     `xml:"name"`
	Uuid    string     `xml:"uuid"`
	Address NetIP      `xml:"ip"` // bridge address
	Mac     MacAddress `xml:"mac"`
	Hosts   []DnsHost  `xml:"dns>host"`
}

// func NewNetwork(name string, address string) Network {
// 	return Network{
// 		Name: name,
// 		Address: IP{
// 			Family:  "ipv4",
// 			Address: address,
// 			Prefix:  24,
// 		},
// 	}
// }

// func (n Network) Create(h *Hypervisor) error {
// 	xml, err := xml.Marshal(n)
// 	if err != nil {
// 		return fmt.Errorf("could not create xml for network: %w", err)
// 	}

// 	fmt.Println(string(xml))

// 	net, err := h.Conn.NetworkCreateXML(string(xml))
// 	if err != nil {
// 		return fmt.Errorf("could not create network on hypervisor: %w", err)
// 	}
// 	defer net.Free()

// 	return nil
// }

type NetIP struct {
	Family  string `xml:"family,attr"`
	Address string `xml:"address,attr"`
	Prefix  int    `xml:"prefix,attr"`
	Netmask string `xml:"netmask,attr"`
}

func (i NetIP) String() string {
	var mask net.IPMask

	if i.Netmask != "" {
		mask = net.IPMask(net.ParseIP(i.Netmask).To4())
	} else {
		mask = net.CIDRMask(i.Prefix, 32)
	}

	n := &net.IPNet{
		IP:   net.ParseIP(i.Address).Mask(mask),
		Mask: mask,
	}

	return n.String()
}

type DnsHost struct {
	Address  string `xml:"ip,attr"`
	Hostname string `xml:"hostname"`
}

func (dom Domain) String() string {
	s := fmt.Sprintf("Domain: %s (%s) %s\n", dom.Name, dom.Type, dom.Uuid)
	s += fmt.Sprintf("  cpu: %d, mem: %d %s\n", dom.Vcpu, dom.Memory.Size, dom.Memory.Unit)
	s += "  disks:\n"
	for _, disk := range dom.Disks {
		switch disk.Type {
		case "volume":
			s += fmt.Sprintf("   - %s: %s/%s - volume: %s::%s\n", disk.Kind, disk.Device.Bus, disk.Device.Dev, disk.Source.Pool, disk.Source.Volume)
		case "file":
			s += fmt.Sprintf("   - %s: %s/%s - path: %s\n", disk.Kind, disk.Device.Bus, disk.Device.Dev, disk.Source.File)
		}
	}
	s += "  interfaces:\n"
	for _, iface := range dom.Ifaces {
		s += fmt.Sprintf("   - %s %s net: %s bridge: %s\n", iface.Device.Dev, iface.Mac.Address, iface.Source.Network, iface.Source.Bridge)
	}

	return s
}

func (n Network) String() string {
	s := fmt.Sprintf("Network: %s %s\n", n.Name, n.Uuid)
	s += fmt.Sprintf("  address: %s\n", n.Address)
	s += "  hosts:\n"
	for _, h := range n.Hosts {
		s += fmt.Sprintf("    %s  %s\n", h.Address, h.Hostname)
	}
	return s
}

func (n Network) LookupDnsHostByName(name string) net.IP {
	for _, entry := range n.Hosts {
		if entry.Hostname == name {
			return net.ParseIP(entry.Address)
		}
	}

	return net.IP{}
}

func NewHypervisor(uri string) (Hypervisor, error) {
	h := Hypervisor{
		Uri: uri,
	}
	conn, err := libvirt.NewConnect(uri)
	if err != nil {
		return h, fmt.Errorf("could not connect to hypervisor: %w", err)
	}
	h.Conn = conn

	return h, nil
}

func (h Hypervisor) Close() (int, error) {
	return h.Conn.Close()
}

func parseDomainXMLDesc(desc string) (Domain, error) {
	v := Domain{}
	err := xml.Unmarshal([]byte(desc), &v)
	if err != nil {
		return v, fmt.Errorf("parseDomainXMLDesc: %w", err)
	}
	return v, nil
}

func parseNetworkXMLDesc(desc string) (Network, error) {
	v := Network{}
	err := xml.Unmarshal([]byte(desc), &v)
	if err != nil {
		return v, fmt.Errorf("parseNetworkXMLDesc: %w", err)
	}
	return v, nil
}

func ListNetworks(h Hypervisor) ([]Network, error) {
	nets, err := h.Conn.ListAllNetworks(0)
	if err != nil {
		return nil, fmt.Errorf("could not list networks: %w", err)
	}

	networks := make([]Network, 0, len(nets))
	for _, net := range nets {
		defer net.Free()

		xml, err := net.GetXMLDesc(0)
		if err != nil {
			log.Println("could not get the XML description of the network: %s", err)
			continue
		}

		network, err := parseNetworkXMLDesc(xml)
		if err != nil {
			log.Println("could not parse the XML description of the network: %s", err)
			continue
		}

		networks = append(networks, network)
	}

	return networks, nil
}

func LookupNetwork(h Hypervisor, name string) (Network, error) {
	net, err := h.Conn.LookupNetworkByName(name)
	if err != nil {
		return Network{}, fmt.Errorf("could not lookup network: %w", err)
	}
	defer net.Free()

	xml, err := net.GetXMLDesc(0)
	if err != nil {
		return Network{}, fmt.Errorf("could not get XML description of the network: %w", err)
	}

	network, err := parseNetworkXMLDesc(xml)
	if err != nil {
		return network, err
	}

	return network, nil
}

func ListDomains(h Hypervisor) ([]Domain, error) {
	doms, err := h.Conn.ListAllDomains(0)
	if err != nil {
		return nil, fmt.Errorf("could not list domains: %w", err)
	}

	domains := make([]Domain, 0, len(doms))
	for _, dom := range doms {
		defer dom.Free()

		xml, err := dom.GetXMLDesc(0)
		if err != nil {
			log.Println("could not get XML description of the domain: %s", err)
			continue
		}

		domain, err := parseDomainXMLDesc(xml)
		if err != nil {
			log.Println(err)
			continue
		}

		active, err := dom.IsActive()
		if err != nil {
			log.Println("could not get status of the domain: %s", err)
			continue
		}

		domain.Status = active
		domains = append(domains, domain)
	}

	return domains, nil
}

func ListDomainsByNetwork(h Hypervisor, network Network) ([]Domain, error) {
	allDomains, err := ListDomains(h)
	if err != nil {
		return nil, err
	}

	domains := make([]Domain, 0, len(allDomains))
	for _, domain := range allDomains {
		for _, iface := range domain.Ifaces {
			if iface.Source.Network == network.Name {
				domains = append(domains, domain)
			}
		}
	}

	return domains, nil
}
