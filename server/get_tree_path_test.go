// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

// The aim of this set of tests is to verify GetTree and GetTreeFull handle
// paths that *could* exist, but don't actually exist.  This is needed to
// ensure that we don't return errors to NETCONF wrongly when a node is valid
// but not configured / has no current state value.  We still return an
// error for XML for now to maintain the existing API behaviour.

import (
	"testing"

	"github.com/danos/configd/rpc"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
	"github.com/danos/yang/schema"
)

func verifyValidPathFullTree(
	t *testing.T, testSchema string, nspec schema.NodeSpec, encoding string,
) {
	verifyPathInternal(t, testSchema, nspec, encoding, true, false)
}

func verifyInvalidPathFullTree(
	t *testing.T, testSchema string, nspec schema.NodeSpec, encoding string,
) {
	verifyPathInternal(t, testSchema, nspec, encoding, false, false)
}

func verifyValidPathConfigOnly(
	t *testing.T, testSchema string, nspec schema.NodeSpec, encoding string,
) {
	verifyPathInternal(t, testSchema, nspec, encoding, true, true)
}

func verifyInvalidPathConfigOnly(
	t *testing.T, testSchema string, nspec schema.NodeSpec, encoding string,
) {
	verifyPathInternal(t, testSchema, nspec, encoding, false, true)
}

func verifyPathInternal(
	t *testing.T,
	testSchema string,
	nspec schema.NodeSpec,
	encoding string,
	pathIsValid bool,
	configOnly bool,
) {
	opts := make(map[string]interface{})
	opts["Defaults"] = true

	testSchemas := genTestSchema(schemaTemplate, testSchema)
	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(testSchemas).
			SetConfig(emptyConfig))

	var err error
	if configOnly {
		_, err = d.TreeGet(rpc.RUNNING, "sid",
			pathutil.Pathstr(nspec.Path),
			encoding,
			opts)
	} else {
		_, err, _ = d.TreeGetFullWithWarnings(rpc.RUNNING, "sid",
			pathutil.Pathstr(nspec.Path),
			encoding,
			opts)
	}

	if pathIsValid {
		if err != nil {
			t.Fatalf("Unable to verify path: %s\n", err)
			return
		}
		return
	}

	if err == nil {
		t.Fatalf("Expected path to be invalid.\n")
	}
}

type pathExistsTest struct {
	name,
	path string
}

const testSchema = `
container topPresence {
	presence true;
}
container top {
	leaf topLeaf {
		type string {
			pattern "[a-z]*";
		}
	}
	list outerList {
		key outerName;
		leaf outerName {
			type string;
		}
		leaf-list outerLL {
			type string;
		}
		leaf outerLeaf {
			type string;
		}
		list innerList {
			key innerName;
			leaf innerName {
				type string;
			}
			leaf innerLeaf {
				type string;
			}
		}
	}
	container state {
		config false;
		list stateList {
			key stateKey;
			leaf stateKey {
				type string;
			}
			leaf stateLeafInList {
				type string;
			}
		}
		leaf stateLeaf {
			type string;
		}
	}
}`

const schemaTemplate = `%s`

// For NETCONF, we need to return no error if a requested path could exist,
// but currently doesn't.  Paths for GetTree / GetTreeFull need to be in
// configd format, ie list entry name inserted as 'parent' of nodes that are
// siblings in YANG (eg interface address etc).
func TestPathsThatCouldExist(t *testing.T) {

	tests := []pathExistsTest{
		{
			name: "container (NP, no content)",
			path: "/top",
		},
		{
			name: "container leaf (no value)",
			path: "/top/topLeaf",
		},
		{
			name: "container leaf (valid value matches pattern stmt)",
			path: "/top/topLeaf/aaa",
		},
		{
			name: "container",
			path: "/topPresence",
		},
		{
			name: "List name - no value",
			path: "/top/outerList",
		},
		{
			name: "List key",
			path: "/top/outerList/listEntry",
		},
		{
			name: "List leaf (no value)",
			path: "/top/outerList/listEntry/outerLeaf",
		},
		{
			name: "List leaf",
			path: "/top/outerList/listEntry/outerLeaf/value",
		},
		{
			name: "List leaf-list (no value)",
			path: "/top/outerList/listEntry/outerLL",
		},
		{
			name: "List leaf-list",
			path: "/top/outerList/listEntry/outerLL/value",
		},
		{
			name: "Nested list (no key)",
			path: "/top/outerList/listEntry/innerList",
		},
		{
			name: "Nested list key",
			path: "/top/outerList/listEntry/innerList/listKey",
		},
		{
			name: "Nested list leaf (no value)",
			path: "/top/outerList/listEntry/innerList/listKey/innerLeaf",
		},
		{
			name: "Nested list leaf",
			path: "/top/outerList/listEntry/innerList/listKey/innerLeaf/value",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifyValidPathFullTree(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"netconf")
			verifyValidPathConfigOnly(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"netconf")

			verifyInvalidPathFullTree(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"xml")
			verifyInvalidPathConfigOnly(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"xml")
		})
	}
}

func TestPathsThatCantExist(t *testing.T) {

	tests := []pathExistsTest{
		{
			name: "container leaf (invalid value)",
			path: "/top/topLeaf/AAA",
		},
		{
			name: "Totally non-existent",
			path: "/nonexistent",
		},
		{
			name: "Partially non-existent",
			path: "/top/nonexistent",
		},
		{
			name: "List leaf value, no key",
			path: "/top/outerList/outerLeaf/leafVal1",
		},
		{
			name: "List leaf-list value, no key",
			path: "/top/outerList/outerLL/llVal1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			verifyInvalidPathFullTree(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"netconf")
			verifyInvalidPathFullTree(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"xml")
		})
	}
}

func TestStatePathsThatCouldExist(t *testing.T) {

	tests := []pathExistsTest{
		{
			name: "State leaf",
			path: "/top/state/stateLeaf",
		},
		{
			name: "State container",
			path: "/top/state",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			verifyValidPathFullTree(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"netconf")
			verifyInvalidPathConfigOnly(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"netconf")

			verifyInvalidPathFullTree(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"xml")
			verifyInvalidPathConfigOnly(t, testSchema,
				schema.NodeSpec{Path: pathutil.Makepath(test.path)},
				"xml")
		})
	}
}
