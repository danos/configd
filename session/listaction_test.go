// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
)

var testlistpath_foo = pathutil.CopyAppend(testlistpath, "foo")
var testlistpath_bar = pathutil.CopyAppend(testlistpath, "bar")

//////////////////////////////////////////////////////////////////////
// Create tests

func TestListActionsCreateSystem(t *testing.T) {
	t.Log("Verify create actions for system ordered list")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expout = `[]

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
create

[testcontainer testlist bar]
end

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
create

[testcontainer testlist foo]
end

[]

`

	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsCreateSystemKey(t *testing.T) {
	t.Log("Verify create actions for system ordered list key node")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by system;
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expout = `[]

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
create

[testcontainer testlist bar name bar]
end

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
create

[testcontainer testlist foo name foo]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsCreateSystemPriority(t *testing.T) {
	t.Log("Verify create action for system ordered list with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by system;
		configd:priority "400";
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expout = `[]

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
create

[testcontainer testlist bar]
end

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
create

[testcontainer testlist foo]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsCreateSystemKeyPriority(t *testing.T) {
	t.Log("Verify create actions for system ordered list key node with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by system;
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expout = `[]

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
create

[testcontainer testlist bar name bar]
end

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
create

[testcontainer testlist foo name foo]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsCreateUser(t *testing.T) {
	t.Log("Verify create actions for user ordered list")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist baz
	testlist foo
	testlist bar
}
`
	const expout = `[]

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
end

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
end

[]

`

	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsCreateUserKey(t *testing.T) {
	t.Skip("List does not support descendant actions for 'ordered-by user'")
	t.Log("Verify create actions for user ordered list key node")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by user;
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist baz
	testlist foo
	testlist bar
}
`
	const expout = `[]

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
create

[testcontainer testlist foo name foo]
end

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
create

[testcontainer testlist bar name bar]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsCreateUserPriority(t *testing.T) {
	t.Log("Verify create action for user ordered list with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by user;
		configd:priority "400";
		configd:begin "echo begin";
		configd:end "echo end";
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist baz
	testlist foo
	testlist bar
}
`
	const expout = `[]

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
end

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsCreateUserKeyPriority(t *testing.T) {
	t.Skip("List does not support descendant actions for 'ordered-by user'")
	t.Log("Verify create actions for user ordered list key node with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by user;
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testlist baz
}
`
	const expconfig = `testcontainer {
	testlist baz
	testlist foo
	testlist bar
}
`
	const expout = `[]

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
create

[testcontainer testlist foo name foo]
end

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
create

[testcontainer testlist bar name bar]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateSet(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateSet(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

//////////////////////////////////////////////////////////////////////
// Delete tests

func TestListActionsDeleteSystem(t *testing.T) {
	t.Log("Verify delete actions for system ordered list")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by system;
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
delete

[testcontainer testlist bar]
end

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
delete

[testcontainer testlist foo]
end

[]

`

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsDeleteSystemKey(t *testing.T) {
	t.Log("Verify delete actions for system ordered list key node")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by system;
	}
}
`
	const config = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
delete

[testcontainer testlist bar name bar]
end

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
delete

[testcontainer testlist foo name foo]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsDeleteSystemPriority(t *testing.T) {
	t.Log("Verify delete action for system ordered list with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by system;
		configd:priority "400";
		configd:begin "echo begin";
		configd:end "echo end";
		configd:create "echo create";
		configd:update "echo update";
		configd:delete "echo delete";
	}
}
`
	const config = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
delete

[testcontainer testlist bar]
end

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
delete

[testcontainer testlist foo]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsDeleteSystemKeyPriority(t *testing.T) {
	t.Log("Verify delete actions for system ordered list key node with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by system;
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testlist bar
	testlist baz
	testlist foo
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
delete

[testcontainer testlist bar name bar]
end

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
delete

[testcontainer testlist foo name foo]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsDeleteUser(t *testing.T) {
	t.Skip("List does not support 'ordered-by user'")
	t.Log("Verify delete actions for user ordered list")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by user;
		configd:begin "echo begin";
		configd:end "echo end";
	}
}
`
	const config = `testcontainer {
	testlist foo
	testlist baz
	testlist bar
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
end

[testcontainer testlist baz]
begin

[testcontainer testlist baz]
end

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
end

[testcontainer testlist baz]
begin

[testcontainer testlist baz]
end

[]

`

	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsDeleteUserKey(t *testing.T) {
	t.Skip("List does not support descendant actions for 'ordered-by user'")
	t.Log("Verify delete actions for user ordered list key node")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by user;
	}
}
`
	const config = `testcontainer {
	testlist foo
	testlist baz
	testlist bar
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
delete

[testcontainer testlist foo name foo]
end

[testcontainer testlist baz name baz]
begin

[testcontainer testlist baz name baz]
delete

[testcontainer testlist baz name baz]
end

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
delete

[testcontainer testlist bar name bar]
end

[testcontainer testlist baz name baz]
begin

[testcontainer testlist baz name baz]
create

[testcontainer testlist baz name baz]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsDeleteUserPriority(t *testing.T) {
	t.Skip("List does not support 'ordered-by user'")
	t.Log("Verify delete action for user ordered list with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
		}
		ordered-by user;
		configd:priority "400";
		configd:begin "echo begin";
		configd:end "echo end";
	}
}
`
	const config = `testcontainer {
	testlist foo
	testlist baz
	testlist bar
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist foo]
begin

[testcontainer testlist foo]
end

[testcontainer testlist bar]
begin

[testcontainer testlist bar]
end

[testcontainer testlist baz]
begin

[testcontainer testlist baz]
end

[testcontainer testlist baz]
begin

[testcontainer testlist baz]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

func TestListActionsDeleteUserKeyPriority(t *testing.T) {
	t.Skip("List does not support descendant actions for 'ordered-by user'")
	t.Log("Verify delete actions for user ordered list key node with priority")
	const schema = `
container testcontainer {
	list testlist {
		key name;
		leaf name {
			type string;
			configd:begin "echo begin";
			configd:end "echo end";
			configd:create "echo create";
			configd:update "echo update";
			configd:delete "echo delete";
		}
		ordered-by user;
		configd:priority "400";
	}
}
`
	const config = `testcontainer {
	testlist foo
	testlist baz
	testlist bar
}
`
	const expconfig = `testcontainer {
	testlist baz
}
`
	const expout = `[]

[testcontainer testlist foo name foo]
begin

[testcontainer testlist foo name foo]
delete

[testcontainer testlist foo name foo]
end

[testcontainer testlist bar name bar]
begin

[testcontainer testlist bar name bar]
delete

[testcontainer testlist bar name bar]
end

[testcontainer testlist baz name baz]
begin

[testcontainer testlist baz name baz]
delete

[testcontainer testlist baz name baz]
end

[testcontainer testlist baz name baz]
begin

[testcontainer testlist baz name baz]
create

[testcontainer testlist baz name baz]
end

[]

`
	srv, sess := TstStartup(t, schema, config)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_bar, false)
	ValidateDelete(t, sess, srv.Ctx, testlistpath_foo, false)
	ValidateCommit(t, sess, srv.Ctx, true, expconfig, expout)
	sess.Kill()
}

//////////////////////////////////////////////////////////////////////
// Update tests

// TODO
