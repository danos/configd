// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests on XPATH 'when' and 'must' statements that
// require an active session to be running.  The tests are on basic
// functionality and only use a single YANG module.

// Test procedure is the same for each node type.  We create an entry, and
// ensure 'when' fails, and that a subsequent 'must' statement that would
// fail is not checked (not necessary).  We then change config so that the
// second 'must' will fail (so we verify that multiple musts are checked).
// Finally we get all 3 checks to pass.

package session_test

import (
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

const contSchema = `
container currentCont {
	presence "must and when testing";
	when "count(aLeafList) < 3";
	must "local-name(.) = 'currentCont'";
	must "not(contains(., 'foo'))";
	leaf aLeaf {
		type string;
	}
	leaf-list aLeafList {
		type uint16;
	}
}`

func TestContainerWhenFail(t *testing.T) {
	const baseCfg = "currentCont"

	// Add 3 leaflist entries to get 'when' to fail.  Add leaf entry
	// so we verify 'must' failure is not triggered ('when' takes
	// precedence.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaflist entry 1",
			"currentCont/aLeafList/321", SetPass),
		createValOpTbl("Add leaflist entry 2",
			"currentCont/aLeafList/3210", SetPass),
		createValOpTbl("Add leaflist entry 3 to create WHEN failure",
			"currentCont/aLeafList/4321", SetPass),
		createValOpTbl("Add leafEntry 'foo' for (unseen) must failure",
			"currentCont/aLeaf/foo", SetPass),
	}

	// This also shows that neither must statement was run
	test_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont",
		"count(aLeafList) < 3").
		RawErrorStrings()

	contTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, baseCfg, test_expOut),
	}

	runXpathTestsCheckOutput(t, contSchema, baseCfg, contTests)
}

func TestContainerMustFail(t *testing.T) {
	const baseCfg = "currentCont"

	// Make second must statement fail.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Make second must statement fail",
			"currentCont/aLeaf/foo", SetPass),
	}
	test_expOut := errtest.NewMustDefaultError(t,
		"/currentCont",
		"not(contains(., 'foo'))").
		RawErrorStrings()

	contTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, baseCfg, test_expOut),
	}

	runXpathTestsCheckOutput(t, contSchema, baseCfg, contTests)
}

func TestContainerWhenAndMustPass(t *testing.T) {
	const baseCfg = "currentCont"

	// Make sure we can get must and when conditions to pass!
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaflist entry 1",
			"currentCont/aLeafList/321", SetPass),
		createValOpTbl("Add leaflist entry 2 (2 entries ok, 3 would fail)",
			"currentCont/aLeafList/3210", SetPass),
		createValOpTbl("Add leafEntry for good measure.",
			"currentCont/aLeaf/oo", SetPass),
	}
	test_cfg := `currentCont {
	aLeaf oo
	aLeafList 321
	aLeafList 3210
}
`
	contTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, test_cfg, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, contSchema, baseCfg, contTests)
}

const listSchema = `
container currentCont {
	list aList {
		key name;
		leaf name {
			type string;
		}
		when "count(../anotherLeafList) < 3";
		must "local-name(.) = 'aList'";
		must "not(contains(., 'oops'))";
	}
	leaf-list anotherLeafList {
		type uint16;
	}
}`

func TestListWhenFail(t *testing.T) {
	const baseCfg = `currentCont {
        aList {
            listEntry
        }
    }`

	// T1: Add 3 leaflist entries to get 'when' to fail.  Last one contains
	//     '4321' to verify 'must' failure is not triggered ('when' takes
	//     precedence.
	test1_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaflist entry 1",
			"currentCont/anotherLeafList/321", SetPass),
		createValOpTbl("Add leaflist entry 2",
			"currentCont/anotherLeafList/3210", SetPass),
		createValOpTbl("Add entry 3 for WHEN and (hidden) MUST failure",
			"currentCont/anotherLeafList/4321", SetPass),
	}
	// This also shows that neither must statement was run
	test1_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aList/listEntry",
		"count(../anotherLeafList) < 3").
		RawErrorStrings()

	listTests := []xpathTestEntry{
		newXpathTestEntry(test1_setTbl, nil, CommitFail, baseCfg, test1_expOut),
	}

	runXpathTestsCheckOutput(t, listSchema, baseCfg, listTests)
}

func TestListMustFail(t *testing.T) {
	const baseCfg = `currentCont {
        aList {
            listEntry
        }
    }`

	// Make second must statement fail for each of 2 elements
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Make second must statement fail",
			"currentCont/aList/oops", SetPass),
		createValOpTbl("Make second must statement fail",
			"currentCont/aList/oopsAgain", SetPass),
	}
	test_expOut :=
		errtest.NewMustDefaultError(t,
			"/currentCont/aList/oops",
			"not(contains(., 'oops'))").
			RawErrorStrings()
	test_expOut = append(test_expOut,
		errtest.NewMustDefaultError(t,
			"/currentCont/aList/oopsAgain",
			"not(contains(., 'oops'))").
			RawErrorStrings()...)

	listTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, baseCfg, test_expOut),
	}

	runXpathTestsCheckOutput(t, listSchema, baseCfg, listTests)
}

func TestListWhenAndMustPass(t *testing.T) {
	const baseCfg = `currentCont {
        aList {
            listEntry
        }
    }`
	// Make sure we can get must and when conditions to pass!
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaflist entry 1",
			"currentCont/anotherLeafList/321", SetPass),
		createValOpTbl("Add leaflist entry 2 (2 is ok, 3 would fail)",
			"currentCont/anotherLeafList/3210", SetPass),
	}
	test_cfg := `currentCont {
	aList listEntry
	anotherLeafList 321
	anotherLeafList 3210
}
`
	listTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, test_cfg, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, listSchema, baseCfg, listTests)
}

const leafListSchema = `
container currentCont {
	leaf-list aLeafList {
		type uint16;
		when "count(../aList) < 3";
		must "local-name(.) = 'aLeafList'";
		must "contains(., '321')";
	}
	list aList {
		key name;
		leaf name {
			type string;
		}
	}
}`

func TestLeafListWhenFail(t *testing.T) {
	// Add 3 list entries to get 'when' to fail. Our 'trigger' leaf list
	// entry has invalid content to trigger second must (but we won't hit
	// that as when takes precedence.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add bad leaf-list entry for hidden MUST failure",
			"currentCont/aLeafList/432", SetPass),
		createValOpTbl("Add list entry 1",
			"currentCont/aList/listEntry1", SetPass),
		createValOpTbl("Add list entry 2",
			"currentCont/aList/listEntry2", SetPass),
		createValOpTbl("Add list entry 3 for WHEN failure",
			"currentCont/aList/listEntry3", SetPass),
	}
	// This also shows that neither must statement was run
	test_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aLeafList/432",
		"count(../aList) < 3").
		RawErrorStrings()

	leafListTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, leafListSchema, emptyconfig, leafListTests)
}

func TestLeafListMustFail(t *testing.T) {
	// Make second must statement fail ... but only for second element.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Acceptable leaf list entry",
			"currentCont/aLeafList/321", SetPass),
		createValOpTbl("Make second must statement fail",
			"currentCont/aLeafList/432", SetPass),
	}
	test_expOut := errtest.NewMustDefaultError(t,
		"/currentCont/aLeafList/432",
		"contains(., '321')").
		RawErrorStrings()

	leafListTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, leafListSchema, emptyconfig, leafListTests)
}

func TestLeafListWhenAndMustPass(t *testing.T) {
	// Now make sure we can get must and when conditions to pass!
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaflist entry 1",
			"currentCont/aLeafList/321", SetPass),
	}
	test_cfg := `currentCont {
	aLeafList 321
}
`
	leafListTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, test_cfg, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, leafListSchema, emptyconfig, leafListTests)
}

const leafSchema = `
container currentCont {
	leaf aLeaf {
		type string;
		when "count(../aList) < 3";
		must "local-name(.) = 'aLeaf'";
		must "contains(., 'oo')";
	}
	list aList {
		key name;
		leaf name {
			type string;
		}
	}
}`

func TestLeafWhenFail(t *testing.T) {
	// Add 3 list entries to get 'when' to fail. Our 'trigger' leaf list
	// entry has invalid content to trigger second must (but we won't hit
	// that as when takes precedence.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add bad leaf entry for hidden MUST failure",
			"currentCont/aLeaf/leafEntryBad", SetPass),
		createValOpTbl("Add list entry 1",
			"currentCont/aList/listEntry1", SetPass),
		createValOpTbl("Add list entry 2",
			"currentCont/aList/listEntry2", SetPass),
		createValOpTbl("Add list entry 3 for WHEN failure",
			"currentCont/aList/listEntry3", SetPass),
	}
	// This also shows that neither must statement was run
	test_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aLeaf/leafEntryBad",
		"count(../aList) < 3").
		RawErrorStrings()

	leafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, leafSchema, emptyconfig, leafTests)
}

func TestLeafMustFail(t *testing.T) {
	// Make second must statement fail
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Bad leaf entry",
			"currentCont/aLeaf/leafEntryBad", SetPass),
	}
	test_expOut := errtest.NewMustDefaultError(t,
		"/currentCont/aLeaf/leafEntryBad",
		"contains(., 'oo')").
		RawErrorStrings()

	leafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, leafSchema, emptyconfig, leafTests)
}

func TestLeafWhenAndMustPass(t *testing.T) {
	// Now make sure we can get must and when conditions to pass!
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaflist entry 1",
			"currentCont/aLeaf/oo", SetPass),
	}
	test_cfg := `currentCont {
	aLeaf oo
}
`
	leafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, test_cfg, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, leafSchema, emptyconfig, leafTests)
}

const listKeyLeafSchema = `
container currentCont {
	list aList {
		key name;
		leaf name {
			type string;
			when "count(../../yetAnotherLeafList) < 2";
			must "local-name(.) = 'name'";
			must "contains(., 'listEntry')";
		}
		when "count(../anotherLeafList) < 3";
		must "local-name(.) = 'aList'";
		must "not(contains(., 'oops'))";
	}
	leaf-list anotherLeafList {
		type uint16;
	}
	leaf-list yetAnotherLeafList {
		type uint16;
	}
}`

func TestListKeyLeafWhenFail(t *testing.T) {
	const baseCfg = `currentCont {
        aList {
            listEntry
        }
    }`

	// Add 3 list entries to get 'when' to fail. Our 'trigger' leaf list
	// entry has invalid content to trigger second must (but we won't hit
	// that as when takes precedence.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add bad list entry for hidden MUST failure",
			"currentCont/aList/bad", SetPass),
		createValOpTbl("Add leaf-list entry 1 for WHEN test",
			"currentCont/yetAnotherLeafList/321", SetPass),
		createValOpTbl("Add leaf-list entry 2 for WHEN test",
			"currentCont/yetAnotherLeafList/3210", SetPass),
		createValOpTbl("Add leaf-list entry 3 for WHEN failure",
			"currentCont/yetAnotherLeafList/4321", SetPass),
	}
	// This also shows that neither must statement was run
	test_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aList/bad",
		"count(../../yetAnotherLeafList) < 2").
		RawErrorStrings()

	listKeyLeafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, listKeyLeafSchema, emptyconfig,
		listKeyLeafTests)
}

func TestListKeyLeafMustFail(t *testing.T) {
	// Make second must statement fail
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Bad list entry - fails MUST",
			"currentCont/aList/bad", SetPass),
	}
	test_expOut := errtest.NewMustDefaultError(t,
		"/currentCont/aList/bad",
		"contains(., 'listEntry')").
		RawErrorStrings()

	listKeyLeafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, listKeyLeafSchema, emptyconfig,
		listKeyLeafTests)
}

func TestListKeyLeafWhenAndMustPass(t *testing.T) {
	// Now make sure we can get must and when conditions to pass!
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add list entry 1",
			"currentCont/aList/listEntry1", SetPass),
	}
	test_cfg := `currentCont {
	aList listEntry1
}
`
	listKeyLeafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, test_cfg, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, listKeyLeafSchema, emptyconfig,
		listKeyLeafTests)
}

const emptyLeafSchema = `
container currentCont {
	leaf aLeaf {
		type string;
		when "../emptyLeaf";
	}
	leaf emptyLeaf {
		type empty;
	}
}`

// 'when' statement refers to an empty leaf.
func TestWhenRefToEmptyLeaf(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Add aLeaf to fail due to no empty leaf",
			"currentCont/aLeaf/leafEntryBad", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aLeaf/leafEntryBad",
		"../emptyLeaf").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Add aLeaf to pass with empty leaf configured",
			"currentCont/aLeaf/leafEntryGood", SetPass),
		createValOpTbl("Add empty leaf so when statement passes",
			"currentCont/emptyLeaf", SetPass),
	}
	expCfgB := `currentCont {
	aLeaf leafEntryGood
	emptyLeaf
}
`

	emptyLeafTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, expCfgB, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, emptyLeafSchema, emptyconfig, emptyLeafTests)
}

const emptyLeafSchema2 = `
container currentCont {
	leaf aLeaf {
		type string;
	}
	leaf emptyLeaf {
		type empty;
		when "../aLeaf";
	}
}`

// 'when' statement is on the empty leaf, referencing another leaf.
func TestWhenInEmptyLeaf(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Add emptyLeaf to fail due to no aleaf",
			"currentCont/emptyLeaf", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/emptyLeaf",
		"../aLeaf").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Add aLeaf so when statement passes",
			"currentCont/aLeaf/leafEntryGood", SetPass),
		createValOpTbl("Add empty leaf to pass with aLeaf configured",
			"currentCont/emptyLeaf", SetPass),
	}
	expCfgB := `currentCont {
	aLeaf leafEntryGood
	emptyLeaf
}
`

	emptyLeafTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, expCfgB,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, emptyLeafSchema2, emptyconfig, emptyLeafTests)
}

const leafDefaultSchema = `
container currentCont {
	leaf aLeaf {
		type string;
		when "../defaultLeaf > 10";
	}
	leaf defaultLeaf {
		type uint16;
		default 5;
	}
}`

func TestDefaultLeafWhenFail(t *testing.T) {
	// Add leaf entry so when fails due to default value blocking it.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaf entry to fail WHEN",
			"currentCont/aLeaf/foo", SetPass),
	}

	test_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aLeaf/foo",
		"../defaultLeaf > 10").
		RawErrorStrings()

	defaultLeafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutput(t, leafDefaultSchema, emptyconfig,
		defaultLeafTests)
}

func TestDefaultLeafWhenPass(t *testing.T) {
	// Add leaf entry AND change default leaf so it should pass.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaf entry",
			"currentCont/aLeaf/foo", SetPass),
		createValOpTbl("Change defaultLeaf so aLeaf entry is allowed",
			"currentCont/defaultLeaf/15", SetPass),
	}

	expCfg := `currentCont {
	aLeaf foo
	defaultLeaf 15
}
`

	defaultLeafTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expCfg, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, leafDefaultSchema, emptyconfig,
		defaultLeafTests)
}

const leafDeleteSchema = `
container currentCont {
	leaf aLeaf {
		type string;
		when "../secondLeaf";
	}
	leaf secondLeaf {
		type string;
	}
}`

// Verify we handle deleted leaves correctly (don't count them)
func TestDeleteLeafWhenFail(t *testing.T) {
	test1_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaf entry which passes as secondleaf present",
			"currentCont/aLeaf/foo", SetPass),
		createValOpTbl("Add leaf entry to allow first leaf to pass",
			"currentCont/secondLeaf/bar", SetPass),
	}

	expCfg1 := `currentCont {
	aLeaf foo
	secondLeaf bar
}
`

	test2_delTbl := []ValidateOpTbl{
		createValOpTbl("Remove leaf entry to verify first leaf fails.",
			"currentCont/secondLeaf/bar", SetPass),
	}

	test2_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aLeaf/foo",
		"../secondLeaf").
		RawErrorStrings()

	deleteLeafTests := []xpathTestEntry{
		newXpathTestEntry(test1_setTbl, nil, CommitPass, expCfg1, expOutAllOK),
		newXpathTestEntry(nil, test2_delTbl, CommitFail, expCfg1, test2_expOut),
	}

	runXpathTestsCheckOutput(t, leafDeleteSchema, emptyconfig,
		deleteLeafTests)
}

const listDeleteSchema = `
container currentCont {
	leaf aLeaf {
		type string;
		when "count(../aList) > 1";
	}
    list aList {
		key "name";
		leaf "name" {
			type string;
		}
	}
}`

// Verify we handle deleted list entries ok.  Note that we need to leave a
// list entry to ensure we cover the (up to this point) untested code
// checking for deleted elements as if we completely remove the list, it's
// handled by other (already tested) code.
func TestDeleteListWhenFail(t *testing.T) {
	test1_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leaf entry which passes as list entry present",
			"currentCont/aLeaf/foo", SetPass),
		createValOpTbl("Add list entries to allow first leaf to pass",
			"currentCont/aList/someListEntry", SetPass),
		createValOpTbl("Add list entries to allow first leaf to pass",
			"currentCont/aList/anotherListEntry", SetPass),
	}

	expCfg1 := `currentCont {
	aLeaf foo
	aList anotherListEntry
	aList someListEntry
}
`

	test2_delTbl := []ValidateOpTbl{
		createValOpTbl("Remove list entry to verify first leaf fails.",
			"currentCont/aList/someListEntry", SetPass),
	}

	test2_expOut := errtest.NewWhenDefaultError(t,
		"/currentCont/aLeaf/foo",
		"count(../aList) > 1").
		RawErrorStrings()

	deleteListTests := []xpathTestEntry{
		newXpathTestEntry(test1_setTbl, nil, CommitPass, expCfg1, expOutAllOK),
		newXpathTestEntry(nil, test2_delTbl, CommitFail, expCfg1, test2_expOut),
	}

	runXpathTestsCheckOutput(t, listDeleteSchema, emptyconfig,
		deleteListTests)
}

const contErrMsgSchema = `
	container currentCont {
	presence "must custom-error testing";
	must "not(contains(., 'foo'))" {
		error-message "Must not contain 'foo'";
	}
	leaf aLeaf {
		type string;
	}
}`

func TestContainerMustFailCustomError(t *testing.T) {
	const baseCfg = "currentCont"

	// Add leaf entry to trigger must failure with custom error message.
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Add leafEntry 'foo' for (unseen) must failure",
			"currentCont/aLeaf/foo", SetPass),
	}

	test_expOut := errtest.NewMustCustomError(t,
		"/currentCont",
		"Must not contain 'foo").
		RawErrorStrings()

	contTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail, baseCfg, test_expOut),
	}

	runXpathTestsCheckOutput(t, contErrMsgSchema, baseCfg, contTests)
}

// Probably doesn't add a lot to the tests, but useful as proof of concept
// regarding a useful multicast filter that can replace a hideously ugly
// regexp (not, not the pattern here - something far worse!).
//
func TestMulticastXpathFilter(t *testing.T) {
	const schema = `
	typedef ipv4-address {
		type string {
			pattern '(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.){3}'
				+  '([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])';
			configd:pattern-help "<x.x.x.x>";
			configd:help "IPv4 Prefix";
		}
    }
	container interfaces {
	    list dataplane {
			key "name";
			leaf name {
				type string;
			}
			leaf-list address {
				type ipv4-address;
				must "(substring-before(., '.') >= 224) and " +
					"(substring-before(., '.') <= 239) and " +
					"(not(starts-with(., '224.0.0')))";
			}
		}
	}
	`

	const config = `interfaces {
		dataplane dp0s3
	}`

	// T1: Lowest and highest valid mcast addresses
	test1_setTbl := []ValidateOpTbl{
		createValOpTbl("Set lowest valid mcast address.",
			"interfaces/dataplane/dp0s3/address/224.1.0.0", SetPass),
		createValOpTbl("Set highest valid mcast address.",
			"interfaces/dataplane/dp0s3/address/239.255.255.255", SetPass),
	}

	// T2: Invalid mcast addresses
	test2_setTbl := []ValidateOpTbl{
		createValOpTbl("Set too low address.",
			"interfaces/dataplane/dp0s3/address/223.1.1.1", SetPass),
		createValOpTbl("Set too low address.",
			"interfaces/dataplane/dp0s3/address/224.0.0.1", SetPass),
		createValOpTbl("Set too high address.",
			"interfaces/dataplane/dp0s3/address/240.0.0.0", SetPass),
	}
	test_expOut2 :=
		errtest.NewMustDefaultError(t,
			"/interfaces/dataplane/dp0s3/address/223.1.1.1",
			"(substring-before(., '.') >= 224) and "+
				"(substring-before(., '.') <= 239) and "+
				"(not(starts-with(., '224.0.0')))").
			RawErrorStrings()
	test_expOut2 = append(test_expOut2,
		errtest.NewMustDefaultError(t,
			"/interfaces/dataplane/dp0s3/address/224.0.0.1",
			"(substring-before(., '.') >= 224) and "+
				"(substring-before(., '.') <= 239) and "+
				"(not(starts-with(., '224.0.0')))").
			RawErrorStrings()...)
	test_expOut2 = append(test_expOut2,
		errtest.NewMustDefaultError(t,
			"/interfaces/dataplane/dp0s3/address/240.0.0.0",
			"(substring-before(., '.') >= 224) and "+
				"(substring-before(., '.') <= 239) and "+
				"(not(starts-with(., '224.0.0')))").
			RawErrorStrings()...)

	const mcastCfg = `interfaces {
	dataplane dp0s3 {
		address 224.1.0.0
		address 239.255.255.255
	}
}
`

	mcastTests := []xpathTestEntry{
		newXpathTestEntry(test1_setTbl, nil, CommitPass, mcastCfg,
			expOutAllOK),
		newXpathTestEntry(test2_setTbl, nil, CommitFail, mcastCfg,
			test_expOut2),
	}

	runXpathTestsCheckOutput(t, schema, config, mcastTests)
}

func TestFirewallAlg(t *testing.T) {
	const fwAlgSchema = `
		container alg {
			presence "true";
			container tftp {
				presence "true";
				leaf-list port { // default 69
					max-elements 32;
					type uint16 {
						range 1..65535;
					}
                    must "not(current() = ../../ftp/port)" {
                        error-message "ALG ports must be unique";
                    }
                    must "(count(../../ftp/port) > 0) or current() != 21" {
                        error-message "TFTP port cannot be set to default " +
                            "FTP port (21) unless FTP set to non-default port";
                    }
                    must "not(current() = ../../sip/port)" {
                        error-message "ALG ports must be unique";
                    }
                    must "(count(../../sip/port) > 0) or current() != 5060" {
                        error-message "TFTP port cannot be set to default " +
                           "SIP port (5060) unless SIP set to non-default port";
                    }
				}
			}
			container ftp {
				presence "true";
				leaf-list port { // default 21
					max-elements 32;
					type uint16 {
						range 1..65535;
					}
                    must "not(current() = ../../sip/port)" {
                        error-message "ALG ports must be unique";
                    }
                    must "(count(../../sip/port) > 0) or current() != 5060" {
                        error-message "TFTP port cannot be set to default " +
                           "SIP port (5060) unless SIP set to non-default port";
                    }
                    must "not(current() = ../../tftp/port)" {
                        error-message "ALG ports must be unique";
                    }
                    must "(count(../../tftp/port) > 0) or current() != 69" {
                        error-message "FTP port cannot be set to default " +
                           "TFTP port (21) unless TFTP set to non-default port";
                    }
				}
			}
			container sip {
				presence "true";
				leaf-list port { // default 5060
					max-elements 32;
					type uint16 {
						range 1..65535;
					}
                    must "not(current() = ../../ftp/port)" {
                        error-message "ALG ports must be unique";
                    }
                    must "(count(../../ftp/port) > 0) or current() != 21" {
                        error-message "TFTP port cannot be set to default " +
                            "FTP port (21) unless FTP set to non-default port";
                    }
                    must "not(current() = ../../tftp/port)" {
                        error-message "ALG ports must be unique";
                    }
                    must "(count(../../tftp/port) > 0) or current() != 69" {
                        error-message "SIP port cannot be set to default " +
                           "TFTP port (21) unless TFTP set to non-default port";
                    }
				}
			}
		}
`

	// T1: 3 different values for tftp, sip and ftp - pass
	test1_setTbl := []ValidateOpTbl{
		createValOpTbl("Set FTP port.",
			"alg/ftp/port/123", SetPass),
		createValOpTbl("Set SIP port.",
			"alg/sip/port/124", SetPass),
		createValOpTbl("Set TFTP port.",
			"alg/tftp/port/125", SetPass),
	}

	const test_cfg1 = `alg {
	ftp {
		port 123
	}
	sip {
		port 124
	}
	tftp {
		port 125
	}
}
`

	// T2: SIP set to tftp default, tftp not set - fail
	test2_setTbl := []ValidateOpTbl{
		createValOpTbl("Set SIP port to TFTP default, no TFTP port set.",
			"alg/sip/port/69", SetPass),
	}
	test_expOut2 :=
		errtest.NewMustCustomError(t,
			"/alg/sip/port/69",
			"SIP port cannot be set to default TFTP port (21) "+
				"unless TFTP set to non-default port").
			RawErrorStrings()

	// T3: SIP set to tftp default, tftp set to non-default - pass
	test3_setTbl := []ValidateOpTbl{
		createValOpTbl("Set SIP port to TFTP default.",
			"alg/sip/port/69", SetPass),
		createValOpTbl("Set TFTP port to non-default value.",
			"alg/tftp/port/70", SetPass),
	}
	const test_cfg3 = `alg {
	sip {
		port 69
	}
	tftp {
		port 70
	}
}
`

	// T4: SIP and tftp both set to same, non-default, value - fail
	test4_setTbl := []ValidateOpTbl{
		createValOpTbl("Set SIP port.",
			"alg/sip/port/333", SetPass),
		createValOpTbl("Set TFTP port to same as SIP.",
			"alg/tftp/port/333", SetPass),
	}
	test_expOut4 :=
		errtest.NewMustCustomError(t,
			"/alg/sip/port/333",
			"ALG ports must be unique").
			RawErrorStrings()
	test_expOut4 = append(test_expOut4,
		errtest.NewMustCustomError(t,
			"/alg/tftp/port/333",
			"ALG ports must be unique").
			RawErrorStrings()...)

	clearTbl := []ValidateOpTbl{
		createValOpTbl("Clear ALG container",
			"alg", SetPass),
	}

	fwAlgTests := []xpathTestEntry{
		newXpathTestEntry(test1_setTbl, nil, CommitPass, test_cfg1,
			expOutAllOK),
		// We're not building on test1 config, so remove it.
		newXpathTestEntry(nil, clearTbl, CommitPass, emptyconfig,
			expOutAllOK),
		newXpathTestEntry(test2_setTbl, nil, CommitFail, emptyconfig,
			test_expOut2),
		newXpathTestEntry(test3_setTbl, nil, CommitPass, test_cfg3,
			expOutAllOK),
		// We're not building on test3 config, so remove it.
		newXpathTestEntry(nil, clearTbl, CommitPass, emptyconfig,
			expOutAllOK),
		newXpathTestEntry(test4_setTbl, nil, CommitFail, emptyconfig,
			test_expOut4),
	}

	runXpathTestsCheckOutput(t, fwAlgSchema, emptyconfig, fwAlgTests)
}

// Example from OSPF ... using a predicate to ensure the network used for
// one area is not set on any other area.
func TestPredicateBlockDuplicateNetwork(t *testing.T) {
	const schema = `
	typedef ipv4-address {
		type string {
			pattern '(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.){3}'
				+  '([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])';
			configd:pattern-help "<x.x.x.x>";
			configd:help "IPv4 Prefix";
		}
    }

    container ospf {
    	list area {
    		key id;
    		leaf id {
    			type uint16;
    		}
    		leaf address {
    			type ipv4-address;
                // Beware: 'current() !=' is VERY different to 'not(current() ='
    			must "not(current() = ../../area[id != current()/../id]/address)" {
    				error-message "Cannot use same network as another area";
    			}
                // Let's skin the cat a different way now ...
                must "1 = count(../../area[address = current()])" {
                    error-message "More than one usage of same address";
                }
    		}
    	}
    }
`

	// T1: Set non-conflicting networks on 2 areas
	test1_setTbl := []ValidateOpTbl{
		createValOpTbl("Set Area 0 network.",
			"ospf/area/0/address/10.0.0.0", SetPass),
		createValOpTbl("Set Area 1 network.",
			"ospf/area/1/address/20.0.0.0", SetPass),
	}

	const ospfConfig = `ospf {
	area 0 {
		address 10.0.0.0
	}
	area 1 {
		address 20.0.0.0
	}
}
`

	// T2: Set conflicting network on 3rd area.
	test2_setTbl := []ValidateOpTbl{
		createValOpTbl("Set Area 51 network.",
			"ospf/area/51/address/10.0.0.0", SetPass),
	}
	test_expOut2 :=
		errtest.NewMustCustomError(t,
			"/ospf/area/0/address/10.0.0.0",
			"Cannot use same network as another area").
			RawErrorStrings()
	test_expOut2 = append(test_expOut2,
		errtest.NewMustCustomError(t,
			"/ospf/area/0/address/10.0.0.0",
			"More than one usage of same address").
			RawErrorStrings()...)
	test_expOut2 = append(test_expOut2,
		errtest.NewMustCustomError(t,
			"/ospf/area/51/address/10.0.0.0",
			"Cannot use same network as another area").
			RawErrorStrings()...)
	test_expOut2 = append(test_expOut2,
		errtest.NewMustCustomError(t,
			"/ospf/area/51/address/10.0.0.0",
			"More than one usage of same address").
			RawErrorStrings()...)

	ospfTests := []xpathTestEntry{
		newXpathTestEntry(test1_setTbl, nil, CommitPass, ospfConfig,
			expOutAllOK),
		newXpathTestEntry(test2_setTbl, nil, CommitFail, emptyconfig,
			test_expOut2),
	}

	runXpathTestsCheckOutput(t, schema, emptyconfig, ospfTests)
}
