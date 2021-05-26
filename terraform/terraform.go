// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terraform

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Init runs terraform init in the target directory, the main.tf configuration
// file must exist in the directory
func Init(binDir, dir string) error {
	return runTerraform(binDir, dir, "init")
}

func Apply(binDir, dir string) error {
	return runTerraform(binDir, dir, "apply", "-auto-approve")
}

func Destroy(binDir, dir string) error {
	return runTerraform(binDir, dir, "destroy", "-auto-approve")
}

func runTerraform(binDir, dir string, args ...string) error {
	tfcmd := exec.Command(filepath.Join(binDir, "terraform"), args...)
	tfcmd.Dir = dir
	tfcmd.Stdout = os.Stdout
	tfcmd.Stderr = os.Stderr
	err := tfcmd.Run()

	return err
}
