// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
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

type copyConfigErrTest struct {
	name,
	sourceDatastore,
	sourceConfig,
	sourceURL,
	targetDatastore,
	targetURL,
	errPath,
	errType,
	errTag,
	errMsg string
	errInfoTags []*mgmterror.MgmtErrorInfoTag
}

func wrapInConfigTags(input string) string {
	return fmt.Sprintf("<config>%s</config>", input)
}

func TestCopyConfigErrorHandling(t *testing.T) {
	testCases := []copyConfigErrTest{
		{
			name:      "Source URL provided",
			sourceURL: "sourceURL",
			errType:   "application",
			errTag:    "operation-not-supported",
			errMsg:    "URL capability is not supported",
		},
		{
			name:      "Target URL provided",
			targetURL: "targetURL",
			errType:   "application",
			errTag:    "operation-not-supported",
			errMsg:    "URL capability is not supported",
		},
		{
			name:            "Source datastore provided",
			sourceDatastore: "candidate",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Source must be specified in <config> tags.",
		},
		{
			name:    "Source config not provided",
			errType: "application",
			errTag:  "missing-element",
			errMsg:  "An expected element is missing",
			errInfoTags: []*mgmterror.MgmtErrorInfoTag{
				mgmterror.NewMgmtErrorInfoTag("", "bad-element", "<source>")},
		},
		{
			name:            "Target datastore not candidate",
			sourceConfig:    "not empty",
			targetDatastore: "running",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Target datastore only supports candidate, not running",
		},
		{
			name: "Invalid value for node",
			sourceConfig: wrapInConfigTags(
				"<testavailable>neither-true-nor-false</testavailable>"),
			targetDatastore: "candidate",
			errPath:         "/testavailable/neither-true-nor-false",
			errType:         "application",
			errTag:          "invalid-value",
			errMsg:          "Must have one of the following values: true, false",
		},
		{
			name: "Invalid permissions for node",
			sourceConfig: wrapInConfigTags(
				"<testhidden>false</testhidden>"),
			targetDatastore: "candidate",
			errType:         "application",
			errTag:          "operation-failed",
			errMsg: "Access to the requested protocol operation or " +
				"data model is denied",
		},
	}

	limitedAuth := auth.NewTestAuther(
		auth.NewTestRule(auth.Deny, auth.AllOps, "/testhidden"),
		auth.NewTestRule(auth.Allow, auth.AllOps, "*"))

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			oc := newOutputChecker(t).
				setSchema(copyConfigTestSchema).
				setInitConfig(copyConfigTestConfig).
				setAuther(limitedAuth, !ConfigdUser, InSecretsGroup)

			oc.copyConfig(
				test.sourceDatastore,
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
		})
	}
}
