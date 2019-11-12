// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"bufio"
	"fmt"
	"os"
)

type normalizationFn func(string) string
type normalizationType struct {
	name     string
	help     string
	function normalizationFn
}

var typeTable = []normalizationType{
	{"legacy", "Best effort matching of type", NormalizeString},
	{"mac", "MAC address", NormalizeMac},
	{"ip", "IPv4 or IPv6 address or CIDR", NormalizeIP},
	{"ipv4", "IPv4 address or CIDR", NormalizeIPv4},
	{"ipv6", "IPv6 address or CIDR", NormalizeIPv6},
	{"ipv4-prefix", "IPv4 prefix", NormalizeIPv4prefix},
	{"ipv6-prefix", "IPv6 prefix", NormalizeIPv6prefix},
	{"ip-prefix", "IP prefix", NormalizeIPprefix},
	{"neg-ipv4", "IPv4 address or CIDR", NormalizeNegIPv4},
	{"neg-ipv6", "IPv6 address or CIDR", NormalizeNegIPv6},
	{"neg-ipv4-prefix", "IPv4 prefix", NormalizeNegIPv4prefix},
	{"neg-ipv6-prefix", "IPv6 prefix", NormalizeNegIPv6prefix},
	{"neg-ip-prefix", "IP prefix", NormalizeNegIPprefix},
}

func showUsageAndExit() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "    %s <type>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  Supported types\n")
	for _, v := range typeTable {
		fmt.Fprintf(os.Stderr, "    %12s - %s\n", v.name, v.help)
	}
	os.Exit(1)
}

func getNormalizeFn() normalizationFn {

	if len(os.Args) != 2 {
		showUsageAndExit()
	}

	request := os.Args[1]

	for _, v := range typeTable {
		if v.name == request {
			return v.function
		}
	}

	showUsageAndExit()
	return nil
}

func main() {
	normalize_fn := getNormalizeFn()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		output := normalize_fn(input)
		fmt.Println(output)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}
