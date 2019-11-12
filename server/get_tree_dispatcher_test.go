// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

// The aim of this set of tests is to verify that when we call GetTreeFull,
// we get as much state / config info back as possible, and don't let any
// configd:get-state script failure(s) cause us to get no useful output.
// The only valid reason for complete failure should be a request for the
// tree on an invalid path.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/danos/config/testutils"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

// There are 3 interface types, in alphabetical order.  The one we play with
// for testing is in the middle, so we can check that output generated before
// and after errors here is included in the final output.
var intfSchemaTemplate = `
container interfaces {
	container dataplane-state {
		config false;
		configd:get-state 'echo {"dataplanes":[{"tagnode":"dp0s1"}]}';
		list dataplanes {
			key tagnode;
			leaf tagnode {
				type string;
			}
		}
	}
	container switch-state {
		config false;
		configd:get-state '%s';
		list switches {
			key bridge-name;
			leaf bridge-name {
				type string;
			}
			leaf bridge-id {
				type uint32;
			}
		}
		leaf status {
			type string;
		}
	}
	container tunnel-state {
		config false;
		configd:get-state 'echo {"tunnels":[{"tagnode":"tun22"}]}';
		list tunnels {
			key tagnode;
			leaf tagnode {
				type string;
			}
		}
	}
}`

const (
	// Scripts inserted into schemas
	echo                = "echo "
	nonExistentScript   = "non_existent_script"
	emptyString         = "echo"
	emptyJson           = "echo {}"
	emptyList           = `echo {"switches":[]}`
	emptyListOtherState = `echo {"switches":[],"status":"up"}`
	oneSwitch           = `echo {"switches":[{"bridge-name":"br1"}]}`
	oneSwitchWithId     = `echo {"switches":[{"bridge-name":"br1","bridge-id":11}]}`
	twoSwitches         = `echo {"switches":[{"bridge-name":"br1"},{"bridge-name":"br2"}]}`
	badJson             = `{"swatches"}`
	intfState           = `{"dataplanes":[{"tagnode":"dp0s1"}]}`
)

const (
	// Paths we request full config + state tree for
	rootPath        = "/"
	intfPath        = "/interfaces"
	switchStatePath = "/interfaces/switch-state"
)

const (
	// JSON for the various interface-types' state (building blocks for the
	// subsequent strings.
	emptyState       = "{}"
	dpState          = `"dataplane-state":{"dataplanes":[{"tagnode":"dp0s1"}]}`
	tunState         = `"tunnel-state":{"tunnels":[{"tagnode":"tun22"}]}`
	swStatusUp       = `"switch-state":{"status":"up"}`
	oneSwState       = `"switch-state":{"switches":[{"bridge-name":"br1"}]}`
	oneSwWithIdState = `"switch-state":{"switches":[{"bridge-name":"br1","bridge-id":11}]}`
	twoSwState       = `"switch-state":{"switches":[{"bridge-name":"br1"},{"bridge-name":"br2"}]}`

	// Overall expected JSON returned for various scenarios from switch-state
	// node.
	oneSwitchJsonFromSwState        = `{` + oneSwState + `}`
	oneSwitchJsonWithIdFromSwState  = `{` + oneSwWithIdState + `}`
	twoSwitchJsonFromSwState        = `{` + twoSwState + `}`
	twoSwitchOneFailJsonFromSwState = `{"switch-state":{"switches":[{"bridge-name":"br0"},{"bridge-name":"br666"}]}}`

	// ... and from root path
	nonSwitchJsonFromRoot = `{"interfaces":{` + dpState + `,` + tunState + `}}`

	otherStateJsonFromRoot = `{"interfaces":{` + dpState + `,` +
		swStatusUp + `,` + tunState + `}}`
	otherStateJsonFromSwState   = `{` + swStatusUp + `}`
	oneSwitchJsonWithDPFromRoot = `{"interfaces":{` +
		dpState + `,` + oneSwState + `,` + tunState + `}}`
	oneSwitchJsonWithIDAndDPFromRoot = `{"interfaces":{` +
		dpState + `,` + oneSwWithIdState + `,` + tunState + `}}`

	twoSwitchJsonWithDPFromRoot = `{"interfaces":{` +
		dpState + `,` + twoSwState + `,` + tunState + `}}`
)

const (
	doesntMatchSchema           = ": Doesn't match schema"
	failedToProcessReturnedData = "Failed to process returned data"
	failedToRunStateFn          = "Failed to run state fn."
	invalidlyFormattedData      = "Invalidly formatted data returned"
	noInfo                      = ""
	emptyXMLState               = "<data></data>"
	jsonEncoding                = "json"
	xmlEncoding                 = "xml"
)

func genIntfTestSchema(input string) []sessiontest.TestSchema {
	return genTestSchema(intfSchemaTemplate, input)
}

func genTestSchema(template, state string) []sessiontest.TestSchema {
	return []sessiontest.TestSchema{
		{
			Name: sessiontest.NameDef{
				Namespace: "vyatta-test-validation-v1",
				Prefix:    "validation",
			},
			SchemaSnippet: fmt.Sprintf(template, state),
		},
	}
}

func runTest(
	t *testing.T,
	schema []sessiontest.TestSchema,
	path, config, encoding string,
) (string, error, []error, bytes.Buffer) {

	opts := make(map[string]interface{})
	opts["Defaults"] = true

	var logBuf bytes.Buffer

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(schema).
			SetConfig(config).
			SetSessionMgrLog(&logBuf))

	out, err, warns := d.TreeGetFullWithWarnings(rpc.RUNNING, "sid",
		path,
		encoding,
		opts)

	return out, err, warns, logBuf
}

func expectPass(
	t *testing.T,
	schema []sessiontest.TestSchema,
	path, expOutput, encoding string) {
	output, err, warns, _ := runTest(t, schema, path, emptyConfig, encoding)
	if err != nil {
		t.Fatalf("Unexpected failure:\nPath:\t%s\nErr:\t%s\n",
			path, err.Error())
		return
	}

	if len(warns) != 0 {
		t.Fatalf("Unexpected warnings:\nPath:\t%s\nWarns:\n%v\n",
			path, warns)
		return
	}

	if output != expOutput {
		t.Fatalf("Path:\t%s\nExp:\t%s\nGot:\t%s\n", path, expOutput, output)
	}
}

func expectWarn(
	t *testing.T,
	schema []sessiontest.TestSchema,
	path, expOutput, encoding string,
	expWarns []*errtest.ExpMgmtError,
) {
	expectWarnWithCfg(
		t, schema, emptyConfig, path, expOutput, encoding, expWarns)
}

func expectWarnWithCfg(
	t *testing.T,
	schema []sessiontest.TestSchema,
	config, path, expOutput, encoding string,
	expWarns []*errtest.ExpMgmtError,
) {
	output, err, warns, log := runTest(t, schema, path, config, encoding)
	if err != nil {
		t.Fatalf("Unexpected failure:\nPath:\t%s\nErr:\t%s\n",
			path, err.Error())
		return
	}

	if output != expOutput {
		t.Fatalf("Path:\t%s\nExp:\t%s\nGot:\t%s\n", path, expOutput, output)
	}

	if len(expWarns) != len(warns) {
		t.Fatalf("Got %d warnings, but expected %d\n",
			len(warns), len(expWarns))
		return
	}

	errtest.CheckMgmtErrors(t, expWarns, warns)
	errtest.CheckMgmtErrorsInLog(t, log, expWarns)
}

func expectFail(
	t *testing.T,
	schema []sessiontest.TestSchema,
	path, expFail, encoding string,
) {
	_, err, _, _ := runTest(t, schema, path, emptyConfig, encoding)
	if err == nil {
		t.Fatalf("Unexpected success:\nPath:\t%s\n", path)
		return
	}
	if !strings.Contains(err.Error(), expFail) {
		t.Fatalf("Wrong Error:\nExp:\t%s\nGot:\t%s\n", expFail, err.Error())
	}
}

func TestRootEmptyState(t *testing.T) {
	schema := genIntfTestSchema(emptyString)

	expectPass(t, schema, rootPath, nonSwitchJsonFromRoot, jsonEncoding)
}

func TestRootBadJson(t *testing.T) {
	schema := genIntfTestSchema(echo + badJson)

	expectWarn(t, schema, rootPath, nonSwitchJsonFromRoot, jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					invalidlyFormattedData,
					"(*schema.container)switch-state",
					badJson,
				},
				switchStatePath,
				noInfo),
		})
}

func TestRootEmptyJson(t *testing.T) {
	schema := genIntfTestSchema(emptyJson)

	expectPass(t, schema, rootPath, nonSwitchJsonFromRoot, jsonEncoding)
}

// Checks we cope with empty list
func TestRootEmptyList(t *testing.T) {
	schema := genIntfTestSchema(emptyList)

	expectPass(t, schema, rootPath, nonSwitchJsonFromRoot, jsonEncoding)
}

// Checks we don't ignore other state in any JSON that contains an empty list
func TestRootEmptyListWithOtherState(t *testing.T) {
	schema := genIntfTestSchema(emptyListOtherState)

	expectPass(t, schema, rootPath, otherStateJsonFromRoot, jsonEncoding)
}

func TestRootSingleSwitch(t *testing.T) {
	schema := genIntfTestSchema(oneSwitch)

	expectPass(t, schema, rootPath, oneSwitchJsonWithDPFromRoot, jsonEncoding)
}

func TestRootSingleSwitchWithId(t *testing.T) {
	schema := genIntfTestSchema(oneSwitchWithId)

	expectPass(t, schema, rootPath, oneSwitchJsonWithIDAndDPFromRoot,
		jsonEncoding)
}

func TestRootTwoSwitches(t *testing.T) {
	schema := genIntfTestSchema(twoSwitches)

	expectPass(t, schema, rootPath, twoSwitchJsonWithDPFromRoot, jsonEncoding)
}

func TestRootValidPathWithWrongButWellFormedJSON(t *testing.T) {
	schema := genIntfTestSchema(echo + intfState)

	expectWarn(t, schema, rootPath, nonSwitchJsonFromRoot, jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToProcessReturnedData,
					"(*schema.container)switch-state",
					"/dataplanes" + doesntMatchSchema,
					intfState,
				},
				switchStatePath,
				noInfo),
		})
}

func TestRootScriptReturnsErrorCode(t *testing.T) {
	schema := genIntfTestSchema(nonExistentScript)

	expectWarn(t, schema, rootPath, nonSwitchJsonFromRoot, jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToRunStateFn,
				},
				switchStatePath,
				noInfo),
		})
}

// Now repeat, calling directly on /interfaces/switch-state path.
func TestSwitchStateEmptyState(t *testing.T) {
	schema := genIntfTestSchema(emptyString)

	expectPass(t, schema, switchStatePath, emptyState, jsonEncoding)
}

func TestSwitchStateBadJson(t *testing.T) {
	schema := genIntfTestSchema(echo + badJson)

	expectWarn(t, schema, switchStatePath, emptyState, jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					invalidlyFormattedData,
					"(*schema.container)switch-state",
					badJson,
				},
				switchStatePath,
				noInfo),
		})
}

func TestSwitchStateEmptyJson(t *testing.T) {
	schema := genIntfTestSchema(emptyJson)

	expectPass(t, schema, switchStatePath, emptyState, jsonEncoding)
}

func TestSwitchStateEmptyList(t *testing.T) {
	schema := genIntfTestSchema(emptyList)

	expectPass(t, schema, switchStatePath, emptyState, jsonEncoding)
}

// Checks we don't ignore other state in any JSON that contains an empty list
func TestSwitchStatetEmptyListWithOtherState(t *testing.T) {
	schema := genIntfTestSchema(emptyListOtherState)

	expectPass(t, schema, switchStatePath, otherStateJsonFromSwState,
		jsonEncoding)
}

func TestSwitchStateSingleSwitch(t *testing.T) {
	schema := genIntfTestSchema(oneSwitch)

	expectPass(t, schema, switchStatePath, oneSwitchJsonFromSwState,
		jsonEncoding)
}

func TestSwitchStateSingleSwitchWithId(t *testing.T) {
	schema := genIntfTestSchema(oneSwitchWithId)

	expectPass(t, schema, switchStatePath, oneSwitchJsonWithIdFromSwState,
		jsonEncoding)
}

func TestSwitchStateTwoSwitches(t *testing.T) {
	schema := genIntfTestSchema(twoSwitches)

	expectPass(t, schema, switchStatePath, twoSwitchJsonFromSwState,
		jsonEncoding)
}

func TestNonExistentPath(t *testing.T) {
	schema := genIntfTestSchema("arbitrary")

	expectFail(t, schema, "/nonexistent",
		"/nonexistent: An unexpected element is present.", jsonEncoding)
}

func TestSwitchStateValidPathWithWrongButWellFormedJSON(t *testing.T) {
	schema := genIntfTestSchema(echo + intfState)

	expectWarn(t, schema, switchStatePath, emptyState, jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToProcessReturnedData,
					"(*schema.container)switch-state",
					"/dataplanes" + doesntMatchSchema,
					intfState,
				},
				switchStatePath,
				noInfo),
		})
}

func TestSwitchStateScriptReturnsErrorCode(t *testing.T) {
	schema := genIntfTestSchema(nonExistentScript)

	expectWarn(t, schema, switchStatePath, emptyState, jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToRunStateFn,
				},
				switchStatePath,
				noInfo),
		})
}

var switchOnlySchemaTemplate = `
container interfaces {
	container switch-state {
		config false;
		configd:get-state 'echo {"switches":[{"bridge-name":"br0"}]}';
		configd:get-state '%s';
		configd:get-state 'echo {"switches":[{"bridge-name":"br666"}]}';
		list switches {
			key bridge-name;
			leaf bridge-name {
				type string;
			}
			leaf bridge-id {
				type uint32;
			}
		}
	}
}`

// Ensure we run all scripts on a node where one of the scripts fails.
func TestAllScriptsRunOnNodeWhenOneFails(t *testing.T) {
	schema := genTestSchema(switchOnlySchemaTemplate, nonExistentScript)

	expectWarn(t, schema, switchStatePath, twoSwitchOneFailJsonFromSwState,
		jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToRunStateFn,
				},
				switchStatePath,
				noInfo),
		})
}

var scriptsUnderScriptsTemplate = `
container interfaces {
	container switch-state {
		config false;
		configd:get-state '%s';
		list switches {
			key bridge-name;
			leaf bridge-name {
				type string;
			}
			leaf bridge-id {
				type uint32;
			}
		}
		container switches-info {
			configd:get-state 'echo {"info":"infoVal"}';
			leaf info { type string; }
		}
	}
}`

func TestScriptFailureDoesntStopScriptsOnChildNodesBeingRun(t *testing.T) {
	schema := genTestSchema(scriptsUnderScriptsTemplate, nonExistentScript)

	expJson := `
		{
		"interfaces":{
			"switch-state":{
				"switches-info":{
					"info":"infoVal"
				}
			}
		}
	}`

	expectWarn(t, schema, rootPath, stripWS(expJson), jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToRunStateFn,
				},
				switchStatePath,
				noInfo),
		})
}

var scriptAboveSwitchStateTemplate = `
container interfaces {
	config false;
	configd:get-state '%s';
	container switch-state {
		config false;
		configd:get-state 'echo {"switches":[{"bridge-name":"br0"}]}';
		list switches {
			key bridge-name;
			leaf bridge-name {
				type string;
			}
			leaf bridge-id {
				type uint32;
			}
		}
	}
}`

func TestScriptCalledDuringWalkToTargetNodeFails(t *testing.T) {
	schema := genTestSchema(scriptAboveSwitchStateTemplate, nonExistentScript)

	expJson := `
		{
		"switch-state":{
			"switches":[{
				"bridge-name":"br0"
			}]
		}
	}`

	expectWarn(t, schema, switchStatePath, stripWS(expJson), jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToRunStateFn,
				},
				"/interfaces",
				noInfo),
		})
}

// The next few tests check that we deal with child nodes (both with active
// configuration, and state-only) that generate invalid output, without
// impacting the reporting from other config and state nodes.
var childNodeFailureTemplate = `
container top {
	leaf aaaTopLeaf {
		type string;
	}
	container configAndStateChild {
		leaf configLeaf {
			type string;
		}
		container stateInfo {
			config false;
			configd:get-state '%s';
			leaf stateLeaf {
				type string;
			}
		}
	}
	container stateOnlyChild {
		config false;
		configd:get-state '%s';
		leaf stateOnlyChildLeaf {
			type string;
		}
	}
	container zzzLastCont {
		description "Last, always ok, used to check we don't return early";
		leaf zzzConfigLeaf {
			type string;
		}
		configd:get-state 'echo {"stateOnlyLeaf":"someValue"}';
		leaf stateOnlyLeaf {
			config false;
			type string;
		}
	}
} `

const (
	stateInfoScriptFail = `echo {"badJson"}`
	stateOnlyScriptPass = `echo {"stateOnlyChildLeaf":"stateOnlyChildLeafVal"}`
	stateInfoScriptPass = `echo {"stateLeaf":"stateLeafValue"}`
	stateOnlyScriptFail = `echo {"badJson"}`
)

var childOperStateConfig = testutils.Root(
	testutils.Cont("top",
		testutils.Leaf("aaaTopLeaf", "topValue"),
		testutils.Cont("configAndStateChild",
			testutils.Leaf("configLeaf", "configLeafValue")),
		testutils.Cont("zzzLastCont",
			testutils.Leaf("zzzConfigLeaf", "zzzConfigValue"))))

func genChildNodeFailTestSchema(
	template, script1, script2 string,
) []sessiontest.TestSchema {

	return []sessiontest.TestSchema{
		{
			Name: sessiontest.NameDef{
				Namespace: "vyatta-test-validation-v1",
				Prefix:    "validation",
			},
			SchemaSnippet: fmt.Sprintf(template, script1, script2),
		},
	}
}

// Allows tests to show human-readable JSON, then convert it to machine-
// comparable for running tests.
func stripWS(pretty string) string {
	r := strings.NewReplacer(" ", "", "\n", "", "\t", "")
	return r.Replace(pretty)
}

func TestChildOperStateActiveChildren(t *testing.T) {
	schema := genChildNodeFailTestSchema(childNodeFailureTemplate,
		stateInfoScriptFail, stateOnlyScriptPass)

	expJson := `
	{
		"top":{
			"aaaTopLeaf":"topValue",
			"configAndStateChild":{
				"configLeaf":"configLeafValue"
			},
			"stateOnlyChild":{
				"stateOnlyChildLeaf":"stateOnlyChildLeafVal"
			},
			"zzzLastCont":{
				"stateOnlyLeaf":"someValue",
				"zzzConfigLeaf":"zzzConfigValue"
			}
		}
	}`

	expectWarnWithCfg(t,
		schema, childOperStateConfig, "/top", stripWS(expJson), jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					invalidlyFormattedData,
					`{"badJson"}`,
					"(*schema.container)stateInfo",
				},
				"/top/configAndStateChild/stateInfo",
				noInfo),
		})
}

func TestChildOperStateStateOnlyChildren(t *testing.T) {
	schema := genChildNodeFailTestSchema(childNodeFailureTemplate,
		stateInfoScriptPass, stateOnlyScriptFail)

	expJson := `
	{
		"top":{
			"aaaTopLeaf":"topValue",
			"configAndStateChild":{
				"configLeaf":"configLeafValue",
				"stateInfo":{
					"stateLeaf":"stateLeafValue"
				}
			},
			"zzzLastCont":{
				"stateOnlyLeaf":"someValue",
				"zzzConfigLeaf":"zzzConfigValue"
			}
		}
	}`

	expectWarnWithCfg(t,
		schema, childOperStateConfig, "/top", stripWS(expJson), jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					invalidlyFormattedData,
					`{"badJson"}`,
					"(*schema.container)stateOnlyChild",
				},
				"/top/stateOnlyChild",
				noInfo),
		})
}

func TestMultipleScriptFailuresDetected(t *testing.T) {
	stateInfoScriptFailSchema := `echo {"wrongNode":"wrongValue"}`
	stateOnlyScriptFailFormat := `echo {"badJson"}`

	schema := genChildNodeFailTestSchema(childNodeFailureTemplate,
		stateInfoScriptFailSchema, stateOnlyScriptFailFormat)

	expJson := `
	{
		"top":{
			"aaaTopLeaf":"topValue",
			"configAndStateChild":{
				"configLeaf":"configLeafValue"
			},
			"zzzLastCont":{
				"stateOnlyLeaf":"someValue",
				"zzzConfigLeaf":"zzzConfigValue"
			}
		}
	}`

	expectWarnWithCfg(t,
		schema, childOperStateConfig, "/top", stripWS(expJson), jsonEncoding,
		[]*errtest.ExpMgmtError{
			errtest.NewExpMgmtError(
				[]string{
					failedToProcessReturnedData,
					doesntMatchSchema,
					`{"wrongNode":"wrongValue"}`,
					"(*schema.container)stateInfo",
				},
				"/top/configAndStateChild/stateInfo",
				noInfo),
			errtest.NewExpMgmtError(
				[]string{
					invalidlyFormattedData,
					`{"badJson"}`,
					"(*schema.container)stateOnlyChild",
				},
				"/top/stateOnlyChild",
				noInfo),
		})
}

const simpleSchema = `
container top {
	config false;
	configd:get-state '%s';
	leaf state {
		type string;
	}
}`

func TestNoStateReturnedJson(t *testing.T) {
	schema := genTestSchema(simpleSchema, "echo")

	expectPass(t, schema, rootPath, emptyState, jsonEncoding)
}

func TestNoStateReturnedXml(t *testing.T) {
	schema := genTestSchema(simpleSchema, "echo")

	expectPass(t, schema, rootPath, emptyXMLState, xmlEncoding)
}

func TestComponentState(t *testing.T) {
	t.Skipf("TBD")
}
