// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "carcass",
		Short: "A simple virtual machine management tool",
		Long: `Carcass is a tool to manage sets of libvirt based virtual machines. From
creation to customisation, using libvirt, Terraform, Ansible, CFSSL and a
simple web UI.`,
	}
	Uri     string
	DataDir string
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&Uri, "connect", "c", "qemu:///system", "hypervisor connection URI")
	rootCmd.PersistentFlags().StringVarP(&DataDir, "data-dir", "d", "~/.local/share/carcass", "data directory")
}
