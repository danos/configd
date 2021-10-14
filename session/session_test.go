// Copyright (c) 2017-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/danos/config/testutils"
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/configd/rpc"
	. "github.com/danos/configd/session"
	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
	"github.com/danos/utils/natsort"
	"github.com/danos/utils/pathutil"
)

const emptyschema = ""
const emptyconfig = ""

// Values used in tests
// int ranges are "-50 .. 50 | 52 .. 60 | 70 .. 80"
const (
	intrange1min_minus1  = "-51"
	intrange1min         = "-50"
	intrange1max         = "50"
	intrange1max_plus1   = "51"
	intrange2min         = "52"
	intrangebetween2and3 = "65"
	intrange3max         = "80"
	intrange3maxplus1    = "81"
)

// uint ranges are "1 .. 100 | 150 .. 199 | 220 .. 240"
const (
	uintmin_minus1        = "-1"
	uintmin               = "0"
	uintrange1min_minus1  = "0"
	uintrange1min         = "1"
	uintrange1max         = "100"
	uintrange1max_plus1   = "101"
	uintrangebetween2and3 = "200"
	uintrange3max         = "240"
	uintrange3maxplus1    = "241"
)

// Descriptions for numeric validate set type tests
const validatesetnovalue = "Validate set without value"
const validatesettoosmall = "Validate set too small value"
const validatesetminvalue = "Validate set minimum value"
const validatesetmaxvalue = "Validate set maximum value"
const validatesettoolarge = "Validate set too large value"
const validatesetbelowminrange1 = "Validate set below minimum range 1 value"
const validatesetminrange1 = "Validate set minimum range 1 value"
const validatesetmaxrange1 = "Validate set maximum range 1 value"
const validatesetabovemaxrange1 = "Validate set above maximum range 1 value"
const validatesetbelowminrange2 = "Validate set below minimum range 2 value"
const validatesetminrange2 = "Validate set minimum range 2 value"
const validatesetmaxrange2 = "Validate set maximum range 2 value"
const validatesetabovemaxrange2 = "Validate set above maximum range 2 value"
const validatesetbetweenrange2_3 = "Validate set between range 2 and range 3"
const validatesetbelowminrange3 = "Validate set below minimum range 3 value"
const validatesetminrange3 = "Validate set minimum range 3 value"
const validatesetmaxrange3 = "Validate set maximum range 3 value"
const validatesetabovemaxrange3 = "Validate set above maximum range 3 value"
const validatesetinnerrange = "Validate set inner range value"

// Paths used in tests
const testcontainer = "testcontainer"
const testempty = "testempty"
const testboolean = "testboolean"
const testleaf = "testleaf"
const testleaflistuser = "testleaflistuser"
const testleaflist = "testleaflist"
const testlist = "testlist"
const teststring = "teststring"

var emptypath = []string{}
var invalidpath = []string{"foo", "bar", "baz"}
var rootpath = []string{""}
var testcontainerpath = []string{testcontainer}
var testemptypath = pathutil.CopyAppend(testcontainerpath, testempty)
var testbooleanpath = pathutil.CopyAppend(testcontainerpath, testboolean)
var testleafpath = pathutil.CopyAppend(testcontainerpath, testleaf)
var testleaflistuserpath = pathutil.CopyAppend(testcontainerpath, testleaflistuser)
var testlistpath = pathutil.CopyAppend(testcontainerpath, testlist)
var testlist1path = pathutil.CopyAppend(testlistpath, "list1")
var teststringpath = pathutil.CopyAppend(testcontainerpath, teststring)

// Tests run in the order they are defined

type validateExistsTbl struct {
	path      []string
	expexists bool
}

const existsSchema = `
container testcontainer {
	leaf testempty {
		type empty;
	}
	leaf testboolean {
		type boolean;
		default false;
	}
}
`

func TestExists(t *testing.T) {
	const config = `
testcontainer {
	testempty
}
`
	tbl := []validateExistsTbl{
		{emptypath, true},
		{invalidpath, false},
		{rootpath, false},
		{testemptypath, true},
		{testbooleanpath, true},
	}

	srv, sess := TstStartup(t, existsSchema, config)
	for key, _ := range tbl {
		ValidateExists(t, sess, srv.Ctx, tbl[key].path, tbl[key].expexists)
	}
	sess.Kill()
}

// Check GetTree handles defaults correctly
func TestDefaultExistsGetTree(t *testing.T) {
	srv, sess := TstStartup(t, existsSchema, "")

	opts := &TreeOpts{Defaults: false, Secrets: true}
	if _, err := sess.GetTree(srv.Ctx, testbooleanpath, opts); err == nil {
		t.Fatalf("testboolean should not be found.")
		return
	}

	opts.Defaults = true
	if _, err := sess.GetTree(srv.Ctx, testbooleanpath, opts); err != nil {
		t.Fatalf("testboolean should be found.")
		return
	}
}

// Check GetFullTree handles defaults correctly
func TestDefaultExistsGetFullTree(t *testing.T) {
	// Skip this test until VRVDR-32367 is fixed.
	t.Skip("Skipping until VRVDR-32367 is fixed")
	srv, sess := TstStartup(t, existsSchema, "")

	opts := &TreeOpts{Defaults: false, Secrets: true}
	// TODO - this is returning the default and should not be.
	if _, err, _ := sess.GetFullTree(
		srv.Ctx, testbooleanpath, opts); err == nil {
		t.Fatalf("testboolean should not be found.")
		return
	}

	opts.Defaults = true
	// TODO - this is returning the default even without the fix.  It should
	//        only return the default once the fix is in!
	if _, err, _ := sess.GetFullTree(
		srv.Ctx, testbooleanpath, opts); err != nil {
		t.Fatalf("testboolean should be found.")
		return
	}
}

type validateTypeTbl struct {
	path []string
	exp  rpc.NodeType
}

func validateType(t *testing.T, sess *Session, ctx *configd.Context, tst validateTypeTbl) {
	nt, err := sess.GetType(ctx, tst.path)
	if err != nil {
		t.Errorf("Unable to get type for path [%s]; %s",
			pathutil.Pathstr(tst.path), err)
		testutils.LogStack(t)
	} else if nt != tst.exp {
		t.Errorf("Invalid type %d for path [%s]; expected %d",
			nt, pathutil.Pathstr(tst.path), tst.exp)
		testutils.LogStack(t)
	}
}

func TestGetType(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testempty {
		type empty;
	}
	leaf testboolean {
		type boolean;
		default false;
	}
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
}
`
	const config = `
testcontainer {
	testleaflistuser foo
}
`
	var testbooleanpath_false = pathutil.CopyAppend(testbooleanpath, "false")
	var testleaflistuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")
	tbl := []validateTypeTbl{
		{emptypath, rpc.CONTAINER},
		{invalidpath, rpc.CONTAINER},
		{rootpath, rpc.CONTAINER},
		{testcontainerpath, rpc.CONTAINER},
		{testemptypath, rpc.LEAF},
		{testbooleanpath_false, rpc.LEAF},
		{testlistpath, rpc.LIST},
		{testlist1path, rpc.CONTAINER},
		{testleaflistuserpath, rpc.LEAF_LIST},
		{testleaflistuserpath_foo, rpc.LEAF},
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	for key, _ := range tbl {
		validateType(t, sess, srv.Ctx, tbl[key])
	}
	sess.Kill()
}

type validateDefaultTbl struct {
	path []string
	exp  bool
}

func validateDefault(t *testing.T, sess *Session, ctx *configd.Context, tst validateDefaultTbl) {
	def, err := sess.IsDefault(ctx, tst.path)
	if err != nil {
		t.Errorf("Unable to determine default for path [%s] : %s", pathutil.Pathstr(tst.path), err)
		testutils.LogStack(t)
	} else if def != tst.exp {
		t.Errorf("Incorrect default for path [%s]", pathutil.Pathstr(tst.path))
		testutils.LogStack(t)
	}
}

func TestIsDefault(t *testing.T) {
	const schema = `
typedef testdefaulttype {
	type uint32;
	default 42;
}
container testcontainer {
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf testempty {
		type empty;
	}
	leaf testdefaulttype {
		type testdefaulttype;
	}
}
`
	const config = `
testcontainer {
	testboolean true;
}
`
	var testbooleanpath_true = pathutil.CopyAppend(testbooleanpath, "true")
	var testdefaulttypepath = pathutil.CopyAppend(testcontainerpath, "testdefaulttype")
	tbl := []validateDefaultTbl{
		{emptypath, false},
		{invalidpath, false},
		{rootpath, false},
		{testbooleanpath_true, false},
		{testemptypath, false},
		{testdefaulttypepath, true},
	}
	srv, sess := TstStartup(t, schema, config)
	for key, _ := range tbl {
		validateDefault(t, sess, srv.Ctx, tbl[key])
	}
	sess.Kill()
}

type validateGetTbl struct {
	path []string
	exp  []string
}

func validateGet(t *testing.T, sess *Session, ctx *configd.Context, tst validateGetTbl) {
	val, err := sess.Get(ctx, tst.path)
	if err != nil {
		t.Errorf("Unable to get path [%s] : %s", pathutil.Pathstr(tst.path), err)
		testutils.LogStack(t)
	} else if strings.Join(val, " ") != strings.Join(tst.exp, " ") {
		t.Errorf("Unexpected result from path [%s]",
			pathutil.Pathstr(tst.path))
		t.Logf("Received: %s", val)
		t.Logf("Expected: %s", tst.exp)
		testutils.LogStack(t)
	}
}

func TestGet(t *testing.T) {
	const schema = `
container testcontainer {
    presence "allow config of empty container";
	leaf testboolean {
		type boolean;
		default false;
	}
}
`
	const config = `
testcontainer {
}
`
	tbl := []validateGetTbl{
		{emptypath, []string{testcontainer}},
		{invalidpath, emptypath},
		{rootpath, emptypath},
		{testcontainerpath, []string{testboolean}},
		{testbooleanpath, []string{"false"}},
	}
	srv, sess := TstStartup(t, schema, config)
	for key, _ := range tbl {
		validateGet(t, sess, srv.Ctx, tbl[key])
	}
	sess.Kill()
}

func getLockedState(t *testing.T, sess *Session, ctx *configd.Context) int32 {
	lock, err := sess.Locked(ctx)
	if err != nil {
		t.Fatalf("Unable to get locked state; %s", err)
	}
	return lock
}

func TestLocked(t *testing.T) {
	srv, sess := TstStartup(t, emptyschema, emptyconfig)
	lock := getLockedState(t, sess, srv.Ctx)
	if lock != 0 {
		t.Fatalf("Session incorrectly locked; %d", lock)
	}
	sess.Kill()
}

func TestLock(t *testing.T) {
	srv, sess := TstStartup(t, emptyschema, emptyconfig)
	lock, err := sess.Lock(srv.Ctx)
	if err != nil {
		t.Fatalf("Unable to lock session; %s", err)
	}

	lockpid := getLockedState(t, sess, srv.Ctx)
	if lock != lockpid {
		t.Fatalf("Session incorrectly locked; locked by %d, reported as %d",
			lock, lockpid)
	}

	lock, err = sess.Lock(srv.Ctx)
	if err == nil {
		t.Fatal("Incorrectly locked already locked session")
	}

	ctx := &configd.Context{
		Pid:  int32(5),
		Auth: srv.Auth,
		Dlog: srv.Dlog,
		Elog: srv.Elog,
	}
	lock, err = sess.Lock(ctx)
	if err == nil {
		t.Fatal("Incorrectly locked session locked by different context")
	}
	sess.Kill()
}

func TestUnlock(t *testing.T) {
	srv, sess := TstStartup(t, emptyschema, emptyconfig)

	_, err := sess.Unlock(srv.Ctx)
	if err == nil {
		t.Fatalf("Session incorrectly locked; %s", err)
	}

	var lockpid, unlockpid int32
	lockpid, err = sess.Lock(srv.Ctx)
	if err != nil {
		t.Fatalf("Unable to lock session; %s", err)
	}

	ctx := &configd.Context{
		Pid:  int32(5),
		Auth: srv.Auth,
		Dlog: srv.Dlog,
		Elog: srv.Elog,
	}
	unlockpid, err = sess.Unlock(ctx)
	if err == nil {
		t.Fatalf("Incorrectly unlocked session from different context")
	}

	unlockpid, err = sess.Unlock(srv.Ctx)
	if err != nil {
		t.Fatalf("Unable to unlock session; %s", err)
	}
	if lockpid != unlockpid {
		t.Fatalf("Session was incorrectly locked; locked by %d, unlocked by %d",
			lockpid, unlockpid)
	}

	sess.Kill()
}

func validateSaved(t *testing.T, sess *Session, ctx *configd.Context, exp bool) {
	if sess.Saved(ctx) != exp {
		t.Errorf("Session marked with incorrect saved state; expected %v", exp)
	}
}

func TestSaved(t *testing.T) {
	srv, sess := TstStartup(t, emptyschema, emptyconfig)
	validateSaved(t, sess, srv.Ctx, false)
	sess.MarkSaved(srv.Ctx, true)
	validateSaved(t, sess, srv.Ctx, true)
	sess.MarkSaved(srv.Ctx, false)
	validateSaved(t, sess, srv.Ctx, false)
	sess.Kill()
}

// TODO: move to separate test functions
// validateSetPath(t, sess, srv.ctx, testlistpath, true)
// validateSetPath(t, sess, srv.ctx, testlist1path, false)
func TestValidateSetPath(t *testing.T) {
	const schema = `
container testcontainer {
}
`
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("Validate set without a path", emptypath, "", false),
		NewValOpTblEntry("Validate set invalid path", invalidpath, "", true),
		NewValOpTblEntry("Validate set root path", rootpath, "", true),
		NewValOpTblEntry("Validate set container", testcontainerpath, "", true),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetLeafList(t *testing.T) {
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
}
`
	var testleaflistuserpath_bam = pathutil.CopyAppend(testleaflistuserpath, "bam")
	testleaflistuserpath_bam = pathutil.CopyAppend(testleaflistuserpath_bam, "")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testleaflistuserpath, "", true),
		NewValOpTblEntry("Validate set list-leaf item 1", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("Validate set list-leaf item 2", testleaflistuserpath, "bar", false),
		NewValOpTblEntry("Validate set list-leaf item 3", testleaflistuserpath, "baz", false),
		NewValOpTblEntry("Validate set list-leaf item with trailing /", testleaflistuserpath_bam, "", true),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetList(t *testing.T) {
	const schema = `
container testcontainer {
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
	}
}
`
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testlistpath, "", true),
		NewValOpTblEntry("Validate set list item 1", testlistpath, "foo", false),
		NewValOpTblEntry("Validate set list item 2", testlistpath, "bar", false),
		NewValOpTblEntry("Validate set list item 3", testlistpath, "baz", false),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetUnion(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testunion {
		type union {
			type uint32;
			type string;
		}
	}
}
`
	var testunionpath = pathutil.CopyAppend(testcontainerpath, "testunion")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("Validate set union uint", testunionpath, "10", false),
		NewValOpTblEntry("Validate set union string", testunionpath, "foo", false),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestSet(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testempty {
		type empty;
	}
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf teststring {
		type string;
	}
}
`
	var teststringpath_bam = pathutil.CopyAppend(teststringpath, "bam")
	teststringpath_bam = pathutil.CopyAppend(teststringpath_bam, "")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("Set empty path", emptypath, "", true),
		NewValOpTblEntry("Set invalid path", invalidpath, "", true),
		NewValOpTblEntry("Set root path", rootpath, "", true),
		NewValOpTblEntry("Set empty leaf", testemptypath, "", false),
		NewValOpTblEntry("Set boolean node true", testbooleanpath, "true", false),
		NewValOpTblEntry("Set boolean node false", testbooleanpath, "false", false),
		NewValOpTblEntry("Set string value", teststringpath, "foo", false),
		NewValOpTblEntry("Set string value with trailing /", teststringpath_bam, "", true),
	}
	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestChoiceSet(t *testing.T) {
	var targetpath = []string{"testcontainer", "target", "a-target-value"}
	var abstargetpath = []string{"testcontainer", "abs-target", "a-target-value"}
	var relativetargetpath = []string{"testcontainer", "relative-target", "a-target-value"}
	const schema = `
container testcontainer {
	list target {
		key value;

		leaf value {
			type string;
		}
	}

	choice achoice {
		case one {
			leaf testempty {
				type empty;
			}
			choice alpha {
				leaf alpha-one {
					type string;
				}
				case alpha-case {
					leaf alpha-two {
						type string;
					}
					leaf alpha-three {
						type string;
					}

					leaf abs-target {
						type leafref {
							path "/testcontainer/target/value";
						}
					}
					leaf relative-target {
						type leafref {
							path "../target/value";
						}
					}
				}
			}
			leaf one-one {
				type string;
			}
			leaf one-two {
				type string;
			}
		}
		case two {
			leaf testboolean {
				type boolean;
				default false;
			}
			choice beta {
				leaf beta-one {
					type string;
				}
				case beta-case {
					leaf beta-two {
						type string;
					}
					leaf beta-three {
						type string;
					}
				}
			}
			leaf two-one {
				type string;
			}
			leaf two-two {
				type string;
			}
		}
		leaf teststring {
			type string;
		}
	}
}
`
	var teststringpath_bam = pathutil.CopyAppend(teststringpath, "bam")
	teststringpath_bam = pathutil.CopyAppend(teststringpath_bam, "")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("Set empty path", emptypath, "", true),
		NewValOpTblEntry("Set empty path", targetpath, "", false),
		NewValOpTblEntry("Set empty path", abstargetpath, "", false),
		NewValOpTblEntry("Set empty path", relativetargetpath, "", false),
		NewValOpTblEntry("Set invalid path", invalidpath, "", true),
		NewValOpTblEntry("Set root path", rootpath, "", true),
		NewValOpTblEntry("Set empty leaf", testemptypath, "", false),
		NewValOpTblEntry("Set boolean node true", testbooleanpath, "true", false),
		NewValOpTblEntry("Set boolean node false", testbooleanpath, "false", false),
		NewValOpTblEntry("Set string value", teststringpath, "foo", false),
		NewValOpTblEntry("Set string value with trailing /", teststringpath_bam, "", true),
	}
	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

// Tests that work through a series of set operations
// do verify that other cases in a choice are deleted
func TestChoiceAutoDelete(t *testing.T) {
	const schema = `
	container testcontainer {

	choice achoice {
		case one {
			leaf one-one {
				type string;
			}
		}

		case two {
			leaf one-two {
				type string;
			}
			leaf mand-node {
				mandatory true;
				type string;
			}
		}
		case three {
			container one-three {
				leaf one-three-leaf {
					type string;
				}
			}

			choice anotherchoice {
				container two-one {
					leaf two-one-leaf {
						type string;
					}
				}

				case a {
					container two-two {
						leaf two-two-a {
							type string;
						}
						leaf two-two-b {
							type string;
						}
					}
				}
			}
		}
	}
}
`

	var cOneOne = []string{"testcontainer", "one-one", "11"}
	var cOneTwo = []string{"testcontainer", "one-two", "12"}
	var cMandNode = []string{"testcontainer", "mand-node", "foo"}
	var cOneThreeLeaf = []string{"testcontainer", "one-three", "one-three-leaf", "13"}
	var cTwoOneLeaf = []string{"testcontainer", "two-one", "two-one-leaf", "21"}
	var cTwoTwoA = []string{"testcontainer", "two-two", "two-two-a", "22A"}
	var cTwoTwoB = []string{"testcontainer", "two-two", "two-two-b", "22B"}

	srv, sess := TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	ValidateSet(t, sess, srv.Ctx, cOneOne, false)

	const sOneOne = `testcontainer {
	one-one 11
}
`
	ValidateShow(t, sess, srv.Ctx, emptypath, false, sOneOne, true)

	// Applying this config should remove the one-one config applied earlier
	ValidateSet(t, sess, srv.Ctx, cOneTwo, false)

	const sOneTwo = `testcontainer {
	mand-node foo
	one-two 12
}
`
	// Fails as mand-node is missing
	ValidateCommit(t, sess, srv.Ctx, false, sOneTwo)
	ValidateSet(t, sess, srv.Ctx, cMandNode, false)

	// Success again as mandatory (mand-node) present
	ValidateCommit(t, sess, srv.Ctx, true, sOneTwo)
	ValidateShow(t, sess, srv.Ctx, emptypath, false, sOneTwo, true)

	// this will result in previous config being removed
	ValidateSet(t, sess, srv.Ctx, cOneThreeLeaf, false)

	const sOneThreeLeaf = `testcontainer {
	one-three {
		one-three-leaf 13
	}
}
`
	ValidateShow(t, sess, srv.Ctx, emptypath, false, sOneThreeLeaf, true)

	// Check config in a hierarchical choice behaves correctly
	ValidateSet(t, sess, srv.Ctx, cTwoTwoA, false)
	ValidateSet(t, sess, srv.Ctx, cTwoTwoB, false)

	const sTwoTwo = `testcontainer {
	one-three {
		one-three-leaf 13
	}
	two-two {
		two-two-a 22A
		two-two-b 22B
	}
}
`
	ValidateShow(t, sess, srv.Ctx, emptypath, false, sTwoTwo, true)

	ValidateSet(t, sess, srv.Ctx, cTwoOneLeaf, false)

	const sTwoOneLeaf = `testcontainer {
	one-three {
		one-three-leaf 13
	}
	two-one {
		two-one-leaf 21
	}
}
`
	ValidateShow(t, sess, srv.Ctx, emptypath, false, sTwoOneLeaf, true)
}

// Tests to verify a choice default
// Verify that initial defaults appear in show output
// instantiate values in other cases, and verify the correct
// defaults are shown
func TestChoiceDefaults(t *testing.T) {
	const schema = `

	choice top-level {
		default top-default-seen;

		leaf top-default-seen {
			type string;
			default "seen";
		}
		leaf top-default-hidden {
			type string;
			default "hidden";
		}
	}

	container testcontainer {

	choice achoice {
		default three-four;
		case one {
			leaf one {
				type string;
			}
			leaf default-one {
				type string;
				default "1";
			}
		}

		case two {
			leaf two {
				type string;
			}
			leaf default-two {
				type string;
				default "2";
			}
		}
		case three-four {
			container three {
				leaf three {
					type string;
				}
				leaf default-three {
					type string;
					default "3";
				}
				choice sub-three {
					default sub-three-a;

					case sub-three-a {
						container defaults-seen {
							leaf def-one {
								type string;
								default "1";
							}
							leaf def-two {
								type string;
								default "2";
							}
						}
						container defaults-hidden {
							presence "";

							leaf def-three {
								type string;
								default "3";
							}
							leaf def-four {
								type string;
								default "4";
							}
						}
					}
				}
			}
			container four {
				presence "guard default-four";
				leaf four {
					type string;
				}
				leaf default-four {
					type string;
					default four;
				}
			}
		}
	}
}
`

	srv, sess := TstStartup(t, schema, emptyconfig)
	defer sess.Kill()

	//ValidateSet(t, sess, srv.Ctx, cOneOne, false)

	const initConfig = `testcontainer {
	three {
		default-three 3
		defaults-seen {
			def-one 1
			def-two 2
		}
	}
}
top-default-seen seen
`
	ValidateShowWithDefaults(t, sess, srv.Ctx, emptypath, false, initConfig, true)

	const finalConfig = `testcontainer {
	default-one 1
	one one
}
top-default-hidden override
`
	ValidateSet(t, sess, srv.Ctx, []string{"top-default-hidden", "override"}, false)
	ValidateSet(t, sess, srv.Ctx, []string{"testcontainer", "one", "one"}, false)
	ValidateShowWithDefaults(t, sess, srv.Ctx, emptypath, false, finalConfig, true)

}

func TestSetLeafList(t *testing.T) {
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
	leaf-list testleaflistsystem {
		type string;
		ordered-by system;
	}
}
`
	// TODO: order-by system not supported yet
	// var testleaflistsystempath = pathutil.CopyAppend(testcontainerpath, "testleaflistsystem")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("Set list-leaf without value", testleaflistuserpath, "", true),
		NewValOpTblEntry("Set list-leaf item 1", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("Set list-leaf item 2", testleaflistuserpath, "bar", false),
		NewValOpTblEntry("Set list-leaf item 3", testleaflistuserpath, "baz", false),
		NewValOpTblEntry("Set list-leaf item 4", testleaflistuserpath, "foo", true),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestSetList(t *testing.T) {
	const schema = `
container testcontainer {
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
	}
}
`
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("Set list without value", testlistpath, "", true),
		NewValOpTblEntry("Set list item 1", testlistpath, "foo", false),
		NewValOpTblEntry("Set list item 2", testlistpath, "bar", false),
		NewValOpTblEntry("Set list item 3", testlistpath, "baz", false),
		NewValOpTblEntry("Set list item 4", testlistpath, "foo", true),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

// Checking aspects of leaves' behaviour with defaults and mandatory
// statements:
//
// (a) Non-presence container shows leaf with default
// (b) Presence container doesn't show leaf with default UNLESS
// (c) ... presence container is configured.
// (d) Mandatory leaf inheriting default is accepted and configurable.

func TestDefaultInNonPresenceContainer(t *testing.T) {
	const schema = `
	typedef uint_with_default {
		type uint8;
	    default 66;
	}
	container nonPresenceContainer {
		leaf testLeafInheritsDefault {
			type uint_with_default;
		}
	}`

	// Set up initial empty config.
	srv, sess := TstStartup(t, schema, "")

	// Non-presence showing leaf with default.  Should see '66' as dflt.
	const expNonPresenceConfig = `nonPresenceContainer {
	testLeafInheritsDefault 66
}
`
	ValidateShowWithDefaults(t, sess, srv.Ctx, []string{}, false,
		expNonPresenceConfig, true /* default visible */)

}

func TestDefaultNotShownInUnconfigPresenceContainer(t *testing.T) {
	const schema = `
	typedef uint_with_default {
		type uint8;
	    default 66;
	}
	container presenceContainerWithoutMandatory {
		presence "Present to show defaults hidden";
		leaf testLeafInheritsDefault {
			type uint_with_default;
		}
	}`

	// Set up initial empty config
	srv, sess := TstStartup(t, schema, "")

	// Presence container should not show leaf with default
	const expPresenceConfig = `presenceContainerWithoutMandatory {
	testLeafInheritsDefault 66
}
`
	ValidateShowWithDefaults(t, sess, srv.Ctx, []string{}, false,
		expPresenceConfig, false /* 66 not visible */)

	sess.Kill()
}

func TestDefaultShownInConfiguredPresenceContainer(t *testing.T) {
	const schema = `
	typedef uint_with_default {
		type uint8;
	    default 66;
	}
	container presenceContainerWithoutMandatory {
		presence "Present to show defaults hidden";
		leaf testLeafInheritsDefault {
			type uint_with_default;
		}
	}`

	// Set up initial empty config
	srv, sess := TstStartup(t, schema, "")

	// Now configure presence container and we should see default.
	const cfgPresence = `presenceContainerWithoutMandatory
`
	const expPresenceConfigWithoutMandatory = `presenceContainerWithoutMandatory {
	testLeafInheritsDefault 66
}
`
	tblSetPresenceWithoutMand := []ValidateOpTbl{
		NewValOpTblEntry("Verify set of non-mandatory presence container",
			[]string{"presenceContainerWithoutMandatory"}, "", false),
	}

	ValidateOperationTable(t, sess, srv.Ctx, tblSetPresenceWithoutMand,
		SET)
	ValidateCommit(t, sess, srv.Ctx, true /* expect pass */, cfgPresence)
	ValidateShowWithDefaults(t, sess, srv.Ctx, []string{}, false,
		expPresenceConfigWithoutMandatory, true)

	sess.Kill()
}

func TestMandatoryLeafInheritingDefaultIsConfigurable(t *testing.T) {
	const schema = `
	typedef uint_with_default {
		type uint8;
	    default 66;
	}
	container presenceContainer {
        presence "Show mandatory overrides inherited default.";
        description "Container to show mandatory can override default.";
		leaf testLeafInheritsDefault {
			type uint_with_default;
            mandatory "true";
		}
	}`

	// Set up initial config with mandatory node
	const mandatoryPresenceConfig = `presenceContainer {
	testLeafInheritsDefault 33
}
`
	srv, sess := TstStartup(t, schema, mandatoryPresenceConfig)

	// Non-presence showing leaf with default overridden
	const expPresenceConfig = `presenceContainer {
	testLeafInheritsDefault 33
}
`
	ValidateShowWithDefaults(t, sess, srv.Ctx, []string{}, false,
		expPresenceConfig, true /* 33 visible */)

	sess.Kill()
}

func validateCommitOrdering(t *testing.T, sess *Session, ctx *configd.Context, exp bool, expOut string) {
	ValidateSessOpOutput(t, sess, ctx, exp, expOut, COMMIT)
}

func TestDelete(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testempty {
		type empty;
	}
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
}
`
	const config = `
testcontainer {
	testempty
	testlist foo
	testlist bar
	testlist baz
	testleaflistuser foo
	testleaflistuser bar
	testleaflistuser baz
}
`
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("", emptypath, "", true),
		NewValOpTblEntry("", invalidpath, "", true),
		NewValOpTblEntry("", rootpath, "", true),
		NewValOpTblEntry("", testemptypath, "", false),
		NewValOpTblEntry("", testlistpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "foo", true),
		NewValOpTblEntry("", testlistpath, "baz", false),
		NewValOpTblEntry("", testlistpath, "baz", true),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", true),
		NewValOpTblEntry("", testleaflistuserpath, "baz", false),
		NewValOpTblEntry("", testleaflistuserpath, "baz", true),
	}

	srv, sess := TstStartup(t, schema, config)
	ValidateDeleteTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestDeleteWithDefault(t *testing.T) {
	const schema = `
container testcontainer {
    container cwp {
        presence "Some presence container";
        leaf bar {
           type string;
        }
    }
    container testcontainer2 {
        leaf testdefault {
            type string;
            default hrw;
        }
    }
}
`
	const config = `
testcontainer {
    cwp
}
`
	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, []string{"testcontainer"}, false)
	ValidateExists(t, sess, srv.Ctx, []string{"testcontainer", "cwp"}, false)
	sess.Kill()
}

func validateChanged(t *testing.T, sess *Session, ctx *configd.Context, exp bool) {
	if sess.Changed(ctx) != exp {
		t.Errorf("Session marked with incorrect changed state; expected %v", exp)
	}
}

func TestChanged(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf teststring {
		type string;
	}
}
`
	const config = `
testcontainer {
	teststring foo
}
`
	srv, sess := TstStartup(t, schema, config)

	validateChanged(t, sess, srv.Ctx, false)

	var testbooleanpath_true = pathutil.CopyAppend(testbooleanpath, "true")
	ValidateSet(t, sess, srv.Ctx, testbooleanpath_true, false)
	validateChanged(t, sess, srv.Ctx, true)

	ValidateDelete(t, sess, srv.Ctx, testbooleanpath, false)
	validateChanged(t, sess, srv.Ctx, false)

	var teststringpath_bar = pathutil.CopyAppend(teststringpath, "bar")
	ValidateSet(t, sess, srv.Ctx, teststringpath_bar, false)
	validateChanged(t, sess, srv.Ctx, true)

	err := sess.Discard(srv.Ctx)
	if err != nil {
		t.Errorf("Discard failed; %s", err)
	}
	validateChanged(t, sess, srv.Ctx, false)

	sess.Kill()
}

type validateStatusTbl struct {
	path   []string
	status rpc.NodeStatus
	err    bool
}

func validateStatus(t *testing.T, sess *Session, ctx *configd.Context, exp validateStatusTbl) {
	status, err := sess.GetStatus(ctx, exp.path)
	if (err != nil) != exp.err {
		if err == nil {
			t.Errorf("Unexpected error from get status of  path [%s]",
				pathutil.Pathstr(exp.path))
		} else {
			t.Errorf("Unexpeced error from to get status of path [%s]; %s",
				pathutil.Pathstr(exp.path), err)
		}
		testutils.LogStack(t)
		return
	}
	if status != exp.status {
		statusStr := [...]string{"UNCHANGED", "CHANGED", "ADDED", "DELETED"}
		t.Errorf("Unexpected status from path [%s]", pathutil.Pathstr(exp.path))
		t.Logf("Received: %s(%d)", statusStr[status], status)
		t.Logf("Expected: %s(%d)", statusStr[exp.status], exp.status)
		testutils.LogStack(t)
	}
}

func TestGetStatus(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testempty {
		type empty;
	}
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf teststring {
		type string;
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
	list testlist {
		key name;
		leaf name {
			type string;
		}
		leaf bar {
			type empty;
		}
	}
}
`
	const config = `
testcontainer {
	teststring foo
	testleaflistuser foo
	testleaflistuser bar
	testlist foo
	testlist baz {
		bar
	}
}
`
	var testbooleanpath_true = pathutil.CopyAppend(testbooleanpath, "true")
	var testleaflistuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")
	var testleaflistuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")
	var testlistpath_foo = pathutil.CopyAppend(testlistpath, "foo")
	var testlistpath_foo_bar = pathutil.CopyAppend(testlistpath_foo, "bar")
	var testlistpath_baz = pathutil.CopyAppend(testlistpath, "baz")
	var testlistpath_baz_bar = pathutil.CopyAppend(testlistpath_baz, "bar")
	tbl := []ValidateStatusTbl{
		NewValStatusTblEntry(emptypath, rpc.UNCHANGED, false),
		NewValStatusTblEntry(invalidpath, rpc.UNCHANGED, true),
		NewValStatusTblEntry(rootpath, rpc.UNCHANGED, true),
		NewValStatusTblEntry(testcontainerpath, rpc.CHANGED, false),
		NewValStatusTblEntry(testemptypath, rpc.UNCHANGED, true),
		NewValStatusTblEntry(testbooleanpath_true, rpc.CHANGED, false),
		NewValStatusTblEntry(teststringpath, rpc.DELETED, false),
		NewValStatusTblEntry(testleaflistuserpath, rpc.CHANGED, false),
		NewValStatusTblEntry(testleaflistuserpath_foo, rpc.DELETED, false),
		NewValStatusTblEntry(testleaflistuserpath_bar, rpc.CHANGED, false),
		NewValStatusTblEntry(testlistpath_foo, rpc.CHANGED, false),
		NewValStatusTblEntry(testlistpath_foo_bar, rpc.ADDED, false),
		NewValStatusTblEntry(testlistpath_baz_bar, rpc.DELETED, false),
	}

	srv, sess := TstStartup(t, schema, config)

	ValidateSet(t, sess, srv.Ctx, testbooleanpath_true, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo_bar, false)
	ValidateDelete(t, sess, srv.Ctx, teststringpath, false)
	ValidateDelete(t, sess, srv.Ctx, testleaflistuserpath_foo, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_baz, false)

	for key, _ := range tbl {
		ValidateStatus(t, sess, srv.Ctx, tbl[key])
	}
	sess.Kill()
}

func TestShow(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf teststring {
		type string;
		configd:secret true;
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
	list testlist {
		key name;
		leaf name {
			type string;
		}
		leaf bar {
			type empty;
		}
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
	testlist foo {
		bar
	}
	teststring foo
}
`
	srv, sess := TstStartup(t, schema, config)
	ValidateShow(t, sess, srv.Ctx, emptypath, false, config, true)

	hidcfg := strings.Replace(config, "teststring foo", "teststring \"********\"", 1)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, hidcfg, true)

	expErrs := errtest.
		NewNodeDoesntExistError(t, "/foo").
		RawErrorStrings()

	ValidateShowContains(t, sess, srv.Ctx, invalidpath, false, true, expErrs...)
	sess.Kill()
}

func mkLoadFile(t *testing.T, config string) string {
	f, err := ioutil.TempFile("/tmp", "tmpconfig")
	if err != nil {
		t.Fatal("Unable to create test config file")
		testutils.LogStack(t)
		return ""
	}
	name := f.Name()
	f.WriteString(config)
	f.Close()
	return name
}

func validateLoad(t *testing.T, sess *Session, ctx *configd.Context, cfgfile string) {
	err, invalidPaths := sess.Load(ctx, cfgfile, nil)
	if err != nil {
		t.Errorf("Error loading configuration file %s; %s", cfgfile, err)
		testutils.LogStack(t)
	}
	if len(invalidPaths) > 0 {
		t.Fatalf("Invalid paths when loading configuration file %s:\n%v\n",
			cfgfile, invalidPaths)
		return
	}
}

func TestLoad(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf teststring {
		type string;
		configd:secret true;
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
	list testlist {
		key name;
		leaf name {
			type string;
		}
		leaf bar {
			type empty;
		}
	}
}
`
	const config = `
testcontainer {
	testleaflistuser foo
	testleaflistuser bar
	testlist foo {
		bar
	}
	teststring foo
}
`
	// config has a prepended '\n' so strip it
	expcfg := config[1:]
	srv, sess := TstStartup(t, schema, emptyconfig)

	name := mkLoadFile(t, expcfg)
	if len(name) == 0 {
		return
	}
	validateLoad(t, sess, srv.Ctx, name)
	os.Remove(name)
	ValidateShow(t, sess, srv.Ctx, emptypath, false, expcfg, true)

	sess.Kill()
}

type validateGetTreeTbl struct {
	path     []string
	encoding string
	exptree  string
	expfail  bool
}

func validateGetTree(t *testing.T, sess *Session, ctx *configd.Context, tst validateGetTreeTbl) {
	ut, err := sess.GetTree(ctx, tst.path,
		&TreeOpts{Defaults: false, Secrets: true})
	var tree string
	if err == nil {
		tree, err = ut.Marshal("data", tst.encoding, union.Authorizer(sess.NewAuther(ctx)),
			union.IncludeDefaults)
	}
	if (err != nil) != tst.expfail {
		if err == nil {
			t.Errorf("Unexpected get tree result for path [%s]; \n%s",
				pathutil.Pathstr(tst.path), tree)
		} else {
			t.Errorf("Error getting tree for path loading %s; %s",
				pathutil.Pathstr(tst.path), err)
		}
		testutils.LogStack(t)
		return
	}

	if !tst.expfail && tst.exptree != tree {
		t.Errorf("Unexpected tree returned for path %s", pathutil.Pathstr(tst.path))
		t.Logf("Received:\n%s", tree)
		t.Logf("Expected:\n%s", tst.exptree)
	}
}

func TestGetTree(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf teststring {
		type string;
		configd:secret true;
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
	list testlist {
		key name;
		leaf name {
			type string;
		}
		leaf bar {
			type empty;
		}
	}
	list testlistuser {
		ordered-by "user";
		key name;
		leaf name {
			type string;
		}
		leaf bar {
			type empty;
		}
	}
}
container state {
	config false;
	leaf status {
		type string;
		default "foo";
		configd:get-state "echo {\"status\": \"Should not see this\"}";
	}
}
`
	const config = `
testcontainer {
	testleaflistuser foo
	testleaflistuser bar
	testlist foo {
		bar
	}
	testlist baz {
		bar
	}
	testlist bar {
		bar
	}
	teststring foo
	testlistuser foo {
		bar
	}
	testlistuser baz {
		bar
	}
	testlistuser bar {
		bar
	}
}
`
	const cfg_internal = `{"testcontainer":{"testboolean":false,"testleaflistuser":["foo","bar"],"testlist":{"bar":{"bar":null},"baz":{"bar":null},"foo":{"bar":null}},"testlistuser":{"foo":{"bar":null},"baz":{"bar":null},"bar":{"bar":null}},"teststring":"foo"}}`
	const cfg_json = `{"testcontainer":{"testboolean":false,"testleaflistuser":["foo","bar"],"testlist":[{"name":"bar","bar":null},{"name":"baz","bar":null},{"name":"foo","bar":null}],"testlistuser":[{"name":"foo","bar":null},{"name":"baz","bar":null},{"name":"bar","bar":null}],"teststring":"foo"}}`
	const cfg_xml = `<data><testcontainer xmlns="urn:vyatta.com:test:configd-session"><testboolean xmlns="urn:vyatta.com:test:configd-session">false</testboolean><testleaflistuser xmlns="urn:vyatta.com:test:configd-session">foo</testleaflistuser><testleaflistuser xmlns="urn:vyatta.com:test:configd-session">bar</testleaflistuser><testlist xmlns="urn:vyatta.com:test:configd-session"><name xmlns="urn:vyatta.com:test:configd-session">bar</name><bar xmlns="urn:vyatta.com:test:configd-session"></bar></testlist><testlist xmlns="urn:vyatta.com:test:configd-session"><name xmlns="urn:vyatta.com:test:configd-session">baz</name><bar xmlns="urn:vyatta.com:test:configd-session"></bar></testlist><testlist xmlns="urn:vyatta.com:test:configd-session"><name xmlns="urn:vyatta.com:test:configd-session">foo</name><bar xmlns="urn:vyatta.com:test:configd-session"></bar></testlist><testlistuser xmlns="urn:vyatta.com:test:configd-session"><name xmlns="urn:vyatta.com:test:configd-session">foo</name><bar xmlns="urn:vyatta.com:test:configd-session"></bar></testlistuser><testlistuser xmlns="urn:vyatta.com:test:configd-session"><name xmlns="urn:vyatta.com:test:configd-session">baz</name><bar xmlns="urn:vyatta.com:test:configd-session"></bar></testlistuser><testlistuser xmlns="urn:vyatta.com:test:configd-session"><name xmlns="urn:vyatta.com:test:configd-session">bar</name><bar xmlns="urn:vyatta.com:test:configd-session"></bar></testlistuser><teststring xmlns="urn:vyatta.com:test:configd-session">foo</teststring></testcontainer></data>`
	const enc_internal = "internal"
	const enc_json = "json"
	const enc_xml = "xml"
	const enc_invalid = "invalidencoding"
	tbl := []validateGetTreeTbl{
		{emptypath, enc_invalid, "", true},
		{emptypath, enc_internal, cfg_internal, false},
		{invalidpath, enc_internal, cfg_internal, true},
		{rootpath, enc_internal, cfg_internal, true},
		{testcontainerpath, enc_internal, cfg_internal, false},
		{testcontainerpath, enc_json, cfg_json, false},
		{testcontainerpath, enc_xml, cfg_xml, false},
	}

	srv, sess := TstStartup(t, schema, config)
	for key, _ := range tbl {
		validateGetTree(t, sess, srv.Ctx, tbl[key])
	}
	sess.Kill()
}

func validateValidate(t *testing.T, sess *Session, ctx *configd.Context, exp bool, expOut string) {
	ValidateSessOpOutput(t, sess, ctx, exp, expOut, VALIDATE)
}

// TODO: Since no xpath, need multiple schemas to test validation
// failure and success. Once we have xpath support these can be
// collapsed into a single test schema with xpath expression.
func TestValidateFailure(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testempty {
		type empty;
		configd:validate "false";
	}
}
`
	const emptyout = ""
	srv, sess := TstStartup(t, schema, emptyconfig)

	// Validate locked session
	altctx := &configd.Context{
		Pid:  int32(1),
		Auth: srv.Auth,
		Dlog: srv.Dlog,
		Elog: srv.Elog,
	}
	_, err := sess.Lock(altctx)
	if err != nil {
		t.Fatalf("Unable to lock session; %s", err)
	}
	validateValidate(t, sess, srv.Ctx, false, emptyout)
	_, err = sess.Unlock(altctx)
	if err != nil {
		t.Fatalf("Unable to unlock session; %s", err)
	}

	// Validate no change doesn't generate error first ...
	validateValidate(t, sess, srv.Ctx, true, emptyout)

	// Validate with validation failure
	ValidateSet(t, sess, srv.Ctx, testemptypath, false)
	validateValidate(t, sess, srv.Ctx, false, emptyout)
	sess.Kill()
}

func TestValidate(t *testing.T) {
	const schema = `container testcontainer {
	leaf testempty {
		type empty;
		configd:validate "echo testempty";
	}
	leaf testboolean {
		type boolean;
		default false;
		configd:validate "echo testboolean";
	}
	leaf teststring {
		type string;
		configd:secret true;
		configd:validate "echo teststring";
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:validate "echo testleaflistuser";
	}
	list testlist {
		key name;
		leaf name {
			type string;
			configd:validate "echo testlist key name";
		}
		leaf bar {
			type empty;
			configd:validate "echo testlist leaf bar";
		}
		configd:validate "echo testlist";
	}
}
`
	const config = `testcontainer {
	testempty
	testlist foo {
		bar
	}
}
`
	var expOutput = `[testcontainer testboolean false]
testboolean

[testcontainer testempty]
testempty

[testcontainer testleaflistuser bar]
testleaflistuser

[testcontainer testleaflistuser foo]
testleaflistuser

[testcontainer testlist baz]
testlist

[testcontainer testlist baz bar]
testlist leaf bar

[testcontainer testlist baz name baz]
testlist key name

[testcontainer teststring foo]
teststring

`
	var testleaflistuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")
	var testleaflistuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")
	var testlistpath_foo = pathutil.CopyAppend(testlistpath, "foo")
	var testlistpath_baz = pathutil.CopyAppend(testlistpath, "baz")
	var testlistpath_baz_bar = pathutil.CopyAppend(testlistpath_baz, "bar")
	var teststringpath_foo = pathutil.CopyAppend(teststringpath, "foo")

	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_bar, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_baz_bar, false)
	ValidateSet(t, sess, srv.Ctx, teststringpath_foo, false)
	validateValidate(t, sess, srv.Ctx, true, expOutput)
	sess.Kill()
}

func TestExtensionIfFeatureEnabled(t *testing.T) {
	const schema = `
	feature testfeature {
		description "testfeature";
	}

	augment /testcontainer {
		if-feature testfeature;
		configd:validate "echo testcontainer if-feature";
	}

container testcontainer {
	leaf testempty {
		type empty;
		configd:validate "echo testempty";
	}
	leaf testboolean {
		type boolean;
		default false;
		configd:validate "echo testboolean";
	}
	leaf teststring {
		type string;
		configd:secret true;
		configd:validate "echo teststring";
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:validate "echo testleaflistuser";
	}
	list testlist {
		key name;
		leaf name {
			type string;
			configd:validate "echo testlist key name";
		}
		leaf bar {
			type empty;
			configd:validate "echo testlist leaf bar";
		}
		configd:validate "echo testlist";
	}
}
`
	const config = `testcontainer {
	testempty
	testlist foo {
		bar
	}
}
`
	var expOutput = `[testcontainer]
testcontainer if-feature

[testcontainer testboolean false]
testboolean

[testcontainer testempty]
testempty

[testcontainer testleaflistuser bar]
testleaflistuser

[testcontainer testleaflistuser foo]
testleaflistuser

[testcontainer testlist baz]
testlist

[testcontainer testlist baz bar]
testlist leaf bar

[testcontainer testlist baz name baz]
testlist key name

[testcontainer teststring foo]
teststring

`
	var testleaflistuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")
	var testleaflistuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")
	var testlistpath_foo = pathutil.CopyAppend(testlistpath, "foo")
	var testlistpath_baz = pathutil.CopyAppend(testlistpath, "baz")
	var testlistpath_baz_bar = pathutil.CopyAppend(testlistpath_baz, "bar")
	var teststringpath_foo = pathutil.CopyAppend(teststringpath, "foo")

	srv, sess := TstStartupWithCapabilities(t, schema, config,
		"testdata/extensionFeatures/capsAll")
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_bar, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_baz_bar, false)
	ValidateSet(t, sess, srv.Ctx, teststringpath_foo, false)
	validateValidate(t, sess, srv.Ctx, true, expOutput)
	sess.Kill()
}
func TestExtensionIfFeatureDisabled(t *testing.T) {
	const schema = `
	feature testfeature {
		description "testfeature";
	}

	augment /testcontainer {
		if-feature testfeature;
		configd:validate "echo testcontainer if-feature";
	}

container testcontainer {
	leaf testempty {
		type empty;
		configd:validate "echo testempty";
	}
	leaf testboolean {
		type boolean;
		default false;
		configd:validate "echo testboolean";
	}
	leaf teststring {
		type string;
		configd:secret true;
		configd:validate "echo teststring";
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:validate "echo testleaflistuser";
	}
	list testlist {
		key name;
		leaf name {
			type string;
			configd:validate "echo testlist key name";
		}
		leaf bar {
			type empty;
			configd:validate "echo testlist leaf bar";
		}
		configd:validate "echo testlist";
	}
}
`
	const config = `testcontainer {
	testempty
	testlist foo {
		bar
	}
}
`
	var expOutput = `[testcontainer testboolean false]
testboolean

[testcontainer testempty]
testempty

[testcontainer testleaflistuser bar]
testleaflistuser

[testcontainer testleaflistuser foo]
testleaflistuser

[testcontainer testlist baz]
testlist

[testcontainer testlist baz bar]
testlist leaf bar

[testcontainer testlist baz name baz]
testlist key name

[testcontainer teststring foo]
teststring

`
	var testleaflistuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")
	var testleaflistuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")
	var testlistpath_foo = pathutil.CopyAppend(testlistpath, "foo")
	var testlistpath_baz = pathutil.CopyAppend(testlistpath, "baz")
	var testlistpath_baz_bar = pathutil.CopyAppend(testlistpath_baz, "bar")
	var teststringpath_foo = pathutil.CopyAppend(teststringpath, "foo")

	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_bar, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_baz_bar, false)
	ValidateSet(t, sess, srv.Ctx, teststringpath_foo, false)
	validateValidate(t, sess, srv.Ctx, true, expOutput)
	sess.Kill()
}

func TestCommit(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testboolean {
		type boolean;
		default false;
	}
	leaf teststring {
		type string;
		configd:secret true;
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
	}
	list testlist {
		key name;
		leaf name {
			type string;
		}
		leaf bar {
			type empty;
		}
	}
}
`
	const config = `testcontainer {
	testboolean true
	testleaflistuser foo
	testleaflistuser bar
	testlist foo {
		bar
	}
	teststring foo
}
`
	srv, sess := TstStartup(t, schema, emptyconfig)

	// Commit nothing
	ValidateCommit(t, sess, srv.Ctx, false, emptyconfig)

	// Commit locked session
	altctx := &configd.Context{
		Pid:  int32(1),
		Auth: srv.Auth,
		Dlog: srv.Dlog,
		Elog: srv.Elog,
	}
	_, err := sess.Lock(altctx)
	if err != nil {
		t.Fatalf("Unable to lock session; %s", err)
	}
	ValidateCommit(t, sess, srv.Ctx, false, emptyconfig)
	_, err = sess.Unlock(altctx)
	if err != nil {
		t.Fatalf("Unable to unlock session; %s", err)
	}

	// Commit changes
	var testbooleanpath_true = pathutil.CopyAppend(testbooleanpath, "true")
	var teststringpath_foo = pathutil.CopyAppend(teststringpath, "foo")
	var testleaflistuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")
	var testleaflistuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")
	var testlistpath_foo = pathutil.CopyAppend(testlistpath, "foo")
	var testlistpath_foo_bar = pathutil.CopyAppend(testlistpath_foo, "bar")

	ValidateSet(t, sess, srv.Ctx, testbooleanpath_true, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistuserpath_bar, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo_bar, false)
	ValidateSet(t, sess, srv.Ctx, teststringpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, config)

	sess.Kill()
}

/*
 * TestUnique
 *
 * T1: same port, no IP
 * T2: same port, S1 has IP
 * T3: same port, different IP
 * T4: same IP, no port
 * T5: same IP, S1 has port
 * T6: same IP, different port
 * T7: same IP and port (expect FAIL)
 */
func TestUnique(t *testing.T) {
	type validateUniqueTbl struct {
		path []string
		exp  bool
	}

	type uniqueTestTbl struct {
		add_cmds []ValidateOpTbl
		exp      bool
	}

	const schema = `
	container testuniq {
		list server {
			key "name";
			unique "port ip";
			leaf name {
				type string;
			}
			leaf ip {
				type uint32;
			}
			leaf port {
				type uint32 {
					range 1000..9999;
				}
			}
		}
	}
	`

	const config = `testuniq {
	server dummy
}
`

	const server = "server"
	const testuniq = "testuniq"

	var testuniqpath = []string{testuniq}
	var server_path = pathutil.CopyAppend(testuniqpath, server)
	var s1p1 = []string{"testuniq", "server", "s1", "port", "1111"}
	var s1i1 = []string{"testuniq", "server", "s1", "ip", "111"}
	var s2p1 = []string{"testuniq", "server", "s2", "port", "1111"}
	var s2p2 = []string{"testuniq", "server", "s2", "port", "2222"}
	var s2i1 = []string{"testuniq", "server", "s2", "ip", "111"}
	var s2i2 = []string{"testuniq", "server", "s2", "ip", "222"}

	// Always use S1 and S2, so common delete table.
	test_del_tbl := []ValidateOpTbl{
		NewValOpTblEntry("", server_path, "s1", true /* commit should pass */),
		NewValOpTblEntry("", server_path, "s2", true /* commit should pass */),
	}

	// T1: same port, no IP
	test1_tbl := []ValidateOpTbl{
		NewValOpTblEntry("", s1p1, "", false /* set should PASS */),
		NewValOpTblEntry("", s2p1, "", false),
	}

	// T2: same port, S1 has IP
	test2_tbl := []ValidateOpTbl{
		NewValOpTblEntry("", s1p1, "", false),
		NewValOpTblEntry("", s1i1, "", false),
		NewValOpTblEntry("", s2p1, "", false),
	}

	// T3: same port, different IP
	test3_tbl := []ValidateOpTbl{
		NewValOpTblEntry("", s1p1, "", false),
		NewValOpTblEntry("", s1i1, "", false),
		NewValOpTblEntry("", s2p1, "", false),
		NewValOpTblEntry("", s2i2, "", false),
	}

	// T4: same IP, no port
	test4_tbl := []ValidateOpTbl{
		NewValOpTblEntry("", s1i1, "", false),
		NewValOpTblEntry("", s2i1, "", false),
	}

	// T5: same IP, S1 has port
	test5_tbl := []ValidateOpTbl{
		NewValOpTblEntry("", s1p1, "", false),
		NewValOpTblEntry("", s1i1, "", false),
		NewValOpTblEntry("", s2i1, "", false),
	}

	// T6: same IP, different port
	test6_tbl := []ValidateOpTbl{
		NewValOpTblEntry("", s1p1, "", false),
		NewValOpTblEntry("", s1i1, "", false),
		NewValOpTblEntry("", s2p2, "", false),
		NewValOpTblEntry("", s2i1, "", false),
	}

	// T7: same IP and port (expect FAIL)
	test7_tbl_fail := []ValidateOpTbl{
		NewValOpTblEntry("", s1p1, "", false),
		NewValOpTblEntry("", s1i1, "", false),
		NewValOpTblEntry("", s2p1, "", false),
		NewValOpTblEntry("", s2i1, "", false),
	}

	// List of tests + results.  Ideally we'd have the test definitions
	// above including their result, but for now this will do.  Note use
	// of _fail as suffix for tests expected to fail to try to ensure
	// correct results listed below.
	uniq_tests := []uniqueTestTbl{
		{test1_tbl, true /* commit should pass */},
		{test2_tbl, true},
		{test3_tbl, true},
		{test4_tbl, true},
		{test5_tbl, true},
		{test6_tbl, true},
		{test7_tbl_fail, false /* commit should fail */},
	}

	srv, sess := TstStartup(t, schema, config)

	// For each test case, set all commands, then commit, then delete
	// and (un)commit to leave a clean config for next test.
	for _, test := range uniq_tests {
		ValidateOperationTable(t, sess, srv.Ctx, test.add_cmds, SET)
		ValidateCommit(t, sess, srv.Ctx, test.exp)
		ValidateOperationTable(t, sess, srv.Ctx, test_del_tbl,
			DELETE_AND_COMMIT)
	}

	sess.Kill()
}

type leaflistpath []string

func (p leaflistpath) Generate(rand *rand.Rand, size int) reflect.Value {
	p = pathutil.CopyAppend([]string{testleaflist},
		fmt.Sprintf("%d", rand.Uint32()))
	return reflect.ValueOf(p)
}

func (p leaflistpath) String() string {
	var b bytes.Buffer
	for i, s := range p {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s)
	}
	return b.String()
}

func TestLeafListUserOrder(t *testing.T) {
	if testing.Short() {
		// Ironically, 'quick' test takes 1.75s!
		t.Skipf("Skip LeafListUser Order test for 'short' tests")
	}

	const schema = `
	leaf-list testleaflist {
		type string;
		ordered-by user;
	}
`
	srv, sess := TstStartup(t, schema, emptyconfig)
	defer sess.Kill()
	check := func(paths []leaflistpath) bool {
		defer sess.Discard(srv.Ctx)
		var exp bytes.Buffer
		for _, p := range paths {
			if err := sess.Set(srv.Ctx, p); err != nil {
				t.Fatal(err)
			}
			exp.WriteString(p.String())
			exp.WriteByte('\n')
		}
		cfg, err := sess.Show(srv.Ctx, emptypath, true, false)
		if err != nil {
			t.Fatal(err)
		}
		return cfg == exp.String()
	}

	seed := time.Now().UnixNano()
	qcfg := quick.Config{
		Rand: rand.New(rand.NewSource(seed)),
	}
	if err := quick.Check(check, &qcfg); err != nil {
		t.Logf("Seed %v", seed)
		t.Error(err)
	}
}

func TestLeafListSystemOrder(t *testing.T) {
	if testing.Short() {
		// Ironically, 'quick' test takes 1.75s!
		t.Skipf("Skip LeafListSystem Order test for 'short' tests")
	}

	const schema = `
	leaf-list testleaflist {
		type string;
		ordered-by system;
	}
`
	srv, sess := TstStartup(t, schema, emptyconfig)
	defer sess.Kill()
	check := func(paths []leaflistpath) bool {
		defer sess.Discard(srv.Ctx)
		cfgPaths := make([]string, len(paths))
		for i, p := range paths {
			if err := sess.Set(srv.Ctx, p); err != nil {
				t.Fatal(err)
			}
			cfgPaths[i] = p.String()
		}
		cfg, err := sess.Show(srv.Ctx, emptypath, true, false)
		if err != nil {
			t.Fatal(err)
		}
		natsort.Sort(cfgPaths)
		var exp bytes.Buffer
		for _, p := range cfgPaths {
			exp.WriteString(p)
			exp.WriteByte('\n')
		}
		return cfg == exp.String()
	}

	seed := time.Now().UnixNano()
	qcfg := quick.Config{
		Rand: rand.New(rand.NewSource(seed)),
	}
	if err := quick.Check(check, &qcfg); err != nil {
		t.Logf("Seed %v", seed)
		t.Error(err)
	}
}

func TestLeafListOrder_VRVDR2911(t *testing.T) {
	const schema = `
container testcontainer {
	list testlist {
		key id;
		leaf id {
			type uint32;
		}
		leaf-list testleaflistuser {
			type string;
			ordered-by user;
		}
	}
}
`
	const config = `testcontainer {
	testlist 0 {
		testleaflistuser foo
		testleaflistuser bar
	}
}
`
	const expconfig = `testcontainer {
	testlist 0 {
		testleaflistuser baz
		testleaflistuser bar
	}
}
`
	var testlistpath_0 = pathutil.CopyAppend(testlistpath, "0")
	var testleaflistpath = pathutil.CopyAppend(testlistpath_0, testleaflistuser)
	var testleaflistpath_bar = pathutil.CopyAppend(testleaflistpath, "bar")
	var testleaflistpath_baz = pathutil.CopyAppend(testleaflistpath, "baz")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistpath_baz, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig)
	ValidateDelete(t, sess, srv.Ctx, testlistpath, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistpath_baz, false)
	ValidateSet(t, sess, srv.Ctx, testleaflistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, false, expconfig)
	sess.Kill()
}

type listpath []string

func (p listpath) Generate(rand *rand.Rand, size int) reflect.Value {
	p = pathutil.CopyAppend([]string{testlist},
		fmt.Sprintf("%d", rand.Uint32()))
	return reflect.ValueOf(p)
}

func (p listpath) String() string {
	var b bytes.Buffer
	for i, s := range p {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s)
	}
	return b.String()
}

func TestListUserOrder(t *testing.T) {
	if testing.Short() {
		// Ironically, 'quick' test takes 1.75s!
		t.Skipf("Skip ListUser Order test for 'short' tests")
	}

	const schema = `
	list testlist {
		ordered-by user;
		key name;
		leaf name {
			type string;
		}
	}
`
	srv, sess := TstStartup(t, schema, emptyconfig)
	defer sess.Kill()
	check := func(paths []listpath) bool {
		defer sess.Discard(srv.Ctx)
		var exp bytes.Buffer
		for _, p := range paths {
			if err := sess.Set(srv.Ctx, p); err != nil {
				t.Fatal(err)
			}
			exp.WriteString(p.String())
			exp.WriteByte('\n')
		}
		cfg, err := sess.Show(srv.Ctx, emptypath, true, false)
		if err != nil {
			t.Fatal(err)
		}
		return cfg == exp.String()
	}

	seed := time.Now().UnixNano()
	qcfg := quick.Config{
		Rand: rand.New(rand.NewSource(seed)),
	}
	if err := quick.Check(check, &qcfg); err != nil {
		t.Logf("Seed %v", seed)
		t.Error(err)
	}
}

func TestListSystemOrder(t *testing.T) {
	if testing.Short() {
		// Ironically, 'quick' test takes 1.75s!
		t.Skipf("Skip ListSystem Order test for 'short' tests")
	}

	const schema = `
	list testlist {
		ordered-by system;
		key name;
		leaf name {
			type string;
		}
	}
`
	srv, sess := TstStartup(t, schema, emptyconfig)
	defer sess.Kill()
	check := func(paths []listpath) bool {
		defer sess.Discard(srv.Ctx)
		cfgPaths := make([]string, len(paths))
		for i, p := range paths {
			if err := sess.Set(srv.Ctx, p); err != nil {
				t.Fatal(err)
			}
			cfgPaths[i] = p.String()
		}
		cfg, err := sess.Show(srv.Ctx, emptypath, true, false)
		if err != nil {
			t.Fatal(err)
		}
		natsort.Sort(cfgPaths)
		var exp bytes.Buffer
		for _, p := range cfgPaths {
			exp.WriteString(p)
			exp.WriteByte('\n')
		}
		return cfg == exp.String()
	}

	seed := time.Now().UnixNano()
	qcfg := quick.Config{
		Rand: rand.New(rand.NewSource(seed)),
	}
	if err := quick.Check(check, &qcfg); err != nil {
		t.Logf("Seed %v", seed)
		t.Error(err)
	}
}

func TestGuessSecrets_VRVDR3934(t *testing.T) {
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		leaf secret {
			type string;
			configd:secret true;
		}
	}
}
`
	const config = `testcontainer {
	testlist foo {
		secret bar
	}
}
`
	const expconfig = `testcontainer {
	testlist foo {
		secret "********"
	}
}
`
	var testlistpath_foo = pathutil.CopyAppend(testlistpath, "foo")
	var secretpath = pathutil.CopyAppend(testlistpath_foo, "secret")
	var secretpath_bar = pathutil.CopyAppend(secretpath, "bar")
	var secretpath_baz = pathutil.CopyAppend(secretpath, "baz")

	srv, sess := TstStartup(t, schema, config)
	altctx := srv.Ctx
	altctx.Configd = false
	ValidateSet(t, sess, altctx, secretpath_bar, false)
	ValidateCommit(t, sess, altctx, false, emptyconfig)
	ValidateSet(t, sess, altctx, secretpath_baz, false)
	ValidateCommit(t, sess, altctx, true, expconfig)

	ValidateSet(t, sess, srv.Ctx, secretpath_baz, false)
	ValidateCommit(t, sess, srv.Ctx, false, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, secretpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig)
	sess.Kill()
}

// configd scripts can be invoked in one of 2 ways - via execfn, or via
// execCmd.  Environment variables for each are now sourced from the same
// location; previously they were sourced differently.
//
// This test ensures they are accessible via execfn() calls.  If I could
// work out how to get a configd:syntax call to echo the environment variables
// (well, simply to run w/o failing in a UT environment) then I'd add a test
// for that as well.  Suggestions on a postcard to the author ...
func TestConfigdExecFnEnvVars(t *testing.T) {
	t.Log("Verify env vars are set correctly for scripts using execfn().")

	const schema = `
container testcontainer {
	leaf testleaf {
		type string;
		configd:begin "env";
	}
}
`
	const expconfig = `testcontainer {
	testleaf foo
}
`
	const expout = `[]

[testcontainer testleaf foo]
vyatta_htmldir=/opt/vyatta/share/html
vyatta_datadir=/opt/vyatta/share
vyatta_op_templates=/opt/vyatta/share/vyatta-op/templates
vyatta_sysconfdir=/opt/vyatta/etc
vyatta_sharedstatedir=/opt/vyatta/com
vyatta_sbindir=/opt/vyatta/sbin
vyatta_cfg_templates=/opt/vyatta/share/vyatta-cfg/templates
vyatta_bindir=/opt/vyatta/bin
vyatta_libdir=/opt/vyatta/lib
vyatta_localstatedir=/opt/vyatta/var
vyatta_libexecdir=/opt/vyatta/libexec
vyatta_prefix=/opt/vyatta
vyatta_datarootdir=/opt/vyatta/share
vyatta_configdir=/opt/vyatta/config
vyatta_infodir=/opt/vyatta/share/info
vyatta_localedir=/opt/vyatta/share/locale
PATH=/usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin:/opt/vyatta/bin:/opt/vyatta/bin/sudo-users:/opt/vyatta/sbin
PERL5LIB=/opt/vyatta/share/perl5
VYATTA_CONFIG_SID=TEST
COMMIT_ACTION=SET
CONFIGD_PATH=/testcontainer/testleaf/foo
CONFIGD_EXT=begin

[]

`
	var testleafpath_foo = pathutil.CopyAppend(testleafpath, "foo")

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testleafpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

// TODO: test authorization access in APIs
// TODO: test order of action execution
//        - create commit dry-run that returns called scripts
//        - or action scripts that just echo a string that we can compare with expected output
// TODO: test node priority

// TODO
// func TestComment(t *testing.T) {
