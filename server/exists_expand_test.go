// Copyright (c) 2017-2019, AT&T Intellectual Property Inc. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/server"
	"github.com/danos/mgmterror"
	"github.com/danos/mgmterror/errtest"
)

const (
	ConfigdUser    = true
	InSecretsGroup = true
)

const existExpandConfig = `
	testContainer {
		testLeaf foo
	}`

func TestExistsPass(t *testing.T) {

	testPath := "testContainer/testLeaf"

	oc := newOutputChecker(t).
		setSchema(defaultSchema).
		setInitConfig(existExpandConfig)

	oc = oc.exists(testPath)

	oc.verifyStatusOkNoError()
}

func TestValueExistsPass(t *testing.T) {
	testPath := "testContainer/testLeaf/foo"

	oc := newOutputChecker(t).
		setSchema(defaultSchema).
		setInitConfig(existExpandConfig)

	oc = oc.exists(testPath)

	oc.verifyStatusOkNoError()
}

func TestExistsFailNoAuth(t *testing.T) {

	testPath := "testContainer/testLeaf"

	oc := newOutputChecker(t).
		setSchema(defaultSchema).
		setInitConfig(existExpandConfig).
		setAuther(auth.TestAutherDenyAll(), ConfigdUser, InSecretsGroup)

	oc = oc.exists(testPath)

	oc.setExpErr(errtest.NewAccessDeniedError(t, testPath))
	oc.verifyStatusFail().verifyRawError()
}

func TestExistsFailInvalidPathLeaf(t *testing.T) {

	testPath := "testContainer/testInvalid"

	oc := newOutputChecker(t).
		setSchema(defaultSchema).
		setInitConfig(existExpandConfig)

	oc = oc.exists(testPath)

	expErrs := errtest.NewExpectedFormattedErrors(t).
		AddNode("testContainer",
			errtest.NewErrDesc(
				errtest.InvalidPath, "testContainer testInvalid"))
	oc.verifyStatusFail().verifyMgmtErrors(expErrs)
}

func TestExistsFailInvalidPathContainer(t *testing.T) {

	testPath := "testInvalid/testLeaf"

	oc := newOutputChecker(t).
		setSchema(defaultSchema).
		setInitConfig(existExpandConfig)

	oc = oc.exists(testPath)

	expErrs := errtest.NewExpectedFormattedErrors(t).
		AddNode("testInvalid",
			errtest.NewErrDesc(
				errtest.InvalidPath, "testInvalid"))
	oc.verifyStatusFail().verifyMgmtErrors(expErrs)
}

type expandCompletionTest struct {
	name,
	path,
	prefix string
	pos       int
	expOut    string
	expErr    *errtest.TestError
	extraErrs []string
}

func TestExpandCompletionSuccess(t *testing.T) {

	tests := []expandCompletionTest{
		// Expand()
		{
			name:   "Leaf completion",
			path:   "tes/testLe",
			prefix: server.NoPrefix,
			pos:    server.InvalidPos,
			expOut: "/testContainer/testLeaf",
		},
		{
			name:   "List completion",
			path:   "tes/testLi/foo/fie/foo",
			prefix: server.NoPrefix,
			pos:    server.InvalidPos,
			expOut: "/testContainer/testList/foo/field/foo",
		},
		// ExpandWithPrefix()
		{
			name:   "Mid-word completion, valid prefix, 1st word",
			path:   "tesXXX/testLi/foo",
			prefix: "tes",
			pos:    0,
			expOut: "/testContainerXXX/testList/foo",
		},
		{
			name:   "Mid-word completion, valid prefix, 2nd word",
			path:   "test/testLiXXX/foo",
			prefix: "testLi",
			pos:    1,
			expOut: "/testContainer/testListXXX/foo",
		},
		{
			name:   "End-word completion, valid prefix, 2nd word",
			path:   "test/testLi/foo",
			prefix: "testLi",
			pos:    1,
			expOut: "/testContainer/testList/foo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oc := newOutputChecker(t).
				setSchema(defaultSchema).
				setInitConfig(existExpandConfig)

			oc = oc.expandWithPrefix(test.path, test.prefix, test.pos)

			oc = oc.verifyOutputOkNoError(test.expOut)
		})
	}
}

func TestExpandCompletionFailure(t *testing.T) {

	tests := []expandCompletionTest{
		{
			name:   "Mid-2nd-word, invalid prefix",
			path:   "test/testLiXXX/foo",
			prefix: "testLix",
			pos:    1,
			expErr: errtest.NewInvalidNodeError(t, "testContainer/testLiXXX"),
		},
		{
			name:   "Mid-3rd-word, invalid preceding word",
			path:   "test/testLiXXX/foo",
			prefix: "fo",
			pos:    2,
			expErr: errtest.NewInvalidNodeError(t, "testContainer/testLiXXX"),
		},
		{
			name:   "Mid-2nd-word ambiguous prefix",
			path:   "test/testLiXXX/foo",
			prefix: "test",
			pos:    1,
			expErr: errtest.NewPathAmbiguousError(t, "testContainer/testLiXXX"),
			extraErrs: []string{
				"testBadList", "testBadScript", "testLeaf", "testList"},
		},
		{
			name:   "Invalid path past empty leaf",
			path:   "testContainer/emptyLeaf/foo",
			prefix: server.NoPrefix,
			pos:    server.InvalidPos,
			expErr: errtest.NewInvalidNodeError(
				t, "testContainer/emptyLeaf/foo"),
		},
		{
			name:   "Invalid path",
			path:   "testContainer/testInvalid",
			prefix: server.NoPrefix,
			pos:    server.InvalidPos,
			expErr: errtest.NewInvalidNodeError(t, "testContainer/testInvalid"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oc := newOutputChecker(t).
				setSchema(defaultSchema).
				setInitConfig(existExpandConfig)

			oc = oc.expandWithPrefix(test.path, test.prefix, test.pos)

			oc = oc.setExpErr(test.expErr).
				addExtraErrs(test.extraErrs...).
				setUnexpErrs(setFailedStr, validationFailedStr,
					doubleIsntValidStr)

			oc.verifyCLIError()
		})
	}
}

func TestExpandFailNoAuth(t *testing.T) {

	t.Skip("Expansion is not policed")

	testPath := "tes/testL"
	expect := assert.NewExpectedError(
		mgmterror.NewAccessDeniedApplicationError().Error())

	d := newTestDispatcher(
		t, auth.TestAutherDenyAll(), defaultSchema, existExpandConfig)
	_, actual := d.Expand(testPath)

	expect.Matches(t, actual)
}
