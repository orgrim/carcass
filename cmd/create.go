// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orgrim/carcass/terraform"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&NetCIDR, "net", "n", "", "CIDR Network for the environment")
}

var (
	createCmd = &cobra.Command{
		Use:   "create env",
		Short: "Create a new environment",
		Long:  "Create a new environment empty environment with only the virtual network",
		RunE:  create,
	}
	NetCIDR string
)

func create(cmd *cobra.Command, args []string) error {
	// prepare a directory for the env
	if len(args) == 0 {
		return fmt.Errorf("missing environment name")
	}

	envName := args[0]
	if hasForbiddenChars(envName) || len(envName) == 0 {
		return fmt.Errorf("invalid environment name")
	}

	envPath, err := environmentDir(DataDir, envName)
	if err != nil {
		return fmt.Errorf("invalid data directory: %w", err)
	}

	_, err = os.Stat(envPath)
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("environment directory already exist")
	}

	err = os.MkdirAll(envPath, 0755)
	if err != nil {
		return fmt.Errorf("could not create environment directory: %w", err)
	}

	// create a terraform config with only the network
	tfModule, err := terraformModDir(DataDir, "bones")
	if err != nil {
		return fmt.Errorf("invalid data directory: %w", err)
	}

	tfConfig, err := terraform.NewConfiguration(tfModule, envName, NetCIDR)
	if err != nil {
		return err
	}

	// create the dnsmasq configuration
	err = configureDnsmasq("/etc/dnsmasq.d/", envName)
	if err != nil {
		return err
	}

	// get defaults from the user config file
	if userConfig, err := loadConfig(); err == nil {
		tfConfig.Module.StoragePool = userConfig.StoragePool
		tfConfig.Module.Username = userConfig.Username
		tfConfig.Module.SshPubKey = userConfig.SshPubKey
	}

	tfConfigDir := filepath.Join(envPath, "terraform")
	err = os.MkdirAll(tfConfigDir, 0755)
	if err != nil {
		return err
	}

	tfConfigPath := filepath.Join(tfConfigDir, "main.tf")

	dst, err := os.Create(tfConfigPath)
	if err != nil {
		return fmt.Errorf("could not create terraform configuration: %w", err)
	}
	defer dst.Close()

	err = terraform.WriteModuleConfig(dst, tfConfig)
	if err != nil {
		return err
	}

	// terraform init + apply
	binDir, _ := binaryDir(DataDir)

	err = terraform.Init(binDir, tfConfigDir)
	if err != nil {
		return err
	}

	err = terraform.Apply(binDir, tfConfigDir)
	if err != nil {
		return err
	}

	return nil
}

func configureDnsmasq(destdir string, name string) error {
	tmp := tempFile("carcassXXXXX.conf")
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("could not create tempfile: %w", err)
	}

	fmt.Fprintf(f, "local=/%s/\naddn-hosts=/var/lib/libvirt/dnsmasq/%s.addnhosts\n", name, name)

	f.Close()

	dst := filepath.Join(destdir, fmt.Sprintf("%s.conf", name))

	sudoCmd := exec.Command("sudo", "cp", tmp, dst)
	fmt.Println("running:", strings.Join(sudoCmd.Args, " "))
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	err = sudoCmd.Run()
	if err != nil {
		return fmt.Errorf("could not create %s: %w", dst, err)
	}

	os.Remove(tmp)

	err = restartDnsmasq()
	if err != nil {
		return err
	}

	return nil
}

func listNets(ones int) {
	if ones > 32 && ones < 24 {
		return
	}
	h := 256 / (1 << (32 - ones))
	for i := 0; i < h; i++ {
		fmt.Println(i * (1 << (32 - ones)))
	}
}
