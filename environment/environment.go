// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package environment

import (
	"fmt"

	"github.com/orgrim/carcass/hv"
	"github.com/orgrim/carcass/infra"
)

// An Environment holds everything needed to provision a working service
// spanning multiple VM.
//
// The configuration of the virtual machines is done with Ansible.
type Environment struct {
	Name        string
	Description string
	Domain      string                // DNS domain
	Network     string                // CIDR network address
	Infra       *infra.Infrastructure //
	// ansible
}

//

func (e Environment) String() string {
	s := fmt.Sprintf("Environment: %s", e.Name)
	if e.Description == "" {
		s += "\n"
	} else {
		s += fmt.Sprintf(" (%s)\n", e.Description)
	}

	s += fmt.Sprintf("  Network: %s  %s\n", e.Infra.Network.Name, e.Infra.Network.Address)
	s += "  Machines:\n"

	width := 0
	for _, d := range e.Infra.Machines {
		if len(d.Name) > width {
			width = len(d.Name)
		}
	}

	for _, d := range e.Infra.Machines {
		s += fmt.Sprintf("    - %s", d.Name)

		for i := 0; i < (width - len(d.Name)); i++ {
			s += fmt.Sprintf(" ")
		}

		s += fmt.Sprintf("  %s", e.Infra.Network.LookupDnsHostByName(d.Name))

		for _, disk := range d.Disks {
			if disk.Source.BackingVolName != "" {
				s += fmt.Sprintf("  %s", infra.ImageNameFromVolume(disk.Source.BackingVolName))
			}
		}

		if d.Status {
			s += fmt.Sprintln("  active")
		} else {
			s += fmt.Sprint("\n")
		}

	}
	return s
}

func Lookup(h *hv.Hypervisor, name string) (*Environment, error) {

	is, err := infra.LookupByName(h, name)
	if err != nil {
		return nil, fmt.Errorf("could not load environment: %w", err)
	}

	env := &Environment{
		Name:   name,
		Domain: name,
		Infra:  is,
	}

	return env, nil
}

func NewEnvironment(h *hv.Hypervisor, name string, desc string) *Environment {
	return &Environment{
		Name:        name,
		Description: desc,
		Domain:      name,
		Infra:       infra.NewInfrastructure(h),
	}
}

func (e *Environment) Create() error {
	// network := hv.NewNetwork(e.Name, "10.0.10.0")
	// return network.Create(e.HV)
	// terrafrom stuff
	return nil
}

func (e *Environment) Start() {
	e.Infra.StartAll()
}

func (e *Environment) Stop(force bool) {
	e.Infra.StopAll(force)
}
