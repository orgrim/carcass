// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terraform

import (
	"errors"
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"io"
	"os"
	"os/user"
)

type Config struct {
	Provider Provider `hcl:"provider,block"`
	Module   Module   `hcl:"module,block"`
	Meta     Meta     `hcl:"terraform,block"`
}

type Provider struct {
	Name string `hcl:"name,label"`
	Uri  string `hcl:"uri"`
}

type Module struct {
	Name        string             `hcl:"name,label"`
	Source      string             `hcl:"source"`
	StoragePool string             `hcl:"storage_pool,optional"`
	Username    string             `hcl:"user_name,optional"`
	SshPubKey   string             `hcl:"user_pubkey,optional"`
	Domain      string             `hcl:"dns_domain"`
	NetworkName string             `hcl:"net_name,optional"`
	NetworkCIDR string             `hcl:"net_cidr"`
	Machines    map[string]Machine `hcl:"vms"` // hostname -> Machine
}

// Since Machine is an object inside terraform, we need to use tags of the
// lower level "github.com/zclconf/go-cty/cty" module used by hcl to load it.
type Machine struct {
	IPAddress    string `cty:"ip"`      // "10.10.0.3"
	Distrib      string `cty:"distrib"` // "debian10"
	Vcpus        int    `cty:"vcpu"`
	Memory       int    `cty:"memory"`
	DataDiskSize int    `cty:"data_size"`
	Iface        string `cty:"iface"`
}

type Meta struct {
	TfVersion    string      `hcl:"required_version"`
	ReqProviders ReqProvider `hcl:"required_providers,block"`
}

type ReqProvider struct {
	Libvirt ProviderInfo `hcl:"libvirt,optional"`
}

type ProviderInfo struct {
	Source  string `cty:"source"`
	Version string `cty:"version"`
}

func ParseModuleConfig(path string) (Config, error) {

	parser := hclparse.NewParser()
	f, diags := parser.ParseHCLFile(path)

	wr := hcl.NewDiagnosticTextWriter(
		os.Stderr,      // writer to send messages to
		parser.Files(), // the parser's file cache, for source snippets
		78,             // wrapping width
		true,           // generate colored/highlighted output
	)

	if diags.HasErrors() {
		wr.WriteDiagnostics(diags)
		return Config{}, fmt.Errorf("could not parse HCL configuration")
	}

	var c Config
	moreDiags := gohcl.DecodeBody(f.Body, nil, &c)
	diags = append(diags, moreDiags...)

	if diags.HasErrors() {
		wr.WriteDiagnostics(diags)
		return Config{}, fmt.Errorf("could not parse HCL configuration")
	}

	return c, nil
}

func WriteModuleConfig(dst io.Writer, conf Config) error {

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(&conf, f.Body())

	_, err := f.WriteTo(dst)
	if err != nil {
		return err
	}

	return nil
}

func NewConfiguration(modulePath string, domain string, netCIDR string) (Config, error) {
	_, err := os.Stat(modulePath)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("module path does not exist")
	}

	username := "carcass"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	vms := make(map[string]Machine)

	mod := Module{
		Name:        "carcass",
		Source:      modulePath,
		StoragePool: "default",
		Username:    username,
		Domain:      domain,
		NetworkName: domain,
		NetworkCIDR: netCIDR,
		Machines:    vms,
	}

	prov := Provider{
		Name: "libvirt",
		Uri:  "qemu:///system", // XXX var
	}

	meta := Meta{
		TfVersion: ">= 0.13",
		ReqProviders: ReqProvider{
			Libvirt: ProviderInfo{
				Source:  "dmacvicar/libvirt",
				Version: "~> 0.6.3",
			},
		},
	}

	cfg := Config{
		Provider: prov,
		Module:   mod,
		Meta:     meta,
	}

	return cfg, nil
}
