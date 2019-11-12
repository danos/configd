// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	client "github.com/danos/configd/client"
)

var spath string
var ctxdiff bool
var socketpath string

func init() {
	flag.StringVar(
		&spath,
		"spath",
		"",
		"Path at which comparison starts",
	)

	flag.BoolVar(
		&ctxdiff,
		"ctxdiff",
		false,
		"Show contextual differences",
	)
	flag.StringVar(
		&socketpath,
		"socket",
		"/run/vyatta/configd/main.sock",
		"Path to the socket we should write to",
	)
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] file1 file2\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 || len(args) > 3 {
		flag.Usage()
		os.Exit(1)
	}

	files := []string{args[0], args[1]}
	data := make([]string, len(files))
	for i, file := range files {
		f, err := os.Open(file)
		if err != nil {
			fatal(err)
		}
		out, err := ioutil.ReadAll(f)
		if err != nil {
			fatal(err)
		}
		data[i] = string(out)
		f.Close()
	}

	cl, err := client.Dial("unix", socketpath,
		os.ExpandEnv("$VYATTA_CONFIG_SID"))
	out, err := cl.Compare(data[0], data[1], spath, ctxdiff)
	if err != nil {
		fatal(err)
	}

	fmt.Print(out)
}
