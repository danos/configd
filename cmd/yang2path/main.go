// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	cfgSchema "github.com/danos/config/schema"
	"github.com/danos/config/yangconfig"
	"github.com/danos/utils/natsort"
	"github.com/danos/yang/compile"
	"github.com/danos/yang/parse"
	"github.com/danos/yang/schema"
)

type ByName []cfgSchema.Node

func (b ByName) Len() int           { return len(b) }
func (b ByName) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByName) Less(i, j int) bool { return natsort.Less(b[i].Name(), b[j].Name()) }

var directory string
var capabilities string
var systemcfg bool

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [list of files to evaluate]\n", os.Args[0])
	}
	flag.StringVar(&directory, "dir",
		"",
		"Directory containing YANG files")
	flag.StringVar(&capabilities, "capabilities",
		compile.DefaultCapsLocation,
		"File specifying system capabilities")
	flag.BoolVar(&systemcfg, "system",
		false,
		"Use system Yang and Capabilities")
}

func isElemOf(list []string, elem string) bool {
	for _, v := range list {
		if v == elem {
			return true
		}
	}
	return false
}

func appendPathStr(path, elem string) string {
	if path == "" {
		return elem
	}
	return path + " " + elem
}

func walkChildren(n cfgSchema.Node, path string) {
	walkChildrenSkip(n, path, []string{})
}

func walkChildrenSkip(n cfgSchema.Node, path string, skiplist []string) {
	chs := n.Children()
	children := make([]cfgSchema.Node, len(chs))
	for i, ch := range chs {
		children[i] = ch.(cfgSchema.Node)
	}
	sort.Sort(ByName(children))
	for _, v := range children {
		if isElemOf(skiplist, v.Name()) {
			continue
		}
		walk(v, path)
	}
}

func handleHelp(n interface {
	HelpMap() map[string]string
}, path string) string {
	help := n.HelpMap()
	if len(help) == 0 {
		path = appendPathStr(path, "<value>")
	} else {
		vs := make([]string, 0, len(help))
		for v, _ := range help {
			vs = append(vs, v)
		}
		sort.Strings(vs)
		out := strings.Join(vs, "|")
		if len(vs) > 1 {
			out = "(" + out + ")"
		}
		path = appendPathStr(path, out)
	}
	return path
}

func handleLeaf(n cfgSchema.Leaf, path string) {
	path = appendPathStr(path, n.Name())
	typ := n.Type()
	if _, ok := typ.(schema.Empty); !ok {
		path = handleHelp(n, path)
	}
	fmt.Println(path)
}

func walk(n cfgSchema.Node, path string) {
	switch sn := n.(type) {
	case cfgSchema.List:
		path = appendPathStr(path, n.Name())
		path = handleHelp(n, path)
		fmt.Println(path)
		walkChildrenSkip(n, path, sn.Keys())
	case cfgSchema.Leaf:
		handleLeaf(sn, path)
	case cfgSchema.LeafList:
		path = appendPathStr(path, n.Name())
		path = handleHelp(n, path)
		fmt.Println(path)
	case cfgSchema.Container:
		path = appendPathStr(path, n.Name())
		if sn.HasPresence() {
			fmt.Println(path)
		}
		walkChildren(n, path)
	default:
		walkChildren(n, path)
	}
}

func processStdin() {
	fname := "stdin"
	data, err := ioutil.ReadAll(os.Stdin)
	if err != io.EOF {
		handleError(err)
	}

	t, err := parse.Parse(fname, string(data), nil)
	handleError(err)

	stree, err := cfgSchema.CompileModules(map[string]*parse.Tree{fname: t}, "", true, compile.IsConfig, nil)
	handleError(err)

	walk(stree, "")
	os.Exit(0)
}

func handleError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()
	args := flag.Args()

	if systemcfg || directory != "" {

		ycfg := yangconfig.NewConfig()
		if systemcfg {
			ycfg = ycfg.SystemConfig()
		} else {
			ycfg = ycfg.IncludeYangDirs(directory).
				IncludeFeatures(capabilities)
		}
		stree, err := cfgSchema.CompileDir(
			&compile.Config{
				YangLocations: ycfg.YangLocator(),
				Features:      ycfg.FeaturesChecker(),
				Filter:        compile.IsConfig},
			nil)
		handleError(err)
		walk(stree, "")
		os.Exit(0)
	}

	if len(args) == 0 {
		processStdin()
	}

	mods, err := cfgSchema.ParseModules(args...)
	handleError(err)

	stree, err := cfgSchema.CompileModules(mods, "", true, compile.IsConfig, nil)
	handleError(err)

	walk(stree, "")
	os.Exit(0)
}
