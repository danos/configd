// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014, 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package main

import (
	"flag"
	"fmt"
	"os"

	client "github.com/danos/configd/client"
)

var raw bool

func init() {
	flag.BoolVar(&raw, "raw", false, "Read raw file")
}

func handleError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	var out = ""
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage of cfgread:\n")
		fmt.Fprintf(os.Stderr, "    cfgread [-raw] filename\n")
		os.Exit(1)
	}
	cl, err := client.Dial("unix", "/run/vyatta/configd/main.sock", "")
	defer cl.Close()
	handleError(err)
	if raw {
		out, err = cl.ReadConfigFileRaw(args[0])

	} else {
		out, err = cl.ReadConfigFile(args[0])
	}
	handleError(err)
	fmt.Println(out)
	os.Exit(0)
}
