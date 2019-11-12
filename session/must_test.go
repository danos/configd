// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests relating to the must statements on non-presence
// containers.  Other aspects of must statements are tested in other files.

package session_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/yang/testutils"
)

func createMustTestSchema(schemaSnippet string) *sessiontest.TestSchema {
	return sessiontest.NewTestSchema("vyatta-test-must-v1", "must").
		AddSchemaSnippet(schemaSnippet)
}

type expErr struct {
	path,
	msg string
}

func verifyMustOnNPContErrors(
	t *testing.T,
	schemaSnippet,
	initConfig,
	failConfig string,
	expErrsInAnyOrder []expErr,
) {
	srv, sess := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef([]*sessiontest.TestSchema{
			createMustTestSchema(schemaSnippet)}).
		SetConfig(initConfig).
		Init()

	srv.LoadConfig(t, failConfig, sess)

	_, actErrs, ok := sess.Validate(srv.Ctx)
	if ok {
		t.Fatalf("Expected commit to fail due to must on NP container")
		return
	}
	if len(actErrs) != len(expErrsInAnyOrder) {
		t.Fatalf("Unexpected number of errors.  Expected %d, got %d\n",
			len(expErrsInAnyOrder), len(actErrs))
	}
	for _, expErr := range expErrsInAnyOrder {
		matchMsg, matchPath := false, false
		for _, actErr := range actErrs {
			if strings.Contains(actErr.Error(), expErr.msg) {
				matchMsg = true
			}
			if strings.Contains(
				actErr.Error(), fmt.Sprintf("Error: %s:", expErr.path)) {
				matchPath = true
			}
		}

		if !matchMsg {
			t.Logf("Expected error not found: '%s'\n", expErr.msg)
			t.Fatalf("Actual errors:\n%v\n", actErrs)
		}
		if !matchPath {
			t.Logf("Expected error found but wrong path: '%s (%s)'\n",
				expErr.msg, expErr.path)
			t.Fatalf("Actual errors:\n%v\n", actErrs)
		}
	}
}

func genMustErr(mustStmt string) string {
	return fmt.Sprintf("'must' condition is false: '%s'", mustStmt)
}

// Test that initial config is accepted with NP container configured, then
// remove config to get must statement to fail.
const topLevelNPContSchemaSnippet = `
container topNPCont {
	must "topNPLeaf";
	leaf topNPLeaf {
		type string;
	}
}
container topPCont {
	presence "Test presence container";
	leaf topPLeaf {
		type string;
	}
}`

var topPContConfig = testutils.Root(
	testutils.Cont("topPCont",
		testutils.Leaf("topPLeaf", "configured")))

var topNPContConfig = testutils.Root(
	testutils.Cont("topNPCont",
		testutils.Leaf("topNPLeaf", "configured")))

func TestMustTopLevelNPContainer(t *testing.T) {

	verifyMustOnNPContErrors(t,
		topLevelNPContSchemaSnippet,
		topNPContConfig,
		topPContConfig,
		[]expErr{
			{
				path: "/topNPCont",
				msg:  genMustErr("topNPLeaf"),
			},
		})

}

// Verify musts several levels down within non-presence containers are run.
const nestedNPContSchemaSnippet = `
container topNPCont {
	container level2NPCont {
		must "level2Leaf" {
			error-message "Need to configure level2Leaf";
		}
		leaf level2Leaf {
			type string;
		}
		container level3NPCont {
			must "level3Leaf";
			leaf level3Leaf {
				type string;
			}
		}
	}
}
container topPCont {
	presence "Test presence container";
	leaf topPLeaf {
		type string;
	}
}`

var nestedPassMustConfig = testutils.Root(
	testutils.Cont("topNPCont",
		testutils.Cont("level2NPCont",
			testutils.Leaf("level2Leaf", "configured"),
			testutils.Cont("level3NPCont",
				testutils.Leaf("level3Leaf", "configured")))))

func TestMustNestedNPContainer(t *testing.T) {

	verifyMustOnNPContErrors(t,
		nestedNPContSchemaSnippet,
		nestedPassMustConfig,
		topPContConfig,
		[]expErr{
			{
				path: "/topNPCont/level2NPCont",
				msg:  "Need to configure level2Leaf",
			},
			{
				path: "/topNPCont/level2NPCont/level3NPCont",
				msg:  genMustErr("level3Leaf"),
			},
		})
}

// Verify we run all must statements on a single non-presence container.
const multipleMustsSchemaSnippet = `
container topNPCont {
	must "topNPLeaf";
	must "../topPCont/topPLeaf";
	leaf topNPLeaf {
		type string;
	}
}
container topPCont {
	presence "Test presence container";
	leaf topPLeaf {
		type string;
	}
}`

var multipleMustsPassConfig = testutils.Root(
	testutils.Cont("topNPCont",
		testutils.Leaf("topNPLeaf", "configured")),
	testutils.Cont("topPCont",
		testutils.Leaf("topPLeaf", "alsoConfigured")))

func TestMustsMultipleOnSameContainer(t *testing.T) {

	verifyMustOnNPContErrors(t,
		multipleMustsSchemaSnippet,
		multipleMustsPassConfig,
		emptyConfig,
		[]expErr{
			{
				path: "/topNPCont",
				msg:  genMustErr("topNPLeaf"),
			},
			{
				path: "/topNPCont",
				msg:  genMustErr("../topPCont/topPLeaf"),
			},
		})
}

// Verify we run must statements on multiple non-presence containers at the
// same level in the tree.
const multipleNPContsSchemaSnippet = `
container topPCont {
	presence "Test presence container";
	leaf topPLeaf {
		type string;
	}
	container npCont1 {
		must "../topPLeaf = 'unconfigured value'" {
			error-message "Wrong value for topPLeaf";
		}
		leaf npcLeaf1 {
			type string;
		}
	}
	container npCont2 {
		must "../npCont1";
		leaf npcLeaf2 {
			type string;
		}
	}
}`

func TestMustsMultipleNPContainersSameLevel(t *testing.T) {

	verifyMustOnNPContErrors(t,
		multipleNPContsSchemaSnippet,
		emptyConfig,
		topPContConfig,
		[]expErr{
			{
				path: "/topPCont/npCont2",
				msg:  genMustErr("../npCont1"),
			},
			{
				path: "/topPCont/npCont1",
				msg:  "Wrong value for topPLeaf",
			},
		})
}

// Check that we can't refer to ourselves and pass.  This test ensures that
// the node we create ephemerally to run the must statement on can't be seen
// by any XPATH operations.
const ephemeralNodeRefSchemaSnippet = `
container topPCont {
	presence "Test presence container";
	leaf topPLeaf {
		type string;
	}
	container npContDot {
		must ".";
		leaf npcLeaf {
			type string;
		}
	}
	container npContCurrent {
		must "current()";
		leaf npcLeaf {
			type string;
		}
	}
	container npContUpDownRef {
		must "../npContUpDownRef";
		leaf npcLeaf1 {
			type string;
		}
	}
	container npContChildRef {
		must "npcLeaf";
		leaf npcLeaf {
			type string;
		}
	}
	container npContDotChildRef {
		must "./npcLeaf";
		leaf npcLeaf {
			type string;
		}
	}
}`

func TestMustReferencingEphemeralNode(t *testing.T) {

	verifyMustOnNPContErrors(t,
		ephemeralNodeRefSchemaSnippet,
		emptyConfig,
		topPContConfig,
		[]expErr{
			{
				path: "/topPCont/npContDot",
				msg:  genMustErr("."),
			},
			{
				path: "/topPCont/npContCurrent",
				msg:  genMustErr("current()"),
			},
			{
				path: "/topPCont/npContUpDownRef",
				msg:  genMustErr("../npContUpDownRef"),
			},
			{
				path: "/topPCont/npContChildRef",
				msg:  genMustErr("npcLeaf"),
			},
			{
				path: "/topPCont/npContDotChildRef",
				msg:  genMustErr("./npcLeaf"),
			},
		})
}

// Where musts are needed on non-presence containers, but are only to be run
// when the container is configured, the 'not(.) or ...' style is used.  This
// test verifies that this works as expected.
//
// This shows that the 'recommended format' given out by the Configd team
// to YANG authors when we weren't running the validation on unconfigured
// NP containers works as expected.
const recommendedFormatSchemaSnippet = `
container topNPCont {
	leaf topNPLeaf {
		type string;
	}
	container npCont {
		must "not(.) or mandatoryLeaf";
		must "not(current()) or mandatoryLeaf";
		leaf mandatoryLeaf {
			type string;
		}
		leaf optionalLeaf {
			type string;
		}
	}
}`

var optionalLeafConfig = testutils.Root(
	testutils.Cont("topNPCont",
		testutils.Cont("npCont",
			testutils.Leaf("optionalLeaf", "someValue"))))

func TestMustRecommendedFormatForNPContainer(t *testing.T) {

	verifyMustOnNPContErrors(t,
		recommendedFormatSchemaSnippet,
		emptyConfig,
		optionalLeafConfig,
		[]expErr{
			{
				path: "/topNPCont/npCont",
				msg:  genMustErr("not(.) or mandatoryLeaf"),
			},
			{
				path: "/topNPCont/npCont",
				msg:  genMustErr("not(current()) or mandatoryLeaf"),
			},
		})
}
