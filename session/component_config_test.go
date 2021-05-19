// Copyright (c) 2017-2019,2021, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests relating to the ModelSet extension, and more
// specifically to handling of configuration get/set operations across
// multiple components.

package session_test

import (
	"strings"
	"testing"

	"github.com/danos/config/data"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/configd/session"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
	"github.com/danos/vci/conf"
	"github.com/danos/yang/testutils"
)

const (
	emptyConfig  = ""
	emptyCfgJson = "{}"
)

var firstTestComp = conf.CreateTestDotComponentFile("first").
	AddBaseModel()
var secondTestComp = conf.CreateTestDotComponentFile("second").
	AddBaseModel()
var thirdTestComp = conf.CreateTestDotComponentFile("third").
	AddBaseModel().
	SetBefore("first").
	SetAfter("second")

var schemas = []*sessiontest.TestSchema{
	sessiontest.NewTestSchema("vyatta-test-first-v1", "first").
		AddSchemaSnippet(firstSchema),
	sessiontest.NewTestSchema("vyatta-test-second-v1", "second").
		AddSchemaSnippet(secondSchema),
	sessiontest.NewTestSchema("vyatta-test-third-v1", "third").
		AddImport("vyatta-test-first-v1", "first").
		AddSchemaSnippet(thirdSchema),
}

const firstSchema = `
container first {
	leaf firstLeaf {
		type string;
	}
	container empty-leaf {
		leaf empty-leaf {
			type empty;
		}
	}
	container big-number {
		leaf big-number {
			type int64;
		}
	}
	container little-number {
		leaf little-number {
			type int8;
		}
	}
	list userList {
		key name;
		leaf name {
			type string;
		}
		ordered-by user;
	}
	list systemList {
		key name;
		leaf name {
			type string;
		}
		ordered-by system;
	}
}`

const secondSchema = `
container second {
	leaf secondLeaf {
		type string;
	}
}`

const thirdSchema = `
augment /first:first {
	container third {
		leaf thirdLeaf {
			type string;
		}
	}
}`

var config = testutils.Root(
	testutils.Cont("first",
		testutils.Leaf("firstLeaf", "someValue"),
		testutils.Cont("third",
			testutils.Leaf("thirdLeaf", "anotherValue"))),
	testutils.Cont("second",
		testutils.Leaf("secondLeaf", "someValue")))

const (
	firstCompCfgJson  = "{\"vyatta-test-first-v1:first\":{\"firstLeaf\":\"someValue\"}}"
	secondCompCfgJson = "{\"vyatta-test-second-v1:second\":{\"secondLeaf\":\"someValue\"}}"
	thirdCompCfgJson  = "{\"vyatta-test-first-v1:first\":{\"vyatta-test-third-v1:third\":{\"thirdLeaf\":\"anotherValue\"}}}"
)

type compConfigTest struct {
	name       string
	config     []string
	logEntries []schema.TestLogEntry
}

func serialiseCfg(cfgTree *data.Node, ms schema.ModelSet) string {

	root := union.NewNode(cfgTree, nil, ms, nil, 0)
	var b union.StringWriter
	root.Serialize(&b, nil, union.IncludeDefaults)
	return b.String()
}

// Verify Before and After requirements are met for components.
func TestConfigSetOrder(t *testing.T) {

	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(schemas).
		SetComponents(
			conf.BaseModelSet,
			[]string{
				firstTestComp.String(),
				secondTestComp.String(),
				thirdTestComp.String()})
	srv, sess := ts.Init()

	srv.LoadConfig(t, config, sess)

	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("Errors: %v\n", errs)
		return
	}

	ts.CheckCompLogEntries(
		"Config Set Order",
		schema.SetRunning,
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.second", secondCompCfgJson),
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.third", thirdCompCfgJson),
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.first", firstCompCfgJson))
}

func TestConfigSubsequentDeleteOrder(t *testing.T) {

	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(schemas).
		SetComponents(
			conf.BaseModelSet,
			[]string{
				firstTestComp.String(),
				secondTestComp.String(),
				thirdTestComp.String()}).
		SetConfig(config)
	srv, sess := ts.Init()

	srv.LoadConfig(t, emptyConfig, sess)

	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("Errors: %v\n", errs)
		return
	}

	ts.CheckCompLogEntries(
		"Config Subsequent Delete Order",
		schema.SetRunning,
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.second", emptyCfgJson),
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.third", emptyCfgJson),
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.first", emptyCfgJson))
}

func TestConfigActionScriptsCalledInOrder(t *testing.T) {
	// These will only apply to provisiond, using existing code, so there
	// may not be a lot of point testing these.
	t.Skipf("TBD")
}

func TestConfigGetRecombinedCorrectly(t *testing.T) {
	// Nested augments (parent, augment, sub-augment) with components
	// ordered any which way.

	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(schemas).
		SetComponents(
			conf.BaseModelSet,
			[]string{
				firstTestComp.String(),
				secondTestComp.String(),
				thirdTestComp.String()}).
		SetConfig(emptyConfig)
	srv, sess := ts.Init()

	// Where we rely on a commit to set up the config in the test CompMgr,
	// we need to explicitly commit as the initial setup doesn't commit in the
	// same way.
	srv.LoadConfig(t, config, sess)

	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("Unable to commit: %s\n", errs)
	}

	cfgTree, err := ts.GetCompMgr().ComponentGetRunning(
		srv.Ms, union.UnmarshalJSONConfigsWithoutValidation)
	if err != nil {
		t.Fatalf("Unable to get running config: %s", err.Error())
		return
	}
	actCfg := serialiseCfg(cfgTree, srv.Ms)
	if actCfg != config {
		t.Fatalf("Config mismatch.\nExp:\n%s\nGot:\n%s\n", config, actCfg)
		return
	}
}

var firstCompOrderedListCfg = testutils.Root(
	testutils.Cont("first",
		testutils.List("systemList",
			testutils.ListEntry("alpha"),
			testutils.ListEntry("bravo"),
			testutils.ListEntry("charlie"),
			testutils.ListEntry("delta")),
		testutils.List("userList",
			testutils.ListEntry("firstEntry"),
			testutils.ListEntry("secondEntry"),
			testutils.ListEntry("thirdEntry"),
			testutils.ListEntry("fourthEntry"))))

func TestConfigForOrderedListsRetrievedCorrectly(t *testing.T) {
	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(schemas).
		SetComponents(conf.BaseModelSet, []string{firstTestComp.String()}).
		SetConfig(emptyConfig)
	srv, sess := ts.Init()

	srv.LoadConfig(t, firstCompOrderedListCfg, sess)

	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("Unable to commit: %s\n", errs)
	}

	cfgTree, err := ts.GetCompMgr().ComponentGetRunning(
		srv.Ms, union.UnmarshalJSONConfigsWithoutValidation)
	if err != nil {
		t.Fatalf("Unable to get running config: %s", err.Error())
		return
	}
	actCfg := serialiseCfg(cfgTree, srv.Ms)
	if actCfg != firstCompOrderedListCfg {
		t.Fatalf("Config mismatch.\nExp:\n%s\nGot:\n%s\n",
			firstCompOrderedListCfg, actCfg)
		return
	}
}

var defaultCfgTestComp = conf.CreateTestDotComponentFile("default").
	AddModel(
		conf.BaseNameAndModelPrefix+".default",
		nil,
		[]string{conf.BaseModelSet}).
	SetDefault()

var mainCfgTestComp = conf.CreateTestDotComponentFile("main").
	AddBaseModel().
	SetAfter("default")

var augmentCfgTestComp = conf.CreateTestDotComponentFile("augment").
	AddBaseModel().
	SetAfter("main")

const defaultCfgTestSchema = `
	container dfltCont {
	leaf dfltLeaf {
		type string;
	}
}`

const mainCfgTestSchema = `
container mainPCont {
    presence "Test presence container";
	leaf mainPLeaf {
		type string;
	}
	list mainList {
		key name;
		leaf name {
			type string;
		}
		leaf mainListLeaf {
			type int16;
		}
	}
}
container mainNPCont {
leaf mainNPLeaf {
		type string;
	}
}`

const augmentCfgTestSchema = `
augment /main:mainPCont {
	leaf augPLeaf {
		type string;
	}
	leaf-list augPLeafList {
		type string;
	}
}
augment /main:mainNPCont {
	leaf augNPLeaf {
		type string;
	}
	leaf-list augNPLeafList {
		type string;
	}
}
augment /main:mainPCont/main:mainList {
	leaf augListLeaf {
		type boolean;
	}
}`

var cfgTestSchemas = []*sessiontest.TestSchema{
	sessiontest.NewTestSchema("vyatta-test-default-v1", "default").
		AddSchemaSnippet(defaultCfgTestSchema),
	sessiontest.NewTestSchema("vyatta-test-main-v1", "main").
		AddSchemaSnippet(mainCfgTestSchema),
	sessiontest.NewTestSchema("vyatta-test-augment-v1", "augment").
		AddImport("vyatta-test-main-v1", "main").
		AddSchemaSnippet(augmentCfgTestSchema),
}

const (
	defaultTestCompCfgJson = "{}"
	mainOnlyCompCfgJson    = "{\"vyatta-test-main-v1:mainNPCont\":{\"mainNPLeaf\":\"npleafvalue\"}}"
)

var mainOnlyCfg = testutils.Root(
	testutils.Cont("mainNPCont",
		testutils.Leaf("mainNPLeaf", "npleafvalue")))

func TestConfigCreateOnlyWrittenToConfiguredComponents(t *testing.T) {
	// 2 components, one configured
	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(cfgTestSchemas).
		SetComponents(
			conf.BaseModelSet,
			[]string{
				defaultCfgTestComp.String(),
				mainCfgTestComp.String(),
				augmentCfgTestComp.String()})
	srv, sess := ts.Init()

	srv.LoadConfig(t, mainOnlyCfg, sess)

	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("Errors: %v\n", errs)
		return
	}

	ts.CheckCompLogEntries(
		"Config Create Only Written To Configured Components",
		schema.SetRunning,
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.default", defaultTestCompCfgJson),
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.main", mainOnlyCompCfgJson))
}

var defaultCompLogEntry = schema.NewTestLogEntry(
	schema.SetRunning, "net.vyatta.test.default",
	defaultTestCompCfgJson)
var mainPContLogEntry = schema.NewTestLogEntry(
	schema.SetRunning, "net.vyatta.test.main",
	"{\"vyatta-test-main-v1:mainPCont\":{}}")
var mainPContEmptyLogEntry = schema.NewTestLogEntry(
	schema.SetRunning, "net.vyatta.test.main", "{}")
var mainNPContLogEntry = schema.NewTestLogEntry(
	schema.SetRunning, "net.vyatta.test.main",
	"{\"vyatta-test-main-v1:mainNPCont\":{}}")
var augCompEmptyLogEntry = schema.NewTestLogEntry(
	schema.SetRunning, "net.vyatta.test.augment", "{}")

// This table allows us to efficiently work through a set of changes to a
// presence container with augmented child nodes in a different component and
// verify that we get updates for the expected components only.
var presenceContTests = []compConfigTest{
	{
		name: "Create main P container",
		config: []string{
			"set mainPCont"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			mainPContLogEntry,
		},
	},
	{
		name: "Create child of P container (P and child diff comps)",
		config: []string{
			"set mainPCont/augPLeaf/augPLeafValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{\"vyatta-test-augment-v1:augPLeaf\":\"augPLeafValue\"}}"),
		},
	},
	{
		name: "Update child of P container (P and child diff comps)",
		config: []string{
			"set mainPCont/augPLeaf/augPLeafNewValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{\"vyatta-test-augment-v1:augPLeaf\":\"augPLeafNewValue\"}}"),
		},
	},
	{
		name: "Create 2nd child of P container (P and child diff comps)",
		config: []string{
			"set mainPCont/augPLeafList/augPLeafListValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{"+
					"\"vyatta-test-augment-v1:augPLeaf\":\"augPLeafNewValue\","+
					"\"vyatta-test-augment-v1:augPLeafList\":[\"augPLeafListValue\"]"+
					"}}"),
		},
	},
	{
		name: "Delete 2nd child of P container (P and child diff comps)",
		config: []string{
			"delete mainPCont/augPLeafList/augPLeafListValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{\"vyatta-test-augment-v1:augPLeaf\":\"augPLeafNewValue\"}}"),
		},
	},
	{
		name: "Delete 1st child of P container (P and child diff comps)",
		config: []string{
			"delete mainPCont/augPLeaf/augPLeafNewValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			augCompEmptyLogEntry,
		},
	},
	{
		name: "Delete P container",
		config: []string{
			"delete mainPCont"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			mainPContEmptyLogEntry,
		},
	},
	{
		name: "Create child of P container (diff comps)",
		config: []string{
			"set mainPCont/augPLeaf/augPLeafValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			mainPContLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{\"vyatta-test-augment-v1:augPLeaf\":\"augPLeafValue\"}}"),
		},
	},
	{
		name: "Delete P container",
		config: []string{
			"delete mainPCont"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			mainPContEmptyLogEntry,
			augCompEmptyLogEntry,
		},
	},
}

func TestConfigWrittenToCorrectComponentsForPresenceContainer(t *testing.T) {
	runTests(t, presenceContTests)
}

// Same idea but for a non-presence container which never receives
// notifications for its own creation / deletion - the component is only
// notified if children in the same component are configured.
var nonpresenceContTests = []compConfigTest{
	{
		name: "Create child of NP container (NP and child diff comps)",
		config: []string{
			"set mainNPCont/augNPLeaf/augNPLeafValue"},
		logEntries: []schema.TestLogEntry{
			// NB: as the only config in 'main' is a non-presence
			//     container, there is no actual config to write to
			//     the component.
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainNPCont\":{\"vyatta-test-augment-v1:augNPLeaf\":\"augNPLeafValue\"}}"),
		},
	},
	{
		name: "Update child of NP container (NP and child diff comps)",
		config: []string{
			"set mainNPCont/augNPLeaf/augNPLeafNewValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainNPCont\":{\"vyatta-test-augment-v1:augNPLeaf\":\"augNPLeafNewValue\"}}"),
		},
	},
	{
		name: "Create 2nd child of NP container (NP and child diff comps)",
		config: []string{
			"set mainNPCont/augNPLeafList/augNPLeafListValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainNPCont\":{\"vyatta-test-augment-v1:augNPLeaf\":\"augNPLeafNewValue\","+
					"\"vyatta-test-augment-v1:augNPLeafList\":[\"augNPLeafListValue\"]}}"),
		},
	},
	{
		name: "Delete 2nd child of NP container (NP and child diff comps)",
		config: []string{
			"delete mainNPCont/augNPLeaf/augNPLeafNewValue"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainNPCont\":"+
					"{\"vyatta-test-augment-v1:augNPLeafList\":[\"augNPLeafListValue\"]}}"),
		},
	},
	{
		name: "Delete 1st child of NP container",
		config: []string{
			"delete mainNPCont/augNPLeafList"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			augCompEmptyLogEntry,
		},
	},
}

func TestConfigWrittenToCorrectComponentsForNonPresenceContainer(t *testing.T) {
	runTests(t, nonpresenceContTests)
}

var listTests = []compConfigTest{
	{
		name: "Create first list entry",
		config: []string{
			"set mainPCont/mainList/firstEntry"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.main",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":[{\"name\":\"firstEntry\"}]}}"),
		},
	},
	{
		name: "Augment first list entry",
		config: []string{
			"set mainPCont/mainList/firstEntry/augListLeaf/true"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"firstEntry\","+
					"\"vyatta-test-augment-v1:augListLeaf\":true}]}}"),
		},
	},
	{
		name: "Update first list entry",
		config: []string{
			"set mainPCont/mainList/firstEntry/mainListLeaf/666"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.main",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"firstEntry\","+
					"\"mainListLeaf\":666}]}}"),
		},
	},
	{
		name: "Add second list entry",
		config: []string{
			"set mainPCont/mainList/secondEntry"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.main",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"firstEntry\","+
					"\"mainListLeaf\":666},"+
					"{\"name\":\"secondEntry\"}]}}"),
		},
	},
	{
		name: "Add third list entry by adding augmented leaf directly",
		config: []string{
			"set mainPCont/mainList/thirdEntry/augListLeaf/false"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.main",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"firstEntry\","+
					"\"mainListLeaf\":666},"+
					"{\"name\":\"secondEntry\"},"+
					"{\"name\":\"thirdEntry\"}]}}"),
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"firstEntry\","+
					"\"vyatta-test-augment-v1:augListLeaf\":true},"+
					"{\"name\":\"thirdEntry\","+
					"\"vyatta-test-augment-v1:augListLeaf\":false}]}}"),
		},
	},
	{
		name: "Remove augment on first list entry",
		config: []string{
			"delete mainPCont/mainList/firstEntry/augListLeaf/true"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"thirdEntry\","+
					"\"vyatta-test-augment-v1:augListLeaf\":false}]}}"),
		},
	},
	{
		name: "Delete third list entry (includes augmented leaf)",
		config: []string{
			"delete mainPCont/mainList/thirdEntry"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.main",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"firstEntry\","+
					"\"mainListLeaf\":666},"+
					"{\"name\":\"secondEntry\"}]}}"),
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.augment",
				"{}"),
		},
	},
	{
		name: "Delete second list entry",
		config: []string{
			"delete mainPCont/mainList/secondEntry"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.main",
				"{\"vyatta-test-main-v1:mainPCont\":{\"mainList\":"+
					"[{\"name\":\"firstEntry\","+
					"\"mainListLeaf\":666}]}}"),
		},
	},
	{
		name: "Delete first list entry",
		config: []string{
			"delete mainPCont/mainList/firstEntry"},
		logEntries: []schema.TestLogEntry{
			defaultCompLogEntry,
			schema.NewTestLogEntry(schema.SetRunning,
				"net.vyatta.test.main",
				"{\"vyatta-test-main-v1:mainPCont\":{}}"),
		},
	},
}

func TestConfigWrittenToCorrectComponentsForList(t *testing.T) {
	runTests(t, listTests)
}

func runTests(t *testing.T, tests []compConfigTest) {

	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(cfgTestSchemas).
		SetComponents(
			conf.BaseModelSet,
			[]string{
				defaultCfgTestComp.String(),
				mainCfgTestComp.String(),
				augmentCfgTestComp.String()})
	srv, sess := ts.Init()

	for _, test := range tests {
		runTest(t, srv, sess, ts, test)
	}
}

func runTest(
	t *testing.T,
	srv *sessiontest.TstSrv,
	sess *session.Session,
	ts *sessiontest.TestSpec,
	test compConfigTest,
) {
	// Apply config sets and deletes
	for _, cfg := range test.config {
		cmd := strings.Split(cfg, " ")[0]
		path := pathutil.Makepath(strings.Split(cfg, " ")[1])
		if cmd == "set" {
			sess.Set(srv.Ctx, path)
		} else {
			sess.Delete(srv.Ctx, path)
		}
	}

	// Commit config
	ts.ClearCompLogEntries()
	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("%s: Errors: %v\n", test.name, errs)
		return
	}

	// Verify expected config notified to correct components.
	ts.CheckCompLogEntries(
		test.name, schema.SetRunning, test.logEntries...)
	ts.ClearCompLogEntries()
}

var rfc7951config = testutils.Root(
	testutils.Cont("first",
		testutils.Leaf("firstLeaf", "someValue"),
		testutils.Cont("empty-leaf",
			testutils.Leaf("empty-leaf", "")),
		testutils.Cont("big-number",
			testutils.Leaf("big-number", "1357908642")),
		testutils.Cont("little-number",
			testutils.Leaf("little-number", "86")),
		testutils.Cont("third",
			testutils.Leaf("thirdLeaf", "anotherValue"))),
	testutils.Cont("second",
		testutils.Leaf("secondLeaf", "someValue")))

func TestRFC7951IsUsed(t *testing.T) {
	ts := sessiontest.NewTestSpec(t).
		SetSchemaDefsByRef(schemas).
		SetComponents(
			conf.BaseModelSet,
			[]string{
				firstTestComp.String(),
				secondTestComp.String(),
				thirdTestComp.String()})
	srv, sess := ts.Init()

	srv.LoadConfig(t, rfc7951config, sess)

	_, errs, ok := sess.Commit(srv.Ctx, "message", false /* No debug */)
	if !ok {
		t.Fatalf("Errors: %v\n", errs)
		return
	}

	ts.CheckCompLogEntries(
		"RFC7951 Is Used", schema.SetRunning,
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.second", secondCompCfgRfc7951),
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.third", thirdCompCfgRfc7951),
		schema.NewTestLogEntry(schema.SetRunning,
			"net.vyatta.test.first", firstCompCfgRfc7951))
}

const (
	firstCompCfgRfc7951  = "{\"vyatta-test-first-v1:first\":{\"big-number\":{\"big-number\":\"1357908642\"},\"empty-leaf\":{\"empty-leaf\":[null]},\"firstLeaf\":\"someValue\",\"little-number\":{\"little-number\":86}}}"
	secondCompCfgRfc7951 = "{\"vyatta-test-second-v1:second\":{\"secondLeaf\":\"someValue\"}}"
	thirdCompCfgRfc7951  = "{\"vyatta-test-first-v1:first\":{\"vyatta-test-third-v1:third\":{\"thirdLeaf\":\"anotherValue\"}}}"
)
