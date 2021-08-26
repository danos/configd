// Copyright (c) 2017-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests on handling of prefixed and unprefixed path
// elements within when and must statements used in conjunction with the
// 'uses' and 'refine' statements across multiple modules.

package session_test

import (
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

// The following tests verify the interaction of prefixes in when and must
// statements with the refine statement.  Specifically:
//
// - when/must in refine of imported grouping referring to prefixed nodes
//
// - when/must in refine of imported grouping referring to unprefixed nodes
//
// - when/must in original grouping, subsequently imported, referring to
//   original grouping explicitly.
//
// - when/must in original grouping, subsequently imported, referring to
//   unprefixed nodes.
//

// Refine test
//
// Here we are testing that when inside a 'used' imported grouping, and a must
// inside a refine inside the 'uses' statement can correctly reference
// prefixed nodes.
const localRefineSchema = `
container localCont {
	uses remLcl:remGroup {
		when "/remLcl:remCont/remLcl:remLeaf";
		refine remGroupLeaf {
			must "/remLcl:remCont/remLcl:remLeafList = 'foo'" {
				error-message "refLeafList must have element 'foo'";
			}
		}
	}
}`

const remoteRefineSchema = `
grouping remGroup {
	leaf remGroupLeaf {
		type string;
	}
}
container remCont {
	leaf remLeaf {
		type string;
	}
	leaf-list remLeafList {
		type string;
	}
}`

const initRefineGroupingConfig = `remCont {
	remLeaf foo
	remLeafList {
		bar
	}
}
`

const expRefineGroupingConfig = `localCont {
	remGroupLeaf someValue
}
remCont {
	remLeaf foo
	remLeafList bar
	remLeafList foo
}
`

func TestRefineGrouping(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remGroupLeaf - will fail must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewMustCustomError(t,
		"/localCont/remGroupLeaf/someValue",
		"refLeafList must have element 'foo'").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remGroupLeafList so must will pass",
			"remCont/remLeafList/foo", SetPass),
		createValOpTbl("Set remGroupLeaf - will now pass must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	refineTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail,
			initRefineGroupingConfig, testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass,
			expRefineGroupingConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{
					{Namespace: "prefix-remote", Prefix: "remLcl"}},
				SchemaSnippet: localRefineSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: remoteRefineSchema,
			},
		},
		initRefineGroupingConfig, refineTests)
}

// RefineUnprefixed test
//
// Here we are testing that when inside a 'used' imported grouping, and a must
// inside a refine inside the 'uses' statement can correctly reference
// unprefixed nodes.
const localRefineUnprefixedSchema = `
container localCont {
	uses remLcl:remGroup {
		when "/localCont/localLeaf";
		refine remGroupLeaf {
			must "../localLeaf = 'foo'" {
				error-message "localLeaf must have element 'foo'";
			}
		}
	}
	leaf localLeaf {
		type string;
	}
}`

const remoteRefineUnprefixedSchema = `
grouping remGroup {
	leaf remGroupLeaf {
		type string;
	}
}`

const initRefineGroupingUnprefixedConfig = `localCont {
	localLeaf bar
}
`

const expRefineGroupingUnprefixedConfig = `localCont {
	localLeaf foo
	remGroupLeaf someValue
}
`

func TestRefineGroupingUnprefixed(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remGroupLeaf - will fail must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewMustCustomError(t,
		"/localCont/remGroupLeaf/someValue",
		"localLeaf must have element 'foo'").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set localLeaf so must will pass",
			"localCont/localLeaf/foo", SetPass),
		createValOpTbl("Set remGroupLeaf - will now pass must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	refineUnprefixedTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail,
			initRefineGroupingUnprefixedConfig, testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass,
			expRefineGroupingUnprefixedConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{
					{Namespace: "prefix-remote", Prefix: "remLcl"}},
				SchemaSnippet: localRefineUnprefixedSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: remoteRefineUnprefixedSchema,
			},
		},
		initRefineGroupingUnprefixedConfig, refineUnprefixedTests)
}

// WhenMustInRemoteGrouping
//
// Here the must is in the remote grouping we import.  This first
// test has prefixed paths.
const localMustInRemoteGroupingSchema = `
container localCont {
	uses remLcl:remGroup;
}`

const remoteMustInRemoteGroupingSchema = `
grouping remGroup {
	leaf remGroupLeaf {
		type string;
		must "/remote:remCont/remote:remLeafList = 'foo'" {
            error-message "remLeafList should have element 'foo'";
        }
	}
}
container remCont {
	leaf remLeaf {
		type string;
	}
	leaf-list remLeafList {
		type string;
	}
}`

const initMustInRemoteGroupingConfig = `remCont {
	remLeaf foo
	remLeafList {
		bar
	}
}
`

const expMustInRemoteGroupingConfig = `localCont {
	remGroupLeaf someValue
}
remCont {
	remLeaf foo
	remLeafList bar
	remLeafList foo
}
`

func TestMustInRemoteGrouping(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remGroupLeaf - will fail must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewMustCustomError(t,
		"/localCont/remGroupLeaf/someValue",
		"remLeafList should have element 'foo'").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remGroupLeafList so must will pass",
			"remCont/remLeafList/foo", SetPass),
		createValOpTbl("Set remGroupLeaf - will now pass must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	whenMustInRemoteGroupingTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail,
			initRefineGroupingConfig, testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass,
			expRefineGroupingConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{
					{Namespace: "prefix-remote", Prefix: "remLcl"}},
				SchemaSnippet: localMustInRemoteGroupingSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: remoteMustInRemoteGroupingSchema,
			},
		},
		initMustInRemoteGroupingConfig, whenMustInRemoteGroupingTests)
}

// MustInRemoteGroupingUnprefixed
//
// Here the must is in the remote grouping we import.  This second
// test has unprefixed paths.
const localMustInRemoteGroupingUnprefixedSchema = `
container localCont {
	uses remLcl:remGroup;
}`

const remoteMustInRemoteGroupingUnprefixedSchema = `
grouping remGroup {
	leaf remGroupLeaf {
		type string;
		must "../remGroupCont/remGroupContLeaf = 'foo'" {
            error-message "remGroupContLeaf must be 'foo'";
        }
	}
	container remGroupCont {
		leaf remGroupContLeaf {
			type string;
		}
	}
}`

const initMustInRemoteGroupingUnprefixedConfig = `localCont {
	remGroupCont {
		remGroupContLeaf bar
	}
}
`

const expMustInRemoteGroupingUnprefixedConfig = `localCont {
	remGroupCont {
		remGroupContLeaf foo
	}
	remGroupLeaf someValue
}
`

func TestMustInRemoteGroupingUnprefixed(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remGroupLeaf - will fail must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewMustCustomError(t,
		"/localCont/remGroupLeaf/someValue",
		"remGroupContLeaf must be 'foo'").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remGroupContLEaf so must will pass",
			"localCont/remGroupCont/remGroupContLeaf/foo", SetPass),
		createValOpTbl("Set remGroupLeaf - will now pass must",
			"localCont/remGroupLeaf/someValue", SetPass),
	}

	mustInRemoteGroupingUnprefixedTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail,
			initMustInRemoteGroupingUnprefixedConfig, testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass,
			expMustInRemoteGroupingUnprefixedConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{
					{Namespace: "prefix-remote", Prefix: "remLcl"}},
				SchemaSnippet: localMustInRemoteGroupingUnprefixedSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: remoteMustInRemoteGroupingUnprefixedSchema,
			},
		},
		initMustInRemoteGroupingUnprefixedConfig,
		mustInRemoteGroupingUnprefixedTests)
}

// This test case is rather more complex as it was the original scenario that
// uncovered the problem.  It adds to the mix above by augmenting back to the
// original 'remote' (routing in this case) module via 2 further containers in
// 2 further modules.
const refinePolicySchema = `
	container policy;
`

const refinePolicyRouteSchema = `
augment /policy:policy {
	container route;
}`

const refinePolicyPbrSchema = `
grouping rule-pbr {
	uses routing:routing-instance-or-default-leaf {
		refine routing-instance {
			must "(current() = 'default') or " +
				"(/routing:routing" +
				"/routing:routing-instance[routing:instance-name = current()]"+
				"/routing:instance-type = 'vrf')" {
                error-message "Routing instance must be of type VRF";
            }
		}
	}
}

augment /policy:policy/policy-route:route {
	list pbr {
		key tagnode;
		leaf tagnode {
			type string;
		}
		list rule {
			configd:help "Rule number";
			key "tagnode";
			leaf tagnode {
				type string;
			}
			uses rule-pbr; // This should anchor 'routing' prefix to here.
		}
	}
}`

const refineRoutingSchema = `
	typedef routing-instance-name {
	type string;
}
grouping routing-instance-or-default-leaf {
	leaf routing-instance {
		type union {
			type routing-instance-name;
			type enumeration {
				enum default;
			}
		}
		// Refs here are to 'self' so this statement always refers back to
		// /routing/routing-instance/instance-name in this file.
		must "(current() = " +
			"/vyatta-routing-v1:routing/vyatta-routing-v1:routing-instance" +
            "/vyatta-routing-v1:instance-name) " +
			" or (current() = 'default')";
	}
}

container routing {
	list routing-instance {
		key instance-name;
		leaf instance-name {
			type routing-instance-name;
		}
		leaf instance-type {
			type enumeration {
                enum "vrf";
                enum "not-vrf";
            }
		}
	}
}`

const initRefineConfig = `
routing {
	routing-instance blue {
		instance-type vrf
	}
	routing-instance green {
		instance-type vrf
	}
	routing-instance red {
		instance-type not-vrf
	}
}
`

const modifiedRefineConfig = `policy {
	route {
		pbr pbrGrp1 {
			rule R10 {
				routing-instance blue
			}
		}
	}
}` + initRefineConfig

func TestComplexGroupingRefine(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Try to create non-vrf entry.",
			"policy/route/pbr/pbrGrp1/rule/R10/routing-instance/red", SetPass),
	}

	testA_expOut := errtest.NewMustCustomError(t,
		"/policy/route/pbr/pbrGrp1/rule/R10/routing-instance/red",
		"Routing instance must be of type VRF").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Name matches existing interface name.",
			"policy/route/pbr/pbrGrp1/rule/R10/routing-instance/blue", SetPass),
	}

	groupingRefineTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, initRefineConfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, modifiedRefineConfig,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{
					Namespace: "vyatta-policy-pbr-v1",
					Prefix:    "vyatta-policy-pbr-v1"},
				Imports: []NameDef{
					{Namespace: "vyatta-routing-v1", Prefix: "routing"},
					{Namespace: "vyatta-policy-v1", Prefix: "policy"},
					{Namespace: "vyatta-policy-route-v1", Prefix: "policy-route"}},
				SchemaSnippet: refinePolicyPbrSchema,
			},
			{
				Name: NameDef{
					Namespace: "vyatta-routing-v1", Prefix: "vyatta-routing-v1"},
				SchemaSnippet: refineRoutingSchema,
			},
			{
				Name: NameDef{
					Namespace: "vyatta-policy-v1", Prefix: "vyatta-policy-v1"},
				SchemaSnippet: refinePolicySchema,
			},
			{
				Name: NameDef{Namespace: "vyatta-policy-route-v1",
					Prefix: "vyatta-policy-route-v1"},
				Imports: []NameDef{
					{Namespace: "vyatta-policy-v1", Prefix: "policy"}},
				SchemaSnippet: refinePolicyRouteSchema,
			},
		},
		initRefineConfig, groupingRefineTests)
}
