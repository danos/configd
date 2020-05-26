// Copyright (c) 2017-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// Tests on load / merge functionality as reported back to the dispatcher.

package server_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/server"
	"github.com/danos/mgmterror/errtest"
)

type loadOrMergeFn func(string, string) (bool, error)

func commitAndVerifyConfig(t *testing.T, d *server.Disp, expConfig string) {

	_, err := d.Commit(testSID, "message", false /* no debug */)
	if err != nil {
		t.Fatalf("Unable to commit config: %s", err)
	}

	actConfig, _ := d.Show(rpc.RUNNING, testSID, "",
		false /* Don't hide secrets */)
	if actConfig != expConfig {
		t.Fatalf("Config mismatch:\nExp\n%s\n\nGot:\n%s\n\n",
			expConfig, actConfig)
	}
}

// To make tests more readable, we specify the content of the config to
// be loaded as a string, and convert it locally to a file.  We use
// ioutil.TempFile to ensure we get a unique filename and also use defer
// to make sure we remove this.
//
func dispTestLoadOrMergeWriteConfigToFile(config string) (string, error) {
	// Convert config to a file.
	tmpfile, err := ioutil.TempFile("", "dispTestLoadOrMerge")
	if err != nil {
		return "",
			fmt.Errorf("Unable to create temp file for load/merge test: %s\n",
				err.Error())
	}

	if _, err := tmpfile.WriteString(config); err != nil {
		os.Remove(tmpfile.Name())
		return "",
			fmt.Errorf("Unable to write temp file for load/merge test: %s\n",
				err.Error())
	}
	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		return "",
			fmt.Errorf("Unable to close temp file for load/merge test: %s\n",
				err.Error())
	}

	return tmpfile.Name(), nil
}

func dispTestLoadOrMergeCommon(
	t *testing.T,
	loadOrMerge loadOrMergeFn,
	sid, config string,
) (bool, error) {

	file, err := dispTestLoadOrMergeWriteConfigToFile(config)
	if err != nil {
		return false, err
	}
	defer os.Remove(file)

	return loadOrMerge(sid, file)
}

// Generate a loadOrMergeFn closure which wraps around an existing loadOrMergeFn.
// After calling loadOrMerge the closure checks that the expected command
// authorization requests were seen.
func dispLoadOrMergeFnCheckCmdAuthz(t *testing.T, tAuth auth.TestAuther,
	loadOrMerge loadOrMergeFn,
	cmd string) loadOrMergeFn {
	if cmd != "load" && cmd != "merge" {
		panic("Expected cmd == load || cmd == merge")
	}

	return func(sid string, file string) (bool, error) {
		res, err := loadOrMerge(sid, file)
		assertCommandAaaNoSecrets(t, tAuth, []string{cmd, file})
		return res, err
	}
}

func dispTestLoad(t *testing.T, d *server.Disp, sid, config string) {
	handleDispTestLoadOrMergePass(t, d.Load, sid, config)
}

func dispTestLoadFails(
	t *testing.T,
	d *server.Disp,
	sid, config string,
	errMsgs *assert.ExpectedMessages,
) {
	handleDispTestLoadOrMergeFails(t, d.Load, sid, config, errMsgs)
}

// Load or Merge function returns { true, nil }
func handleDispTestLoadOrMergePass(
	t *testing.T,
	loadOrMerge loadOrMergeFn,
	sid, config string,
) {
	ok, err := dispTestLoadOrMergeCommon(t, loadOrMerge, sid, config)
	if err != nil {
		t.Fatalf(err.Error())
		return
	}
	if !ok {
		t.Fatalf("Load failed but no error given.")
		return
	}
}

// Load or Merge function returns { true, warnings }
func handleDispTestLoadOrMergeReportsWarnings(
	t *testing.T,
	loadOrMerge loadOrMergeFn,
	sid, config string,
	warnMsgs *assert.ExpectedMessages,
) {
	ok, err := dispTestLoadOrMergeCommon(t, loadOrMerge, sid, config)
	if !ok {
		t.Fatalf("Load / merge wrongly reported failure for warnings.")
		return
	}
	if err == nil {
		t.Fatalf("Load / merge should have returned warnings.")
		return
	}
	warnMsgs.ContainedIn(t, err.Error())
}

// Load or Merge function returns { false, errors }
func handleDispTestLoadOrMergeFails(
	t *testing.T,
	loadOrMerge loadOrMergeFn,
	sid, config string,
	errMsgs *assert.ExpectedMessages,
) {
	ok, err := dispTestLoadOrMergeCommon(t, loadOrMerge, sid, config)
	if err != nil {
		errMsgs.ContainedIn(t, err.Error())
		return
	}

	if ok {
		t.Fatalf("\nUnexpected success loading config.\n")
		return
	}
}

var loadOrMergeSchema = `
leaf testbool {
	type boolean;
}
leaf teststring {
	type string;
}
leaf testint {
	type uint8 {
		range 1..64;
	}
	must "../testbool = true()";
}`

var initConfig = `
	teststring stuff
`

func createLoadTestDispatcherAndSession(
	t *testing.T,
	schema, initConfig, sid string,
) *server.Disp {
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), schema, initConfig)
	dispTestSetupSession(t, d, sid)
	return d
}

// TESTS

func TestLoadSuccess(t *testing.T) {

	d := createLoadTestDispatcherAndSession(
		t, loadOrMergeSchema, initConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergePass(t, d.Load, testSID, testConfig)
}

func TestLoadSuccessCommandAuthz(t *testing.T) {

	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	loadFn := dispLoadOrMergeFnCheckCmdAuthz(t, a, d.Load, "load")
	handleDispTestLoadOrMergePass(t, loadFn, testSID, testConfig)
}

func TestMergeSuccess(t *testing.T) {

	d := createLoadTestDispatcherAndSession(
		t, loadOrMergeSchema, initConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergePass(t, d.Merge, testSID, testConfig)
}

func TestMergeSuccessCommandAuthz(t *testing.T) {

	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	mergeFn := dispLoadOrMergeFnCheckCmdAuthz(t, a, d.Merge, "merge")
	handleDispTestLoadOrMergePass(t, mergeFn, testSID, testConfig)
}

func TestLoadInvalidPathIgnoresError(t *testing.T) {

	d := createLoadTestDispatcherAndSession(
		t, loadOrMergeSchema, initConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testnonexistentleaf", "8"))

	handleDispTestLoadOrMergePass(t, d.Load, testSID, testConfig)
}

func TestMergeInvalidPathIgnoresError(t *testing.T) {

	d := createLoadTestDispatcherAndSession(
		t, loadOrMergeSchema, initConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testnonexistentleaf", "8"))

	handleDispTestLoadOrMergePass(t, d.Merge, testSID, testConfig)
}

func TestLoadFails(t *testing.T) {

	d := newTestDispatcherWithCustomAuth(
		t, auth.TestAutherDenyAll(),
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)
	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergeFails(t, d.Load, testSID, testConfig,
		assert.NewExpectedMessages("authorization failed"))
}

func TestMergeFails(t *testing.T) {

	d := newTestDispatcherWithCustomAuth(
		t, auth.TestAutherDenyAll(),
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)
	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergeFails(t, d.Merge, testSID, testConfig,
		assert.NewExpectedMessages("authorization failed"))
}

func TestLoadReportWarningSuccess(t *testing.T) {

	d := createLoadTestDispatcherAndSession(
		t, loadOrMergeSchema, initConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergePass(t, d.LoadReportWarnings, testSID, testConfig)
}

func TestLoadReportWarningSuccessCommandAuthz(t *testing.T) {

	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	loadFn := dispLoadOrMergeFnCheckCmdAuthz(t, a, d.LoadReportWarnings, "load")
	handleDispTestLoadOrMergePass(t, loadFn, testSID, testConfig)
}

func TestMergeReportWarningSuccess(t *testing.T) {

	d := createLoadTestDispatcherAndSession(
		t, loadOrMergeSchema, initConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergePass(t, d.MergeReportWarnings, testSID, testConfig)
}

func TestMergeReportWarningSuccessCommandAuthz(t *testing.T) {

	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	mergeFn := dispLoadOrMergeFnCheckCmdAuthz(t, a, d.MergeReportWarnings, "merge")
	handleDispTestLoadOrMergePass(t, mergeFn, testSID, testConfig)
}

func TestLoadReportWarningFail(t *testing.T) {

	d := newTestDispatcherWithCustomAuth(
		t, auth.TestAutherDenyAll(),
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)
	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergeFails(t, d.LoadReportWarnings, testSID, testConfig,
		assert.NewExpectedMessages("authorization failed"))
}

func TestMergeReportWarningFail(t *testing.T) {

	d := newTestDispatcherWithCustomAuth(
		t, auth.TestAutherDenyAll(),
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)
	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	handleDispTestLoadOrMergeFails(
		t, d.MergeReportWarnings, testSID, testConfig,
		assert.NewExpectedMessages("authorization failed"))
}

func TestLoadReportWarningShowsTopLevelInvalidPathWarning(t *testing.T) {

	testpath := "/nonexistentleaf"

	oc := newOutputChecker(t).
		setSchema(loadOrMergeSchema).
		setInitConfig(initConfig)

	testConfig := testutils.Root(
		testutils.Leaf("nonexistentleaf", "8"))

	oc.loadConfig(testConfig)

	oc.setExpErr(errtest.NewInvalidPathError(t, testpath)).
		addExtraErrs(errtest.WarningsGeneratedStr).
		addPathPrefix(testpath).
		setUnexpErrs(validationFailedStr)

	oc.verifyCLIError()
}

func TestMergeReportWarningShowsTopLevelInvalidPathWarning(t *testing.T) {

	testpath := "/nonexistentleaf"

	oc := newOutputChecker(t).
		setSchema(loadOrMergeSchema).
		setInitConfig(initConfig)

	testConfig := testutils.Root(
		testutils.Leaf("nonexistentleaf", "8"))

	oc.mergeConfig(testConfig)

	oc.setExpErr(errtest.NewInvalidPathError(t, testpath)).
		addExtraErrs(errtest.WarningsGeneratedStr).
		addPathPrefix(testpath).
		setUnexpErrs(validationFailedStr)

	oc.verifyCLIError()
}

// We've tested all the various combinations of returned params for Load
// and Merge (basic and ReportWarnings variants).  Now we need to check
// warnings are correctly handled in various scenarios, but need only do
// it for either load or merge, not both, as it's common code we've now
// tested above.

func TestLoadReportWarningShowsMultipleWarnings(t *testing.T) {

	d := createLoadTestDispatcherAndSession(
		t, loadOrMergeSchema, initConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("nonexistentLeaf", "8"),
		testutils.Leaf("anotherNonexistentLeaf", "66"))

	expMsgs := errtest.NewInvalidPathError(t,
		"/nonexistentLeaf").SetCliErrorStrings()
	expMsgs = append(expMsgs, errtest.NewInvalidPathError(t,
		"/anotherNonexistentLeaf").SetCliErrorStrings()...)
	expMsgs = append(expMsgs, errtest.WarningsGeneratedStr)
	expWarnings := assert.NewExpectedMessages(expMsgs...)

	handleDispTestLoadOrMergeFails(
		t, d.LoadReportWarnings, testSID, testConfig,
		expWarnings)
}

func TestLoadReportWarningShowsRangeWarning(t *testing.T) {

	testpath := "/testint/99"

	oc := newOutputChecker(t).
		setSchema(loadOrMergeSchema).
		setInitConfig(initConfig)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "99"))

	oc.loadConfig(testConfig)

	oc.setExpErr(errtest.NewInvalidRangeError(t, testpath, 1, 64)).
		addExtraErrs(errtest.WarningsGeneratedStr).
		addPathPrefix(testpath).
		addExtraErrs(validationFailedStr)

	oc.verifyCLIError()
}

func TestLoadReportWarningShowsMissingChildWarning(t *testing.T) {

	testpath := "/testint"

	oc := newOutputChecker(t).
		setSchema(loadOrMergeSchema).
		setInitConfig(initConfig)

	oc.loadConfig("testint\n")

	oc.setExpErr(errtest.NewNodeRequiresValueError(t, testpath)).
		addExtraErrs(errtest.WarningsGeneratedStr).
		addPathPrefix(testpath).
		addExtraErrs(validationFailedStr)

	oc.verifyCLIError()
}

func TestMergeReportWarningShowsTypeWarning(t *testing.T) {

	testpath := "/testint/notAnInt"

	oc := newOutputChecker(t).
		setSchema(loadOrMergeSchema).
		setInitConfig(initConfig)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "notAnInt"))

	oc.mergeConfig(testConfig)

	oc.setExpErr(errtest.NewInvalidTypeError(
		t, testpath, "an uint8")).
		addExtraErrs(errtest.WarningsGeneratedStr).
		addPathPrefix(testpath).
		addExtraErrs(validationFailedStr)

	oc.verifyCLIError()
}

func TestMergeMustDefaultErrorValidationWarning(t *testing.T) {

	oc := newOutputChecker(t).
		setSchema(loadOrMergeSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	oc.mergeConfig(testConfig)
	oc.verifyNoError()

	errList := []*errtest.ExpMgmtError{
		errtest.MustViolationMgmtErr(
			"'must' condition is false: '../testbool = true()'",
			"/testint/8"),
	}

	oc.validate()
	oc.verifyMgmtErrorList(errList)
}

var normalizeSchema = `
leaf testLeaf {
	type string;
}
leaf-list testAddress {
	type union {
		type enumeration {
			enum dhcp;
			enum dhcpv6;
		}
		type string {
			pattern '(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.){3}'
				+  '([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])'
				+ '/(([0-9])|([1-2][0-9])|(3[0-2]))';
			configd:normalize "normalize ipv4";
		}
		type string {
			pattern '((:|[0-9a-fA-F]{0,4}):)([0-9a-fA-F]{0,4}:){0,5}'
				+ '((([0-9a-fA-F]{0,4}:)?(:|[0-9a-fA-F]{0,4}))|'
				+ '(((25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.){3}'
				+ '(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])))'
				+ '(/(([0-9])|([0-9]{2})|(1[0-1][0-9])|(12[0-8])))';
			pattern '(([^:]+:){6}(([^:]+:[^:]+)|(.*\..*)))|'
				+ '((([^:]+:)*[^:]+)?::(([^:]+:)*[^:]+)?)'
				+ '(/.+)';
		}
		configd:normalize 'echo %s';
	}
}
`

var initNormalizeConfig = testutils.Root(
	testutils.Leaf("testLeaf", "just-testing"))

// Ensure load operation runs normalize script.
func TestLoadNormalizesIPv6Addresses(t *testing.T) {

	testSchema := fmt.Sprintf(normalizeSchema, "2001::1/64")

	d := createLoadTestDispatcherAndSession(
		t, testSchema, initNormalizeConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testAddress", "2001:0::1/64"))

	handleDispTestLoadOrMergePass(t, d.Load, testSID, testConfig)

	expConfig := testutils.Root(
		testutils.Leaf("testAddress", "2001::1/64"))

	commitAndVerifyConfig(t, d, expConfig)
}

func TestMergeNormalizesIPv6Addresses(t *testing.T) {

	testSchema := fmt.Sprintf(normalizeSchema, "2001::1/64")

	d := createLoadTestDispatcherAndSession(
		t, testSchema, initNormalizeConfig, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testAddress", "2001:0::1/64"))

	handleDispTestLoadOrMergePass(t, d.Merge, testSID, testConfig)

	expConfig := testutils.Root(
		testutils.Leaf("testAddress", "2001::1/64"),
		testutils.Leaf("testLeaf", "just-testing"))

	commitAndVerifyConfig(t, d, expConfig)
}

func TestLoadWarningsForDifferentErrorTypes(t *testing.T) {
	t.Skipf("TBD")
	// interfaces lopbock (1 level down invalid)
	// interfaces loopback lo2 descroption (3 level down invalid)
	// interfaces loopback lo999999 (value validation fail)
	// interfaces loopback lo2 description (node requires value)
	// pattern error ...
}
