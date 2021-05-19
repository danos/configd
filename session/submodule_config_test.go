// Copyright (c) 2017,2019, 2021, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// This file contains tests for configuration of submodules where these
// belong to different components to the parent module.

package session_test

import (
	"testing"

	"github.com/danos/config/schema"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/vci/conf"
	"github.com/danos/yang/testutils"
)

const submoduleHasNoPrefix = ""

var parentTestComp = conf.CreateTestDotComponentFile("parent").
	AddBaseModel()
var childTestComp = conf.CreateTestDotComponentFile("child").
	AddBaseModel().
	SetAfter("parent")
var grandchildTestComp = conf.CreateTestDotComponentFile("grandchild").
	AddBaseModel().
	SetAfter("child")

var submoduleSchemas = []*sessiontest.TestSchema{
	sessiontest.NewTestSchema("vyatta-test-parent-v1", "parent").
		AddInclude("vyatta-test-child-v1").
		AddInclude("vyatta-test-grandchild-v1").
		AddSchemaSnippet(parentSchema),
	sessiontest.NewTestSchema("vyatta-test-child-v1", submoduleHasNoPrefix).
		AddBelongsTo("vyatta-test-parent-v1", "parent").
		AddSchemaSnippet(childSchema),
	sessiontest.NewTestSchema(
		"vyatta-test-grandchild-v1", submoduleHasNoPrefix).
		AddBelongsTo("vyatta-test-parent-v1", "parent").
		AddInclude("vyatta-test-child-v1").
		AddSchemaSnippet(grandchildSchema),
}

const parentSchema = `
container parentCont {
	leaf parentLeaf {
		type string;
	}
}`

const childSchema = `
container childCont {
	leaf childLeaf {
		type string;
	}
}`

const grandchildSchema = `
augment /childCont {
	container gcCont {
		leaf gcLeaf {
			type string;
		}
	}
}`

var submoduleConfig = testutils.Root(
	testutils.Cont("parentCont",
		testutils.Leaf("parentLeaf", "parentValue")),
	testutils.Cont("childCont",
		testutils.Leaf("childLeaf", "childValue"),
		testutils.Cont("gcCont",
			testutils.Leaf("gcLeaf", "gcValue"))))

const (
	parentCfgJson = "{\"vyatta-test-parent-v1:parentCont\":{\"parentLeaf\":\"parentValue\"}}"
	childCfgJson  = "{\"vyatta-test-child-v1:childCont\":{\"childLeaf\":\"childValue\"}}"
	gcCfgJson     = "{\"vyatta-test-child-v1:childCont\":{\"vyatta-test-grandchild-v1:gcCont\":{\"gcLeaf\":\"gcValue\"}}}"
)

func TestConfigSetToSubmodules(t *testing.T) {
	// Parent, child, grandchild each with own config and, in grandchild
	// case, augmented into child config too.
	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(submoduleSchemas).
		SetComponents(
			conf.BaseModelSet,
			[]string{
				parentTestComp.String(),
				childTestComp.String(),
				grandchildTestComp.String()})
	srv, sess := ts.Init()

	srv.LoadConfig(t, submoduleConfig, sess)

	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("Errors: %v\n", errs)
		return
	}

	ts.CheckCompLogEntries(
		"Config Set to Submodules",
		schema.SetRunning,
		schema.NewTestLogEntry("SetRunning", "net.vyatta.test.parent",
			parentCfgJson),
		schema.NewTestLogEntry("SetRunning", "net.vyatta.test.child",
			childCfgJson),
		schema.NewTestLogEntry("SetRunning", "net.vyatta.test.grandchild",
			gcCfgJson))
}

// submodules not assigned to provisiond
// config addressability to module:submodule
//   - how is config marked as belonging to submodule?
// module with multiple submodules, only some of which belong to diff
//   components, so we ensure some submodule config gets to parent module
// Does VCI file specify Module, with submodule name, or specify as Submodule?
//   If one comp has module and one submodule, and another comp has other
//     submodule, ensure this works

func TestStateMuxFromSubmodules(t *testing.T) {
	// Check interwoven trees correctly reconstituted.
	// Also check only live components / services queried.
}

func TestRPCDemuxForSubmodules(t *testing.T) {
	t.Skipf("TBD")
}

func TestNotificationDemuxForSubmodules(t *testing.T) {
	t.Skipf("TBD")
}
