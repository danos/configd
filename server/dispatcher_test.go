// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"reflect"
	"testing"

	"github.com/danos/config/auth"
	. "github.com/danos/config/testutils"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/server"
	"github.com/danos/configd/session/sessiontest"
)

const (
	submoduleHasNoPrefix = ""
)

const defaultSchema = `
	container testContainer {
		leaf testLeaf {
			type string {
				configd:normalize "sed -e s/bar/foo/";
			}
		}
		leaf testBadScript {
			type string {
				configd:normalize "notfound";
			}
		}
		list testList {
			key name;
			leaf name {
				type string {
					configd:normalize "sed -e s/bar/foo/";
				}
			}
			leaf field {
				type string {
					configd:normalize "sed -e s/bar/foo/";
				}
			}
			leaf badField {
				type string {
					configd:normalize "notfound";
				}
			}
		}
		list testBadList {
			key name;
			leaf name {
				type string {
					configd:normalize "notfound";
				}
			}
			leaf field {
				type string;
			}
		}
		leaf emptyLeaf {
			type empty;
		}
	}`

const (
	emptyConfig = ""
)

func newTestDispatcher(
	t *testing.T,
	a auth.Auther,
	schema, config string,
) *server.Disp {
	return newTestDispatcherWithCustomAuth(t, a, schema, config, true, true)
}

func newTestDispatcherWithCustomAuth(
	t *testing.T,
	a auth.Auther,
	schema string,
	config string,
	isConfigdUser bool,
	inSecretsGroup bool,
) *server.Disp {
	testspec := sessiontest.NewTestSpec(t).
		SetSingleSchema(schema).
		SetConfig(config).
		SetAuther(a, isConfigdUser, inSecretsGroup)
	return newTestDispatcherFromTestSpec(testspec)
}

func newTestDispatcherWithMultipleSchemas(
	t *testing.T,
	a auth.Auther,
	schemaDefs []sessiontest.TestSchema,
	config string,
) *server.Disp {
	testspec := sessiontest.NewTestSpec(t).
		SetSchemaDefs(schemaDefs).
		SetConfig(config).
		SetAuther(a, sessiontest.ConfigdUser, sessiontest.NotInSecretsGroup)
	return newTestDispatcherFromTestSpec(testspec)
}

func newTestDispatcherFromTestSpec(ts *sessiontest.TestSpec) *server.Disp {
	srv, _ := ts.Init()
	return server.NewDispatcher(srv.Smgr, srv.Cmgr, srv.Ms, srv.MsFull,
		srv.Ctx)
}

func dispTestCommit(t *testing.T, d *server.Disp, sid string) {
	if _, err := d.Commit(sid, "", false); err != nil {
		t.Fatalf("Commit error: %s\n", err)
	}
}

func dispTestSet(t *testing.T, d *server.Disp, sid, path string) {
	if _, err := d.Set(sid, path); err != nil {
		t.Fatalf("\nUnable to configure '%s'. \nError: %s\n", path, err.Error())
	}
}

func dispTestSetFails(
	t *testing.T,
	d *server.Disp,
	sid, path string,
	errMsgs *assert.ExpectedMessages,
) {
	if _, err := d.Set(sid, path); err != nil {
		errMsgs.ContainedIn(t, err.Error())
		return
	}

	t.Fatalf("\nUnexpected success configuring '%s'", path)
}

func dispTestDelete(t *testing.T, d *server.Disp, sid, path string) {
	if _, err := d.Delete(sid, path); err != nil {
		t.Fatalf("\nUnable to delete '%s'. \nError: %s\n", path, err.Error())
	}
}

func dispTestDeleteFails(
	t *testing.T,
	d *server.Disp,
	sid, path string,
	errMsgs *assert.ExpectedMessages,
) {
	if _, err := d.Delete(sid, path); err != nil {
		errMsgs.ContainedIn(t, err.Error())
		return
	}

	t.Fatalf("\nUnexpected success deleting '%s'", path)
}

func dispTestExists(t *testing.T, d *server.Disp, db rpc.DB, sid, path string, expected bool) {
	t.Helper()
	exists, err := d.Exists(db, sid, path)
	if err != nil {
		t.Fatalf("\nUnable to check existence of '%s'. \nError: %s\n", path, err.Error())
	}
	if exists != expected {
		msg := "'%s' unexpectedly "
		if exists {
			msg += "exists"
		} else {
			msg += "does not exist"
		}
		t.Fatalf(msg, path)
	}
}

func dispTestValidate(t *testing.T, d *server.Disp, sid string) {
	if _, err := d.Validate(sid); err != nil {
		t.Fatalf("\nUnable to validate changeset. \nError: %s\n", err.Error())
	}
}

func checkValidateFails(
	t *testing.T,
	d *server.Disp,
	sid string,
	expErrs *assert.ExpectedMessages,
) {
	_, err := d.Validate(sid)
	if err == nil {
		t.Fatalf("\nUnexpected validation success.\n")
	}
	expErrs.ContainedIn(t, err.Error())
}

func dispTestShowCommon(t *testing.T, d *server.Disp, db rpc.DB,
	sid, path, expConfig string, showDflts bool, withCtx bool) {
	var act string
	var err error
	if withCtx {
		act, err = d.ShowConfigWithContextDiffs(sid, path, showDflts)
	} else if showDflts {
		act, err = d.ShowDefaults(db, sid, path, false)
	} else {
		act, err = d.Show(db, sid, path, false)
	}
	if err != nil {
		t.Fatalf("\nUnable to show changeset. \nError: %s\n", err.Error())
	}
	if act != expConfig {
		t.Fatalf("Exp:\n%s\nGot:\n%s\n", expConfig, act)
	}
}

func dispTestShow(t *testing.T, d *server.Disp, db rpc.DB,
	sid, path, expConfig string) {
	dispTestShowCommon(t, d, db, sid, path, expConfig, false, false)
}

func dispTestShowDefaults(t *testing.T, d *server.Disp, db rpc.DB,
	sid, path, expConfig string) {
	dispTestShowCommon(t, d, db, sid, path, expConfig, true, false)
}

func dispTestShowConfigWithContextDiffs(t *testing.T, d *server.Disp,
	sid, path, expConfig string, showDflts bool) {
	dispTestShowCommon(t, d, rpc.AUTO, sid, path, expConfig, showDflts, true)
}

func dispTestSetupSession(t *testing.T, d *server.Disp, sid string) {
	if _, err := d.SessionSetup(sid); err != nil {
		t.Fatalf("\nUnable to setup session. \nError: %s\n", err.Error())
	}
}

const showConfigWithContextDiffsTestSchema = `
container interfaces {
    presence "For contrast with protocols";
	list dataplane {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf mtu {
			type uint16;
			default 1500;
		}
	}
}
container protocols {
	list bgp {
		key id;
		leaf id {
			type uint32;
		}
	}
}
`

var showConfigWithContextDiffsInterfacesConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"))))

var showConfigWithContextDiffsConfig = showConfigWithContextDiffsInterfacesConfig +
	Root(
		Cont("protocols",
			List("bgp",
				ListEntry("100"))))

var showConfigWithContextDiffsInterfacesConfigDef = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1", Leaf("mtu", "1500")))))

var showConfigWithContextDiffsConfigDef = showConfigWithContextDiffsInterfacesConfigDef +
	Root(
		Cont("protocols",
			List("bgp",
				ListEntry("100"))))

var showConfigWithContextDiffsInterfacesDiff = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"),
			Add(ListEntry("dp0s2")))))

var showConfigWithContextDiffsDiff = showConfigWithContextDiffsInterfacesDiff +
	Root(
		Rem(Cont("protocols",
			List("bgp",
				ListEntry("100")))))

var showConfigWithContextDiffsInterfacesDiffDef = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1", Leaf("mtu", "1500")),
			Add(ListEntry("dp0s2", Leaf("mtu", "1500"))))))

var showConfigWithContextDiffsDiffDef = showConfigWithContextDiffsInterfacesDiffDef +
	Root(
		Rem(Cont("protocols",
			List("bgp",
				ListEntry("100")))))

func showConfigWithContextDiffsTest(t *testing.T, setDel, defaults bool, expConfig, configPath string, expCmd []string) {
	t.Helper()
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		showConfigWithContextDiffsTestSchema, showConfigWithContextDiffsConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	expConfig = FormatAsDiffNoTrailingLine(expConfig)

	dispTestSetupSession(t, d, testSID)

	if setDel {
		dispTestSet(t, d, testSID, "interfaces/dataplane/dp0s2")
		dispTestDelete(t, d, testSID, "protocols")

		// Set/Delete will have generated some requests
		clearAllCmdRequestsAndUserAuditLogs(a)
	}

	dispTestShowConfigWithContextDiffs(t, d, testSID, configPath, expConfig, defaults)

	assertCommandAaaNoSecrets(t, a, expCmd)
}

func TestShowConfigWithContextDiffs(t *testing.T) {
	showConfigWithContextDiffsTest(t, false, false,
		showConfigWithContextDiffsConfig, "", []string{"show"})
}

func TestShowConfigWithContextDiffsPath(t *testing.T) {
	showConfigWithContextDiffsTest(t, false, false,
		showConfigWithContextDiffsInterfacesConfig,
		"interfaces", []string{"show", "interfaces"})
}

func TestShowConfigWithContextDiffsDefault(t *testing.T) {
	showConfigWithContextDiffsTest(t, false, true,
		showConfigWithContextDiffsConfigDef, "", []string{"show", "-all"})
}

func TestShowConfigWithContextDiffsDefaultPath(t *testing.T) {
	showConfigWithContextDiffsTest(t, false, true,
		showConfigWithContextDiffsInterfacesConfigDef,
		"interfaces", []string{"show", "-all", "interfaces"})
}

func TestShowConfigWithContextDiffsDiff(t *testing.T) {
	showConfigWithContextDiffsTest(t, true, false,
		showConfigWithContextDiffsDiff, "", []string{"show"})
}

func TestShowConfigWithContextDiffsDiffPath(t *testing.T) {
	showConfigWithContextDiffsTest(t, true, false,
		showConfigWithContextDiffsInterfacesDiff,
		"interfaces", []string{"show", "interfaces"})
}

func TestShowConfigWithContextDiffsDiffDefault(t *testing.T) {
	showConfigWithContextDiffsTest(t, true, true,
		showConfigWithContextDiffsDiffDef, "", []string{"show", "-all"})
}

func TestShowConfigWithContextDiffsDiffDefaultPath(t *testing.T) {
	showConfigWithContextDiffsTest(t, true, true,
		showConfigWithContextDiffsInterfacesDiffDef,
		"interfaces", []string{"show", "-all", "interfaces"})
}

func dispTestCompareSessionChanges(t *testing.T, d *server.Disp, sid, expOut string) {
	t.Helper()
	act, err := d.CompareSessionChanges(sid)
	if err != nil {
		t.Fatalf("\nUnable to show changeset. \nError: %s\n", err.Error())
	}
	if act != expOut {
		t.Fatalf("Exp:\n%s\nGot:\n%s\n", expOut, act)
	}
}

var compareSessionChangesConfigAdd = Root(
	List("dataplane",
		Add(ListEntry("dp0s2"))))

var compareSessionChangesConfigDel = Root(
	Rem(Cont("protocols",
		List("bgp",
			ListEntry("100")))))

func TestCompareSessionChanges(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		showConfigWithContextDiffsTestSchema,
		showConfigWithContextDiffsConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	dispTestCompareSessionChanges(t, d, testSID, "")

	assertCommandAaaNoSecrets(t, a, []string{"compare"})
}

func TestCompareSessionChangesDiff(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		showConfigWithContextDiffsTestSchema,
		showConfigWithContextDiffsConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	dispTestSet(t, d, testSID, "interfaces/dataplane/dp0s2")
	dispTestDelete(t, d, testSID, "protocols")

	// Set/Delete will have generated some requests
	clearAllCmdRequestsAndUserAuditLogs(a)

	expOut := FormatCtxDiffHunk("interfaces", compareSessionChangesConfigAdd) +
		FormatCtxDiffHunk("", compareSessionChangesConfigDel)

	dispTestCompareSessionChanges(t, d, testSID, expOut)

	assertCommandAaaNoSecrets(t, a, []string{"compare"})
}

func TestCompareConfigRevisionsSavedCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.CompareConfigRevisions(testSID, "session", "saved")

	assertCommandAaaNoSecrets(t, a, []string{"compare", "saved"})
}

func TestCompareConfigRevisionsSessionRevCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.CompareConfigRevisions(testSID, "session", "2")

	assertCommandAaaNoSecrets(t, a, []string{"compare", "2"})
}

func TestCompareConfigRevisionsRevRevCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.CompareConfigRevisions(testSID, "1", "2")

	assertCommandAaaNoSecrets(t, a, []string{"compare", "1", "2"})
}

func dispTestGetCompletions(t *testing.T, d *server.Disp, sid, path string, fromSchema bool) map[string]string {
	comps, err := d.GetCompletions(sid, fromSchema, path)
	if err != nil {
		t.Fatalf("\nUnable to get completions '%s'. \nError: %s\n", path, err.Error())
	}
	return comps
}

func checkCompletionsMap(t *testing.T, actual, expected map[string]string) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Completions mismatch, expected:\n%v\ngot:\n%v", expected, actual)
	}
}

const completionsTestSchema = `
container interfaces {
    configd:help "Interfaces";
	list dataplane {
		configd:help "Dataplane";
		configd:allowed "echo foo bar baz";
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			configd:help "Address";
			type string;
		}
		leaf mtu {
			configd:help "MTU";
			type uint16;
			default 1500;
		}
		leaf mtu-ref {
			configd:help "MTU ref";
			type leafref {
				path "../mtu";
			}
		}
	}
}
`

var initCompletionsConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"),
			ListEntry("dp0s2"))))

var completionsValueConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"),
			ListEntry("dp0s2",
				Leaf("mtu", "1234"),
				Leaf("mtu-ref", "1234"),
				LeafList("address",
					LeafListEntry("1.1.1.1"))))))

func checkGetCompletions(t *testing.T, schema bool) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		completionsTestSchema, completionsValueConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	comps := dispTestGetCompletions(t, d, testSID, "", schema)
	expectedComps := map[string]string{
		"interfaces": "Interfaces",
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID, "interfaces", schema)
	expectedComps = map[string]string{
		"dataplane": "Dataplane",
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID, "interfaces/dataplane", schema)
	expectedComps = map[string]string{
		"dp0s1": "Dataplane",
		"dp0s2": "Dataplane",
	}
	if schema {
		expectedComps["<text>"] = "Dataplane"
		expectedComps["foo"] = ""
		expectedComps["bar"] = ""
		expectedComps["baz"] = ""
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID, "interfaces/dataplane/dp0s2", schema)
	expectedComps = map[string]string{
		"address": "Address",
		"mtu":     "MTU",
		"mtu-ref": "MTU ref",
	}
	if schema {
		expectedComps["<Enter>"] = "Execute the current command"
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID,
		"interfaces/dataplane/dp0s2/address", schema)
	expectedComps = map[string]string{
		"1.1.1.1": "Address",
	}
	if schema {
		expectedComps["<text>"] = "Address"
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID,
		"interfaces/dataplane/dp0s2/address/1.1.1.1", schema)
	if schema {
		expectedComps = map[string]string{
			"<Enter>": "Execute the current command",
		}
	} else {
		expectedComps = map[string]string{}
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID,
		"interfaces/dataplane/dp0s2/mtu", schema)
	expectedComps = map[string]string{
		"1234": "MTU",
	}
	if schema {
		expectedComps["<0..65535>"] = "MTU"
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID,
		"interfaces/dataplane/dp0s2/mtu/1234", schema)
	if schema {
		expectedComps = map[string]string{
			"<Enter>": "Execute the current command",
		}
	} else {
		expectedComps = map[string]string{}
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID,
		"interfaces/dataplane/dp0s2/mtu-ref", schema)
	if schema {
		expectedComps = map[string]string{
			"1234":   "",
			"<text>": "MTU ref",
		}
	} else {
		expectedComps = map[string]string{
			"1234": "MTU ref",
		}
	}
	checkCompletionsMap(t, comps, expectedComps)

	comps = dispTestGetCompletions(t, d, testSID,
		"interfaces/dataplane/dp0s2/mtu-ref/1234", schema)
	if schema {
		expectedComps = map[string]string{
			"<Enter>": "Execute the current command",
		}
	} else {
		expectedComps = map[string]string{}
	}
	checkCompletionsMap(t, comps, expectedComps)
}

func TestGetCompletionsOutputSchema(t *testing.T) {
	checkGetCompletions(t, true)
}

func TestGetCompletionsOutput(t *testing.T) {
	checkGetCompletions(t, false)
}

func TestRollbackCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.Rollback(testSID, "1", "", false)

	assertCommandAaaNoSecrets(t, a, []string{"rollback", "1"})
}

func TestRollbackWithCommentCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.Rollback(testSID, "2", "revert to working config", false)

	assertCommandAaaNoSecrets(t, a,
		[]string{"rollback", "2", "comment", "revert to working config"})
}

func TestCommitCommandAaa(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "testContainer/testLeaf/foo")
	clearAllCmdRequestsAndUserAuditLogs(a) // Set will have generated requests

	_, err := d.Commit(testSID, "", false)
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}

	assertCommandAaaNoSecrets(t, a, []string{"commit"})
}

func TestCommitWithCommentCommandAaa(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "testContainer/testLeaf/foo")
	clearAllCmdRequestsAndUserAuditLogs(a) // Set will have generated requests

	_, err := d.Commit(testSID, "foo bar baz", false)
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}

	assertCommandAaaNoSecrets(t, a,
		[]string{"commit", "comment", "foo bar baz"})
}

func TestCommitConfirmCommandAaa(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "testContainer/testLeaf/foo")
	clearAllCmdRequestsAndUserAuditLogs(a) // Set will have generated requests

	_, err := d.CommitConfirm(testSID, "", false, 1)
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}
	defer d.Confirm(testSID)

	assertCommandAaaNoSecrets(t, a, []string{"commit-confirm", "1"})
}

func TestCommitConfirmWithCommentCommandAaa(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "testContainer/testLeaf/foo")
	clearAllCmdRequestsAndUserAuditLogs(a) // Set will have generated requests

	_, err := d.CommitConfirm(testSID, "baz bar foo", false, 10)
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}
	defer d.Confirm(testSID)

	assertCommandAaaNoSecrets(t, a,
		[]string{"commit-confirm", "10", "comment", "baz bar foo"})
}

func TestConfirmCommandAaa(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.Confirm(testSID)

	assertCommandAaaNoSecrets(t, a, []string{"confirm"})
}

func TestDiscardCommandAaa(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	success, err := d.Discard(testSID)
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}
	if !success {
		t.Fatalf("discard failed unexpectedly")
	}

	assertCommandAaaNoSecrets(t, a, []string{"discard"})
}

func TestValidateCommandAaa(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		defaultSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "testContainer/testLeaf/foo")
	clearAllCmdRequestsAndUserAuditLogs(a) // Set will have generated requests
	dispTestValidate(t, d, testSID)

	assertCommandAaaNoSecrets(t, a, []string{"validate"})
}

// Need to test we get asterisks or encrypted string:
//
// - If we are configd user then we see encrypted
// - If not, then if not allowed, we see nothing
// - If not configd, and allowed, then we see asterisks unless we are in
//   secrets group.

const authSchema = `
	container testContainer {
		leaf secretLeaf {
			type string;
			configd:secret "true";
		}
	}`

type authTest struct {
	allowed      bool
	cfgdUser     bool
	inSecretsGrp bool
	expectedCfg  string
}

func TestReadConfigFileAuthorisation(t *testing.T) {

	expectNothing := ""
	expectAsterisk :=
		"testContainer {\n\tsecretLeaf \"********\"\n}\n"
	expectEncrypted :=
		"testContainer {\n\tsecretLeaf $1$abcdef123456\n}\n"

	testTbl := []authTest{
		{
			allowed:      false, // Wins
			cfgdUser:     false,
			inSecretsGrp: false,
			expectedCfg:  expectNothing,
		},
		{
			allowed:      true, // Wins
			cfgdUser:     false,
			inSecretsGrp: false,
			expectedCfg:  expectAsterisk,
		},
		{
			allowed:      false,
			cfgdUser:     true, // Wins
			inSecretsGrp: false,
			expectedCfg:  expectEncrypted,
		},
		{
			allowed:      false, // Wins
			cfgdUser:     false,
			inSecretsGrp: true,
			expectedCfg:  expectNothing,
		},
		{
			allowed:      true,
			cfgdUser:     true, // Wins
			inSecretsGrp: false,
			expectedCfg:  expectEncrypted,
		},
		{
			allowed:      false,
			cfgdUser:     true, // Wins
			inSecretsGrp: true,
			expectedCfg:  expectEncrypted,
		},
		{
			allowed:      true,
			cfgdUser:     false,
			inSecretsGrp: true, // Wins
			expectedCfg:  expectEncrypted,
		},
		{
			allowed:      true,
			cfgdUser:     true, // Wins
			inSecretsGrp: true,
			expectedCfg:  expectEncrypted,
		},
	}

	for _, test := range testTbl {
		d := newTestDispatcherWithCustomAuth(
			t, auth.TestAutherAllowOrDenyAll(test.allowed),
			authSchema, emptyConfig, test.cfgdUser, test.inSecretsGrp)
		actual, err := d.ReadConfigFile("test_files/cfgfiletest")

		if err != nil {
			t.Fatalf("Unexpected error reading Config File\n"+
				"    Allowed: %v\n    CfgdUser: %v\n    Secrets: %v\n"+
				"    Error: %s\n",
				test.allowed, test.cfgdUser, test.inSecretsGrp, err.Error())
		}
		if actual != test.expectedCfg {
			t.Fatalf("Unexpected result reading Config File\n"+
				"    Allowed: %v\n    CfgdUser: %v\n    Secrets: %v\n"+
				"    Expect: %s\n"+
				"    Actual: %s\n",
				test.allowed, test.cfgdUser, test.inSecretsGrp,
				test.expectedCfg, actual)
		}
	}
}
