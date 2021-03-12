// Copyright (c) 2020-2021, AT&T Intellectual Property. All rights reserved.
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

const copyConfigTestSchema = `
leaf testhidden {
	type boolean;
}
leaf testavailable {
	type boolean;
}`

const copyConfigTestConfig = `
testhidden true
testavailable true
`

const (
	noDatastore = ""
	noConfig    = ""
	noURL       = ""
)

var copyConfigCommand = []string{"load", "encoding", "xml", "copy-config"}
var copyConfigCommandJSON = []string{"load", "encoding", "json", "copy-config"}
var copyConfigCommandRFC7951 = []string{"load", "encoding", "rfc7951", "copy-config"}
var copyConfigCommandNilEnc = []string{"load", "copy-config"}

type copyConfigErrTest struct {
	name,
	sourceDatastore,
	sourceEncoding,
	sourceConfig,
	sourceURL,
	targetDatastore,
	targetURL,
	errPath,
	errType,
	errTag,
	errMsg string
	errInfoTags  []*mgmterror.MgmtErrorInfoTag
	blockCommand bool
	authCmd      []string
}

func wrapInConfigTags(input string) string {
	return fmt.Sprintf("<config>%s</config>", input)
}

func TestCopyConfigErrorHandling(t *testing.T) {
	testCases := []copyConfigErrTest{
		{
			name:           "Source URL provided",
			sourceEncoding: "xml",
			authCmd:        copyConfigCommand,
			sourceURL:      "sourceURL",
			errType:        "application",
			errTag:         "operation-not-supported",
			errMsg:         "URL capability is not supported",
		},
		{
			name:           "Target URL provided",
			sourceEncoding: "xml",
			authCmd:        copyConfigCommand,
			targetURL:      "targetURL",
			errType:        "application",
			errTag:         "operation-not-supported",
			errMsg:         "URL capability is not supported",
		},
		{
			name:            "Source datastore provided",
			sourceEncoding:  "xml",
			authCmd:         copyConfigCommand,
			sourceDatastore: "candidate",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Source must be specified in <config> tags.",
		},
		{
			name:           "Source config not provided",
			sourceEncoding: "xml",
			authCmd:        copyConfigCommand,
			errType:        "application",
			errTag:         "missing-element",
			errMsg:         "An expected element is missing",
			errInfoTags: []*mgmterror.MgmtErrorInfoTag{
				mgmterror.NewMgmtErrorInfoTag("", "bad-element", "<source>")},
		},
		{
			name:            "Target datastore not candidate",
			authCmd:         copyConfigCommand,
			sourceEncoding:  "xml",
			sourceConfig:    "not empty",
			targetDatastore: "running",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Target datastore only supports candidate, not running",
		},
		{
			name:    "Invalid value for node - Unknown encoding",
			authCmd: copyConfigCommandNilEnc,
			sourceConfig: wrapInConfigTags(
				"<testavailable>neither-true-nor-false</testavailable>"),
			targetDatastore: "candidate",
			errType:         "application",
			errTag:          "operation-failed",
			errMsg:          "Unknown encoding",
		},
		{
			name:           "Invalid value for node",
			authCmd:        copyConfigCommand,
			sourceEncoding: "xml",
			sourceConfig: wrapInConfigTags(
				"<testavailable>neither-true-nor-false</testavailable>"),
			targetDatastore: "candidate",
			errPath:         "/testavailable/neither-true-nor-false",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Must have one of the following values: true, false",
		},
		{
			name:            "Invalid value for node - JSON",
			authCmd:         copyConfigCommandJSON,
			sourceEncoding:  "json",
			sourceConfig:    "{\"testavailable\":\"neither-true-nor-false\"}",
			targetDatastore: "candidate",
			errPath:         "/testavailable/neither-true-nor-false",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Must have one of the following values: true, false",
		},
		{
			name:            "Invalid value for node - RFC7951",
			authCmd:         copyConfigCommandRFC7951,
			sourceEncoding:  "rfc7951",
			sourceConfig:    "{\"vyatta-test-validation-v1:testavailable\":\"neither-true-nor-false\"}",
			targetDatastore: "candidate",
			errPath:         "/testavailable/neither-true-nor-false",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Must have one of the following values: true, false",
		},
		{
			name:           "Invalid permissions for user",
			authCmd:        copyConfigCommand,
			sourceEncoding: "xml",
			sourceConfig: wrapInConfigTags(
				"<testavailable>false</testavailable>"),
			targetDatastore: "candidate",
			errType:         "application",
			errTag:          "access-denied",
			errMsg: "Access to the requested protocol operation or " +
				"data model is denied",
			blockCommand: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			limitedAuth := auth.NewTestAuther(
				auth.NewTestRule(auth.Deny, auth.AllOps, "/testhidden"),
				auth.NewTestRule(auth.Allow, auth.AllOps, "*"))
			if test.blockCommand {
				limitedAuth.AddBlockedCommand(test.authCmd)
			}

			oc := newOutputChecker(t).
				setSchema(copyConfigTestSchema).
				setInitConfig(copyConfigTestConfig).
				setAuther(limitedAuth, !ConfigdUser, InSecretsGroup)

			oc.copyConfig(
				test.sourceDatastore,
				test.sourceEncoding,
				test.sourceConfig,
				test.sourceURL,
				test.targetDatastore,
				test.targetURL)

			expErr := errtest.NewExpMgmtError(
				[]string{test.errMsg},
				test.errPath,
				test.errInfoTags).
				SetType(test.errType).
				SetTag(test.errTag)

			oc.verifyMgmtError(expErr)

			// 'secrets' here relates to arguments being secret, not that we
			// are in the secrets group.
			if !test.blockCommand {
				assertCommandAaaNoSecrets(
					t, limitedAuth, test.authCmd)
			}
			clearAllCmdRequestsAndUserAuditLogs(limitedAuth)
		})
	}
}
