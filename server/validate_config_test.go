// Copyright (c) 2021, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"fmt"
	"testing"

	"github.com/danos/mgmterror/errtest"

	"github.com/danos/config/auth"
	"github.com/danos/mgmterror"
)

const validateConfigTestSchema = `
container mandatory {
	presence "";
	leaf mand-leaf {
		type string;
		mandatory true;
	}

	list mand-list {
		key key;
		min-elements 1;
		leaf key {
			type string;
		}
		leaf value {
			type string;
		}
	}
}

container musts {
	presence "";

	leaf val1 {
		type string;
		must "(../val2)" {
			error-message "Must have val2";
		}
	}

	leaf val2 {
		type string;
	}
}

leaf testhidden {
	type string;
}

leaf testrange {
	type uint64 {
		range 50..100;
	}
}`

const validateConfigTestConfig = `
mandatory {
	mand-leaf amandleaf
	mand-list one {
		value numberone
	}
	mand-list two {
		value numbertwo
	}
}
musts {
	val1 firstvalue
	val2 secondvalue
}
testrange 80
testhidden invisible
`

var validateConfigCommand = []string{"load", "encoding", "xml", "validate"}

type validateConfigTests struct {
	name,
	sourceEncoding,
	sourceConfig,
	errPath,
	errType,
	errTag,
	errMsg string
	errInfoTags []*mgmterror.MgmtErrorInfoTag
	blockCommand,
	noErr bool
}

func wrapWithConfigTags(input string) string {
	return fmt.Sprintf("<config>%s</config>", input)
}

func TestValidateConfig(t *testing.T) {
	testCases := []validateConfigTests{
		{
			name:           "Invalid value for node",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				"<testrange>42</testrange>"),
			errPath: "/testrange/42",
			errType: "application",
			errTag:  "invalid-value",
			errMsg:  "Must have value between 50 and 100",
		},
		{
			name:           "Invalid value for node JSON",
			sourceEncoding: "json",
			sourceConfig:   `{"testrange": 42}`,
			errPath:        "/testrange/42",
			errType:        "application",
			errTag:         "invalid-value",
			errMsg:         "Must have value between 50 and 100",
		},
		{
			name:           "Invalid value for node RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig:   "{\"vyatta-test-validation-v1:testrange\": 42}",
			errPath:        "/testrange/42",
			errType:        "application",
			errTag:         "invalid-value",
			errMsg:         "Must have value between 50 and 100",
		},
		{
			name:           "Invalid permissions for node - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				"<testhidden>false</testhidden>"),
			errType: "application",
			errTag:  "access-denied",
			errMsg: "Access to the requested protocol operation or " +
				"data model is denied",
		},
		{
			name:           "Invalid permissions for node - JSON",
			sourceEncoding: "json",
			sourceConfig:   "{\"testhidden\":\"false\"}",
			errType:        "application",
			errTag:         "access-denied",
			errMsg: "Access to the requested protocol operation or " +
				"data model is denied",
		},
		{
			name:           "Invalid permissions for node - RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig:   "{\"vyatta-test-validation-v1:testhidden\":\"false\"}",
			errType:        "application",
			errTag:         "access-denied",
			errMsg: "Access to the requested protocol operation or " +
				"data model is denied",
		},
		{
			name:           "Missing mandatory leaf - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				`<mandatory>
				<mand-list>
				<key>foo</key>
				<value>bar</value>
				</mand-list>
				</mandatory>`),
			errPath: "/mandatory",
			errType: "application",
			errTag:  "operation-failed",
			errMsg:  "Missing mandatory node mand-leaf",
		},
		{
			name:           "Missing mandatory leaf - JSON",
			sourceEncoding: "json",
			sourceConfig:   `{"mandatory" : { "mand-list":[{ "key": "foo", "value":"bar"}]}}`,
			errPath:        "/mandatory",
			errType:        "application",
			errTag:         "operation-failed",
			errMsg:         "Missing mandatory node mand-leaf",
		},
		{
			name:           "Missing mandatory leaf - RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig:   `{"vyatta-test-validation-v1:mandatory" : { "mand-list":[{ "key": "foo", "value":"bar"}]}}`,
			errPath:        "/mandatory",
			errType:        "application",
			errTag:         "operation-failed",
			errMsg:         "Missing mandatory node mand-leaf",
		},
		{
			name:           "Missing mandatory list - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				`<mandatory><mand-leaf>foo</mand-leaf></mandatory>`),
			errPath: "/mandatory",
			errType: "application",
			errTag:  "operation-failed",
			errMsg:  "Missing mandatory node mand-list",
		},
		{
			name:           "Missing mandatory list - JSON",
			sourceEncoding: "json",
			sourceConfig:   `{"mandatory":{"mand-leaf":"foo"}}`,
			errPath:        "/mandatory",
			errType:        "application",
			errTag:         "operation-failed",
			errMsg:         "Missing mandatory node mand-list",
		},
		{
			name:           "Missing mandatory list - RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig:   `{"vyatta-test-validation-v1:mandatory":{"mand-leaf":"foo"}}`,
			errPath:        "/mandatory",
			errType:        "application",
			errTag:         "operation-failed",
			errMsg:         "Missing mandatory node mand-list",
		},
		{
			name:           "Must statement not satisfied - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				"<musts><val1>foo</val1></musts>"),
			errPath: "/musts/val1/foo",
			errType: "application",
			errTag:  "operation-failed",
			errMsg:  "Must have val2",
		},
		{
			name:           "Must statement not satisfied - JSON",
			sourceEncoding: "json",
			sourceConfig:   `{"musts": {"val1":"foo"}}`,
			errPath:        "/musts/val1/foo",
			errType:        "application",
			errTag:         "operation-failed",
			errMsg:         "Must have val2",
		},
		{
			name:           "Must statement not satisfied - RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig:   `{"vyatta-test-validation-v1:musts": {"val1":"foo"}}`,
			errPath:        "/musts/val1/foo",
			errType:        "application",
			errTag:         "operation-failed",
			errMsg:         "Must have val2",
		},
		{
			name:           "non-existent nodes - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				"<musts><val>foo</val></musts>"),
			errPath: "/musts",
			errType: "application",
			errTag:  "unknown-element",
			errMsg:  "musts [val] is not valid",
			errInfoTags: []*mgmterror.MgmtErrorInfoTag{
				mgmterror.NewMgmtErrorInfoTag("", "bad-element", "val")},
		},
		{
			name:           "non-existent nodes - JSON",
			sourceEncoding: "json",
			sourceConfig:   `{"musts":{"val": "foo"}}`,
			errPath:        "/musts",
			errType:        "application",
			errTag:         "unknown-element",
			errMsg:         "musts [val] is not valid",
			errInfoTags: []*mgmterror.MgmtErrorInfoTag{
				mgmterror.NewMgmtErrorInfoTag("", "bad-element", "val")},
		},
		{
			name:           "non-existent nodes - RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig:   `{"vyatta-test-validation-v1:musts":{"val": "foo"}}`,
			errPath:        "/musts",
			errType:        "application",
			errTag:         "unknown-element",
			errMsg:         "musts [val] is not valid",
			errInfoTags: []*mgmterror.MgmtErrorInfoTag{
				mgmterror.NewMgmtErrorInfoTag("", "bad-element", "val")},
		},
		{
			name:           "Must statement satisfied - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				"<musts><val1>foo</val1><val2>bar</val2></musts>"),
			noErr: true,
		},
		{
			name:           "Must statement satisfied - JSON",
			sourceEncoding: "json",
			sourceConfig:   `{"musts":{"val1":"foo","val2":"bar"}}`,
			noErr:          true,
		},
		{
			name:           "Must statement satisfied - RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig:   `{"vyatta-test-validation-v1:musts":{"val1":"foo","val2":"bar"}}`,
			noErr:          true,
		},
		{
			name:           "Missing list key - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				"<musts><val1>foo</val1><val2>bar</val2></musts>" +
					"<mandatory><mand-leaf>foo</mand-leaf>" +
					"<mand-list><value>foobar</value></mand-list>" +
					"<mand-list><key>bar</key><value>barbaz</value></mand-list>" +
					"</mandatory>"),
			errPath: "/key",
			errType: "application",
			errTag:  "operation-failed",
			errMsg:  "List entry is missing key",
		},
		{
			name:           "Valid full config - XML",
			sourceEncoding: "xml",
			sourceConfig: wrapWithConfigTags(
				"<musts><val1>foo</val1><val2>bar</val2></musts>" +
					"<mandatory><mand-leaf>foo</mand-leaf>" +
					"<mand-list><key>foo</key><value>foobar</value></mand-list>" +
					"<mand-list><key>bar</key><value>barbaz</value></mand-list>" +
					"</mandatory>"),
			noErr: true,
		},
		{
			name:           "Valid full config - JSON",
			sourceEncoding: "json",
			sourceConfig: `{"musts":{"val1":"foo","val2":"bar"},
				"mandatory":{"mand-leaf":"foo",
				"mand-list":[{"key":"foo","value":"foobar"}],
				"mand-list":[{"key":"bar","value":"barbaz"}]}}`,
			noErr: true,
		},
		{
			name:           "Valid full config - RFC7951",
			sourceEncoding: "rfc7951",
			sourceConfig: `{"vyatta-test-validation-v1:musts":{"val1":"foo","val2":"bar"},
				"vyatta-test-validation-v1:mandatory":{"mand-leaf":"foo",
				"mand-list":[{"key":"foo","value":"foobar"}],
				"mand-list":[{"key":"bar","value":"barbaz"}]}}`,
			noErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			limitedAuth := auth.NewTestAuther(
				auth.NewTestRule(auth.Deny, auth.AllOps, "/testhidden"),
				auth.NewTestRule(auth.Allow, auth.AllOps, "*"))
			if test.blockCommand {
				limitedAuth.AddBlockedCommand(validateConfigCommand)
			}

			oc := newOutputChecker(t).
				setSchema(validateConfigTestSchema).
				setInitConfig(validateConfigTestConfig).
				setAuther(limitedAuth, !ConfigdUser, InSecretsGroup)

			oc.validateConfig(test.sourceEncoding, test.sourceConfig)

			if test.noErr {
				if oc.actErr != nil {
					t.Fatalf("Seen unexpected error: %s\n", oc.actErr)
				}
			} else {
				expErr := errtest.NewExpMgmtError(
					[]string{test.errMsg},
					test.errPath,
					test.errInfoTags).
					SetType(test.errType).
					SetTag(test.errTag)

				oc.verifyMgmtError(expErr)
			}
			clearAllCmdRequestsAndUserAuditLogs(limitedAuth)
		})
	}
}
