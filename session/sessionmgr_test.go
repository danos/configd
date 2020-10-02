// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"testing"

	"github.com/danos/configd"
	"github.com/danos/configd/session"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror"
)

const (
	testSessName = "1234"
)

func newTestSession(
	t *testing.T, srv *sessiontest.TstSrv, sid string,
) *session.Session {
	sess, err := srv.Smgr.Create(srv.Ctx, sid, srv.Cmgr, srv.Ms, srv.MsFull)
	if sess == nil || err != nil {
		t.Fatalf("Unexpected nil session, err: %v", err)
	}
	return sess
}

func TestSessionMgrGetNonExistent(t *testing.T) {
	srv, _ := sessiontest.NewTestSpec(t).Init()

	sess, err := srv.Smgr.Get(srv.Ctx, testSessName)
	if sess != nil {
		t.Fatalf("Unexpectedly retrieved session: %v", sess)
	}

	expErr := mgmterror.NewOperationFailedApplicationError()
	expErr.Message = "session " + testSessName + " does not exist"
	if err == nil || err.Error() != expErr.Error() {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func sameCtx(ref *configd.Context) *configd.Context {
	new := *ref
	return &new
}

// Make a copy of the reference Context, adjusting it so
// it represents that of the (implicitly auth'd) "configd" user.
func configdCtx(ref *configd.Context) *configd.Context {
	new := *ref
	new.Configd = true
	new.Superuser = false
	new.Uid += 1
	new.User = "configd"
	return &new
}

// Make a copy of the reference Context, adjusting it so
// it represents that of a user of superuser level.
func superuserCtx(ref *configd.Context) *configd.Context {
	new := *ref
	new.Configd = false
	new.Superuser = true
	new.Uid += 1
	new.User += "SUPER"
	return &new
}

// Make a copy of the reference Context, adjusting it so
// it represents that of a regular user.
func regularCtx(ref *configd.Context) *configd.Context {
	new := *ref
	new.Configd = false
	new.Superuser = false
	new.Uid += 1
	new.User += "REG"
	return &new
}

type sessionMgrPermTestCase struct {
	ctxSwitcher func(*configd.Context) *configd.Context
	expSess     bool
	expErr      error
}

func runSessionMgrPermTestCases(
	t *testing.T,
	refCtx *configd.Context,
	testCases []sessionMgrPermTestCase,
	expSess *session.Session,
	mgrOp func(*configd.Context) (*session.Session, error)) {

	for i, tc := range testCases {
		tcErrorf := func(format string, args ...interface{}) {
			t.Errorf("Test case %d: "+format,
				append([]interface{}{i}, args...)...)
		}

		if tc.expSess && tc.expErr != nil {
			tcErrorf("Expected a session and an error. Invalid test case?")
			break
		}

		actSess, err := mgrOp(tc.ctxSwitcher(refCtx))

		if tc.expSess && actSess != expSess {
			tcErrorf("Unexpected session (%p != %p)", actSess, expSess)
		} else if !tc.expSess && actSess != nil {
			tcErrorf("Unexpectedly got session %p", actSess)
		}

		if tc.expErr == nil && err == nil {
			continue
		} else if tc.expErr == nil && err != nil {
			tcErrorf("Unexpectedly got err: %v", err)
		} else if err == nil && tc.expErr != nil {
			tcErrorf("Unexpectedly got no error")
		} else if tc.expErr.Error() != err.Error() {
			tcErrorf("Got err: %v; expected: %v", err, tc.expErr)
		}
	}
}

var existingSessTcs = []sessionMgrPermTestCase{
	{sameCtx, true, nil},
	{configdCtx, true, nil},
	{superuserCtx, true, nil},
	{regularCtx, true, nil},
}

func TestSessionMgrCreateExisting(t *testing.T) {
	srv, _ := sessiontest.NewTestSpec(t).Init()
	expSess := newTestSession(t, srv, testSessName)
	defer srv.Smgr.Destroy(srv.Ctx, testSessName)

	runSessionMgrPermTestCases(t, srv.Ctx, existingSessTcs, expSess,
		func(ctx *configd.Context) (*session.Session, error) {
			return srv.Smgr.Create(ctx, testSessName,
				srv.Cmgr, srv.Ms, srv.MsFull)
		})
}

func TestSessionMgrGet(t *testing.T) {
	srv, _ := sessiontest.NewTestSpec(t).Init()
	expSess := newTestSession(t, srv, testSessName)
	defer srv.Smgr.Destroy(srv.Ctx, testSessName)

	runSessionMgrPermTestCases(t, srv.Ctx, existingSessTcs, expSess,
		func(ctx *configd.Context) (*session.Session, error) {
			return srv.Smgr.Get(ctx, testSessName)
		})
}

// Little bit of a hack... Destroy() never returns a
// Session reference, so no test case ever expects one.
var destroySessTcs = []sessionMgrPermTestCase{
	{sameCtx, false, nil},
	{configdCtx, false, nil},
	{superuserCtx, false, nil},
	{regularCtx, false, nil},
}

func TestSessionMgrDestroy(t *testing.T) {
	srv, _ := sessiontest.NewTestSpec(t).Init()

	runSessionMgrPermTestCases(t, srv.Ctx, destroySessTcs, nil,
		func(ctx *configd.Context) (*session.Session, error) {
			_ = newTestSession(t, srv, testSessName)
			defer srv.Smgr.Destroy(srv.Ctx, testSessName) // test cleanup

			// This is the actual test. The deferred call is just
			// cleanup in case of an (expected) failure here.
			err := srv.Smgr.Destroy(ctx, testSessName)
			return nil, err
		})
}
