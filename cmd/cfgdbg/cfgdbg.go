// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danos/configd/client"
)

var logType string
var logLevel string

func usage() {
	_, file := filepath.Split(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage of %s [flags]:\n\n", file)

	flag.PrintDefaults()

	usageInfo := `
  This utility allows users to view and change configd debug/log settings.
  Run with no params to view settings (and to see valid options)
`
	fmt.Fprintf(os.Stderr, usageInfo)

}

func init() {
	flag.StringVar(&logType, "log-type", "",
		"Name of debug/log to set")
	flag.StringVar(&logLevel, "log-level", "",
		"Log level")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	cl, err := client.Dial("unix", "/run/vyatta/configd/main.sock",
		os.ExpandEnv("$VYATTA_CONFIG_SID"))

	out, err := cl.SetConfigDebug(logType, logLevel)
	if logType == "" && logLevel == "" {
		fmt.Fprintf(os.Stdout, "%s\n", out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(0)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "%s\n", out)
	os.Exit(0)
}
