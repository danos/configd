// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-17 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// Tests relating to configd:priority

package session_test

import (
	"testing"

	. "github.com/danos/config/testutils"
	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
)

// Check that the order in which nodes are commited in the
// correct order as defined by configd:priority in the schema
func TestPriority(t *testing.T) {
	type validatePriTbl struct {
		schema string
		opTbl  []ValidateOpTbl
		expOut string
	}

	const testcontainertwo = "testcontainertwo"
	var testcontainertwopath = []string{testcontainertwo}
	var testleaftwopath = pathutil.CopyAppend(testcontainertwopath, testleaf)
	const testcontainerthree = "testcontainerthree"
	var testcontainerthreepath = []string{testcontainerthree}
	var testleafthreepath = pathutil.CopyAppend(testcontainerthreepath, testleaf)
	const testcontainerfour = "testcontainerfour"
	var testcontainerfourpath = []string{testcontainerfour}
	var testleaffourpath = pathutil.CopyAppend(testcontainerfourpath, testleaf)
	const testcontainerfive = "testcontainerfive"
	var testcontainerfivepath = []string{testcontainerfive}
	var testleaffivepath = pathutil.CopyAppend(testcontainerfivepath, testleaf)
	const testcontainersix = "testcontainersix"
	var testcontainersixpath = []string{testcontainersix}
	var testleafsixpath = pathutil.CopyAppend(testcontainersixpath, testleaf)

	const schema = `
container testcontainer {
	configd:priority "500";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafOne-one";
		configd:end "echo commitTestleafOne-two";
	}
	leaf teststring {
		type string;
		configd:end "echo commitTeststringOne";
		configd:priority "550";
	}
	leaf testboolean {
		type boolean;
		configd:priority "590";
		configd:end "echo commitTestbooleanOne";
	}
}
container testcontainertwo {
	configd:priority "900";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafTwo";
	}
}
container testcontainerthree {
	configd:priority "200";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafThree";
	}
}
container testcontainerfour {
	configd:priority "100";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafFour";
	}
}
container testcontainerfive {
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafFive";
	}
}
container testcontainersix {
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafSix";
	}
}
`
	const schemaTwo = `
container testcontainerfive {
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafFive";
	}
}
container testcontainersix {
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafSix";
	}
}
container testcontainer {
	configd:priority "500";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafOne";
	}
	leaf teststring {
		type string;
		configd:end "echo commitTeststringOne";
		configd:priority "590";
	}
	leaf testboolean {
		type boolean;
		configd:priority "550";
		configd:end "echo commitTestbooleanOne";
	}
}
container testcontainertwo {
	configd:priority "100";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafTwo";
	}
}
container testcontainerthree {
	configd:priority "600";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafThree";
	}
}
container testcontainerfour {
	configd:priority "900";
	leaf testleaf {
		type string;
		configd:end "echo commitTestleafFour";
	}
}
`
	const expctOne = `[]

[testcontainerfive testleaf foo]
commitTestleafFive

[testcontainersix testleaf foo]
commitTestleafSix

[testcontainerfour testleaf foo]
commitTestleafFour

[testcontainerthree testleaf foo]
commitTestleafThree

[testcontainer testleaf foo]
commitTestleafOne-one

[testcontainer testleaf foo]
commitTestleafOne-two

[testcontainer teststring foo]
commitTeststringOne

[testcontainer testboolean true]
commitTestbooleanOne

[testcontainertwo testleaf foo]
commitTestleafTwo

[]

`

	const expctTwo = `[]

[testcontainerfive testleaf foo]
commitTestleafFive

[testcontainersix testleaf foo]
commitTestleafSix

[testcontainertwo testleaf foo]
commitTestleafTwo

[testcontainer testleaf foo]
commitTestleafOne

[testcontainer testboolean true]
commitTestbooleanOne

[testcontainer teststring foo]
commitTeststringOne

[testcontainerthree testleaf foo]
commitTestleafThree

[testcontainerfour testleaf foo]
commitTestleafFour

[]

`
	tblPriOne := []ValidateOpTbl{
		{"", testleafpath, "foo", false},
		{"", testleafsixpath, "foo", false},
		{"", testleaftwopath, "foo", false},
		{"", testleafthreepath, "foo", false},
		{"", testleaffourpath, "foo", false},
		{"", testleaffivepath, "foo", false},
		{"", teststringpath, "foo", false},
		{"", testbooleanpath, "true", false},
	}
	tblPriTwo := []ValidateOpTbl{
		{"", teststringpath, "foo", false},
		{"", testbooleanpath, "true", false},
		{"", testleaffourpath, "foo", false},
		{"", testleafthreepath, "foo", false},
		{"", testleaffivepath, "foo", false},
		{"", testleaftwopath, "foo", false},
		{"", testleafsixpath, "foo", false},
		{"", testleafpath, "foo", false},
	}
	tblPriThree := []ValidateOpTbl{
		{"", testleaffivepath, "foo", false},
		{"", testleafsixpath, "foo", false},
		{"", testleafthreepath, "foo", false},
		{"", testleafpath, "foo", false},
		{"", testleaffourpath, "foo", false},
		{"", testleaftwopath, "foo", false},
		{"", testbooleanpath, "true", false},
		{"", teststringpath, "foo", false},
	}

	priTbl := []validatePriTbl{
		{schema, tblPriOne, expctOne},
		{schema, tblPriTwo, expctOne},
		{schema, tblPriThree, expctOne},
		{schemaTwo, tblPriOne, expctTwo},
		{schemaTwo, tblPriTwo, expctTwo},
		{schemaTwo, tblPriThree, expctTwo},
	}
	t.Log("priTbl")
	for key, _ := range priTbl {
		srv, sess := TstStartup(t, priTbl[key].schema, emptyconfig)
		ValidateOperationTable(t, sess, srv.Ctx, priTbl[key].opTbl, SET)
		validateCommitOrdering(t, sess, srv.Ctx, true, priTbl[key].expOut)
		sess.Kill()
	}
}

func TestPriorityInversion(t *testing.T) {
	const schema = `
container testcontainer {
	configd:priority 300;
	configd:end "echo testcontainer:300";
	container testcontainer {
		configd:priority 200;
		configd:end "echo testcontainer:200";
		container testcontainer {
			configd:priority 100;
			configd:end "echo testcontainer:100";
			leaf testleaf {
				type string;
			}
		}
	}
}
`
	const expOut = `[]

[testcontainer]
testcontainer:300

[testcontainer testcontainer]
testcontainer:200

[testcontainer testcontainer testcontainer]
testcontainer:100

[]

`
	var testleaffoo = []string{testcontainer, testcontainer, testcontainer, testleaf, "foo"}
	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testleaffoo, false)
	validateCommitOrdering(t, sess, srv.Ctx, true, expOut)
	sess.Kill()
}

// Check relative order of scripts being called for multiple list entries.
// This test was added following a problem with OSPF areas where they had
// begin and end scripts, but still needed to get area deletes for all
// areas send to ZebOS before any creates, or in the case where area 1 was
// being deleted then area 0 created, using same network, validation passed
// but the ZebOS back-end complained and failed post-commit.  Careful use
// of priority allowed the script order to be tweaked, and to ensure we don't
// accidentally change current order, these tests 'bake in' the expected
// behaviour.  Scenarios are:
//
// (a) Without begin and/or end, we expect all delete scripts for all list
//     entries to be called before the create scripts. (NoPrio test)
//
// (b) If we then add in begin and end scripts, we instead expect scripts
//     for each list entry to be called in full before the next list entry
//     is processed.  Within each list entry, delete is before create.
//     (NoPrioBeginAndEnd)
//
// (c) Finally, if we then add priority statements, we can cause all the
//     deletes to be called before all the creates.  Note that the order
//     relative to begin/end is not exactly optimal, but so long as it is
//     consistent and we can document it, then it will do.  Priority should
//     be going away with VCI and the idealised configuration model, so this
//     can be considered as just preserving legacy behaviour.
//     (PrioBeginAndEnd)
//
// For each scenario we test with both area 0 being deleted and area 1 created
// and then the reverse (area 1 deleted, area 0 created).

const ospfSchemaNoPrio = `
	container protocols {
	container ospf {
		presence "Enable OSPF";
		list area {
			key "tagnode";
			leaf tagnode {
				type uint32;
			}
			leaf-list network {
				type string;
				configd:create "echo 'create network'";
				configd:delete "echo 'delete network'";
			}
			configd:create "echo 'create area'";
			configd:delete "echo 'delete area'";
		}
	}
}`

func TestSetVsDeleteOrderNoPriorityMoveNet0To1(t *testing.T) {
	startupCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("0",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	setTbl := []ValidateOpTbl{
		createValOpTbl("set protocols ospf area 1 network 1.1.1.1",
			"protocols/ospf/area/1/network/1.1.1.1", SetPass),
	}
	delTbl := []ValidateOpTbl{
		createValOpTbl("delete protocols ospf area 0",
			"protocols/ospf/area/0", SetPass),
	}

	expCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("1",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	expOut := []string{`[]

[protocols ospf area 0 network 1.1.1.1]
'delete network'

[protocols ospf area 0]
'delete area'

[protocols ospf area 1]
'create area'

[protocols ospf area 1 network 1.1.1.1]
'create network'

[]

`}

	tests := []xpathTestEntry{
		newXpathTestEntry(setTbl, delTbl, CommitPass, expCfg, expOut),
	}

	runXpathTestsCheckOutput(t, ospfSchemaNoPrio, startupCfg, tests)
}

func TestSetVsDeleteOrderNoPriorityMoveNet1To0(t *testing.T) {
	startupCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("1",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	setTbl := []ValidateOpTbl{
		createValOpTbl("set protocols ospf area 0 network 1.1.1.1",
			"protocols/ospf/area/0/network/1.1.1.1", SetPass),
	}
	delTbl := []ValidateOpTbl{
		createValOpTbl("delete protocols ospf area 1",
			"protocols/ospf/area/1", SetPass),
	}

	expCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("0",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	expOut := []string{`[]

[protocols ospf area 1 network 1.1.1.1]
'delete network'

[protocols ospf area 1]
'delete area'

[protocols ospf area 0]
'create area'

[protocols ospf area 0 network 1.1.1.1]
'create network'

[]

`}

	tests := []xpathTestEntry{
		newXpathTestEntry(setTbl, delTbl, CommitPass, expCfg, expOut),
	}

	runXpathTestsCheckOutput(t, ospfSchemaNoPrio, startupCfg, tests)
}

const ospfSchemaNoPrioBeginAndEnd = `
	container protocols {
	container ospf {
		presence "Enable OSPF";
		list area {
			key "tagnode";
			leaf tagnode {
				type uint32;
			}
			leaf-list network {
				type string;
				configd:create "echo 'create network'";
				configd:delete "echo 'delete network'";
			}
			configd:begin "echo 'begin area'";
			configd:create "echo 'create area'";
			configd:delete "echo 'delete area'";
			configd:end "echo 'end area'";
		}
	}
}`

func TestSetVsDeleteOrderNoPriorityBeginAndEndMoveNet0To1(t *testing.T) {
	startupCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("0",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	setTbl := []ValidateOpTbl{
		createValOpTbl("set protocols ospf area 1 network 1.1.1.1",
			"protocols/ospf/area/1/network/1.1.1.1", SetPass),
	}
	delTbl := []ValidateOpTbl{
		createValOpTbl("delete protocols ospf area 0",
			"protocols/ospf/area/0", SetPass),
	}

	expCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("1",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	expOut := []string{`[]

[protocols ospf area 0]
'begin area'

[protocols ospf area 0 network 1.1.1.1]
'delete network'

[protocols ospf area 0]
'delete area'

[protocols ospf area 0]
'end area'

[protocols ospf area 1]
'begin area'

[protocols ospf area 1]
'create area'

[protocols ospf area 1 network 1.1.1.1]
'create network'

[protocols ospf area 1]
'end area'

[]

`}

	tests := []xpathTestEntry{
		newXpathTestEntry(setTbl, delTbl, CommitPass, expCfg, expOut),
	}

	runXpathTestsCheckOutput(t, ospfSchemaNoPrioBeginAndEnd, startupCfg, tests)
}

func TestSetVsDeleteOrderNoPriorityBeginAndEndMoveNet1To0(t *testing.T) {
	startupCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("1",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	setTbl := []ValidateOpTbl{
		createValOpTbl("set protocols ospf area 0 network 1.1.1.1",
			"protocols/ospf/area/0/network/1.1.1.1", SetPass),
	}
	delTbl := []ValidateOpTbl{
		createValOpTbl("delete protocols ospf area 1",
			"protocols/ospf/area/1", SetPass),
	}

	expCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("0",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	expOut := []string{`[]

[protocols ospf area 0]
'begin area'

[protocols ospf area 0]
'create area'

[protocols ospf area 0 network 1.1.1.1]
'create network'

[protocols ospf area 0]
'end area'

[protocols ospf area 1]
'begin area'

[protocols ospf area 1 network 1.1.1.1]
'delete network'

[protocols ospf area 1]
'delete area'

[protocols ospf area 1]
'end area'

[]

`}

	tests := []xpathTestEntry{
		newXpathTestEntry(setTbl, delTbl, CommitPass, expCfg, expOut),
	}

	runXpathTestsCheckOutput(t, ospfSchemaNoPrioBeginAndEnd, startupCfg, tests)
}

const ospfSchemaPrio = `
	container protocols {
	container ospf {
		list area {
			key "tagnode";
			leaf tagnode {
				type uint32;
			}
			configd:priority "630";
			leaf-list network {
				type string;
				configd:create "echo 'create network'";
				configd:delete "echo 'delete network'";
				configd:priority "631";
			}
			configd:begin "echo 'begin area'";
			configd:create "echo 'create area'";
			configd:delete "echo 'delete area'";
			configd:end "echo 'end area'";
		}
	}
}`

func TestSetVsDeleteOrderPriorityMoveNet0To1(t *testing.T) {
	// set protocols ospf area 0 network 1.1.1.1/32
	startupCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("0",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	// delete protocols ospf area 0
	// set protocols ospf area 1 network 1.1.1.1/32
	setTbl := []ValidateOpTbl{
		createValOpTbl("set protocols ospf area 1 network 1.1.1.1",
			"protocols/ospf/area/1/network/1.1.1.1", SetPass),
	}
	delTbl := []ValidateOpTbl{
		createValOpTbl("delete protocols ospf area 0",
			"protocols/ospf/area/0", SetPass),
	}

	expCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("1",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	expOut := []string{`[]

[protocols ospf area 0 network 1.1.1.1]
'delete network'

[protocols ospf area 0]
'begin area'

[protocols ospf area 0]
'delete area'

[protocols ospf area 0]
'end area'

[protocols ospf area 1]
'begin area'

[protocols ospf area 1]
'create area'

[protocols ospf area 1]
'end area'

[protocols ospf area 1 network 1.1.1.1]
'create network'

[]

`}

	tests := []xpathTestEntry{
		newXpathTestEntry(setTbl, delTbl, CommitPass, expCfg, expOut),
	}

	runXpathTestsCheckOutput(t, ospfSchemaPrio, startupCfg, tests)
}

func TestSetVsDeleteOrderPriorityMoveNet1To0(t *testing.T) {
	startupCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("1",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	setTbl := []ValidateOpTbl{
		createValOpTbl("set protocols ospf area 0 network 1.1.1.1",
			"protocols/ospf/area/0/network/1.1.1.1", SetPass),
	}
	delTbl := []ValidateOpTbl{
		createValOpTbl("delete protocols ospf area 1",
			"protocols/ospf/area/1", SetPass),
	}

	expCfg :=
		Cont("protocols",
			Cont("ospf",
				List("area",
					ListEntry("0",
						LeafList("network",
							LeafListEntry("1.1.1.1"))))))

	expOut := []string{`[]

[protocols ospf area 1 network 1.1.1.1]
'delete network'

[protocols ospf area 1]
'begin area'

[protocols ospf area 1]
'delete area'

[protocols ospf area 1]
'end area'

[protocols ospf area 0]
'begin area'

[protocols ospf area 0]
'create area'

[protocols ospf area 0]
'end area'

[protocols ospf area 0 network 1.1.1.1]
'create network'

[]

`}

	tests := []xpathTestEntry{
		newXpathTestEntry(setTbl, delTbl, CommitPass, expCfg, expOut),
	}

	runXpathTestsCheckOutput(t, ospfSchemaPrio, startupCfg, tests)
}
