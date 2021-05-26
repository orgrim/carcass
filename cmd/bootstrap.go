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
	binDir, err := binaryDir(DataDir)
	if err != nil {
		log.Fatalln("invalid data directory:", err)
	}

	err = terraform.InstallTerraform(binDir, "0.15.3")
	if err != nil {
		log.Fatalln("could not install terraform:", err)
	}

	err = terraform.InstallLibvirtProvider(binDir, "0.6.3")
	if err != nil {
		log.Fatalln("could not install terraform-libvirt-provider:", err)
	}

	tfBaseDir, err := terraformBaseModDir(DataDir)
	if err != nil {
		log.Fatalln("invalid data directory:", err)
	}

	err = terraform.ExtractModule("bones", tfBaseDir)
	if err != nil {
		log.Fatalln(err)
	}

	err = terraform.LinkLibvirtProvider(binDir, "0.6.3")
	if err != nil {
		log.Fatalln(err)
	}
}
