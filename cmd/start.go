// Copyright 2021 Nicolas Thauvin. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/orgrim/carcass/environment"
	"github.com/orgrim/carcass/hv"
	"github.com/spf13/cobra"
	"log"
)

func init() {
	rootCmd.AddCommand(startCmd)

	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "stop all environments")
	stopCmd.Flags().BoolVarP(&forceStop, "force", "f", false, "send shutdown request even if domain is not active")
	rootCmd.AddCommand(stopCmd)
}

var (
	startCmd = &cobra.Command{
		Use:   "start env [env...]",
		Short: "Start environment",
		Long:  "Start all VM of the environment",
		Run:   start,
	}

	stopCmd = &cobra.Command{
		Use:   "stop env [env...]",
		Short: "Stop environment",
		Long:  "Stop all VM of the environment",
		Run:   stop,
	}

	stopAll   bool
	forceStop bool
)

func start(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		log.Fatalln("missing environment")
	}

	h, err := hv.NewHypervisor(Uri)
	if err != nil {
		log.Fatalln(err)
	}
	defer h.Close()

	for _, e := range args {
		env, err := environment.Lookup(&h, e)
		if err != nil {
			log.Println(err)
			continue
		}
		env.Start()
	}
}

func stop(cmd *cobra.Command, args []string) {
	h, err := hv.NewHypervisor(Uri)
	if err != nil {
		log.Fatalln(err)
	}
	defer h.Close()

	envs := make([]string, 0)
	if stopAll {
		nets, err := hv.ListNetworks(h)
		if err != nil {
			log.Fatalln(err)
		}
		for _, net := range nets {
			envs = append(envs, net.Name)
		}
	} else {
		envs = args
	}

	if len(envs) == 0 {
		log.Fatalln("missing environment")
	}

	for _, e := range envs {
		env, err := environment.Lookup(&h, e)
		if err != nil {
			log.Println(err)
			continue
		}
		env.Stop(forceStop)
	}
}
