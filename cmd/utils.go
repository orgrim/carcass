// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func prepareWorkDir(path string) (string, error) {
	workDir := path

	if strings.HasPrefix(path, "~") {
		var homeDir string

		parts := strings.SplitN(path, "/", 2)
		username := parts[0][1:]

		if username == "" {
			homeDir = os.Getenv("HOME")
			if homeDir == "" {
				u, err := user.Current()
				if err != nil {
					return "", fmt.Errorf("could not expand ~: %w", err)
				}
				homeDir = u.HomeDir
				if homeDir == "" {
					return "", fmt.Errorf("could not expand ~: empty home directory")
				}
			}
		} else {
			u, err := user.Lookup(username)
			if err != nil {
				return "", fmt.Errorf("could not expand ~%s: %w", username, err)
			}
			homeDir = u.HomeDir
			if homeDir == "" {
				return "", fmt.Errorf("could not expand ~%s: empty home directory", username)
			}
		}

		workDir = filepath.Join(homeDir, parts[1])
	}

	return filepath.Clean(workDir), nil
}

func hasForbiddenChars(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '-' || s[i] > '.' && s[i] < '0' || s[i] > '9' &&
			s[i] < 'A' || s[i] > 'Z' && s[i] < 'a' || s[i] > 'z' {
			return true
		}
	}
	return false
}
