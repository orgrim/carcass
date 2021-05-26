// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/orgrim/carcass/terraform"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(destroyCmd)
}

var (
	destroyCmd = &cobra.Command{
		Use:   "destroy env",
		Short: "Remove a new environment",
		Long:  "Remove a new environment",
		Run:   destroy,
	}
)

func destroy(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatalln("missing environment name")
	}

	envName := args[0]
	if hasForbiddenChars(envName) || len(envName) == 0 {
		log.Fatalln("invalid environment name")
	}

	envPath, err := environmentDir(DataDir, envName)
	if err != nil {
		log.Fatalln("invalid data directory:", err)
	}

	_, err = os.Stat(envPath)
	if errors.Is(err, os.ErrNotExist) {
		log.Fatalln("environment does not exist")
	}

	binDir, _ := binaryDir(DataDir)
	tfConfigDir := filepath.Join(envPath, "terraform")

	err = terraform.Destroy(binDir, tfConfigDir)
	if err != nil {
		log.Fatalln(err)
	}

	sudoCmd := exec.Command("sudo", "rm", fmt.Sprintf("/etc/dnsmasq.d/%s.conf", envName))
	fmt.Println("running:", strings.Join(sudoCmd.Args, " "))
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	err = sudoCmd.Run()
	if err != nil {
		log.Fatalln(err)
	}

	err = restartDnsmasq()
	if err != nil {
		log.Fatalln(err)
	}

	err = os.RemoveAll(envPath)
	if err != nil {
		log.Fatalln(err)
	}
}
