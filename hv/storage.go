// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hv

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math"

	libvirt "libvirt.org/go/libvirt"
)

type Pool struct {
	XMLName xml.Name `xml:"pool"`
	Name    string   `xml:"name"`
	Type    string   `xml:"type,attr"`
	Uuid    string   `xml:"uuid"`
	Path    string   `xml:"target>path"`
	Mode    int      `xml:"target>permissions>mode"`
	Uid     int      `xml:"target>permissions>owner"`
	Gid     int      `xml:"target>permissions>group"`
}

type Volume struct {
	XMLName      xml.Name  `xml:"volume"`
	Name         string    `xml:"name"`
	Type         string    `xml:"type,attr"`
	Key          string    `xml:"key"`
	Capacity     int64     `xml:"capacity"`
	Size         int64     `xml:"physical"`
	Path         string    `xml:"target>path"`
	Format       VolFormat `xml:"target>format"`
	BackingStore string    `xml:"backingStore>path"`
}

type VolFormat struct {
	Type string `xml:"type,attr"`
}

// ListPools retrieves all defined storage pools on the hypervisor and decodes
// their XML definition into usable structs
func ListPools(h Hypervisor) ([]Pool, error) {
	sps, err := h.Conn.ListAllStoragePools(0)
	if err != nil {
		return nil, fmt.Errorf("could not list storage pools: %w", err)
	}

	pools := make([]Pool, 0, len(sps))
	for _, sp := range sps {
		defer sp.Free()

		xml, err := sp.GetXMLDesc(0)
		if err != nil {
			log.Println("could not get the XML description of the storage pool: %s", err)
			continue
		}

		pool, err := parsePoolXMLDesc(xml)
		if err != nil {
			log.Println("could not parse the XML description of the storage pool: %s", err)
			continue
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

// LookupPool searches for a pool on the hypervisor and decodes its XML
// definition
func LookupPool(h Hypervisor, name string) (Pool, error) {

	sp, err := h.Conn.LookupStoragePoolByName(name)
	if err != nil {
		return Pool{}, fmt.Errorf("could not lookup storage pool %s: %w", name, err)
	}
	defer sp.Free()

	xml, err := sp.GetXMLDesc(0)
	if err != nil {
		return Pool{}, fmt.Errorf("could not get the XML description of the storage pool %s: %w", name, err)
	}

	pool, err := parsePoolXMLDesc(xml)
	if err != nil {
		return Pool{}, fmt.Errorf("could not parse the XML description of the storage pool %s: %w", name, err)
	}

	return pool, nil
}

func parsePoolXMLDesc(desc string) (Pool, error) {
	v := Pool{}
	err := xml.Unmarshal([]byte(desc), &v)
	if err != nil {
		return v, fmt.Errorf("parsePoolXMLDesc: %w", err)
	}
	return v, nil
}

// ListVolumes retrieves all defined volumes of the pool on the hypervisor and
// decodes their XML definition into usable structs
func ListVolumes(h Hypervisor, poolName string) ([]Volume, error) {
	sp, err := h.Conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return nil, fmt.Errorf("could not lookup storage pool %s: %w", poolName, err)
	}
	defer sp.Free()

	svols, err := sp.ListAllStorageVolumes(0)
	if err != nil {
		return nil, fmt.Errorf("could not list storage volumes from pool %s: %w", poolName, err)
	}

	volumes := make([]Volume, 0, len(svols))
	for _, sv := range svols {
		defer sv.Free()

		xml, err := sv.GetXMLDesc(0)
		if err != nil {
			log.Println("could not get the XML description of the storage volume: %s", err)
			continue
		}

		volume, err := parseVolumeXMLDesc(xml)
		if err != nil {
			log.Println("could not parse the XML description of the storage pool: %s", err)
			continue
		}

		volumes = append(volumes, volume)
	}

	return volumes, nil
}

// LookupVolume searches for a volume in a pool on the hypervisor and decodes
// its XML definition
func LookupVolume(h Hypervisor, poolName string, volName string) (Volume, error) {
	sp, err := h.Conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return Volume{}, fmt.Errorf("could not lookup storage pool %s: %w", poolName, err)
	}
	defer sp.Free()

	sv, err := sp.LookupStorageVolByName(volName)
	if err != nil {
		return Volume{}, fmt.Errorf("could not lookup volume %s in pool %s: %w", volName, poolName, err)
	}
	defer sv.Free()

	xml, err := sv.GetXMLDesc(0)
	if err != nil {
		log.Println("could not get the XML description of the storage volume: %s", err)
	}

	volume, err := parseVolumeXMLDesc(xml)
	if err != nil {
		log.Println("could not parse the XML description of the storage volume: %s", err)
	}

	return volume, nil
}

func LookupVolumeByPath(h Hypervisor, path string) (Volume, error) {
	sv, err := h.Conn.LookupStorageVolByPath(path)
	if err != nil {
		return Volume{}, fmt.Errorf("could not lookup volume %s: %w", path, err)
	}
	defer sv.Free()

	xml, err := sv.GetXMLDesc(0)
	if err != nil {
		log.Println("could not get the XML description of the storage volume: %s", err)
	}

	volume, err := parseVolumeXMLDesc(xml)
	if err != nil {
		log.Println("could not parse the XML description of the storage volume: %s", err)
	}

	return volume, nil
}

// CreateVolume creates a new qcow2 volume of the given size (capacity) in the
// pool on the hypervisor
func CreateVolume(h Hypervisor, poolName string, volName string, capacity int64) error {

	// get the storage pool
	sp, err := h.Conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return fmt.Errorf("could not lookup storage pool %s: %w", poolName, err)
	}
	defer sp.Free()

	volDef := Volume{
		Name:     volName,
		Type:     "file",
		Capacity: capacity,
		Format:   VolFormat{Type: "qcow2"},
	}

	xml, err := xml.Marshal(volDef)
	if err != nil {
		return fmt.Errorf("could not create XML volume definition: %w", err)
	}

	sv, err := sp.StorageVolCreateXML(string(xml), 0)
	if err != nil {
		return fmt.Errorf("could not create volume from XML: %w", err)
	}
	defer sv.Free()

	return nil
}

// RemoveVolume removes a volume by name in the pool on the hypervisor
func RemoveVolume(h Hypervisor, poolName string, volName string) error {
	sp, err := h.Conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return fmt.Errorf("could not lookup storage pool %s: %w", poolName, err)
	}
	defer sp.Free()

	sv, err := sp.LookupStorageVolByName(volName)
	if err != nil {
		var virError libvirt.Error
		if errors.As(err, &virError) {
			return fmt.Errorf("could not lookup volume: %w", err)
		}
	}
	defer sv.Free()

	err = sv.Delete(libvirt.STORAGE_VOL_DELETE_NORMAL)
	if err != nil {
		return fmt.Errorf("could not delete volume: %w", err)
	}

	return nil
}

// UploadVolume writes contents from the input io.Reader to an existing volume
// using the hypervisor API. The capacity of the volume must be big enough to
// store all the input data.
func UploadVolume(h Hypervisor, poolName string, volName string, r io.Reader) error {

	stream, err := h.Conn.NewStream(0)
	if err != nil {
		return fmt.Errorf("could not create stream: %w", err)
	}
	defer stream.Free()

	sp, err := h.Conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return fmt.Errorf("could not lookup storage pool %s: %w", poolName, err)
	}
	defer sp.Free()

	sv, err := sp.LookupStorageVolByName(volName)
	if err != nil {
		return fmt.Errorf("could not lookup volume %s in pool %s: %w", volName, poolName, err)
	}
	defer sv.Free()

	// initiate the upload. We still have to send data
	err = sv.Upload(stream, 0, 0, libvirt.STORAGE_VOL_UPLOAD_SPARSE_STREAM)

	buf := make([]byte, 1024)
	for {
		got, err := r.Read(buf)
		if err != nil {
			if err != io.EOF {
				serr := stream.Abort()
				e := fmt.Errorf("could not read input: %w", err)
				if serr != nil {
					return fmt.Errorf("stream abort failed: %w and %w", serr, e)
				}
				return e
			}
		}

		if got == 0 && err == io.EOF {
			break
		}

		offset := 0
		for offset < got {
			// send what we've got
			sent, err := stream.Send(buf[offset:got])
			if err != nil {
				return fmt.Errorf("stream send failed: %w", err)
			}
			offset += sent
		}

	}

	ferr := stream.Finish()
	if ferr != nil {
		var virError libvirt.Error
		if errors.As(ferr, &virError) {
			// do not fail when Finish() not supported by the
			// driver on the side libvirt
			if virError.Code != 3 || virError.Domain != 38 {
				return fmt.Errorf("stream finish failed: %w", ferr)
			}
		}
	}

	return nil
}

func sizePretty(s int) string {
	unit := "B"
	size := float64(s)
	if math.Abs(size) > 1024 {
		size /= 1024
		unit = "kB"
		if math.Abs(size) > 1024 {
			size /= 1024
			unit = "MB"
		}
	}
	return fmt.Sprintf("%.2f %s", size, unit)
}

// VolumeExists checks if a volume exists in the pool on the hypervisor
func VolumeExists(h Hypervisor, poolName string, volName string) (bool, error) {
	sp, err := h.Conn.LookupStoragePoolByName(poolName)
	if err != nil {
		return false, fmt.Errorf("could not lookup storage pool %s: %w", poolName, err)
	}
	defer sp.Free()

	sv, err := sp.LookupStorageVolByName(volName)
	if err != nil {
		var virError libvirt.Error
		if errors.As(err, &virError) {
			if virError.Code != 50 || virError.Domain != 18 {
				return false, fmt.Errorf("could not lookup volume: %w", err)
			}
			return false, nil
		}
	}
	sv.Free()

	return true, nil
}

func parseVolumeXMLDesc(desc string) (Volume, error) {
	v := Volume{}
	err := xml.Unmarshal([]byte(desc), &v)
	if err != nil {
		return v, fmt.Errorf("parseVolumeXMLDesc: %w", err)
	}
	return v, nil
}
