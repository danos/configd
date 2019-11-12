// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danos/config/schema"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

// HELPER FUNCTIONS

const (
	noNamespaceSpecified = ""
	noModelNameSpecified = ""
	emptyConfig          = ""
	validTrueJson        = "{\"yangd-v1:valid\":true}"
	jsonNotParsed        = ""
)

func getModelSet(
	t *testing.T,
	schemaDefs []sessiontest.TestSchema,
	initConfig string,
) (schema.ModelSet, schema.ModelSet) {

	srv, _ := sessiontest.TstStartupMultipleSchemas(
		t, schemaDefs, initConfig)
	return srv.Ms, srv.MsFull
}

func newTestYangd(
	t *testing.T,
	schemaDefs []sessiontest.TestSchema,
	initConfig string,
	compFiles ...string,
) Yangd {
	ms, msFull := getModelSet(t, schemaDefs, initConfig)
	compCfg := getComponentConfigsCheckError(t, compFiles...)
	return NewYangd(ms, msFull, compCfg)
}

func escapeQuotes(input string) string {
	return strings.Replace(input, "\"", "\\\"", -1)
}

func formatRPCJson(modelName, namespace, rpcName, rpcBody string,
) string {
	return fmt.Sprintf(
		"{\"yangd-v1:rpc-module-name\":\"%s\","+
			"\"yangd-v1:rpc-namespace\":\"%s\","+
			"\"yangd-v1:rpc-name\":\"%s\","+
			"\"yangd-v1:rpc-input\":\"%s\"}",
		modelName, namespace, rpcName, escapeQuotes(rpcBody))
}

func checkValidationFails(
	t *testing.T,
	testSchemas []sessiontest.TestSchema,
	testComps []string,
	initConfig string,
	inputJson string,
	expErrors *assert.ExpectedMessages) {

	yd := newTestYangd(t, testSchemas, initConfig,
		testComps...)

	_, err := yd.ValidateRpcInput([]byte(inputJson))

	if err == nil {
		t.Fatalf("Unexpected success.")
		return
	}

	expErrors.ContainedIn(t, err.Error())
}

func checkValidationPasses(
	t *testing.T,
	testSchemas []sessiontest.TestSchema,
	testComps []string,
	initConfig string,
	inputJson string,
	expMsgs *assert.ExpectedMessages) {

	yd := newTestYangd(t, testSchemas, initConfig,
		testComps...)

	out, err := yd.ValidateRpcInput([]byte(inputJson))

	if err != nil {
		t.Fatalf("Unexpected failure: %s", err.Error())
		return
	}

	expMsgs.ContainedIn(t, string(out))
}

// RPC Input Schema For Validation Tests
//
// For most of these we only need a single module, and we have a test
// component and template schema that can be used.

const (
	validationCompModelName = "vyatta-test-validation-v1"
	invalidCompModelName    = "net.vyatta.test.invalid"
	testRpcName             = "testRpc"
	unableToValidate        = "Unable to validate RPC"
)

const validationTestComp = `[Vyatta Component]
Name=net.vyatta.test.validation
Description=Validation Test Component
ExecName=/opt/vyatta/sbin/validation-test
ConfigFile=/etc/vyatta/validation-test.conf

[Model net.vyatta.test.validation]
Modules=vyatta-test-validation-v1
ModelSets=vyatta-v1`

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

func genTestSchema(input string) []sessiontest.TestSchema {
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

// RPC TESTS
//
// Test basic RPC lookup validation.  Only covers cases not covered by the
// subsequent validation cases, so mostly error handling ones here.

func TestFindRpcInvalidModelName(t *testing.T) {

	inputJson := formatRPCJson(
		invalidCompModelName,
		noNamespaceSpecified,
		testRpcName,
		jsonNotParsed)

	expErrs := assert.NewExpectedMessages(
		"Unable to find RPC 'testRpc' for module",
		invalidCompModelName)

	checkValidationFails(t,
		nil,
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expErrs)
}

func TestFindRpcInvalidNamespace(t *testing.T) {
	inputJson := formatRPCJson(
		noModelNameSpecified,
		"urn:vyatta.com:test:vyatta-test-invalid-v1",
		testRpcName,
		jsonNotParsed)

	expErrs := assert.NewExpectedMessages(
		"Unable to find namespace",
		"urn:vyatta.com:test:vyatta-test-invalid-v1")

	checkValidationFails(t,
		nil,
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expErrs)
}

// VALIDATION TESTS

func TestValidationOfBadlyFormattedJson(t *testing.T) {

	inputJson := "invalidlyFormattedJSON"

	expErrs := assert.NewExpectedMessages(
		"Unable to parse request",
		"internal format error",
		"invalid character 'i' looking for beginning of value")

	checkValidationFails(t,
		genTestSchema(sessionCountSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expErrs)
}

func TestValidationOfBadlyFormattedJsonRpcInputField(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		testRpcName,
		"invalidJSON")

	expErrs := assert.NewExpectedMessages(
		unableToValidate,
		validationCompModelName+":"+testRpcName,
		"invalid character 'i' looking for beginning of value")

	checkValidationFails(t,
		genTestSchema(sessionCountSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expErrs)
}

func TestValidationUnknownRPC(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		"unknownRpc",
		`{"session-count": 33}"`)

	expErrs := assert.NewExpectedMessages(
		"Unable to find RPC 'unknownRpc' for module",
		validationCompModelName)

	checkValidationFails(t,
		nil,
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expErrs)
}

func TestValidationWithModelNameOfValidJsonMatchingSchema(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		testRpcName,
		`{"session-count": 33}`)

	expOut := assert.NewExpectedMessages(validTrueJson)

	checkValidationPasses(t,
		genTestSchema(sessionCountSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expOut)
}

func TestValidationWithNamespaceOfValidJsonMatchingSchema(t *testing.T) {

	inputJson := formatRPCJson(
		noModelNameSpecified,
		"urn:vyatta.com:test:vyatta-test-validation-v1",
		testRpcName,
		`{"session-count": 33}`)

	expOut := assert.NewExpectedMessages(validTrueJson)

	checkValidationPasses(t,
		genTestSchema(sessionCountSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expOut)
}

var mandatorySchemaSnippet = `
	leaf required {
		type uint16;
		mandatory "true";
	}
	leaf optional {
		type uint16;
	}`

func TestValidationMissingMandatoryNode(t *testing.T) {
	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		testRpcName,
		`{"optional":123}`)

	expErrs := []string{
		unableToValidate,
		validationCompModelName,
	}
	expErrs = append(expErrs,
		errtest.NewMissingMandatoryNodeError(t, "/required").
			RawErrorStrings()...)

	checkValidationFails(t,
		genTestSchema(mandatorySchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		assert.NewExpectedMessages(expErrs...))

}

func TestValidationWithMandatoryNode(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		testRpcName,
		`{"required":123}`)

	expMsgs := assert.NewExpectedMessages(validTrueJson)

	checkValidationPasses(t,
		genTestSchema(mandatorySchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expMsgs)
}

func TestValidationInvalidNodeValue(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		testRpcName,
		`{"session-count": 333}`)

	expErrors := []string{
		unableToValidate,
		validationCompModelName,
	}
	expErrors = append(expErrors,
		errtest.NewInvalidRangeError(t, "/session-count/333", 0, 255).
			RawErrorStrings()...)

	checkValidationFails(t,
		genTestSchema(sessionCountSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		assert.NewExpectedMessages(expErrors...))

}

func TestValidationUnexpectedNode(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		testRpcName,
		`{"unexpected-node": 33}`)

	expErrors := []string{
		unableToValidate,
		validationCompModelName,
	}
	expErrors = append(expErrors,
		errtest.NewSchemaMismatchError(t, "/unexpected-node").
			RawErrorStrings()...)

	checkValidationFails(t,
		genTestSchema(sessionCountSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		assert.NewExpectedMessages(expErrors...))
}

var mustSchemaSnippet = `
		container port-range {
			must "start <= end";
			leaf start {
				type uint16;
			}
			leaf end {
				type uint16;
			}
		}`

func TestValidationMustStatementPass(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		"testRpc",
		`{"port-range":{"start":1,"end": 555}}`)

	expMsgs := assert.NewExpectedMessages(validTrueJson)

	checkValidationPasses(t,
		genTestSchema(mustSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		expMsgs)
}

func TestValidationMustStatementFail(t *testing.T) {

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		"testRpc",
		`{"port-range":{"start":555,"end": 1}}`)

	expMsgs := []string{
		unableToValidate,
		validationCompModelName + ":" + testRpcName,
	}
	expMsgs = append(expMsgs,
		errtest.NewMustDefaultError(t,
			"/port-range",
			"start <= end").RawErrorStrings()...)

	checkValidationFails(t,
		genTestSchema(mustSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		assert.NewExpectedMessages(expMsgs...))
}

func TestValidationMustDependsOnInstantiatedDefault(t *testing.T) {

	mustDefaultSchemaSnippet := `
		leaf session-count {
			type uint32;
			default 100;
		}
		container port-range {
			must "(end - start + 1) >= ../session-count";
			leaf start {
				type uint16;
				default 8000;
			}
			leaf end {
				type uint16;
				default 8199;
			}
		}`

	inputJson := formatRPCJson(
		validationCompModelName,
		noNamespaceSpecified,
		"testRpc",
		`{"port-range":{"start":1,"end": 99}}`)

	expMsgs := []string{
		unableToValidate,
		validationCompModelName,
	}
	expMsgs = append(expMsgs,
		errtest.NewMustDefaultError(t,
			"/port-range",
			"(end - start + 1) >= ../session-count").
			RawErrorStrings()...)

	checkValidationFails(t,
		genTestSchema(mustDefaultSchemaSnippet),
		[]string{validationTestComp},
		emptyConfig,
		inputJson,
		assert.NewExpectedMessages(expMsgs...))
}
