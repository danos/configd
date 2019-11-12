// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests on XPATH 'when' functionality
// with augment and grouping within a single module that require
// an active session.  It does not validate any prefixed paths - see
// qname_test.go for tests in that area.

package session_test

import (
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

// Summary of tests
// ----------------
//
// Each pair of tests covers essentially the same schema scenario, but in
// the first case uses an absolute path, and in the second a relative path,
// so we can ensure both cases work.  Specific scenarios covered are as
// follows, with the tests mainly concerned with ensuring the context for
// the when statement is correct.  (Some of the later tests only cover
// relative paths as that proves 'current' node must be correct and testing
// for absolute paths doesn't really add a lot, if anything.)
//
// - 'when' statement directly under augment (verifies 'when' is run with
//   correct context as it gets stored on the child of the node it really
//   applies to)
//
// - 'when' in leaf under augment statement
//   node)
//
// - 'when' in a grouping used inside an augment
//
// - 'when' directly under augment combined with 'when' in a leaf (this and
//   next verify correct handling of multiple 'when' statements with different
//   contexts)
//
// - 'when' directly under augment combined with 'when' in grouping used in
//   augment (just in case there are any differences to a normal leaf as
//   grouping / uses statements are handled differently).

const augmentSchemaWhenAbsPath = `
container testCont {
	leaf firstLeaf {
		type string;
	}
}

augment /testCont {
	leaf augmentLeaf {
		type string;
	}

	when "/testCont/firstLeaf = 'leaf1'"; // Absolute path
}`

// Test 'when' directly under augment with absolute path
func TestWhenInAugmentAbsPath(t *testing.T) {
	// Apply illegal config to verify augment leaf cannot be configured.
	// This verifies when on the augmented object.
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Illegal config to fail augment when",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont",
		"/testCont/firstLeaf = 'leaf1'").
		RawErrorStrings()

	// Now try again with value of firstLeaf that will pass augment 'when'.
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Legal config for first augment when",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testB_expCfg := `testCont {
	augmentLeaf arbitraryString
	firstLeaf leaf1
}
`

	augmentWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, augmentSchemaWhenAbsPath, emptyconfig,
		augmentWhenTests)
}

const augmentSchemaWhenRelPath = `
container testCont {
	leaf firstLeaf {
		type string;
	}
}

augment /testCont {
	leaf augmentLeaf {
		type string;
	}

	when "firstLeaf = 'leaf1'"; // Relative path
}`

// Test 'when' directly under augment on relative path
func TestWhenInAugmentRelPath(t *testing.T) {
	// Configure illegal firstLeaf value to fail when statement
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Illegal config to fail augment when",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont",
		"firstLeaf = 'leaf1'").
		RawErrorStrings()

	// Now configure valid firstLeaf so when statement passes.
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Legal config for first augment when",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testB_expCfg := `testCont {
	augmentLeaf arbitraryString
	firstLeaf leaf1
}
`

	augmentWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, augmentSchemaWhenRelPath, emptyconfig,
		augmentWhenTests)
}

const augmentSchemaLeafWhenAbsPath = `
container testCont {
	leaf firstLeaf {
		type string;
	}
}

augment /testCont {
	leaf augmentLeaf {
		type string;
		when "/testCont/firstLeaf = 'leaf1'"; // Absolute path
	}
}`

// Test when inside a grouping, absolute path.
func TestWhenInLeafUnderAugmentAbsPath(t *testing.T) {
	// Apply illegal config to verify augment cannot be configured.
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Illegal config to fail grouping when",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/augmentLeaf/arbitraryString",
		"/testCont/firstLeaf = 'leaf1'").
		RawErrorStrings()

	// Now configure valud firstLeaf value to pass grouping 'when'
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Legal config for grouping when",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Configure augmented leaf to verify when passes",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testB_expCfg := `testCont {
	augmentLeaf arbitraryString
	firstLeaf leaf1
}
`

	groupingWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, augmentSchemaLeafWhenAbsPath, emptyconfig,
		groupingWhenTests)
}

const augmentSchemaLeafWhenRelPath = `
container testCont {
	leaf firstLeaf {
		type string;
	}
}

augment /testCont {
	leaf augmentLeaf {
		type string;
		when "../firstLeaf = 'leaf1'";
	}
}`

// Test 'when' inside a grouping, relative path, used for an augment
func TestWhenInLeafUnderAugmentRelPath(t *testing.T) {
	// First set up failure
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Illegal config to fail grouping when",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/augmentLeaf/arbitraryString",
		"../firstLeaf = 'leaf1'").
		RawErrorStrings()

	// ... then set up pass.
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Legal config for grouping when",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Configure augmented leaf to verify 'when' passes",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testB_expCfg := `testCont {
	augmentLeaf arbitraryString
	firstLeaf leaf1
}
`

	groupingWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, augmentSchemaLeafWhenRelPath, emptyconfig,
		groupingWhenTests)
}

const groupingSchemaWhenAbsPath = `
container testCont {
	leaf firstLeaf {
		type string;
	}
}

grouping leafGrouping {
	leaf groupLeaf {
		type string;
		when "/testCont/firstLeaf = 'leaf1'"; // Absolute path
	}
}

augment /testCont {
	uses leafGrouping;
}`

// Test when inside a grouping, absolute path.
func TestWhenWithGroupingAbsPath(t *testing.T) {
	// Apply illegal config to verify augment cannot be configured.
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Illegal config to fail grouping when",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/groupLeaf/arbitraryString",
		"/testCont/firstLeaf = 'leaf1'").
		RawErrorStrings()

	// Now configure valud firstLeaf value to pass grouping 'when'
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Legal config for grouping when",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Configure augmented leaf to verify when passes",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}
	testB_expCfg := `testCont {
	firstLeaf leaf1
	groupLeaf arbitraryString
}
`

	groupingWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, groupingSchemaWhenAbsPath, emptyconfig,
		groupingWhenTests)
}

const groupingSchemaWhenRelPath = `
container testCont {
	leaf firstLeaf {
		type string;
	}
}

grouping leafGrouping {
	leaf groupLeaf {
		type string;
		when "../firstLeaf = 'leaf1'";
	}
}

augment /testCont {
	uses leafGrouping;
}`

// Test 'when' inside a grouping, relative path, used for an augment
func TestWhenInGroupingRelPath(t *testing.T) {
	// First set up failure
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Illegal config to fail grouping when",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Configure augmented leaf to trigger error",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/groupLeaf/arbitraryString",
		"../firstLeaf = 'leaf1'").
		RawErrorStrings()

	// ... then set up pass.
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Legal config for grouping when",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Configure augmented leaf to verify 'when' passes",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}
	testB_expCfg := `testCont {
	firstLeaf leaf1
	groupLeaf arbitraryString
}
`

	groupingWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, groupingSchemaWhenRelPath, emptyconfig,
		groupingWhenTests)
}

// This set of 2 tests verifies the scenario where we have a 'when'
// statement on a leaf in an augment statement (either 'native' or via
// a 'uses' statement) and we also have a 'when' on the augment itself.
// This second 'when' gets stored on the augmented child(ren) not on the
// container being augmented, but is run in the context of the parent
// container.

const groupingAndAugmentWhenRelPathSchema = `
container testCont {
	leaf firstLeaf {
		type string;
	}
	leaf secondLeaf {
		type string;
	}
}

grouping leafGrouping {
	leaf groupLeaf {
		type string;
		when "../firstLeaf = 'leaf1'";
	}
}

augment /testCont {
	uses leafGrouping;
	when "secondLeaf = 'leaf2'";
}`

func TestWhenOnAugmentAndInGroupingRelPath(t *testing.T) {
	// Configure group, first (bad) and second leaf (bad) - both whens fail
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to fail groupLeaf 'when'",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Second leaf value to fail augment 'when'",
			"testCont/secondLeaf/failWhen", SetPass),
		createValOpTbl("GroupLeaf configured to trigger failures",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/groupLeaf/arbitraryString",
		"../firstLeaf = 'leaf1'").
		RawErrorStrings()
	testA_expOut = append(testA_expOut,
		errtest.NewWhenDefaultError(t,
			"/testCont",
			"secondLeaf = 'leaf2'").
			RawErrorStrings()...)

	// Configure group, first (ok) and second (bad) - only second when fails
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to pass groupLeaf 'when'",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Second leaf value to fail augment 'when'",
			"testCont/secondLeaf/failWhen", SetPass),
		createValOpTbl("GroupLeaf configured to trigger failure",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}
	testB_expOut := errtest.NewWhenDefaultError(t,
		"/testCont",
		"secondLeaf = 'leaf2'").
		RawErrorStrings()

	// Configure group, first (bad) and second (ok) - only first when fails
	testC_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to fail groupLeaf 'when'",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Second leaf value to pass augment 'when'",
			"testCont/secondLeaf/leaf2", SetPass),
		createValOpTbl("GroupLeaf configured to trigger failure",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}
	testC_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/groupLeaf/arbitraryString",
		"../firstLeaf = 'leaf1'").
		RawErrorStrings()

	// Configure group, first (ok) and second (ok) - config passes
	testD_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to pass groupLeaf 'when'",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Second leaf value to pass augment 'when'",
			"testCont/secondLeaf/leaf2", SetPass),
		createValOpTbl("GroupLeaf configured to check all passes",
			"testCont/groupLeaf/arbitraryString", SetPass),
	}

	testD_expCfg := `testCont {
	firstLeaf leaf1
	groupLeaf arbitraryString
	secondLeaf leaf2
}
`

	groupingWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitFail, emptyconfig,
			testB_expOut),
		newXpathTestEntry(testC_setTbl, nil, CommitFail, emptyconfig,
			testC_expOut),
		newXpathTestEntry(testD_setTbl, nil, CommitPass, testD_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, groupingAndAugmentWhenRelPathSchema,
		emptyconfig, groupingWhenTests)
}

const leafAndAugmentWhenRelPathSchema = `
container testCont {
	leaf firstLeaf {
		type string;
	}
	leaf secondLeaf {
		type string;
	}
}

augment /testCont {
	leaf augmentLeaf {
		type string;
		when "../firstLeaf = 'leaf1'";
	}

	when "secondLeaf = 'leaf2'";
}`

func TestWhenOnAugmentAndOnLeafRelPath(t *testing.T) {
	// Configure augment, first (bad) and second leaf (bad) - both whens fail
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to fail augmentLeaf 'when'",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Second leaf value to fail augment 'when'",
			"testCont/secondLeaf/failWhen", SetPass),
		createValOpTbl("Augmentleaf configured to trigger failures",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testA_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/augmentLeaf/arbitraryString",
		"../firstLeaf = 'leaf1'").
		RawErrorStrings()
	testA_expOut = append(testA_expOut,
		errtest.NewWhenDefaultError(t,
			"/testCont",
			"secondLeaf = 'leaf2'").
			RawErrorStrings()...)

	// Configure augment, first (ok) and second (bad) - only second when fails
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to pass augmentleaf 'when'",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Second leaf value to fail augment 'when'",
			"testCont/secondLeaf/failWhen", SetPass),
		createValOpTbl("AugmentLeaf configured to trigger failure",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testB_expOut := errtest.NewWhenDefaultError(t,
		"/testCont",
		"secondLeaf = 'leaf2'").
		RawErrorStrings()

	// Configure augment, first (bad) and second (ok) - only first when fails
	testC_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to fail augmentleaf 'when'",
			"testCont/firstLeaf/failWhen", SetPass),
		createValOpTbl("Second leaf value to pass augment 'when'",
			"testCont/secondLeaf/leaf2", SetPass),
		createValOpTbl("Augmentleaf configured to trigger failure",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}
	testC_expOut := errtest.NewWhenDefaultError(t,
		"/testCont/augmentLeaf/arbitraryString",
		"../firstLeaf = 'leaf1'").
		RawErrorStrings()

	// Configure augment, first (ok) and second (ok) - config passes
	testD_setTbl := []ValidateOpTbl{
		createValOpTbl("First leaf value to pass augmentleaf 'when'",
			"testCont/firstLeaf/leaf1", SetPass),
		createValOpTbl("Second leaf value to pass augment 'when'",
			"testCont/secondLeaf/leaf2", SetPass),
		createValOpTbl("Augmentleaf configured to check all passes",
			"testCont/augmentLeaf/arbitraryString", SetPass),
	}

	testD_expCfg := `testCont {
	augmentLeaf arbitraryString
	firstLeaf leaf1
	secondLeaf leaf2
}
`

	groupingWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitFail, emptyconfig,
			testB_expOut),
		newXpathTestEntry(testC_setTbl, nil, CommitFail, emptyconfig,
			testC_expOut),
		newXpathTestEntry(testD_setTbl, nil, CommitPass, testD_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutput(t, leafAndAugmentWhenRelPathSchema, emptyconfig,
		groupingWhenTests)
}
