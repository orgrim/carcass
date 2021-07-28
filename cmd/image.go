// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"log"

	"github.com/orgrim/carcass/hv"
	"github.com/orgrim/carcass/infra"
	"github.com/spf13/cobra"
)

var (
	imageCmd = &cobra.Command{
		Use:   "image [action]",
		Short: "Manage OS images",
	}

	listImageCmd = &cobra.Command{
		Use:   "list",
		Short: "List image in the given storage pool",
		Run:   listImage,
	}

	addImageCmd = &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Download and store an OS cloud image in the storage pool",
		Run:   addImage,
	}

	rmImageCmd = &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove and OS cloud image from the storage pool",
		Run:   rmImage,
	}

	poolName string
)

func init() {
	listImageCmd.Flags().StringVarP(&poolName, "storage-pool", "p", "default", "operate on this storage pool")
	imageCmd.AddCommand(listImageCmd)

	addImageCmd.Flags().StringVarP(&poolName, "storage-pool", "p", "default", "operate on this storage pool")
	imageCmd.AddCommand(addImageCmd)

	rmImageCmd.Flags().StringVarP(&poolName, "storage-pool", "p", "default", "operate on this storage pool")
	imageCmd.AddCommand(rmImageCmd)

	rootCmd.AddCommand(imageCmd)
}

func listImage(cmd *cobra.Command, args []string) {

	h, err := hv.NewHypervisor(Uri)
	if err != nil {
		log.Fatalln(err)
	}
	defer h.Close()

	dataDir, err := expandDataDir(DataDir)
	if err != nil {
		log.Println("warning: could not expand data-dir:", err)
	}

	images, err := infra.LookupImages(&h, poolName, dataDir)
	if err != nil {
		log.Fatalln(err)
	}

	for _, i := range images {
		fmt.Println(i)
	}

}

func addImage(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		log.Fatalln("missing arguments, name and url")
	}

	name := args[0]
	rawurl := args[1]

	h, err := hv.NewHypervisor(Uri)
	if err != nil {
		log.Fatalln(err)
	}
	defer h.Close()

	image := infra.NewImage(name, poolName)
	err = image.Store(&h, rawurl)
	if err != nil {
		log.Fatalln("could not add image:", err)
	}
	image.Source = rawurl

	dataDir, err := expandDataDir(DataDir)
	if err != nil {
		log.Println("warning: could not expand data-dir:", err)
		return
	}

	err = image.AddSourceMap(dataDir)
	if err != nil {
		log.Println("warning:", err)
	}
}

func rmImage(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Fatalln("missing image name")
	}

	name := args[0]

	h, err := hv.NewHypervisor(Uri)
	if err != nil {
		log.Fatalln(err)
	}
	defer h.Close()

	image := infra.NewImage(name, poolName)

	exists, err := image.Exists(&h)
	if err != nil {
		log.Fatalln("could not check if image exists:", err)
	}

	if !exists {
		log.Fatalf("OS image %s does not exist in pool %s", name, poolName)
	}

	dataDir, err := expandDataDir(DataDir)
	if err != nil {
		log.Println("warning: could not expand data-dir:", err)
		return
	}

	err = image.RemoveSourceMap(dataDir)
	if err != nil {
		log.Println("warning:", err)
	}

	err = image.Drop(&h)
	if err != nil {
		log.Fatalln("could not remove image:", err)
	}

	log.Printf("OS image %s removed from pool %s", name, poolName)
}
