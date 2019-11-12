// Copyright (c) 2017-2019 AT&T Intellectual Property
// All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// These tests verify commit and validation behaviour (success and failure).

package server_test

import (
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
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
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), commitSchema, emptyConfig)
	d.Set("RUNNING", "testContainer/testLeaf/foo")
	_, err := d.Commit("RUNNING", "Test Commit", false /* debug */)
	if err != nil {
		t.Fatalf("TestCommit: err '%s'\n", err.Error())
	}
}

// 'Nothing to commit' should be reported as an error.
func TestNothingToCommitError(t *testing.T) {
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), commitSchema, emptyConfig)
	_, actual := d.Commit("RUNNING", "Test Commit", false /* debug */)

	expErrs := assert.NewExpectedMessages(
		"No configuration changes to commit",
		"Commit failed!")

	expErrs.ContainedIn(t, actual.Error())
}

// Matching previous releases, validation error when running commit is
// reported as a commit error.
func TestCommitValidationError(t *testing.T) {
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), commitSchema, emptyConfig)
	nodePath := "testContainer/generateValidateFail/foo"
	_, err := d.Set("RUNNING", nodePath)
	if err != nil {
		t.Fatalf("Unable to set config for validation error: %s\n",
			err.Error())
		return
	}

	_, actual := d.Commit("RUNNING", "Test Commit", false /* debug */)

	expErrs := assert.NewExpectedMessages(
		errtest.NewMustDefaultError(t,
			nodePath,
			"../nonExistentLeaf").CommitCliErrorStrings()...)

	expErrs.ContainedIn(t, actual.Error())
}

// Other than the 'nothing to commit' error, commits should not generate an
// error.  Instead they generate a warning, and the commit proceeds.
func TestCommitWarning(t *testing.T) {
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), commitSchema, emptyConfig)
	nodePath := "testContainer/generateWarning/foo"
	d.Set("RUNNING", nodePath)
	msg, err := d.Commit("RUNNING", "Test Commit", false /* debug */)

	if err != nil {
		t.Fatalf("Commit shouldn't have failed; err '%s'\n", err.Error())
	} else {
		expMsgs := assert.NewExpectedMessages(
			errtest.NonFatalCommitErrorStrings(t, nodePath)...)
		expMsgs.ContainedIn(t, msg)
	}
}

// Returns no error.
func TestValidateSuccess(t *testing.T) {
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), commitSchema, emptyConfig)
	d.Set("RUNNING", "testContainer/testLeaf/foo")
	msg, err := d.Validate("RUNNING")
	if err != nil {
		t.Fatalf("TestValidate: msg '%s', err '%s'\n", msg, err.Error())
	}
}

// 'Nothing to validate' should NOT be reported as an error.
func TestNothingToValidate(t *testing.T) {
	d := newTestDispatcher(
		t, auth.TestAutherAllowAll(), commitSchema, emptyConfig)

	dispTestValidate(t, d, "RUNNING")
}

// Returns error "Validate failed!"
func TestValidateError(t *testing.T) {
	d := newTestDispatcher(
		t, auth.TestAutherAllowAll(), commitSchema, emptyConfig)
	_, err := d.Set("RUNNING", "testContainer/generateValidateFail/foo")
	if err != nil {
		t.Fatalf("Unable to set config for validation error: %s\n",
			err.Error())
		return
	}

	_, actual := d.Validate("RUNNING")

	expErrs := assert.NewExpectedMessages(
		"testContainer generateValidateFail foo",
		"'must' condition is false: '../nonExistentLeaf'")

	expErrs.ContainedIn(t, actual.Error())
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
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), singleErrorSchema, "")
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}
	if _, err := d.Set(testSID, "currentCont/aLeaf/foo"); err != nil {
		t.Fatalf("Unable to configure session: %s\n", err.Error())
		return
	}

	_, err := d.Commit(testSID, "message", false /* debug */)
	if err == nil {
		t.Fatalf("Expected error running tests.\n")
		return
	}

	expect := errtest.NewExpectedFormattedErrors(t).
		AddNode("currentCont",
			errtest.NewErrDesc(errtest.DefaultMustError,
				"not(contains(., 'foo'))")).
		AddEndMessage(errtest.TestCommitFailStr)
	expect.Matches(err)
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
	d := newTestDispatcher(
		t, auth.TestAutherAllowAll(), multipleErrorSchema, "")
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}
	dispTestSet(t, d, testSID, "currentCont/aLeaf/foo")
	dispTestSet(t, d, testSID, "currentCont/aLeafRef/bar")
	dispTestSet(t, d, testSID,
		"currentCont/aList/first/otherLeaf/something")
	dispTestSet(t, d, testSID,
		"currentCont/aList/second/otherLeaf/something")

	_, err := d.Commit(testSID, "message", false /* debug */)
	if err == nil {
		t.Fatalf("Expected error running tests.\n")
		return
	}

	currentContPath := "currentCont"
	expect := errtest.NewExpectedFormattedErrors(t).
		AddNode(currentContPath,
			errtest.NewErrDesc(errtest.MissingMandatory, "aLeafList")).
		AddNode(currentContPath,
			errtest.NewErrDesc(errtest.DefaultMustError,
				"local-name(.) = 'oldCont'")).
		AddNode(currentContPath,
			errtest.NewErrDesc(errtest.CustomMustError,
				"aLeaf's value may not contain 'foo' anywhere.")).
		AddNode("currentCont aLeafRef bar",
			errtest.NewErrDesc(errtest.LeafrefMissing,
				"currentCont aLeaf bar")).
		AddNode("currentCont aList",
			errtest.NewErrDesc(errtest.NotUnique,
				"otherLeaf something", "first second")).
		AddEndMessage(errtest.TestCommitFailStr)

	expect.Matches(err)
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
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), leafrefErrorSchema, "")
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}
	fooPath := "currentCont/aLeaf/foo"
	dispTestSet(t, d, testSID, fooPath)
	barPath := "currentCont/absLeafref/bar"
	dispTestSet(t, d, testSID, barPath)
	bar2Path := "currentCont/relLeafref/bar2"
	dispTestSet(t, d, testSID, bar2Path)
	bar3Path := "currentCont/keyLeafref/bar3"
	dispTestSet(t, d, testSID, bar3Path)

	_, err := d.Commit(testSID, "message", false /* debug */)
	if err == nil {
		t.Fatalf("Expected error running tests.\n")
		return
	}

	expect := errtest.NewExpectedFormattedErrors(t).
		AddNode(barPath,
			errtest.NewErrDesc(errtest.LeafrefMissing,
				"currentCont aLeaf bar")).
		AddNode(bar3Path,
			errtest.NewErrDesc(errtest.LeafrefMissing,
				"currentCont aList otherLeaf bar3")).
		AddNode(bar2Path,
			errtest.NewErrDesc(errtest.LeafrefMissing,
				"currentCont aLeaf bar2")).
		AddEndMessage(errtest.TestCommitFailStr)

	expect.Matches(err)
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
	d := newTestDispatcherWithMultipleSchemas(
		t, auth.TestAutherAllowAll(), prefixSchemas, "")
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}
	dispTestSet(t, d, testSID, "localCont/localLeaf/nonexistentValue")
	_, err := d.Commit(testSID, "message", false /* debug */)
	if err == nil {
		t.Fatalf("Expected error running tests.\n")
		return
	}

	errtest.NewExpectedFormattedErrors(t).
		AddNode("localCont localLeaf nonexistentValue",
			errtest.NewErrDesc(errtest.LeafrefMissing,
				"remoteCont remoteLeaf nonexistentValue")).
		AddEndMessage(errtest.TestCommitFailStr).
		Matches(err)
}

func TestValidateFailPrefixedLeafref(t *testing.T) {
	d := newTestDispatcherWithMultipleSchemas(
		t, auth.TestAutherAllowAll(), prefixSchemas, "")
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}
	dispTestSet(t, d, testSID, "localCont/localLeaf/nonexistentValue")
	_, err := d.Validate(testSID)
	if err == nil {
		t.Fatalf("Expected error running tests.\n")
		return
	}

	errtest.NewExpectedFormattedErrors(t).
		AddNode("localCont localLeaf nonexistentValue",
			errtest.NewErrDesc(errtest.LeafrefMissing,
				"remoteCont remoteLeaf nonexistentValue")).
		AddEndMessage(errtest.TestValidateFailStr).
		Matches(err)
}
