// Copyright (c) 2020, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodeArgs(t *testing.T) {
	args := []string{"foo", "bar", "baz"}
	str := encodeArgs(args)

	var out argsObject
	dec := json.NewDecoder(strings.NewReader(str))
	dec.Decode(&out)

	compareLists := func(l1, l2 []string) bool {
		if len(l1) != len(l2) {
			return false
		}
		for i, v := range l1 {
			if l2[i] != v {
				return false
			}
		}
		return true
	}
	if !compareLists(out.Args, args) {
		t.Fatalf("got %q, expected %q\n", out.Args, args)
	}
}
