// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terraform

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed data
var data embed.FS

// ExtractData extract the terraform module embedded into the binary to
// a directory
func ExtractModule(module string, destdir string) {

}

// Install downloads, checks and install terraform into the target directory
func InstallTerraform(destdir string, version string) error {
	err := os.MkdirAll(destdir, 0755)
	if err != nil {
		return err
	}

	baseUrl := "https://releases.hashicorp.com/terraform"

	// terraform comes in a zip archive, we need a temporary directory to store it
	archive := fmt.Sprintf("terraform_%s_%s_%s.zip", version, runtime.GOOS, runtime.GOARCH)
	archiveLocation := fmt.Sprintf("%s/%s/%s", baseUrl, version, archive)

	tmppath := filepath.Join(os.TempDir(), archive)
	if _, err := os.Stat(tmppath); err != nil {
		tmppath, err = downloadFile(archiveLocation, "", os.TempDir())
		if err != nil {
			return err
		}
	}

	// get the checksum file
	sumFile := fmt.Sprintf("terraform_%s_SHA256SUMS", version)
	sumFileLocation := fmt.Sprintf("%s/%s/%s", baseUrl, version, sumFile)

	resp, err := http.Get(sumFileLocation)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// get the checksum
	var needH string
	l := make([]byte, 0)
	for _, b := range data {
		if b == '\r' {
			continue
		}
		if b == '\n' {
			// As soon as we have a line, check if it is the one we
			// want, e.g. it is the sha256sum format
			if strings.HasSuffix(string(l), archive) {
				needH = strings.Split(string(l), " ")[0]
				break
			}
			l = make([]byte, 0)
			continue
		}
		l = append(l, b)
	}

	if needH == "" {
		return fmt.Errorf("could not get the sha256 hash of %s from %s", archive, sumFileLocation)
	}

	// check the hash
	if err := checkHash(tmppath, needH); err != nil {
		return err
	}

	// extract the terraform binary to the destination directory
	ar, err := zip.OpenReader(tmppath)
	if err != nil {
		return err
	}

	binPath := filepath.Join(destdir, "terraform")
	file, err := os.Create(binPath)
	if err != nil {
		return err
	}

	defer file.Close()

	found := false
	for _, f := range ar.File {
		if f.Name != "terraform" {
			continue
		}

		found = true

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		_, err = io.Copy(file, rc)
		if err != nil {
			return err
		}
	}

	if !found {
		return fmt.Errorf("could not find the terraform binary in the archive")
	}

	if err := os.Chmod(binPath, 0755); err != nil {
		return err
	}

	return nil
}

//
func InstallLibvirtProvider(destdir string, version string, distrib string) error {
	err := os.MkdirAll(destdir, 0755)
	if err != nil {
		return err
	}

	chksums, err := getLibvirtProviderSumFile(version)
	if err != nil {
		return err
	}

	target := distrib
	if distrib == "Debian" {
		target = "Ubuntu"
	}

	var sum, archive string
	for _, l := range chksums {
		if strings.Contains(l, target) {
			elems := strings.Split(l, " ")
			sum = elems[0]
			archive = elems[len(elems)-1]
			break
		}
	}

	if sum == "" {
		return fmt.Errorf("could not find provider file for distrib: %s", target)
	}

	log.Println("selected:", archive)

	baseUrl := "https://github.com/dmacvicar/terraform-provider-libvirt/releases/download"
	archiveLocation := fmt.Sprintf("%s/v%s/%s", baseUrl, version, archive)

	tmppath := filepath.Join(os.TempDir(), archive)
	if _, err := os.Stat(tmppath); err != nil {
		tmppath, err = downloadFile(archiveLocation, "", os.TempDir())
		if err != nil {
			return err
		}
	}

	if err := checkHash(tmppath, sum); err != nil {
		return err
	}

	// extract
	file, err := os.Open(tmppath)
	if err != nil {
		return err
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	found := false
	binPath := filepath.Join(destdir, "terraform-provider-libvirt")
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		if hdr.Name == "terraform-provider-libvirt" {
			dst, err := os.Create(binPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			if _, err := io.Copy(dst, tr); err != nil {
				return err
			}
			dst.Close()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("could not find the libvirt provider binary in the archive")
	}

	if err := os.Chmod(binPath, 0755); err != nil {
		return err
	}

	// destdir en dur pour ~/.local/share/terraform/plugins/registry.terraform.io/dmacvicar/libvirt/0.6.2/linux_amd64

	return nil

}

func checkHash(path, sum string) error {
	h := sha256.New()

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err := io.Copy(h, file); err != nil {
		return err
	}

	got := fmt.Sprintf("%x", string(h.Sum(nil)))
	if got != sum {
		return fmt.Errorf("checksum mismatch between upstream checksum and downloaded file: %s vs %s", sum, got)
	}

	return nil
}

func getLibvirtProviderSumFile(version string) ([]string, error) {
	baseUrl := "https://github.com/dmacvicar/terraform-provider-libvirt/releases/download"
	sumFile := fmt.Sprintf("terraform-provider-libvirt-%s.sha256", version)
	sumFileLocation := fmt.Sprintf("%s/v%s/%s", baseUrl, version, sumFile)

	// Get the checksum
	resp, err := http.Get(sumFileLocation)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	output := make([]string, 0)
	l := make([]byte, 0)
	for _, b := range data {
		if b == '\r' {
			continue
		}
		if b == '\n' {
			output = append(output, string(l))
			l = make([]byte, 0)
			continue
		}
		l = append(l, b)
	}

	return output, nil
}

func FindHostDistrib() (string, error) {
	prog, err := exec.LookPath("lsb_release")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(prog, "--id", "--short")
	out, err := cmd.Output()
	if err != nil {
		var cerr *exec.ExitError
		if errors.As(err, &cerr) {
			return "", fmt.Errorf("%s:%s", strings.TrimSpace(string(cerr.Stderr)), err)
		}
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// downloadFile gets the file at the given location and stores it in destdir,
// renaming it to name it name is not empty. It returns the path to the
// downloaded file or an error
func downloadFile(location string, name string, destdir string) (string, error) {
	err := os.MkdirAll(destdir, 0755)
	if err != nil {
		return "", err
	}

	if name == "" {
		u, err := url.Parse(location)
		if err != nil {
			return "", err
		}
		name = filepath.Base(u.Path)
	}

	resp, err := http.Get(location)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	file, err := os.Create(filepath.Join(destdir, name))
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Download with a simple percent progress
	var (
		total   int64 = resp.ContentLength
		current int64 = 0
	)
	for {
		read, err := io.CopyN(file, resp.Body, 131072)
		current += read
		fmt.Printf("%s: %d %%\r", name, current*100/total)

		if err != nil {
			fmt.Printf("\n")
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	return filepath.Join(destdir, name), nil
}
