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
	"os"
	"path/filepath"
	"strings"

	"github.com/danos/config/platform"
	"github.com/danos/config/schema"
	"github.com/danos/yang/compile"
	"github.com/danos/yang/xpath"
	"github.com/danos/yang/xpath/xutils"
	"github.com/go-ini/ini"
)

var skip bool
var showNPContainerWarnings bool
var capabilities string
var fullSchema bool
var customXpathFunctions string
var resourcesDir string
var testPlatform string

func usage() {
	_, file := filepath.Split(os.Args[0])
	fmt.Fprintf(os.Stderr, "Usage of %s [flags] [<yang-dir>]:\n\n", file)

	flag.PrintDefaults()

	usageInfo := `
  This utility parses YANG files, checking for any errors.  It has 2 different
  sets of mutually-exclusive parameters for this:

  (a) Simple YANG and feature validation

      Use '-capabilities' (features) and <yang-dir>

  (b) Specific platform validation using 'resources' directory created by
      DRAM or get_iso_yang utilities.

      Use '-resources-dir' (rqd) and '-platform' (optional).  If no platform
      is specified, the 'base' platform will be used.

`
	fmt.Fprintf(os.Stderr, usageInfo)

}

func init() {
	flag.BoolVar(&skip, "i", false, "Ignore unknown symbols")
	flag.StringVar(&capabilities, "capabilities", "",
		"File specifying system capabilities/features")
	flag.BoolVar(&fullSchema, "full", true, "Parse full schema (inc state)")
	flag.BoolVar(&showNPContainerWarnings, "show-np-cont-warnings", false,
		"Show warnings relating to must statements on NP containers")
	flag.StringVar(&customXpathFunctions, "custom-xpath-functions", "",
		"Additional custom XPATH functions")
	flag.StringVar(&resourcesDir, "resources-directory", "",
		"Resources Directory (eg for DRAM)")
	flag.StringVar(&testPlatform, "platform", "",
		"Specific platform to test (default is BASE platform)")
}

const (
	featuresSubdir           = "features"
	vyattaPlatformSubdir     = "vyatta-platform"
	vyattaPlatformYangSubdir = vyattaPlatformSubdir + "/yang"
	xpathPluginDir           = "/lib/xpath/plugins"
)

// getCustomFunctionList - merge functions from INI files with user's list
func getValidCustomFunctionList(customXpathFunctions string) []string {

	functions := getFunctionsFromIniFiles()

	for _, function := range strings.Split(customXpathFunctions, ",") {
		if function != "" {
			functions = append(functions, function)
		}
	}

	return functions
}

func getFunctionsFromIniFiles() []string {
	var functions []string

	files, _ := filepath.Glob(filepath.Join(xpathPluginDir, "*.ini"))
	for _, file := range files {
		iniFile, err := ini.Load(file)
		if err != nil {
			// Simply ignore problematic files.  Best effort here.
			continue
		}
		for _, section := range iniFile.Sections() {
			functions = append(functions, section.Name())
		}
	}

	return functions
}

// getPlatform returns the relevant platform definition
//
// Base platform (empty definition) is returned if the platform name is
// the empty string.
func getPlatform(dir, platName string) (*platform.Definition, error) {

	// Either YangDir specified, or resources with no platform specified.
	// Return the 'base' platform definition in this case.
	if platName == "" {
		return &platform.Definition{}, nil
	}

	// Get platform definitions
	platDir := filepath.Join(dir, vyattaPlatformSubdir)
	plats := platform.NewPlatform().
		PlatformBaseDir(platDir).
		LoadDefinitions()

	// Find specific platform
	plat, ok := plats.Platforms[platName]
	if !ok {
		return nil, fmt.Errorf("Unable to find definition for platform '%s'",
			platName)
	}
	return plat, nil

}

func createYangLocator(
	args []string,
	resources string,
	plat *platform.Definition,
) compile.YangLocator {

	if len(args) > 0 {
		return compile.YangDirs(args[0])
	}

	files := make([]string, 0)
	for _, file := range plat.Yang {
		yangFile := filepath.Join(resources, vyattaPlatformYangSubdir, file)
		files = append(files, yangFile)
	}
	return compile.YangLocations(
		compile.YangDirs(filepath.Join(resources, "yang")),
		compile.YangFiles(files...))
}

func createFeatureChecker(
	caps,
	resources string,
	plat *platform.Definition,
) compile.FeaturesChecker {

	// In basic case, we have a single directory of features, specified by
	// <caps>
	if caps != "" {
		return compile.FeaturesFromLocations(true, caps)
	}

	// If we have a resources directory, features consist of:
	//   <resources>/features
	//     - default enabled features
	//     - user-defined extra enabled features
	//   platform-specific features (enabled and disabled)
	return compile.MultiFeatureCheckers(
		compile.FeaturesFromLocations(
			true, filepath.Join(resourcesDir, featuresSubdir)),
		compile.FeaturesFromNames(true, plat.Features...),
		compile.FeaturesFromNames(false, plat.DisabledFeatures...))
}

// validateArgs - check params are valid, mutexes honoured ...
//
// 	 YangDir  caps  resources platform | Notes
// 1    n    	n	    n        n     | Invalid (no YANG specified)
// 2    n    	n	    n        y     | Invalid (platform w/o resources)
// 3    n    	n	    y        n     | Base Platform with resources
// 4    n    	n	    y        y     | Platform with resources
// 5    n    	y	    n        n     | Invalid (no YangDir specified)
// 6    n    	y	    n        y     | Invalid (platform w/o resources)
// 7    n    	y	    y        n     | Invalid (caps + resources not allowed)
// 8    n    	y	    y        y     | Invalid (caps + resources not allowed)
// 9    y    	n	    n        n     | Base Platform with YangDir
// 10   y    	n	    n        y     | Invalid (plat not valid with YangDir)
// 11   y    	n	    y        n     | Invalid (two sources of YANG)
// 12   y    	n	    y        y     | Invalid (two sources of YANG)
// 13   y    	y	    n        n     | Base Platform with YangDir
// 14   y    	y	    n        y     | Invalid (plat w/o resources)
// 15   y    	y	    y        n     | Invalid (two sources of YANG)
// 16   y    	y	    y        y     | Invalid (two sources of YANG)
func validateArgs(args []string) bool {

	// Exactly one source of YANG
	if (len(args) == 0) == (resourcesDir == "") {
		return false
	}

	// Capabilities only with YangDir not Resources
	if capabilities != "" && resourcesDir != "" {
		return false
	}

	// Platform can only be non-base with Resources
	if testPlatform != "" && resourcesDir == "" {
		return false
	}

	return true
}

func createUserFnChecker(
	customFns string,
) func(name string) (*xpath.Symbol, bool) {

	return func(name string) (*xpath.Symbol, bool) {
		allowedFns := getValidCustomFunctionList(customFns)
		for _, fn := range allowedFns {
			if fn == name {
				return xpath.NewDummyFnSym(name), true
			}
		}
		return nil, false
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if !validateArgs(args) {
		usage()
		os.Exit(1)
	}

	plat, err := getPlatform(resourcesDir, testPlatform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	yangLocator := createYangLocator(args, resourcesDir, plat)

	featChecker := createFeatureChecker(capabilities, resourcesDir, plat)

	userFnChecker := createUserFnChecker(customXpathFunctions)

	_, warns, err := schema.CompileDirWithWarnings(
		&compile.Config{
			YangLocations: yangLocator,
			Features:      featChecker,
			SkipUnknown:   skip,
			Filter:        compile.IsConfigOrState(),
			UserFnCheckFn: userFnChecker},
		nil)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if !showNPContainerWarnings {
		warns = xutils.RemoveNPContainerWarnings(warns)
	}

	if len(warns) != 0 {
		fmt.Fprintf(os.Stdout, "\nXPATH path validation warnings:\n\n")
		for _, warn := range warns {
			fmt.Fprintf(os.Stdout, "%s\n----\n", warn)
		}
		os.Exit(1)
	}
	os.Exit(0)
}
