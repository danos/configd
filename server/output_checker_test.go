// Copyright (c) 2017-2021, AT&T Intellectual Property Inc. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/server"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror"
	"github.com/danos/mgmterror/errtest"
	"github.com/danos/utils/pathutil"
)

// outputChecker - new improved version of the newTestDispatcher functions
//
// This allows the chaining together of methods to set up tests, add
// configuration, expected output, and run tests.  It avoids the need to
// have lots of very similar APIs with subtly different parameters - a new
// option gets its own API that can be added into the chain instead.
//
// Methods are split into 'given', 'when', 'then' sections.
//
type outputChecker struct {
	t              *testing.T
	d              *server.Disp
	schemaSnippet  string
	schemaDefs     []sessiontest.TestSchema
	config         string
	actStatus      bool
	actOutput      string
	actErr         error
	expErr         *errtest.TestError
	extraErrs      []string
	unexpErrs      []string
	comps          []string
	modelSetName   string
	auther         auth.Auther
	isConfigdUser  bool
	inSecretsGroup bool
}

func newOutputChecker(t *testing.T) *outputChecker {
	return &outputChecker{
		t:              t,
		isConfigdUser:  true,
		inSecretsGroup: true,
	}
}

// 'given' - test setup

func (oc *outputChecker) setSchema(schemaSnippet string) *outputChecker {
	oc.schemaSnippet = schemaSnippet
	return oc
}

func (oc *outputChecker) setAuther(
	auther auth.Auther,
	isConfigdUser,
	inSecretsGroup bool,
) *outputChecker {

	oc.auther = auther
	oc.isConfigdUser = isConfigdUser
	oc.inSecretsGroup = inSecretsGroup

	return oc
}

func (oc *outputChecker) setSchemaDefs(
	schemaDefs []sessiontest.TestSchema,
) *outputChecker {

	oc.schemaDefs = schemaDefs
	return oc
}

func (oc *outputChecker) setSchemaDefsByRef(
	schemaDefs []*sessiontest.TestSchema,
) *outputChecker {

	for _, schema := range schemaDefs {
		oc.schemaDefs = append(oc.schemaDefs, *schema)
	}
	return oc
}

func (oc *outputChecker) setComponents(
	modelSetName string,
	comps []string,
) *outputChecker {
	oc.comps = comps
	oc.modelSetName = modelSetName
	return oc
}

func (oc *outputChecker) setInitConfig(config string) *outputChecker {
	oc.config = config
	return oc
}

func (oc *outputChecker) init() *outputChecker {
	if oc.d != nil {
		return oc
	}
	if oc.t == nil {
		panic("Must have 'testing.T' object for outputChecker!!!")
	}
	if oc.schemaSnippet != "" {
		oc.schemaDefs = genSetTestSchema(oc.schemaSnippet)
	}
	if len(oc.schemaDefs) == 0 {
		panic("Must have schema(s) defined for outputChecker!!!")
	}
	oc.d = newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(oc.t).
			SetSchemaDefs(oc.schemaDefs).
			SetConfig(oc.config).
			SetComponents(oc.modelSetName, oc.comps).
			SetAuther(oc.auther, oc.isConfigdUser, oc.inSecretsGroup))

	dispTestSetupSession(oc.t, oc.d, testSID)
	return oc
}

// ACTIONS

func (oc *outputChecker) set(testPath string) *outputChecker {
	oc.init()
	oc.actOutput, oc.actErr = oc.d.Set(testSID, testPath)
	return oc
}

func (oc *outputChecker) validate() *outputChecker {
	oc.init()
	oc.actOutput, oc.actErr = oc.d.Validate(testSID)
	return oc
}

func (oc *outputChecker) commit() *outputChecker {
	oc.init()
	oc.actOutput, oc.actErr = oc.d.Commit(testSID, "commit msg", false)
	return oc
}

func (oc *outputChecker) expand(testPath string) *outputChecker {
	oc.init()
	oc.actOutput, oc.actErr = oc.d.Expand(testPath)
	return oc
}

func (oc *outputChecker) expandWithPrefix(
	testPath, testPrefix string,
	testPos int,
) *outputChecker {

	oc.init()
	oc.actOutput, oc.actErr = oc.d.ExpandWithPrefix(
		testPath, testPrefix, testPos)
	return oc
}

func (oc *outputChecker) exists(testPath string) *outputChecker {
	oc.init()
	oc.actStatus, oc.actErr = oc.d.Exists(rpc.RUNNING, testSID, testPath)
	return oc
}

func (oc *outputChecker) delete(testPath string) *outputChecker {
	oc.init()
	op := false
	op, oc.actErr = oc.d.Delete(testSID, testPath)
	if op {
		oc.t.Fatalf("Expected test to return false.")
		return nil
	}
	return oc
}

func (oc *outputChecker) copyConfig(
	sourceDatastore,
	sourceEncoding,
	sourceConfig,
	sourceURL,
	targetDatastore,
	targetURL string,
) *outputChecker {
	oc.init()
	_, oc.actErr = oc.d.CopyConfig(
		testSID, sourceDatastore, sourceEncoding, sourceConfig,
		sourceURL, targetDatastore, targetURL)

	return oc
}

func (oc *outputChecker) loadConfig(
	testConfig string,
) *outputChecker {
	oc.init()
	return oc.loadOrMergeConfig(
		oc.d.LoadReportWarnings, testConfig)
}

func (oc *outputChecker) mergeConfig(
	testConfig string,
) *outputChecker {
	oc.init()
	return oc.loadOrMergeConfig(
		oc.d.MergeReportWarnings, testConfig)
}

func (oc *outputChecker) loadOrMergeConfig(
	loadOrMerge loadOrMergeFn,
	testConfig string,
) *outputChecker {

	// Convert config to a file.
	tmpfile, err := ioutil.TempFile("", "dispTestLoadOrMerge")
	if err != nil {
		oc.t.Fatalf("Unable to create temp file for load/merge test: %s\n",
			err.Error())
		return nil
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(testConfig); err != nil {
		oc.t.Fatalf("Unable to write temp file for load/merge test: %s\n",
			err.Error())
		return nil
	}
	if err := tmpfile.Close(); err != nil {
		oc.t.Fatalf("Unable to close temp file for load/merge test: %s\n",
			err.Error())
		return nil
	}

	ok, err := loadOrMerge(testSID, tmpfile.Name())
	if !ok {
		oc.t.Fatalf("Unexpected errors during load operation.")
		return nil
	}
	oc.actErr = err
	return oc
}

func (oc *outputChecker) validateConfig(
	encoding,
	sourceConfig string,
) *outputChecker {
	oc.init()
	_, oc.actErr = oc.d.ValidateConfig(testSID, encoding, sourceConfig)

	return oc
}

func (oc *outputChecker) callXmlRpcAndVerifyError(
	moduleOrNamespace,
	rpcName,
	inputXml string,
) *outputChecker {

	_, err := oc.d.CallRpcWithCaller(
		moduleOrNamespace, rpcName,
		inputXml, "xml",
		nil)

	if err == nil {
		oc.t.Fatalf("Unexpected success calling XML RPC")
	}

	return oc.verifyRpcError(err)
}

func (oc *outputChecker) getSchemaAndVerify(
	modOrSubmodName string,
	expSchemaSnippet string,
	unexpSchemaSnippets ...string,
) *outputChecker {

	actSchema, err := oc.d.SchemaGetUnescaped(modOrSubmodName)

	if err != nil {
		oc.t.Fatalf("Unable to get schema: %s\n", err.Error())
		return oc
	}

	if !strings.Contains(actSchema, expSchemaSnippet) {
		oc.t.Logf("Expexcted schema not found in returned schema\n\n")
		oc.t.Fatalf("Exp:\n%s\n\nGot\n%s\n", expSchemaSnippet, actSchema)
		return oc
	}

	for _, unexpSnippet := range unexpSchemaSnippets {
		if strings.Contains(actSchema, unexpSnippet) {
			oc.t.Logf("Unexpexcted schema found in returned schema\n\n")
			oc.t.Fatalf("Not exp:\n%s\n\nGot\n%s\n",
				unexpSnippet, actSchema)
			return oc
		}
	}

	return oc
}

func (oc *outputChecker) getNonExistentSchema(
	modOrSubmodName string,
) *outputChecker {

	_, err := oc.d.SchemaGetUnescaped(modOrSubmodName)

	if err == nil {
		oc.t.Fatalf("Should not have found (sub)module %s\n", modOrSubmodName)
		return oc
	}

	if !strings.Contains(err.Error(),
		fmt.Sprintf("Error: unknown (sub)module %s",
			modOrSubmodName)) {
		oc.t.Fatalf("Wrong error getting unexpected (sub)module: %s\n",
			err.Error())
	}
	return oc
}

func (oc *outputChecker) checkSchemasGettable(
	incSubmods bool,
	schemaNames ...string,
) *outputChecker {

	var schemas string

	if incSubmods {
		schemas, _ = oc.d.GetSchemas()
	} else {
		schemas, _ = oc.d.GetModuleSchemas()
	}

	for _, name := range schemaNames {
		if !strings.Contains(schemas, name) {
			oc.t.Fatalf("GetSchema() output missing %s\nOutput:\n%s\n",
				name, schemas)
			return oc
		}
	}

	return oc
}

func (oc *outputChecker) checkSchemasNotGettable(
	incSubmods bool,
	schemaNames ...string,
) *outputChecker {

	var schemas string

	if incSubmods {
		schemas, _ = oc.d.GetSchemas()
	} else {
		schemas, _ = oc.d.GetModuleSchemas()
	}

	for _, name := range schemaNames {
		if strings.Contains(schemas, name) {
			oc.t.Fatalf("GetSchema() output contains %s\nOutput:\n%s\n",
				name, schemas)
			return oc
		}
	}

	return oc
}

// 'then' - set expected results and verify them

// verifyOutputOkNoError - check no error, and output matches expOut
func (oc *outputChecker) verifyOutputOkNoError(expOut string) *outputChecker {
	if oc.actErr != nil {
		oc.t.Fatalf("Operation failed: %s\n", oc.actErr)
		return nil
	}
	if oc.actOutput != expOut {
		oc.t.Fatalf("Unexpected Output.\nExp: %s\nGot: %s\n",
			expOut, oc.actOutput)
		return nil
	}
	return oc
}

// verifyOutputContentNoError - check no error and output contains exp strings.
// Useful where we don't need (or want) an exact match, eg for multiline output
// or where in testing we get the likes of 'run-parts: failed to open ...'
// errors in commits that we can ignore.
func (oc *outputChecker) verifyOutputContentNoError(
	expOuts []string,
) *outputChecker {

	if oc.actErr != nil {
		oc.t.Fatalf("Operation failed: %s\n", oc.actErr)
		return nil
	}

	expMsgs := assert.NewExpectedMessages(expOuts...)
	expMsgs.ContainedIn(oc.t, oc.actOutput)

	return oc
}

// verifyNoError - check no error. Don't care what's in output.
func (oc *outputChecker) verifyNoError() *outputChecker {
	if oc.actErr != nil {
		oc.t.Fatalf("Operation failed: %s\n", oc.actErr)
		return nil
	}
	return oc
}

// verifyStatusOkNoError - check result is true, and no error
func (oc *outputChecker) verifyStatusOkNoError() *outputChecker {
	if oc.actErr != nil {
		oc.t.Fatalf("Operation failed: %s\n", oc.actErr)
		return nil
	}
	if !oc.actStatus {
		oc.t.Fatalf("Operation failed. No error given.")
		return nil
	}
	return oc
}

// verifyStatusFailWithRawError - check result is false, and check err text
// Here error is not pretty printed, and it may not include the path.
func (oc *outputChecker) verifyStatusFail() *outputChecker {

	if oc.actStatus {
		oc.t.Fatalf("Operation succeeded unexpectedly.")
		return nil
	}
	return oc
}

func (oc *outputChecker) verifyRawError() *outputChecker {
	if oc.actErr == nil {
		oc.t.Fatalf("Operation succeeded unexpectedly: %s\n", oc.actErr)
		return nil
	}
	oc.verifyErrors(oc.actErr, oc.expErr.RawErrorStringsNoPath()...)
	return oc
}

// ExpMgmtErrors are formatted for pretty-print output.
func (oc *outputChecker) verifyMgmtErrors(
	expErrs *errtest.ExpMgmtErrors,
) *outputChecker {
	expErrs.Matches(oc.actErr)
	return oc
}

// Check single MgmtError fields.
func (oc *outputChecker) verifyMgmtError(
	expErr *errtest.ExpMgmtError,
) *outputChecker {
	if oc.actErr == nil {
		oc.t.Fatalf("Expected error but none occurred.")
	}
	switch oc.actErr.(type) {
	case mgmterror.MgmtErrorList:
		oc.verifyMgmtErrorList([]*errtest.ExpMgmtError{expErr})
	default:
		errtest.CheckMgmtErrors(
			oc.t, []*errtest.ExpMgmtError{expErr}, []error{oc.actErr})
	}
	return oc
}

// ExpMgmtError objects are 'raw' MgmtError so we can verify each field in the
// error individually using this method.
func (oc *outputChecker) verifyMgmtErrorList(
	expErrList []*errtest.ExpMgmtError,
) *outputChecker {
	// Check type
	mel, ok := oc.actErr.(mgmterror.MgmtErrorList)
	if !ok {
		oc.t.Fatalf("Actual error was not MgmtErrorList")
	}

	actErrs := mel.Errors()
	if len(actErrs) != len(expErrList) {
		oc.t.Fatalf("Expected %d errors, but got %d\n",
			len(expErrList), len(actErrs))
	}

	errtest.CheckMgmtErrors(oc.t, expErrList, actErrs)
	return oc
}

// verifyCLIError - verify errors seen for set/delete and other CLI operations
func (oc *outputChecker) verifyCLIError() *outputChecker {

	if oc.actOutput != "" {
		oc.t.Fatalf("Expected test to return no non-error output.")
		return nil
	}

	if oc.actErr == nil {
		oc.t.Fatalf("Expected test to return non-nil error.")
		return nil
	}

	oc.verifyErrors(oc.actErr, oc.expErr.SetCliErrorStrings()...)

	return oc
}

// verifyRpcError - verify errors seen for RPC operations
func (oc *outputChecker) verifyRpcError(err error) *outputChecker {
	if err == nil {
		oc.t.Fatalf("Expected test to return non-nil error.")
		return nil
	}

	oc.verifyErrors(err, oc.expErr.RpcErrorStrings()...)

	return oc
}

func (oc *outputChecker) verifyErrors(
	actErr error,
	errList ...string,
) *outputChecker {

	errList = append(errList, oc.extraErrs...)

	expErrs := assert.NewExpectedMessages(errList...)
	expErrs.ContainedIn(oc.t, actErr.Error())

	unexpErrs := assert.NewExpectedMessages(oc.unexpErrs...)
	unexpErrs.NotContainedIn(oc.t, actErr.Error())

	return oc
}

func (oc *outputChecker) setExpErr(expErr *errtest.TestError) *outputChecker {
	oc.expErr = expErr
	return oc
}

func (oc *outputChecker) addExtraErrs(extraErrs ...string) *outputChecker {
	oc.extraErrs = append(oc.extraErrs, extraErrs...)
	return oc
}

// addPathPrefix - for load/merge, add '[<spaced path>]:' to expected errs
// Calling this out separately to addExtraErrs means if we ever decide to
// ditch this prefix or change the format, we only need modify this function
// here in the test code.
func (oc *outputChecker) addPathPrefix(path string) *outputChecker {
	oc.extraErrs = append(oc.extraErrs, fmt.Sprintf("[%s]:",
		strings.Join(pathutil.Makepath(path), " ")))
	return oc
}

func (oc *outputChecker) setUnexpErrs(unexpErrs ...string) *outputChecker {
	oc.unexpErrs = append(oc.unexpErrs, unexpErrs...)
	return oc
}
