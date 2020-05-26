// Copyright (c) 2017-2020 AT&T Intellectual Property
// All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// These tests verify commit and validation behaviour (success and failure).
// They verify error content by individual fields; pretty-printed CLI format
// is not tested here because that is done on the receiving side of the configd
// socket, not the sending side (which is where these tests sit).

package server_test

import (
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

const (
	noPath = ""
)

// Commit and Validation warning / error message testing.
//
// Validation pass
// Validation fail
// Commit pass
// Commit fail (nothing to commit)
// Commit fail (validation)
// Commit non-fatal error
//
const commitSchema = `
	container testContainer {
	leaf testLeaf {
		type string;
	}
	leaf generateWarning {
		type string;
	    configd:end false;
	}
	leaf generateValidateFail {
		type string;
		must "../nonExistentLeaf";
	}
}`

// Successful commit should return a message, no error, and config
// should be changed.
func TestCommitSuccess(t *testing.T) {
	testPath := "testContainer/testLeaf/foo"

	oc := newOutputChecker(t).
		setSchema(commitSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set(testPath)
	oc.verifyOutputOkNoError("")

	oc.commit()
	oc.verifyNoError()
}

// 'Nothing to commit' should be reported as an error.
func TestNothingToCommitError(t *testing.T) {

	oc := newOutputChecker(t).
		setSchema(commitSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	errList := []*errtest.ExpMgmtError{
		errtest.NewExpMgmtError(
			[]string{"No configuration changes to commit"},
			noPath,
			noInfo).
			SetType("protocol").
			SetTag("operation-failed"),
	}

	oc.commit()
	oc.verifyMgmtErrorList(errList)
}

// Matching previous releases, validation error when running commit is
// reported as a commit error.
func TestCommitValidationError(t *testing.T) {

	testPath := "testContainer/generateValidateFail/foo"

	oc := newOutputChecker(t).
		setSchema(commitSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set(testPath)
	oc.verifyOutputOkNoError("")

	errList := []*errtest.ExpMgmtError{
		errtest.MustViolationMgmtErr(
			"'must' condition is false: '../nonExistentLeaf'",
			"/"+testPath),
	}

	oc.commit()
	oc.verifyMgmtErrorList(errList)
}

// Other than the 'nothing to commit' error, commits should not generate an
// error.  Instead they generate a warning, and the commit proceeds.
func TestCommitWarning(t *testing.T) {
	testPath := "testContainer/generateWarning/foo"

	oc := newOutputChecker(t).
		setSchema(commitSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set(testPath)
	oc.verifyOutputOkNoError("")

	oc.commit()
	oc.verifyOutputContentNoError(
		errtest.NonFatalCommitErrorStrings(t, testPath))
}

// Returns no error.
func TestValidateSuccess(t *testing.T) {
	oc := newOutputChecker(t).
		setSchema(commitSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set("testContainer/testLeaf/foo")
	oc.verifyOutputOkNoError("")

	oc.validate()
	oc.verifyNoError()
}

// 'Nothing to validate' should NOT be reported as an error.
func TestNothingToValidate(t *testing.T) {
	oc := newOutputChecker(t).
		setSchema(commitSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.validate()
	oc.verifyNoError()
}

// Returns error "Validate failed!"
func TestValidateError(t *testing.T) {
	testPath := "testContainer/generateValidateFail/foo"

	oc := newOutputChecker(t).
		setSchema(commitSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set(testPath)
	oc.verifyOutputOkNoError("")

	errList := []*errtest.ExpMgmtError{
		errtest.MustViolationMgmtErr(
			"'must' condition is false: '../nonExistentLeaf'",
			"/testContainer/generateValidateFail/foo"),
	}

	oc.validate()
	oc.verifyMgmtErrorList(errList)
}

const singleErrorSchema = `
container currentCont {
	presence "must and when testing";
	must "local-name(.) = 'currentCont'";
	must "not(contains(., 'foo'))";
	leaf aLeaf {
		type string;
	}
}`

const errFmtConfig = "currentCont"

func TestCommitFailSingleErrorFormat(t *testing.T) {
	testPath := "currentCont/aLeaf/foo"

	oc := newOutputChecker(t).
		setSchema(singleErrorSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set(testPath)
	oc.verifyOutputOkNoError("")

	errList := []*errtest.ExpMgmtError{
		errtest.MustViolationMgmtErr(
			"'must' condition is false: 'not(contains(., 'foo'))'",
			"/currentCont"),
	}

	oc.commit()
	oc.verifyMgmtErrorList(errList)
}

const multipleErrorSchema = `
container currentCont {
	presence "must and when testing";
	must "local-name(.) = 'oldCont'";
	must "not(contains(., 'foo'))" {
        error-message "aLeaf's value may not contain 'foo' anywhere.";
    }
	leaf aLeaf {
		type string;
	}
	leaf aLeafRef {
		type leafref {
			path "../aLeaf";
		}
	}
    leaf aLeafList {
        type uint16;
        mandatory "true";
    }
    list aList {
        key name;
        unique otherLeaf;
        leaf name {
            type string;
        }
        leaf otherLeaf {
            type string;
        }
    }
}`

func TestCommitFailMultipleErrorFormat(t *testing.T) {

	oc := newOutputChecker(t).
		setSchema(multipleErrorSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set("currentCont/aLeaf/foo")
	oc.set("currentCont/aLeafRef/bar")
	oc.set("currentCont/aList/first/otherLeaf/something")
	oc.set("currentCont/aList/second/otherLeaf/something")
	oc.verifyOutputOkNoError("")

	errList := []*errtest.ExpMgmtError{
		errtest.MissingMandatoryNodeMgmtErr(
			"aLeafList",
			"/currentCont"),
		errtest.MustViolationMgmtErr(
			"'must' condition is false: 'local-name(.) = 'oldCont''",
			"/currentCont"),
		errtest.MustViolationMgmtErr(
			"aLeaf's value may not contain 'foo' anywhere.",
			"/currentCont"),
		errtest.LeafrefMgmtErr(
			"currentCont aLeaf bar",
			"/currentCont/aLeafRef/bar"),
		errtest.UniqueViolationMgmtErr(
			"otherLeaf something",
			"first second",
			"/currentCont/aList"),
	}

	oc.commit()
	oc.verifyMgmtErrorList(errList)
}

const leafrefErrorSchema = `
container currentCont {
	presence "must and when testing";
	leaf aLeaf {
		type string;
	}
    leaf absLeafref {
        type leafref {
            path "/currentCont/aLeaf";
        }
    }
    leaf relLeafref {
        type leafref {
            path "../aLeaf";
        }
    }
    list aList {
        key name;
        leaf name {
            type string;
        }
        leaf otherLeaf {
            type string;
        }
    }
    leaf keyLeafref {
        type leafref {
            path "../aList[name = current()/../aLeaf]/otherLeaf";
        }
    }
}`

func TestCommitFailLeafrefErrorFormat(t *testing.T) {

	oc := newOutputChecker(t).
		setSchema(leafrefErrorSchema).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set("currentCont/aLeaf/foo")
	oc.set("currentCont/absLeafref/bar")
	oc.set("currentCont/relLeafref/bar2")
	oc.set("currentCont/keyLeafref/bar3")
	oc.verifyOutputOkNoError("")

	errList := []*errtest.ExpMgmtError{
		errtest.LeafrefMgmtErr(
			"currentCont aLeaf bar",
			"/currentCont/absLeafref/bar"),
		errtest.LeafrefMgmtErr(
			"currentCont aList otherLeaf bar3",
			"/currentCont/keyLeafref/bar3"),
		errtest.LeafrefMgmtErr(
			"currentCont aLeaf bar2",
			"/currentCont/relLeafref/bar2"),
	}

	oc.commit()
	oc.verifyMgmtErrorList(errList)
}

const localSchema = `
container localCont {
    leaf localLeaf {
        type leafref {
            path "/remote:remoteCont/remote:remoteLeaf";
        }
    }
}`

const remoteSchema = `
container remoteCont {
    leaf remoteLeaf {
        type string;
    }
}`

var prefixSchemas = []sessiontest.TestSchema{
	{
		Name: sessiontest.NameDef{
			Namespace: "prefix-local",
			Prefix:    "local",
		},
		Imports: []sessiontest.NameDef{
			{"prefix-remote", "remote"}},
		SchemaSnippet: localSchema,
	},
	{
		Name: sessiontest.NameDef{
			Namespace: "prefix-remote",
			Prefix:    "remote",
		},
		SchemaSnippet: remoteSchema,
	},
}

func TestCommitFailPrefixedLeafref(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefs(prefixSchemas).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set("localCont/localLeaf/nonexistentValue")
	oc.verifyOutputOkNoError("")

	errList := []*errtest.ExpMgmtError{
		errtest.LeafrefMgmtErr(
			"remoteCont remoteLeaf nonexistentValue",
			"/localCont/localLeaf/nonexistentValue"),
	}

	oc.commit()
	oc.verifyMgmtErrorList(errList)
}

func TestValidateFailPrefixedLeafref(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefs(prefixSchemas).
		setInitConfig(emptyConfig).
		setAuther(auth.TestAutherAllowAll(), ConfigdUser, InSecretsGroup)

	oc.set("localCont/localLeaf/nonexistentValue")
	oc.verifyOutputOkNoError("")

	errList := []*errtest.ExpMgmtError{
		errtest.LeafrefMgmtErr(
			"remoteCont remoteLeaf nonexistentValue",
			"/localCont/localLeaf/nonexistentValue"),
	}

	oc.validate()
	oc.verifyMgmtErrorList(errList)
}
