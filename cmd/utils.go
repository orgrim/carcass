// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// data storage tree fonctions

func expandDataDir(path string) (string, error) {
	dataDir := path

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

		dataDir = filepath.Join(homeDir, parts[1])
	}

	return filepath.Clean(dataDir), nil
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

func environmentDir(path string, env string) (string, error) {
	baseDir, err := expandDataDir(path)
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "environments", env), nil
}

func binaryDir(path string) (string, error) {
	baseDir, err := expandDataDir(path)
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "bin"), nil
}

func terraformBaseModDir(path string) (string, error) {
	baseDir, err := expandDataDir(path)
	if err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "terraform", "modules"), nil
}

func terraformModDir(path string, module string) (string, error) {
	tfBaseDir, err := terraformBaseModDir(path)
	if err != nil {
		return "", err
	}

	return filepath.Join(tfBaseDir, module), nil
}

func localConfigPath() (string, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		u, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("could not find current user: %w", err)
		}
		homeDir = u.HomeDir
		if homeDir == "" {
			return "", fmt.Errorf("could not find home directory")
		}
	}

	cfg := filepath.Clean(filepath.Join(homeDir, ".config/carcass/local.json"))

	return cfg, nil
}

func tempFile(pattern string) string {

	str := make([]rune, 0, len([]rune(pattern)))

	rand.Seed(time.Now().UnixNano())

	for _, r := range []rune(pattern) {
		if r == 'X' {
			for {
				b := rune(rand.Intn(123))
				if b < 'A' || b > 'Z' && b < 'a' || b > 'z' {
					continue
				}
				str = append(str, b)
				break
			}
			continue
		}
		str = append(str, r)
	}

	return filepath.Clean(filepath.Join(os.TempDir(), string(str)))
}

// config related fonctions

type localConfig struct {
	StoragePool string `json:"storage_pool,omitempty"`
	Username    string `json:"ssh_user,omitempty"`
	SshPubKey   string `json:"ssh_pubkey,omitempty"`
}

func loadConfig() (localConfig, error) {
	cfg, err := localConfigPath()
	if err != nil {
		return localConfig{}, fmt.Errorf("load config error: %w", err)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		return localConfig{}, fmt.Errorf("load config error: %w", err)
	}

	config := localConfig{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return localConfig{}, fmt.Errorf("load config error: %w", err)
	}

	return config, nil
}

// dnsmasq

func restartDnsmasq() error {
	sudoCmd := exec.Command("sudo", "systemctl", "restart", "dnsmasq")
	fmt.Println("running:", strings.Join(sudoCmd.Args, " "))
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	err := sudoCmd.Run()
	if err != nil {
		return fmt.Errorf("could not restart dnsmasq")
	}

	return nil
}
