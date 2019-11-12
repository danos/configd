// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
)

// Containers
const testremote = "test-remote"
const testlocal = "test-local"
const embeddedcontainer = "embedded-container"

//Container Paths
var testremotepath = []string{testremote}
var testlocalpath = []string{testlocal}
var embeddedcontainerpath = pathutil.CopyAppend(testlocalpath, embeddedcontainer)

// Leafs
const testremoteleaf = "test-remote-leaf"
const auglocalimpleaf = "aug-local-imp-leaf"
const auglocalexpleaf = "aug-local-exp-leaf"
const augremoteleaf = "aug-remote-leaf"
const augembedimpleaf = "aug-embed-imp-leaf"
const augembedexpleaf = "aug-embed-exp-leaf"
const augembedmixedoneleaf = "aug-embed-mixedone-leaf"
const augembedmixedtwoleaf = "aug-embed-mixedtwo-leaf"

// Leaf Paths
var testremoteleafpath = pathutil.CopyAppend(testremotepath, testremoteleaf)
var auglocalimpleafpath = pathutil.CopyAppend(testlocalpath, auglocalimpleaf)
var auglocalexpleafpath = pathutil.CopyAppend(testlocalpath, auglocalexpleaf)
var augremoteleafpath = pathutil.CopyAppend(testremotepath, augremoteleaf)
var augembedimpleafpath = pathutil.CopyAppend(embeddedcontainerpath, augembedimpleaf)
var augembedexpleafpath = pathutil.CopyAppend(embeddedcontainerpath, augembedexpleaf)
var augembedmixedoneleafpath = pathutil.CopyAppend(embeddedcontainerpath, augembedmixedoneleaf)
var augembedmixedtwoleafpath = pathutil.CopyAppend(embeddedcontainerpath, augembedmixedtwoleaf)

// Test that an augment of a local node with an implicit prefix works
func TestAugmentLocalImplicit(t *testing.T) {
	tblFeature := []ValidateOpTbl{
		{"", auglocalimpleafpath, "TestData", true},
		{"", augembedimpleafpath, "TestData", true},
		{"", augembedmixedoneleafpath, "TestData", true},
		{"", augembedmixedtwoleafpath, "TestData", true},
	}

	srv, sess := TstStartupSchemaDir(t, "testdata/augmentValid", "", "")
	ValidateOperationTable(t, sess, srv.Ctx, tblFeature,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblFeature,
		DELETE_AND_COMMIT)
	sess.Kill()
}

// Test that an augment of a local node with an explicit prefix works
func TestAugmentLocalExplicit(t *testing.T) {
	tblFeature := []ValidateOpTbl{
		{"", auglocalexpleafpath, "TestData", true},
		{"", augembedexpleafpath, "TestData", true},
	}

	srv, sess := TstStartupSchemaDir(t, "testdata/augmentValid", "", "")
	ValidateOperationTable(t, sess, srv.Ctx, tblFeature,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblFeature,
		DELETE_AND_COMMIT)
	sess.Kill()
}

// Test that an augment of a remote target node works.
func TestAugmentRemote(t *testing.T) {
	tblFeature := []ValidateOpTbl{
		{"", augremoteleafpath, "TestData", true},
	}

	srv, sess := TstStartupSchemaDir(t, "testdata/augmentValid", "", "")
	ValidateOperationTable(t, sess, srv.Ctx, tblFeature,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblFeature,
		DELETE_AND_COMMIT)
	sess.Kill()
}
