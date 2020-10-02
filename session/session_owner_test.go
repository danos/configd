// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"testing"

	. "github.com/danos/configd/session"
	. "github.com/danos/configd/session/sessiontest"
)

func TestSessionNoOwner(t *testing.T) {
	srv, _ := NewTestSpec(t).Init()
	sess := NewSession("test", srv.Cmgr, srv.Ms, srv.MsFull)
	defer sess.Kill()

	if !sess.IsShared() {
		t.Fatalf("New session unexpectedly marked as un-shared")
	}
	if sess.OwnedBy(srv.Ctx.Uid) {
		t.Fatalf("New session unexpectedly owned by creator")
	}
}

func TestSessionOwner(t *testing.T) {
	srv, _ := NewTestSpec(t).Init()
	sess := NewSession(
		"test", srv.Cmgr, srv.Ms, srv.MsFull, WithOwner(srv.Ctx.Uid))
	defer sess.Kill()

	if sess.IsShared() {
		t.Fatalf("Owned session unexpectedly shared")
	}
	if !sess.OwnedBy(srv.Ctx.Uid) {
		t.Fatalf("Session not reported as owned by owner!")
	}
}

func TestSessionOtherOwner(t *testing.T) {
	srv, _ := NewTestSpec(t).Init()
	sess := NewSession(
		"test", srv.Cmgr, srv.Ms, srv.MsFull, WithOwner(srv.Ctx.Uid+10))
	defer sess.Kill()

	if sess.IsShared() {
		t.Fatalf("Owned session unexpectedly shared")
	}
	if !sess.OwnedBy(srv.Ctx.Uid + 10) {
		t.Fatalf("Session not reported as owned by owner!")
	}
	if sess.OwnedBy(srv.Ctx.Uid) {
		t.Fatalf("Session unexpectedly reported as owned by creator")
	}
}
