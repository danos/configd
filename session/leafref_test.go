// Copyright (c) 2017-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains session-level tests for the 'leafref' and 'path' YANG
// statements.
//
// tab-completion is outwith the scope of the session object, and is tested
// by the dispatcher tests.  Tests here verify that known good / bad leafref
// references are correctly identified and handled by the 'validate' logic.

package session_test

import (
	"testing"

	. "github.com/danos/config/testutils"
	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

// These schemas are lifted from RFC 6020 section 9.9 (leafref), albeit
// with a couple of types changed to string, and the default-address
// container is moved to a different module to allow for prefix testing.
const interfaceSchema = `
container intfCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		list address {
			key "ip";
			leaf ip {
				type string;
			}
		}
	}

    leaf mgmt-interface {
		type leafref {
			path "../interface/name";
		}
	}
}`

// Set non-existent value (pass) then commit (fail)
func TestSetAndCommitInvalidRef(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leafref with non-existent reference",
			"intfCont/mgmt-interface/dp0s2", SetPass),
	}

	// Error should indicate leafref is not a valid value.
	test_expOut := errtest.NewLeafrefError(t,
		"/intfCont/mgmt-interface/dp0s2",
		"intfCont/interface/name/dp0s2").
		RawErrorStrings()

	setCommitInvalidTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, interfaceSchema, emptyconfig,
		setCommitInvalidTests)
}

// Set non-existent value (pass) then commit (fail) with 'wrong' ref ie
// not to existing interface.
func TestSetAndCommitWrongRef(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leafref with non-existent reference",
			"intfCont/mgmt-interface/dp0s2", SetPass),
		createValOpTbl("Add interface that we don't reference",
			"intfCont/interface/dp0s3", SetPass),
	}

	// Error should indicate leafref is not a valid value.
	test_expOut := errtest.NewLeafrefError(t,
		"/intfCont/mgmt-interface/dp0s2",
		"intfCont/interface/name/dp0s2").
		RawErrorStrings()

	setCommitWrongRefTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, interfaceSchema, emptyconfig,
		setCommitWrongRefTests)
}

// Set non-existent value (pass) then create that value and commit (pass)
func TestSetNotYetCreatedRefCommitValid(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leafref with not-yet-existent reference",
			"intfCont/mgmt-interface/dp0s2", SetPass),
		createValOpTbl("Add interface for valid reference",
			"intfCont/interface/dp0s2", SetPass),
	}

	expConfig := `intfCont {
	interface dp0s2
	mgmt-interface dp0s2
}
`
	setNotYetCreatedTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, interfaceSchema, emptyconfig,
		setNotYetCreatedTests)
}

// Set already-existing value (pass) then commit (pass)
func TestSetRefExistsCommitValid(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leafref with existing reference",
			"intfCont/mgmt-interface/dp0s2", SetPass),
	}

	baseConfig :=
		Cont("intfCont",
			List("interface",
				ListEntry("dp0s2")))

	expConfig :=
		Cont("intfCont",
			List("interface",
				ListEntry("dp0s2")),
			Leaf("mgmt-interface", "dp0s2"))

	setRefExistsTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, interfaceSchema, baseConfig,
		setRefExistsTests)
}

// Set valid value then remove referenced value and commit (fail)
func TestSetValidCommitInvalidRef(t *testing.T) {
	test_delTbl := []ValidateOpTbl{
		createValOpTbl("Remove node leafref references.",
			"intfCont/interface/dp0s2", SetPass),
	}

	baseConfig := `intfCont {
	interface dp0s2
	mgmt-interface dp0s2
}
`
	test_expOut := errtest.NewLeafrefError(t,
		"/intfCont/mgmt-interface/dp0s2",
		"intfCont/interface/name/dp0s2").
		RawErrorStrings()

	expConfig := `intfCont {
	interface dp0s2
	mgmt-interface dp0s2
}
`
	removeValidRefTests := []xpathTestEntry{
		newXpathTestEntry(nil, test_delTbl, CommitFail, expConfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, interfaceSchema, baseConfig,
		removeValidRefTests)
}

// Set valid value and commit (pass) - multiple valid values available
func TestSetAndCommitMultipleValidRefs(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leafref with not-yet-existent reference",
			"intfCont/mgmt-interface/dp0s2", SetPass),
		createValOpTbl("Add interface for unused reference",
			"intfCont/interface/dp0s3", SetPass),
		createValOpTbl("Add interface for valid reference",
			"intfCont/interface/dp0s2", SetPass),
		createValOpTbl("Add interface for unused reference",
			"intfCont/interface/s2", SetPass),
	}

	expConfig := `intfCont {
	interface dp0s2
	interface dp0s3
	interface s2
	mgmt-interface dp0s2
}
`
	setMultipleRefsTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, interfaceSchema, emptyconfig,
		setMultipleRefsTests)
}

const localRefSchema = `
	container localCont {
	leaf refLeaf {
		type string;
	}
	leaf leafrefLeaf {
		type leafref {
			path "/local:localCont/refLeaf";
		}
	}
}`

// Set valid value and commit (pass) - local prefix
func TestSetAndCommitValidLocalRef(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add interface for unused reference",
			"localCont/refLeaf/refLeafVal", SetPass),
		createValOpTbl("Add leafref.",
			"localCont/leafrefLeaf/refLeafVal", SetPass),
	}

	expConfig := `localCont {
	leafrefLeaf refLeafVal
	refLeaf refLeafVal
}
`
	localRefTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				SchemaSnippet: localRefSchema,
			},
		},
		emptyconfig, localRefTests)
}

const defAddrSchema = `
container default-address {
	leaf ifname {
		type leafref {
			path "/intf:intfCont/intf:interface/intf:name";
		}
	}
	leaf address {
		type leafref {
			path "/intf:intfCont/intf:interface[intf:name = current()/../ifname]"
			+ "/intf:address/intf:ip";
		}
	}
}`

// Set valid value and commit (pass) - remote prefix
func TestSetAndCommitValidRemoteRef(t *testing.T) {
	baseConfig := `intfCont {
	interface lo666
}
`
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add remote reference",
			"default-address/ifname/lo666", SetPass),
	}

	expConfig := `default-address {
	ifname lo666
}
intfCont {
	interface lo666
}
`
	remoteRefTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-intf", Prefix: "intf"},
				SchemaSnippet: interfaceSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-da", Prefix: "da"},
				Imports:       []NameDef{{Namespace: "prefix-intf", Prefix: "intf"}},
				SchemaSnippet: defAddrSchema,
			},
		},
		baseConfig, remoteRefTests)
}

// Set valid value and commit (pass) using predicate notation
func TestSetAndCommitValidRefWithPredicate(t *testing.T) {
	baseConfig := `default-address {
	   	ifname lo666
	   }
	   intfCont {
	   	interface lo666 {
	           address 6666
	       }
	   }
	   `
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add remote reference",
			"default-address/address/6666", SetPass),
	}

	expConfig := `default-address {
	address 6666
	ifname lo666
}
intfCont {
	interface lo666 {
		address 6666
	}
}
`

	remoteRefTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-intf", Prefix: "intf"},
				SchemaSnippet: interfaceSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-da", Prefix: "da"},
				Imports:       []NameDef{{Namespace: "prefix-intf", Prefix: "intf"}},
				SchemaSnippet: defAddrSchema,
			},
		},
		baseConfig, remoteRefTests)
}

// Set invalid value and commit (fail) using predicate notation
func TestSetAndCommitInvalidRefWithPredicate(t *testing.T) {
	baseConfig := `default-address {
	   	ifname lo666
	}
	intfCont {
	   	interface lo666 {
	        address 6666
	    }
	   	interface lo777 {
	        address 7777
	    }
	}`

	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add remote reference",
			"default-address/address/7777", SetPass),
	}

	test_expOut := errtest.NewLeafrefError(t,
		"/default-address/address/7777",
		"/intfCont/interface/address/ip/7777").
		RawErrorStrings()

	remoteRefFailTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, baseConfig,
			test_expOut),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-intf", Prefix: "intf"},
				SchemaSnippet: interfaceSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-da", Prefix: "da"},
				Imports:       []NameDef{{Namespace: "prefix-intf", Prefix: "intf"}},
				SchemaSnippet: defAddrSchema,
			},
		},
		baseConfig, remoteRefFailTests)
}

const missingLeafrefSchema = `
list parent-list {
	key "parent";
	leaf parent {
		type string;
	}
	list ref-list {
		key "ref-leaf";
		leaf ref-leaf {
			type string;
		}
	}
	list using-list {
		key "number";
		leaf number {
			type uint32;
		}
		leaf using-leaf {
			type leafref {
				path "../../ref-list/ref-leaf";
			}
		}
	}
}
`

var missingLeafrefInitConfig = Root(
	List("parent-list",
		ListEntry("Table-A",
			List("ref-list",
				ListEntry("ref-1")),
			List("using-list",
				ListEntry("4",
					Leaf("using-leaf", "ref-1")))),
	))

var missingLeafrefConfig = Root(
	List("parent-list",
		ListEntry("Table-A",
			List("ref-list",
				ListEntry("ref-1")),
			List("using-list",
				ListEntry("4",
					Leaf("using-leaf", "ref-1")))),
		ListEntry("Table-B",
			List("ref-list",
				ListEntry("ref-3")),
			List("using-list",
				ListEntry("15",
					Leaf("using-leaf", "ref-3")))),
	))

func TestMissingLeafref(t *testing.T) {

	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add Table B ref-list",
			"parent-list/Table-B/ref-list/ref-3", SetPass),
		createValOpTbl("Add Table B using-list",
			"parent-list/Table-B/using-list/15/using-leaf/ref-3", SetPass),
	}

	missingLeafrefTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, missingLeafrefConfig,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-pol", Prefix: "pol"},
				SchemaSnippet: missingLeafrefSchema,
			},
		},
		missingLeafrefInitConfig, missingLeafrefTests)
}
