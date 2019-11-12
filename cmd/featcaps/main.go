// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

/*
 featcaps is a program to manage the yang feature capabilities, intended
 to be invoked by the secret config mode commands enable_feature and
 disable_feature

 The existence of a file <capsDirectory>/module_name/feature_name indicates
 that the feature is enabled. Disabling a feature is simply a matter of
 deleting the file <capsDirectory>/module_name/feature_name

 Usage:
   -capabilities <capsDirectory>
	Directory containing the enabled feature definitions
	default is /config/features

   -disable
	Disable the feature instead of enabling the feature

   <featureName>
	A feature formatted as yang_module_name:feature_name
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/danos/yang/compile"
)

var capabilities string
var disable bool

func init() {
	flag.StringVar(&capabilities, "capabilities", compile.DefaultCapsLocation, "Location of system capabilities")
	flag.BoolVar(&disable, "disable", false, "")

}

// Create a file, within the given directory, to indicate that a yang feature
// is enabled
func enableFeature(moduleDir, featurePath string) {
	// Yang module level directory does not exist, create it
	if err := os.MkdirAll(moduleDir, os.ModePerm); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if _, err := os.Stat(featurePath); err != nil {
		os.Create(featurePath)
	}
}

// delete a file, which has been acting as an indicator that a yang feature
// has been enabled. If the deletion results in the containing directory
// being empty, delete the directory
func disableFeature(moduleDir, featurePath string) {
	os.RemoveAll(featurePath)

	fi, err := os.Open(moduleDir)

	if err != nil {
		return
	}
	if _, err = fi.Readdirnames(1); err == io.EOF {
		// Feature directory is now empty, delete it
		os.RemoveAll(moduleDir)
	}
}

func main() {
	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		// only one argument allowed
		return
	}
	feature := strings.Split(args[0], ":")
	if len(feature) != 2 {
		// Badly formatted feature
		return
	}

	moduleDir := capabilities + "/" + feature[0]
	featurePath := moduleDir + "/" + feature[1]
	if disable {
		disableFeature(moduleDir, featurePath)
	} else {
		enableFeature(moduleDir, featurePath)
	}
	os.Exit(0)
}
