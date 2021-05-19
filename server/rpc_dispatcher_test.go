// Copyright (c) 2017-2021, AT&T Intellectual Property Inc. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danos/config/schema"
	"github.com/danos/configd"
	"github.com/danos/configd/server"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

type testRpcCaller struct {
	t            *testing.T
	expInputJson string
	outputJson   string
	outputErr    error
	outputStatus bool // Currently ignored in real code.
}

func (trc *testRpcCaller) CallRpc(ctx *configd.Context, modelName, rpcName, inputTreeJson string,
) (string, error) {
	if inputTreeJson != trc.expInputJson {
		trc.t.Fatalf(
			"Incorrect JSON in call to CallRpc()\nExp:\t%s\nGot:\t%s\n",
			trc.expInputJson, inputTreeJson)
		return "", nil
	}

	return trc.outputJson, trc.outputErr
}

const rpcTestComp = `[Vyatta Component]
Name=net.vyatta.test.validation
Description=Validation Test Component
ExecName=/opt/vyatta/sbin/validation-test
ConfigFile=/etc/vyatta/validation-test.conf

[Model net.vyatta.test.validation]
Modules=vyatta-test-validation-v1
ModelSets=vyatta-v1`

const (
	testRpcNamespace     = "urn:vyatta.com:test:vyatta-test-validation-v1"
	testRpcModule        = "vyatta-test-validation-v1"
	testRpcInvalidModule = "vyatta-test-invalid-v1"
	testRpcName          = "testRpc"
	testRpcInvalidName   = "testRpcInvalid"
	configPath           = "Configuration path: "
)

var rpcTestSchemaTemplate = `
	rpc testRpc {
	input {
%s
	}
    output {
        leaf status {
            type string;
        }
    }
	configd:call-rpc "echo {\"status\":\"ok\"}";
}`

var sessionCountSchemaSnippet = `
		leaf session-count {
			type uint8;
			default 100;
		}`

var validSessionCountXML = `
	<data>
	    <session-count>66</session-count>
	</data>`
var validSessionCountJSON = `{"vyatta-test-validation-v1:session-count":66}`

var invalidSessionCountXML = `
	<data>
	    <session-count>666</session-count>
	</data>`
var invalidSessionCountJSON = `{"session-count":666}`

var invalidInputXML = `
	<data>
	    <non-existent>666</non-existent>
	</data>`

var validStatusReplyJSON = `{"status":"ok"}`
var invalidReplyJSON = `{"status":}`
var replyNotMatchingSchemaJSON = `{"non-existent":"ok"}`

var validStatusReplyXML = `<rpc-reply><status xmlns="urn:vyatta.com:test:vyatta-test-validation-v1">ok</status></rpc-reply>`

func genRpcTestSchema(input string) []sessiontest.TestSchema {
	return []sessiontest.TestSchema{
		{
			Name: sessiontest.NameDef{
				Namespace: "vyatta-test-validation-v1",
				Prefix:    "validation",
			},
			SchemaSnippet: fmt.Sprintf(rpcTestSchemaTemplate, input),
		},
	}
}

// NETCONF RPC TESTS
//
// With the coming of VCI, it is no longer possible to test the full
// processing of a NETCONF RPC from receipt by the dispatcher through
// calling of the right RPC to return of the XML reply.  Instead, the
// testing needs to be split up into the areas handled by different
// daemons and libraries, as follows:
//
// 1) sessiond (this file - configd:configd/server/...)
//
//  - model name lookup
//  - XML conversion to JSON (for NETCONF)
//  - correct RPC call
//  - correct handling of reply, conversion back to XML
//
// 2) VCI library (vci:dbus*.go)
//
//  - RPC method wrapping (dbus_wrap_test.go)
//  - correct operation of wrapped method, including validation call to yangd
//  - correct handling of reply
//
// 3) yangd (configd:configd/cmd/configd:yangd*.go)
//
//  - correct validation of RPC input JSON against schema
//  - who adds defaults, and when (both for input and output)? TODO
//    - possibly wrapped method in VCI:dbus_wrap should do this?
//
// 4) Legacy RPC handling
//
//  - now in provisiond
//  - given correct JSON in, do we do right thing?
//
// 5) Overall NETCONF/callrpc RPC handling
//
//  - now done via ROBOT-based acceptance tests
//

func checkRpcCallPasses(
	t *testing.T,
	d *server.Disp,
	trc *testRpcCaller,
	moduleOrNamespace string,
	rpcName string,
	inputRpcBody string,
	encoding string,
	expReply string) {

	actReply, err := d.CallRpcWithCaller(
		moduleOrNamespace, rpcName,
		inputRpcBody, encoding,
		trc)

	if err != nil {
		t.Fatalf("Unexpected failure: %s\n", err.Error())
	}

	if actReply != expReply {
		t.Fatalf("Unexpected RPC reply:\nExp: %s\nGot: %s\n",
			expReply, actReply)
	}
}

func checkRpcCallFails(
	t *testing.T,
	d *server.Disp,
	trc *testRpcCaller,
	moduleOrNamespace string,
	rpcName string,
	inputRpcBody string,
	encoding string,
	expErrors ...string) {

	_, err := d.CallRpcWithCaller(
		moduleOrNamespace, rpcName,
		inputRpcBody, encoding,
		trc)

	if err == nil {
		t.Fatalf("Unexpected success.")
	}
	for _, expError := range expErrors {
		if !strings.Contains(
			err.Error(), expError) {
			t.Fatalf("Failed to get expected error.\nExp: %s\nGot: %s\n",
				expError, err.Error())
		}
	}
}

func TestXMLRpcPass(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	trc := &testRpcCaller{
		t:            t,
		expInputJson: validSessionCountJSON,
		outputJson:   validStatusReplyJSON}

	checkRpcCallPasses(
		t, d, trc,
		testRpcNamespace, testRpcName,
		validSessionCountXML, "xml",
		validStatusReplyXML)
}

func TestJSONRpcPass(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	trc := &testRpcCaller{
		t:            t,
		expInputJson: validSessionCountJSON,
		outputJson:   validStatusReplyJSON}

	checkRpcCallPasses(
		t, d, trc,
		testRpcModule, testRpcName,
		validSessionCountJSON, "json",
		validStatusReplyJSON)
}

func TestFindRpcJSONLookupFailure(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t))

	expectedError := "Unknown RPC (json) " + testRpcInvalidModule

	checkRpcCallFails(
		t, d, nil,
		testRpcInvalidModule, testRpcName,
		validSessionCountJSON, "json",
		expectedError)
}

func TestFindRpcUnknownEncodingFailure(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	expectedError := "Unknown RPC (unknownEncoding) " + testRpcModule

	checkRpcCallFails(
		t, d, nil,
		testRpcModule, testRpcName,
		validSessionCountJSON, "unknownEncoding",
		expectedError)
}

func TestFindRpcNoRpcsForNamespaceFailure(t *testing.T) {
	noRpcSchema := []sessiontest.TestSchema{
		{
			Name: sessiontest.NameDef{
				Namespace: "vyatta-test-validation-v1",
				Prefix:    "validation",
			},
			SchemaSnippet: "",
		},
	}
	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(noRpcSchema).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	expectedError := "Unknown RPC (xml) " + testRpcNamespace

	checkRpcCallFails(
		t, d, nil,
		testRpcNamespace, testRpcName,
		validSessionCountXML, "xml",
		expectedError)
}

func TestFindRpcLookupFailure(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	expectedError := "Unknown RPC (json) " + testRpcModule + ":" +
		testRpcInvalidName

	checkRpcCallFails(
		t, d, nil,
		testRpcModule, testRpcInvalidName,
		validSessionCountJSON, "json",
		expectedError)
}

func TestCallRpcModelNameLookupFailure(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)))

	expectedError := "Unknown model for RPC " + testRpcNamespace

	checkRpcCallFails(
		t, d, nil,
		testRpcNamespace, testRpcName,
		validSessionCountXML, "xml",
		expectedError)
}

func TestHandleRpcConvertXMLInputDecodeFail(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
		setComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}).
		init()

	oc.setExpErr(errtest.NewInvalidNodeError(t, "/non-existent")).
		setUnexpErrs(configPath)

	oc.callXmlRpcAndVerifyError(testRpcNamespace, testRpcName,
		invalidInputXML)
}

var outOfRangeInputXML = `
	<data>
	    <session-count>257</session-count>
	</data>`

func TestHandleRpcConvertXMLInputDecodeFailOutOfRangeValue(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
		setComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}).
		init()

	oc.setExpErr(errtest.NewInvalidNodeError(t, "/session-count/257")).
		setUnexpErrs(configPath)

	oc.callXmlRpcAndVerifyError(testRpcNamespace, testRpcName,
		outOfRangeInputXML)
}

func TestHandleRpcCallRpcFail(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	expectedError := fmt.Errorf("Oops - vci.CallRPC() failed")

	trc := &testRpcCaller{
		t:            t,
		expInputJson: validSessionCountJSON,
		outputErr:    expectedError,
		outputJson:   validStatusReplyJSON}

	checkRpcCallFails(
		t, d, trc,
		testRpcNamespace, testRpcName,
		validSessionCountXML, "xml",
		expectedError.Error())
}

func TestHandleRpcConvertIllegalJSONOutputUnmarshalFail(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	trc := &testRpcCaller{
		t:            t,
		expInputJson: validSessionCountJSON,
		outputJson:   invalidReplyJSON}

	expectedError := "invalid character '}' looking for beginning of value"

	checkRpcCallFails(
		t, d, trc,
		testRpcNamespace, testRpcName,
		validSessionCountXML, "xml",
		expectedError)
}

func TestHandleRpcConvertMismatchedJSONOutputUnmarshalFail(t *testing.T) {

	d := newTestDispatcherFromTestSpec(
		sessiontest.NewTestSpec(t).
			SetSchemaDefs(genRpcTestSchema(sessionCountSchemaSnippet)).
			SetComponents(schema.VyattaV1ModelSet, []string{rpcTestComp}))

	trc := &testRpcCaller{
		t:            t,
		expInputJson: validSessionCountJSON,
		outputJson:   replyNotMatchingSchemaJSON}

	expectedErrors := errtest.NewSchemaMismatchError(t, "/non-existent").
		RawErrorStrings()

	checkRpcCallFails(
		t, d, trc,
		testRpcNamespace, testRpcName,
		validSessionCountXML, "xml",
		expectedErrors...)
}
