// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"fmt"
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
)

const blankNodeStr = "[]\n\n"

func topAndTailExpOut(expOutSections ...string) string {
	retStr := blankNodeStr
	for _, section := range expOutSections {
		retStr = retStr + section
	}
	return retStr + blankNodeStr
}

func constructExpOutSection(node string, actions ...string) string {
	var retStr string
	for _, action := range actions {
		retStr = retStr + fmt.Sprintf("[%s]\n%s\n\n", node, action)
	}
	return retStr
}

func TestActionsLeafListUserCreate(t *testing.T) {
	t.Log("Verify create actions for user ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`

	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "create", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListUserCreatePriority(t *testing.T) {
	t.Log("Verify create action for user ordered leaf-list with priority")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
		configd:priority "400";
	}
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "create", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListUserUpdate(t *testing.T) {
	t.Log("Verify use of update action if create action does not exist in user ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "update", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListUserDeleteFirst(t *testing.T) {
	t.Log("Verify delete action for first element of user ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser bar
}
`
	expout := topAndTailExpOut(
		constructExpOutSection("testcontainer testleaflistuser foo",
			"begin", "delete", "end"),
		constructExpOutSection("testcontainer testleaflistuser bar",
			"begin", "delete", "end"),
		constructExpOutSection("testcontainer testleaflistuser bar",
			"begin", "create", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListUserDeleteFirstPriority(t *testing.T) {
	t.Log("Verify delete action for first element of user ordered leaf-list with priority")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser bar
}
`
	expout := topAndTailExpOut(
		constructExpOutSection("testcontainer testleaflistuser foo",
			"begin", "delete", "end"),
		constructExpOutSection("testcontainer testleaflistuser bar",
			"begin", "delete", "end"),
		constructExpOutSection("testcontainer testleaflistuser bar",
			"begin", "create", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListUserDeleteSecond(t *testing.T) {
	t.Log("Verify delete action for non-first element of user ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser bar",
		"begin", "delete", "end"))

	var testlistleafuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListUserDeleteSecondPriority(t *testing.T) {
	t.Log("Verify delete action for non-first element of user ordered leaf-list with priority")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser bar",
		"begin", "delete", "end"))

	var testlistleafuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListSystemCreate(t *testing.T) {
	t.Log("Verify create actions for system ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "create", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListSystemCreatePriority(t *testing.T) {
	t.Log("Verify create action for system ordered leaf-list with priority")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
		configd:priority "400";
	}
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "create", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListSystemUpdate(t *testing.T) {
	t.Log("Verify use of update action if create action does not exist for system ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "update", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSet(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListSystemDeleteFirst(t *testing.T) {
	t.Log("Verify delete action for first element of system ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser bar
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "delete", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListSystemDeleteFirstPriority(t *testing.T) {
	t.Log("Verify delete action for first element of system ordered leaf-list with priority")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser bar
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser foo",
		"begin", "delete", "end"))

	var testlistleafuserpath_foo = pathutil.CopyAppend(testleaflistuserpath, "foo")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListSystemDeleteSecond(t *testing.T) {
	t.Log("Verify delete action for non-first element of system ordered leaf-list")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser bar",
		"begin", "delete", "end"))

	var testlistleafuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestActionsLeafListSystemDeleteSecondPriority(t *testing.T) {
	t.Log("Verify delete action for non-first element of system ordered leaf-list with priority")
	const schema = `
container testcontainer {
	leaf-list testleaflistuser {
		type string;
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:delete "echo delete";
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testleaflistuser foo
	testleaflistuser bar
}
`
	const expconfig = `testcontainer {
	testleaflistuser foo
}
`
	expout := topAndTailExpOut(constructExpOutSection(
		"testcontainer testleaflistuser bar",
		"begin", "delete", "end"))

	var testlistleafuserpath_bar = pathutil.CopyAppend(testleaflistuserpath, "bar")

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistleafuserpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}
