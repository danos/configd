// Copyright (c) 2017-2019, AT&T Intellectual Property Inc. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"strings"
	"testing"

	"github.com/danos/config/testutils"
	"github.com/danos/configd"
	"github.com/danos/configd/session"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
	"github.com/danos/utils/pathutil"
)

func checkFullTreeInvalid(
	t *testing.T, sess *session.Session, ctx *configd.Context, find string, expect ...string) {

	_, err, _ := sess.GetFullTree(ctx, pathutil.Makepath(find),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err == nil {
		t.Fatalf("Unexpected success validating full tree.")
		return
	}

	actErr := err.Error()
	for _, expErr := range expect {
		if strings.Contains(actErr, expErr) {
			continue
		}
		t.Fatalf(
			"Unexpected error for invalid full tree.\nExp: '%s'\nGot: '%s'\n",
			expErr, actErr)
		return
	}
}

func validateFullTree(
	t *testing.T, sess *session.Session, ctx *configd.Context, find string, expect ...string) {

	ut, err, warns := sess.GetFullTree(ctx, pathutil.Makepath(find),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if len(warns) != 0 {
		t.Logf("Unexpected warning(s):\n")
		for _, warn := range warns {
			t.Logf("%s\n", warn.Error())
		}
		t.Fatalf("Test FAILED.\n")
		return
	}

	// So we can validate case where there are no nodes, but there *could*
	// be nodes, ensure if nothing expected, there is nothing to find.
	if len(expect) == 0 {
		if ut != nil && ut.NumChildren() > 0 {
			t.Fatalf("No children of '%s' expected, yet some found!", ut.Name())
		}
	}

	for _, v := range expect {
		ps := pathutil.Makepath(v)
		n, err := ut.Descendant(nil, ps)
		if err != nil {
			t.Errorf("Error finding node %s: %s", v, err.Error())
			continue
		}
		if n.NumChildren() > 0 {
			t.Errorf("Unexpected child nodes for %s", v)
			continue
		}
	}
}

func validateFullTreeCheckNodesNotFound(
	t *testing.T,
	sess *session.Session,
	ctx *configd.Context,
	find string,
	unexpect ...string) {

	ut, err, warns := sess.GetFullTree(ctx, pathutil.Makepath(find),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if len(warns) != 0 {
		t.Logf("Unexpected warning(s):\n")
		for _, warn := range warns {
			t.Logf("%s\n", warn.Error())
		}
		t.Fatalf("Test FAILED.\n")
		return
	}

	// So we can validate case where there are no nodes, but there *could*
	// be nodes, ensure if nothing expected, there is nothing to find.
	if len(unexpect) == 0 {
		t.Fatalf("Must specify at least one unexpected node")
	}

	for _, v := range unexpect {
		ps := pathutil.Makepath(v)
		_, err := ut.Descendant(nil, ps)
		if err == nil {
			t.Errorf("Unexpectedly found node %s", v)
			continue
		}
	}
}

func TestGetFullTreeSimple(t *testing.T) {
	const schema = `container mix {
			leaf conf {
				type string;
			}
			container state {
				leaf state-value {
					type string;
					config false;
				}
				configd:get-state "echo {\"state-value\":\"leafvalue\"}";
			}
		}`

	const config = `mix {
			conf stuff;
		}`

	srv, sess := sessiontest.TstStartup(t, schema, config)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"",
		"mix/conf/stuff",
		"mix/state/state-value/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"mix",
		"conf/stuff",
		"state/state-value/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"mix/state",
		"state-value/leafvalue")
}

func TestGetFullTreeWithMust(t *testing.T) {
	const schema = `container mix {
			leaf conf {
				type string;
			}
			container state {
				config false;
				leaf state-value {
					type string;
				}
				must 'state-value';
				configd:get-state "echo {\"state-value\":\"leafvalue\"}";
			}
		}`

	const config = `mix {
			conf stuff;
		}`

	srv, sess := sessiontest.TstStartup(t, schema, config)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"",
		"mix/conf/stuff",
		"mix/state/state-value/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"mix",
		"conf/stuff",
		"state/state-value/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"mix/state",
		"state-value/leafvalue")
}

func TestGetFullTreeList(t *testing.T) {
	const schema = `list mix {
			key "tagnode";
			leaf tagnode {
				type string;
			}
			leaf conf {
				type string;
			}
			leaf state-value {
				type string;
				config false;
				configd:get-state "echo {\"state-value\":\"leafvalue1\"}";
			}
			container state {
				leaf state-value {
					type string;
					config false;
				}
				configd:get-state "echo {\"state-value\":\"leafvalue2\"}";
			}
		}`

	const config = `mix item1 {
			conf stuff;
		}`

	const expect = `{"mix":[{"tagnode":"item1","conf":"stuff","state":{"state-value":"leafvalue2"},"state-value":"leafvalue1"}]}`

	srv, sess := sessiontest.TstStartup(t, schema, config)
	defer sess.Kill()

	ut, err, warns := sess.GetFullTree(srv.Ctx, pathutil.Makepath(""),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if len(warns) != 0 {
		t.Logf("Unexpected warning(s):\n")
		for _, warn := range warns {
			t.Logf("%s\n", warn.Error())
		}
		t.Fatalf("Test FAILED.\n")
		return
	}
	actual := string(ut.ToJSON())
	if actual != expect {
		t.Logf(actual)
		t.Fatalf("Unexpected output found")
	}

	validateFullTree(t, sess, srv.Ctx,
		"",
		"mix/item1/conf/stuff",
		"mix/item1/state/state-value/leafvalue2")
	validateFullTree(t, sess, srv.Ctx,
		"mix/item1",
		"conf/stuff",
		"state/state-value/leafvalue2")
	validateFullTree(t, sess, srv.Ctx,
		"mix/item1/state",
		"state-value/leafvalue2")
}

// This checks we get no error if we request a state node populated by VCI,
// but that we don't get the node returned.
func TestComponentState(t *testing.T) {

	const schema = `
	container config {
		leaf config-leaf {
			type string;
		}
		container state {
			config false;
			leaf state-leaf {
				type string;
			}
		}
	}`

	const config = `config {
			config-leaf stuff;
		}`

	srv, sess := sessiontest.TstStartup(t, schema, config)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"",
		"config/config-leaf/stuff")
	validateFullTreeCheckNodesNotFound(t, sess, srv.Ctx,
		"",
		"config/state")
	validateFullTree(t, sess, srv.Ctx,
		"config",
		"config-leaf/stuff")
	validateFullTree(t, sess, srv.Ctx,
		"config/state")
}

func TestGetFullTreeStateOnly(t *testing.T) {
	const schema = `container stateonly {
			config false;
			leaf state {
				type string;
			}
			configd:get-state "echo {\"state\":\"leafvalue\",\"substate\":{\"state\":\"leafvalue\"}}";
			container substate {
				presence "madebyparent";
				leaf state {
					type string;
				}
				leaf morestate {
					type string;
				}
				configd:get-state "echo {\"morestate\":\"leafvalue\"}";
			}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"",
		"stateonly/state/leafvalue",
		"stateonly/substate/state/leafvalue",
		"stateonly/substate/morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly",
		"state/leafvalue",
		"substate/state/leafvalue",
		"substate/morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/state",
		"leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/substate",
		"state/leafvalue",
		"morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/substate/state",
		"leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/substate/morestate",
		"leafvalue")
}

func TestGetFullTreeStatePresence(t *testing.T) {
	const schema = `container stateonly {
			presence "allstateunderpresence";
			config false;
			leaf state {
				type string;
			}
			configd:get-state "echo {\"state\":\"leafvalue\",\"substate\":{\"state\":\"leafvalue\"}}";
			container substate {
				presence "madebyparent";
				leaf state {
					type string;
				}
				leaf morestate {
					type string;
				}
				configd:get-state "echo {\"morestate\":\"leafvalue\"}";
			}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"",
		"stateonly/state/leafvalue",
		"stateonly/substate/state/leafvalue",
		"stateonly/substate/morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly",
		"state/leafvalue",
		"substate/state/leafvalue",
		"substate/morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/state",
		"leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/substate",
		"state/leafvalue",
		"morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/substate/state",
		"leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"stateonly/substate/morestate",
		"leafvalue")
}

func TestGetFullTreeStateUnderConfigPresenceAbsent(t *testing.T) {
	const schema = `container config-node {
				presence "Config Presence Node";
				container stateonly {
					presence "allstateunderpresence";
					config false;
					leaf state {
						type string;
					}
					configd:get-state "echo {\"state\":\"leafvalue\",\"substate\":{\"state\":\"leafvalue\"}}";
					container substate {
						presence "madebyparent";
						leaf state {
							type string;
						}
						leaf morestate {
							type string;
						}
						configd:get-state "echo {\"morestate\":\"leafvalue\"}";
					}
				}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"")
}

func TestGetFullTreeStateUnderConfigPresenceConfigured(t *testing.T) {
	const schema = `container config-node {
				presence "Configuration test node";
				container stateonly {
					presence "allstateunderpresence";
					config false;
					leaf state {
						type string;
					}
					configd:get-state "echo {\"state\":\"leafvalue\",\"substate\":{\"state\":\"leafvalue\"}}";
					container substate {
						presence "madebyparent";
						leaf state {
							type string;
						}
						leaf morestate {
							type string;
						}
						configd:get-state "echo {\"morestate\":\"leafvalue\"}";
					}
				}
		}`

	const config = `config-node
		`

	srv, sess := sessiontest.TstStartup(t, schema, config)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"",
		"config-node/stateonly/state/leafvalue",
		"config-node/stateonly/substate/state/leafvalue",
		"config-node/stateonly/substate/morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"config-node/stateonly",
		"state/leafvalue",
		"substate/state/leafvalue",
		"substate/morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"config-node/stateonly/state",
		"leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"config-node/stateonly/substate",
		"state/leafvalue",
		"morestate/leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"config-node/stateonly/substate/state",
		"leafvalue")
	validateFullTree(t, sess, srv.Ctx,
		"config-node/stateonly/substate/morestate",
		"leafvalue")
}

func TestGetFullTreeLeafState(t *testing.T) {
	const schema = `container test {
			leaf config {
				type string;
			}
			leaf state {
				config false;
				type string;
			}
			configd:get-state "echo {\"state\":\"leafstate\"}";
		}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"test/state",
		"leafstate")
}

func TestGetFullTreeLeafListState(t *testing.T) {
	const schema = `container test {
			leaf config {
				type string;
			}
			leaf-list state {
				config false;
				type string;
				configd:get-state "echo {\"state\": [\"leafstate1\",\"leafstate2\"]}";
			}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"test/state",
		"leafstate1",
		"leafstate2")
}

var configCfg = testutils.Root(
	testutils.Cont("test",
		testutils.Leaf("config", "configValue")))

func TestGetFullTreeWithConfigNoState(t *testing.T) {
	const schema = `container test {
			leaf config {
				type string;
			}
			leaf-list state {
				config false;
				type string;
			}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, configCfg)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"test/config",
		"configValue")
}

func TestGetFullTreeWithConfigAndState(t *testing.T) {
	const schema = `container test {
			leaf config {
				type string;
			}
			leaf-list state {
				config false;
				type string;
				configd:get-state "echo {\"state\": [\"leafstate1\",\"leafstate2\"]}";
			}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, configCfg)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"test",
		"config/configValue",
		"state/leafstate1",
		"state/leafstate2")
}

func TestGetFullTreeWithMandatoryConfigScriptOnLeaf(t *testing.T) {
	const schema = `container test {
			leaf config {
				type string;
				mandatory true;
			}
			leaf-list state {
				config false;
				type string;
				configd:get-state "echo {\"state\": [\"leafstate1\",\"leafstate2\"]}";
			}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, configCfg)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"test",
		"config/configValue",
		"state/leafstate1",
		"state/leafstate2")
}

// Checks script on leaf/leaf-list works as peer of node, versus previous
// test which check it as child.
func TestGetFullTreeWithMandatoryConfigScriptAtTop(t *testing.T) {
	const schema = `container test {
			configd:get-state "echo {\"state\": [\"leafstate1\",\"leafstate2\"]}";
			leaf config {
				type string;
				mandatory true;
			}
			leaf-list state {
				config false;
				type string;
			}
		}`

	srv, sess := sessiontest.TstStartup(t, schema, configCfg)
	defer sess.Kill()

	validateFullTree(t, sess, srv.Ctx,
		"test",
		"config/configValue",
		"state/leafstate1",
		"state/leafstate2")
}

func TestGetFullTreeWithMandatoryStateMissing(t *testing.T) {
	const schema = `container test {
		configd:get-state "echo {\"stateList\": [\"leafstate1\",\"leafstate2\"]}";
		config false;
		leaf stateLeaf {
			type string;
			mandatory true;
		}
		leaf-list stateList {
			type string;
		}
	}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	checkFullTreeInvalid(t, sess, srv.Ctx,
		"test",
		errtest.NewMissingMandatoryNodeError(t, "/test/stateLeaf").
			RawErrorStrings()...)
}

func TestGetFullTreeListKeyOnlyState(t *testing.T) {
	const schema = `container teststate {
		       config false;
		       list state {
			       key "name";
			       leaf name {
				       type string;
			       }
		       }
		       configd:get-state "echo {\"state\":[{\"name\":\"foo\"}]}";
	       }`
	const expect = `{"state":[{"name":"foo"}]}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	ut, err, warns := sess.GetFullTree(
		srv.Ctx, pathutil.Makepath("/teststate/state"),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if len(warns) != 0 {
		t.Logf("Unexpected warning(s):\n")
		for _, warn := range warns {
			t.Logf("%s\n", warn.Error())
		}
		t.Fatalf("Test FAILED.\n")
		return
	}
	actual := string(ut.ToJSON())
	if actual != expect {
		t.Logf("Expected: %s", expect)
		t.Logf("Actual: %s", actual)
		t.Fatalf("Unexpected output found")
	}
}

func TestGetFullTreeLeafStatePath(t *testing.T) {
	t.Skip("need to get exec working")
	const schema = `container test {
			leaf config {
				type string;
			}
			leaf state {
				config false;
				type string;
			}
            configd:get-state "testdata/get-state-from-path";
		}`
	const expect = `{"state":"/test/state"}`
	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	ut, err, warns := sess.GetFullTree(srv.Ctx,
		pathutil.Makepath("/test/state"),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if len(warns) != 0 {
		t.Logf("Unexpected warning(s):\n")
		for _, warn := range warns {
			t.Logf("%s\n", warn.Error())
		}
		t.Fatalf("Test FAILED.\n")
		return
	}
	actual := string(ut.ToJSON())
	if actual != expect {
		t.Logf("Expected: %s", expect)
		t.Logf("Actual: %s", actual)
		t.Fatalf("Unexpected output found")
	}
}

// The context for evaluating must statements on 'config true' nodes should
// be constrained to all 'config true' nodes and should ignore 'config false'
// nodes.
func TestMustOnConfigIgnoresNonConfigNodes(t *testing.T) {
	const schema = `
	container testCont {
		container subCont {
			leaf cfgLeaf {
				type string;
			}
			leaf stateLeaf {
				config false;
				type string;
			}
			configd:get-state "echo {\"stateLeaf\":\"stateValue\"}";
		}
		leaf mustTestLeaf {
			type string;
			must "count(../subCont/*) = 0" {
				error-message "Can't have any (config) nodes under subCont";
			}
		}
	}`

	var mustConfig = testutils.Root(
		testutils.Cont("testCont",
			testutils.Leaf("mustTestLeaf", "someValue")))

	const expect = `{"testCont":{"mustTestLeaf":"someValue","subCont":{"stateLeaf":"stateValue"}}}`

	srv, sess := sessiontest.TstStartup(t, schema, mustConfig)
	defer sess.Kill()

	ut, err, warns := sess.GetFullTree(srv.Ctx, pathutil.Makepath("/testCont"),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if len(warns) != 0 {
		t.Logf("Unexpected warning(s):\n")
		for _, warn := range warns {
			t.Logf("%s\n", warn.Error())
		}
		t.Fatalf("Test FAILED.\n")
		return
	}
	actual := string(ut.ToJSON())
	if actual != expect {
		t.Logf("Expected: %s", expect)
		t.Logf("Actual: %s", actual)
		t.Fatalf("Unexpected output found")
	}
}

func TestMarshalOfNonPresentDecimal64(t *testing.T) {
	const schema = `container teststate {
				config false;
				list state {
					key "name";
					leaf name {
						type string;
					}
					leaf decimal64leaf {
						type decimal64 {
							fraction-digits 2;
						}
					}
				}
				configd:get-state "echo {\"state\":[{\"name\":\"foo\"}]}";
			}`
	const expect = `{"state":[{"name":"foo"}]}`

	srv, sess := sessiontest.TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	ut, err, warns := sess.GetFullTree(srv.Ctx,
		pathutil.Makepath("/teststate/state"),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
	if len(warns) != 0 {
		t.Logf("Unexpected warning(s):\n")
		for _, warn := range warns {
			t.Logf("%s\n", warn.Error())
		}
		t.Fatalf("Test FAILED.\n")
		return
	}
	actual := string(ut.ToJSON())
	if actual != expect {
		t.Logf("Expected: %s", expect)
		t.Logf("Actual: %s", actual)
		t.Fatalf("Unexpected output found")
	}
}
