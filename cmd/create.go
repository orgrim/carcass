// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
}

var (
	createCmd = &cobra.Command{
		Use:   "create env",
		Short: "Create a new environment",
		Long:  "Create a new environment",
		RunE:  create,
	}
)

func create(cmd *cobra.Command, args []string) error {
	// prepare a directory for the env
	if len(args) == 0 {
		return fmt.Errorf("missing environment name")
	}

	envName := args[0]
	if hasForbiddenChars(envName) {
		return fmt.Errorf("invalid environment name")
	}

	workDir, err := prepareWorkDir(DataDir)
	if err != nil {
		return fmt.Errorf("invalid data directory: %w", err)
	}

	envPath := filepath.Join(workDir, envName)

	_, err = os.Stat(envPath)
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("environment directory already exist")
	}

	err = os.MkdirAll(envPath, 0755)
	if err != nil {
		return fmt.Errorf("could not create environment directory: %w", err)
	}

	// create a terraform config with only the network

	// terraform init + apply
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
