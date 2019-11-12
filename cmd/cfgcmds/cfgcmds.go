// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/danos/config/parse"
	"github.com/danos/utils/pathutil"
)

const (
	specialChars    = "#\\&><!|*}{)(][:?^;"
	whitespaceChars = "\011\012\013\014\015 "
	strongChars     = "$\""
	weakChars       = "'"
)

func quote(in string) string {
	hasSpecial := strings.ContainsAny(in,
		strongChars+weakChars+specialChars+whitespaceChars)
	needsStrongQuote := strings.ContainsAny(in, strongChars)
	needsWeakQuote := strings.Contains(in, weakChars)
	switch {
	case needsStrongQuote && needsWeakQuote:
		out := strings.Replace(in, "'", "\\'", -1)
		return "$'" + out + "'"
	case needsStrongQuote:
		return "'" + in + "'"
	case needsWeakQuote:
		return "\"" + in + "\""
	case hasSpecial:
		return "'" + in + "'"
	default:
		return in
	}
}

func buildpath(n *parse.Node, path []string) []string {
	if n.HasArg {
		return append(path, n.Id, quote(n.Arg))
	} else {
		return append(path, n.Id)
	}
}

func buildpaths(n *parse.Node, path []string) [][]string {
	var paths [][]string
	path = buildpath(n, path)
	if len(n.Children) == 0 {
		paths = append(paths, pathutil.Copypath(path))
	}
	for _, ch := range n.Children {
		paths = append(paths, buildpaths(ch, path)...)
	}
	return paths
}

func main() {
	if len(os.Args) > 2 {
		fmt.Fprintln(os.Stderr, "usage: cfgcmds [filename]")
		fmt.Fprintln(os.Stderr, "    cfgcmds takes a file in the vyatta config format")
		fmt.Fprintln(os.Stderr, "    and prints the structure as set commands,")
		fmt.Fprintln(os.Stderr, "    by default it will read from stdin.")
		os.Exit(1)
	}

	file := os.Stdin
	filename := "stdin"
	if len(os.Args) == 2 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		file = f
		filename = os.Args[1]
	}

	text, err := ioutil.ReadAll(file)
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	t, err := parse.Parse(filename, string(text))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var paths [][]string
	for _, ch := range t.Root.Children {
		paths = append(paths, buildpaths(ch, []string{})...)
	}
	for _, path := range paths {
		fmt.Println("set", strings.Join(path, " "))
	}
}
