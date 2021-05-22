// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package osimage

type Image struct {
	Name     string // name of the image, eg "distrib" in the config of terraform. Terraform expects the filename {{Name}}-base.qcow2
	Source   string // source URL
	StoreDir string // storage directory of the image
}
