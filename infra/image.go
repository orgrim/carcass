// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package infra

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/orgrim/carcass/hv"
)

type Image struct {
	Name string // name of the image, eg "distrib" in the config of
	// terraform. Terraform expects the filename
	// {{Name}}-base.qcow2
	Pool     string // Name of the pool
	Source   string // source URL
	Path     string // path of the image
	Format   string // format of the image qcow2 of other
	Capacity int64
	Size     int64
}

// NewImage initializes a Image struct with the given name and associated
// storage pool
func NewImage(name string, pool string) *Image {
	return &Image{
		Name: name,
		Pool: pool,
	}
}

// ImageGetSource sets up a ReadCloser and gets the total size of the source. The
// format of source is an path or URL, currently supported schemes are file,
// http and https.
func ImageGetSource(source string) (io.ReadCloser, int64, error) {
	var (
		data   io.ReadCloser
		length int64
	)

	// get a Reader to the source data, along with the size of the
	// data. The size is necessary as volume creation requires to declare
	// the capacity of the volume
	u, err := url.Parse(source)
	if err != nil {
		return data, length, fmt.Errorf("image get source: could not parse source: %w", err)
	}

	if strings.HasPrefix(u.Scheme, "http") {
		resp, err := http.Get(source)
		if err != nil {
			return data, length, fmt.Errorf("image get source: http get failed: %w", err)
		}

		data = resp.Body
		length = resp.ContentLength

	} else if u.Scheme == "file" || u.Scheme == "" {
		f, err := os.Open(u.Path)
		if err != nil {
			return data, length, fmt.Errorf("image get source: %s: %w", u.Path, err)
		}

		finfo, err := f.Stat()
		if err != nil {
			return data, length, fmt.Errorf("image get source: stat failed: %w", err)
		}

		data = f
		length = finfo.Size()
	} else {
		return data, length, fmt.Errorf("image get source: unsupported URL: %s", source)
	}

	return data, length, nil
}

// Store downloads or copies the image pointed by the data Reader to the volume
// on the hypervisor. The size of the image must be specified to create the
// volume with the correct capacity on the hypervisor.
func (i *Image) Store(h *hv.Hypervisor, data io.Reader, length int64) error {
	exists, err := i.Exists(h)
	if err != nil {
		return fmt.Errorf("image store: %w", err)
	}

	if exists {
		return fmt.Errorf("image store: image already exists on hypervisor")
	}

	// create the volume and use the API of the hypervisor to send the
	// data, this way we avoid requiring the operation to be done locally
	// on the hypervisor (and potential privilege issues)
	volName := i.volumeName()

	err = hv.CreateVolume(*h, i.Pool, volName, length)
	if err != nil {
		return fmt.Errorf("image store: volume create failed: %w", err)
	}

	err = hv.UploadVolume(*h, i.Pool, volName, data)
	if err != nil {
		// remove the volume in case of failure
		if er := hv.RemoveVolume(*h, i.Pool, volName); er != nil {
			return fmt.Errorf("image store: could not clean volume: %s on upload failure: %w", er, err)
		}
		return fmt.Errorf("image store: volume upload failed: %w", err)
	}

	return nil
}

// Drop removes the volume associated with the image
func (i *Image) Drop(h *hv.Hypervisor) error {
	exists, err := i.Exists(h)
	if err != nil {
		return fmt.Errorf("image drop: %w", err)
	}

	if !exists {
		return nil
	}

	err = hv.RemoveVolume(*h, i.Pool, i.volumeName())
	if err != nil {
		return fmt.Errorf("image drop: %w", err)
	}

	return nil
}

// Exists tests if the image as a storage volume on the hypervisor
func (i *Image) Exists(h *hv.Hypervisor) (bool, error) {
	volName := i.volumeName()

	exists, err := hv.VolumeExists(*h, i.Pool, volName)
	if err != nil {
		return false, fmt.Errorf("could not check if image exists: %w", err)
	}

	return exists, nil
}

// AddSourceMap registers the source of the image into the source map file
// located in the given directory
func (i *Image) AddSourceMap(dir string) error {
	path := sourceMapPath(dir)

	m, err := readSourceMap(path)
	if err != nil {
		perr := errors.Unwrap(err)
		if !errors.Is(perr, os.ErrNotExist) {
			return fmt.Errorf("could not read source map file: %w", err)
		}
	}

	if m == nil {
		m = make(SourceMap)
	}

	if _, ok := m[i.Pool]; !ok {
		m[i.Pool] = make(SourceMapEntry)
	}

	m[i.Pool][i.Name] = i.Source

	err = writeSourceMap(m, path)
	if err != nil {
		return fmt.Errorf("could not save source map file: %w", err)
	}

	return nil
}

// RemoveSourceMap deregisters the source of the image from the source map file
// located in the given directory
func (i *Image) RemoveSourceMap(dir string) error {
	path := sourceMapPath(dir)

	m, err := readSourceMap(path)
	if err != nil {
		perr := errors.Unwrap(err)
		if !errors.Is(perr, os.ErrNotExist) {
			return fmt.Errorf("could not read source map file: %w", err)
		}
	}

	if m == nil {
		return nil
	}

	if _, ok := m[i.Pool]; !ok {
		return nil
	}

	delete(m[i.Pool], i.Name)

	err = writeSourceMap(m, path)
	if err != nil {
		return fmt.Errorf("could not save source map file: %w", err)
	}

	return nil
}

// VolumeName computes the filename of the image on the hypervisor
func (i *Image) volumeName() string {
	format := i.Format
	if format == "" {
		format = "qcow2"
	}

	return fmt.Sprintf("%s-base.%s", i.Name, format)
}

// String returns the information on a image in a YAML like format
func (i *Image) String() string {
	return fmt.Sprintf("%s:\n  pool: %s\n  source: %s\n  path: %s\n  format: %s\n  space: %d/%d",
		i.Name, i.Pool, i.Source, i.Path, i.Format, i.Size, i.Capacity)
}

// LookupImages gets the detailed information on images found in the given
// storage pool of hypervisor
func LookupImages(h *hv.Hypervisor, poolName string, sourceMapDir string) ([]*Image, error) {

	vols, err := hv.ListVolumes(*h, poolName)
	if err != nil {
		return nil, fmt.Errorf("Could not list volumes on hypervisor: %w", err)
	}

	var (
		m       SourceMap
		sources SourceMapEntry
	)

	if sourceMapDir != "" {
		m, _ = readSourceMap(sourceMapPath(sourceMapDir))
	}

	sources = make(SourceMapEntry)
	if m != nil {
		if s, ok := m[poolName]; ok {
			sources = s
		}
	}

	images := make([]*Image, 0)
	for _, vol := range vols {
		if strings.HasSuffix(vol.Name, "-base.qcow2") {

			distrib := strings.TrimSuffix(vol.Name, "-base.qcow2")
			i := Image{
				Name:     distrib,
				Pool:     poolName,
				Path:     vol.Path,
				Format:   vol.Format.Type,
				Capacity: vol.Capacity,
				Size:     vol.Size,
				Source:   sources[distrib],
			}
			images = append(images, &i)
		}
	}

	return images, nil
}

type SourceMapEntry map[string]string
type SourceMap map[string]SourceMapEntry

func sourceMapPath(dir string) string {
	return filepath.Clean(filepath.Join(dir, "image-sources.json"))
}

func readSourceMap(path string) (SourceMap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open resource map file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not read resource map file: %w", err)
	}

	var m SourceMap

	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("could not decode map file: %w", err)
	}

	return m, nil
}

func writeSourceMap(m SourceMap, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode resource map to json: %w", err)
	}

	err = os.WriteFile(path, data, 0666)
	if err != nil {
		return fmt.Errorf("could not create resource map file: %w", err)
	}

	return nil
}
