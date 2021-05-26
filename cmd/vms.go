// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"github.com/orgrim/carcass/terraform"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
)

func init() {
	rootCmd.AddCommand(addVmCmd)
	addVmCmd.Flags().StringVar(&ipAddress, "ip", "", "IP Address of the VM in the network of the environment")
	addVmCmd.Flags().StringVar(&distrib, "distrib", "debian10", "Codename of the OS of the VM. See image")
	addVmCmd.Flags().IntVar(&vcpu, "vcpu", 2, "Number of vCPUs")
	addVmCmd.Flags().IntVar(&memory, "ram", 2048, "Amount of RAM in Megabytes")
	addVmCmd.Flags().IntVar(&dataSize, "data", 8, "Size of the data disk in Gigabytes")
}

var (
	addVmCmd = &cobra.Command{
		Use:   "addvm env vmname",
		Short: "Add a VM to an environment",
		Run:   addvm,
	}
	ipAddress string
	distrib   string
	vcpu      int
	memory    int
	dataSize  int
)

func addvm(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		log.Fatalln("missing environment name ov vm name")
	}

	envName := args[0]
	if hasForbiddenChars(envName) || len(envName) == 0 {
		log.Fatalln("invalid environment name")
	}

	vmName := args[1]
	if hasForbiddenChars(vmName) || len(vmName) == 0 {
		log.Fatalln("invalid vm name")
	}

	envPath, err := environmentDir(DataDir, envName)
	if err != nil {
		log.Fatalln("invalid data directory:", err)
	}

	_, err = os.Stat(envPath)
	if errors.Is(err, os.ErrNotExist) {
		log.Fatalln("environment does not exist")
	}

	if ipAddress == "" {
		log.Fatalln("missing IP address")
	}

	tfConfigDir := filepath.Join(envPath, "terraform")
	tfConfigPath := filepath.Join(tfConfigDir, "main.tf")

	conf, err := terraform.ParseModuleConfig(tfConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	conf.Module.Machines[vmName] = terraform.Machine{
		IPAddress:    ipAddress,
		Distrib:      distrib,
		Vcpus:        vcpu,
		Memory:       memory,
		DataDiskSize: dataSize * 1024 * 1024 * 1024,
		Iface:        selectIFace(distrib),
	}

	dst, err := os.Create(tfConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	if err := terraform.WriteModuleConfig(dst, conf); err != nil {
		dst.Close()
		log.Fatalln(err)
	}

	dst.Close()

	binDir, _ := binaryDir(DataDir)
	if err := terraform.Apply(binDir, tfConfigDir); err != nil {
		log.Fatalln(err)
	}

	// Force dnsmasq to re-read the addn-hosts file so that we can resolv
	// the name of the vm
	if err := restartDnsmasq(); err != nil {
		log.Fatalln(err)
	}
}

func selectIFace(distrib string) string {
	switch distrib {
	case "debian10":
		return "ens3"
	case "centos7":
		return "eth0"
	case "centos8":
		return "eth0"
	default:
		return "eth0"
	}
}
