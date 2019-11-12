// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"testing"
)

/*
 * USAGE:
 *   ln -s <your-yang-dir> yang
 *   go test -cpuprofile=cprof -memprofile=mprof
 *   go tool pprof --text yangc.test mprof
 *   go tool pprof --text yangc.test cprof
 */

func TestCompile(t *testing.T) {
	t.Skip("compileYangDir does not compile")
	for i := 0; i < 10; i++ {
		compileYangDir()
	}
}

// This is currently broken
func compileYangDir() {
	// _, err := compile.CompileDir(&compile.Config{
	// 	YangDir:      "yang",
	// 	CapsLocation: "",
	// 	SkipUnknown:  false,
	// 	Filter:   compile.ConfigAndState(),
	// })
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// }
}

type validateArgsTestCase struct {
	desc      string
	args      []string
	features, // aka capabilities
	resources,
	platform string
	valid bool
}

const (
	SET     = "set to some value"
	NOT_SET = ""
)

func TestValidateArgs(t *testing.T) {
	testCases := []validateArgsTestCase{
		{
			desc:  "1: Invalid - no source of YANG specified",
			valid: false,
		},
		{
			desc:     "2: Invalid - platform w/o resources",
			platform: SET,
			valid:    false,
		},
		{
			desc:      "3: Valid - base platform with resources",
			resources: SET,
			valid:     true,
		},
		{
			desc:      "4: Valid - non-base platform with resources",
			resources: SET,
			platform:  SET,
			valid:     true,
		},
		{
			desc:     "5: Invalid - no source of YANG specified",
			features: SET,
			valid:    false,
		},
		{
			desc:     "6: Invalid - platform w/o resources",
			features: SET,
			platform: SET,
			valid:    false,
		},
		{
			desc:      "7: Invalid - features not allowed with resources",
			features:  SET,
			resources: SET,
			valid:     false,
		},
		{
			desc:      "8: Invalid - features not allowed with resources",
			features:  SET,
			resources: SET,
			platform:  SET,
			valid:     false,
		},
		{
			desc:  "9: Base platform with YangDir",
			args:  []string{SET},
			valid: true,
		},
		{
			desc:     "10: Invalid - platform w/o resources",
			args:     []string{SET},
			platform: SET,
			valid:    false,
		},
		{
			desc:      "11: Invalid - two sources of YANG",
			args:      []string{SET},
			resources: SET,
			valid:     false,
		},
		{
			desc:      "12: Invalid - two sources of YANG",
			args:      []string{SET},
			resources: SET,
			platform:  SET,
			valid:     false,
		},
		{
			desc:     "13: Valid - base platform with YangDir and features",
			args:     []string{SET},
			features: SET,
			valid:    true,
		},
		{
			desc:     "14: Invalid - platform w/o resources",
			args:     []string{SET},
			features: SET,
			platform: SET,
			valid:    false,
		},
		{
			desc:      "15: Invalid - two sources of YANG",
			args:      []string{SET},
			features:  SET,
			resources: SET,
			valid:     false,
		},
		{
			desc:      "16: Invalid - two sources of YANG",
			args:      []string{SET},
			features:  SET,
			resources: SET,
			platform:  SET,
			valid:     false,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			capabilities = test.features
			resourcesDir = test.resources
			testPlatform = test.platform

			actValid := validateArgs(test.args)

			if actValid != test.valid {
				t.Fatalf("%s: Expected %t, got %t", test.desc,
					test.valid, actValid)
			}
		})
	}
}
