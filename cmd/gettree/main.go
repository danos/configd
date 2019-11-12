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
	"os"

	client "github.com/danos/configd/client"
	"github.com/danos/configd/rpc"
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
	fmt.Fprintf(os.Stderr, "    %s <path> <encoding:json//rfc7951/xml/internal>\n", os.Args[0])
	os.Exit(1)
}

func getEncoding(encoding string) (string, error) {
	switch encoding {
	case "xml", "json", "rfc7951", "internal":
	default:
		return "",
			fmt.Errorf("Invalid encoding: json, xml, rfc7951 or internal expected")
	}
	return encoding, nil
}

func main() {

	if len(os.Args) != 3 {
		showUsageAndExit()
	}

	encoding, err := getEncoding(os.Args[2])
	handleError(err)

	cl, err := client.Dial("unix", "/run/vyatta/configd/main.sock", "")
	defer cl.Close()
	handleError(err)

	out, err := cl.TreeGetFull(rpc.RUNNING, os.Args[1], encoding)
	handleError(err)
	fmt.Println(out)
	os.Exit(0)
}
