// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"log"

	"github.com/orgrim/carcass/terraform"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}

var (
	bootstrapCmd = &cobra.Command{
		Use:   "bootstrap",
		Short: "Setup third party tools",
		Long: `Check, download and install third party tool used by other command, such as
Terraform, Ansible, CFSSL`,
		Run: bootstrap,
	}
)

func bootstrap(cmd *cobra.Command, args []string) {
	distrib, err := terraform.FindHostDistrib()
	if err != nil {
		log.Println(err)
	}

	err = terraform.InstallTerraform("_work/bin", "0.15.3")
	if err != nil {
		log.Println(err)
	}

	err = terraform.InstallLibvirtProvider("_work/bin", "0.6.3", distrib)
	if err != nil {
		log.Println(err)
	}
}
