// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"github.com/orgrim/carcass/environment"
	"github.com/orgrim/carcass/hv"
	"github.com/spf13/cobra"
	"log"
)

func init() {
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "show details of all environments")
	rootCmd.AddCommand(listCmd)
}

var (
	listCmd = &cobra.Command{
		Use:   "list [options] [env...]",
		Short: "Show environments",
		Long:  `Display environments with details`,
		Run:   list,
	}
	listAll bool
)

func list(cmd *cobra.Command, args []string) {

	h, err := hv.NewHypervisor(Uri)
	if err != nil {
		log.Fatalln(err)
	}
	defer h.Close()

	if len(args) == 0 {
		nets, err := hv.ListNetworks(h)
		if err != nil {
			log.Fatalln(err)
		}
		for _, net := range nets {
			if listAll {
				args = append(args, net.Name)
				continue
			}
			fmt.Println(net.Name)
		}
	}

	for _, a := range args {
		env, err := environment.Lookup(&h, a)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println(env)
	}
}
