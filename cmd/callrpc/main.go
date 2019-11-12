// Copyright (c) 2017,2019, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2015 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	client "github.com/danos/configd/client"
)

func handleError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

func showUsageAndExit() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    %s <namespace> <rpc-name> <encoding:json/rfc7951/xml> [<input json/rfc7951/xml>]\n", os.Args[0])
	os.Exit(1)
}

func getEncoding(encoding string) (string, error) {
	switch encoding {
	case "xml":
	case "json":
	case "rfc7951":
	default:
		return "", fmt.Errorf("Invalid encoding. json, rfc7951 or xml allowed")
	}
	return encoding, nil
}

func main() {
	var inputArgs string
	var getInput func() string

	switch len(os.Args) {
	case 4:
		// Delay processing stdin until the args have been checked
		getInput = func() string {
			in, err := ioutil.ReadAll(os.Stdin)
			if err != nil && err != io.EOF {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return string(in)
		}
	case 5:
		getInput = func() string { return os.Args[4] }
	default:
		showUsageAndExit()
	}

	encoding, err := getEncoding(os.Args[3])
	handleError(err)

	inputArgs = getInput()
	ns := os.Args[1]
	rpc := os.Args[2]

	var out string
	cl, err := client.Dial("unix", "/run/vyatta/configd/main.sock", "")
	defer cl.Close()
	handleError(err)

	out, err = cl.CallRpc(ns, rpc, inputArgs, encoding)
	handleError(err)

	fmt.Println(out)
	os.Exit(0)
}
