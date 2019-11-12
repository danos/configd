// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/danos/yang/compile"
)

var writeDir string
var write bool

func init() {
	flag.StringVar(&writeDir, "d", "", "Directory to write revision files into")
	flag.BoolVar(&write, "w", false, "Write revision files")
}

func handleError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func writeRevs(mods []string) {
	var prefix = ""
	if writeDir != "" {
		handleError(os.MkdirAll(writeDir, os.ModePerm))
		prefix = writeDir + "/"
	}
	for _, m := range mods {
		f, err := os.Create(fmt.Sprintf("%s%s", prefix, m))
		handleError(err)
		f.Close()
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	st, err := compile.CompileDir(nil, &compile.Config{YangDir: args[0], Filter: compile.IsConfig})
	handleError(err)
	mods := make([]string, 0, len(st.Modules()))
	for _, m := range st.Modules() {
		mods = append(mods, fmt.Sprintf("%s@%s", m.Identifier(), m.Version()))
	}
	if write {
		writeRevs(mods)
		os.Exit(0)
	}
	fmt.Printf("/* === vyatta-config-version: \"%s\" === */\n", strings.Join(mods, ":"))
	os.Exit(0)
}
