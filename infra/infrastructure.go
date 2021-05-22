// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package infra

import (
	"fmt"
	"log"

	"github.com/orgrim/carcass/hv"
	"github.com/orgrim/carcass/terraform"
)

// An Infrastructure is a set of virtual machines connected to a virtual
// network on an hypervisor.
//
// An Infrastructure is built from a Terraform configuration, on the local
// libvirt hypervisor.
type Infrastructure struct {
	Config   terraform.Config
	HV       *hv.Hypervisor
	Network  hv.Network
	Machines []hv.Domain
}

// NewInfrastructure creates a new, empty Infrastructure on the given hypervisor
func NewInfrastructure(h *hv.Hypervisor) *Infrastructure {
	return &Infrastructure{
		HV: h,
	}
}

// LookupByName searches for an existing Infrastructure on the given hypervisor
// with the name of the network and returns network and objects, the
// configuration must be loaded explictly.
func LookupByName(h *hv.Hypervisor, netName string) (*Infrastructure, error) {

	network, err := hv.LookupNetwork(*h, netName)
	if err != nil {
		return nil, fmt.Errorf("could not find network on hypervisor: %w", err)
	}

	domains, err := hv.ListDomainsByNetwork(*h, network)
	if err != nil {
		return nil, fmt.Errorf("could not find vms on hypervisor: %w", err)
	}

	is := &Infrastructure{
		HV:       h,
		Network:  network,
		Machines: domains,
	}

	return is, nil
}

// LoadConfig parses the configuration file pointed by path into i
//
// A valid Config is required for other operations.
func (i *Infrastructure) LoadConfig(path string) error {
	return nil
}

// RefreshRessources searches and loads current network and machines from the hypervisor
//
// It uses the configuration
func (i *Infrastructure) RefreshRessources() error {
	return nil
}

// Control
func (i *Infrastructure) Start(name string) error {
	for _, m := range i.Machines {

		if m.Name != name {
			continue
		}

		dom, err := i.HV.Conn.LookupDomainByName(m.Name)
		if err != nil {
			log.Printf("could not lookup domain %s: %s", m.Name, err)
			continue
		}
		defer dom.Free()

		log.Printf("request start of: %s", m.Name)
		err = dom.Create()
		if err != nil {
			log.Printf("could not start domain %s: %s", m.Name, err)
			continue
		}
	}

	return nil
}

func (i *Infrastructure) StartAll() error {
	for _, m := range i.Machines {
		dom, err := i.HV.Conn.LookupDomainByName(m.Name)
		if err != nil {
			log.Printf("could not lookup domain %s: %s", m.Name, err)
			continue
		}
		defer dom.Free()

		log.Printf("request start of: %s", m.Name)
		err = dom.Create()
		if err != nil {
			log.Printf("could not start domain %s: %s", m.Name, err)
			continue
		}
	}

	return nil
}

func (i *Infrastructure) Stop(name string, force bool) error {
	for _, m := range i.Machines {
		if m.Name != name {
			continue
		}

		dom, err := i.HV.Conn.LookupDomainByName(m.Name)
		if err != nil {
			log.Printf("could not lookup domain %s: %s", m.Name, err)
			continue
		}
		defer dom.Free()

		active, err := dom.IsActive()
		if err != nil {
			log.Printf("could not get status of domain: %s", err)
			continue
		}

		if active || force {
			log.Printf("request shutdown of: %s", m.Name)
			err = dom.Shutdown()
			if err != nil {
				log.Printf("could not shutdown domain %s: %s", m.Name, err)
				continue
			}
		}
	}
	return nil
}

func (i *Infrastructure) StopAll(force bool) error {
	for _, m := range i.Machines {

		dom, err := i.HV.Conn.LookupDomainByName(m.Name)
		if err != nil {
			log.Printf("could not lookup domain %s: %s", m.Name, err)
			continue
		}
		defer dom.Free()

		active, err := dom.IsActive()
		if err != nil {
			log.Printf("could not get status of domain: %s", err)
			continue
		}

		if active || force {
			log.Printf("request shutdown of: %s", m.Name)
			err = dom.Shutdown()
			if err != nil {
				log.Printf("could not shutdown domain %s: %s", m.Name, err)
				continue
			}
		}
	}
	return nil
}
