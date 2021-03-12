// Copyright (c) 2018-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-17 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"strings"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/configd"
	"github.com/danos/configd/session"
	. "github.com/danos/configd/session/sessiontest"
)

const loadTestSchema = `
leaf testhidden {
	type boolean;
}
leaf testavailable {
	type boolean;
}`

const loadTestConfig = `
testhidden true
testavailable true
`

var limitedAuth = auth.NewTestAuther(
	auth.NewTestRule(auth.Deny, auth.AllOps, "/testhidden"),
	auth.NewTestRule(auth.Allow, auth.AllOps, "*"))

var fullAuth = auth.TestAutherAllowAll()

func assertValue(t *testing.T, sess *session.Session, ctx *configd.Context, pstr, value string) {
	path := strings.Split(pstr, "/")

	val, err := sess.Get(ctx, path)
	if err != nil {
		t.Errorf("Unable to get path [%s] : %s", path, err)
	} else if len(val) == 0 {
		t.Errorf("Value is missing for path [%s]", path)
	} else if val[0] != value {
		t.Errorf("Unexpected result from path [%s]\n"+
			"    expect: %s"+
			"    actual: %s",
			path, value, val[0])
	}
}

func TestLoadWithAuth(t *testing.T) {

	srv, sess := TstStartupWithCustomAuth(
		t, loadTestSchema, loadTestConfig, limitedAuth, false, true)

	// Set up partial and full authorisation. Nil gives full access
	limitedCtx := *srv.Ctx
	fullCtx := *srv.Ctx
	fullCtx.Auth = fullAuth

	// Check both values are as expected before loading the partial config
	assertValue(t, sess, &fullCtx, "testhidden", "true")
	assertValue(t, sess, &fullCtx, "testavailable", "true")

	sess.Load(&limitedCtx, "testdata/load_test/TestLoadWithAuth.config", nil)

	// Check missing hidden value is retained
	assertValue(t, sess, &fullCtx, "testhidden", "true")

	// Check loaded value has changed
	assertValue(t, sess, &fullCtx, "testavailable", "false")
}

func TestCopyConfigWithAuth(t *testing.T) {

	srv, sess := TstStartupWithCustomAuth(
		t, loadTestSchema, loadTestConfig, limitedAuth, false, true)

	// Set up partial and full authorisation. Nil gives full access
	limitedCtx := *srv.Ctx
	fullCtx := *srv.Ctx
	fullCtx.Auth = fullAuth

	// Check both values are as expected before loading the partial config
	assertValue(t, sess, &fullCtx, "testhidden", "true")
	assertValue(t, sess, &fullCtx, "testavailable", "true")

	newConfig := `
	<config>
		<testavailable>false</testavailable>
	</config>`

	err := sess.CopyConfig(
		&limitedCtx,
		"",          // sourceDatastore
		"xml",       // sourceEncoding
		newConfig,   // sourceConfig
		"",          // sourceURL
		"candidate", // targetDatastore
		"")          // targetURL
	if err != nil {
		t.Fatalf("Error copying config: %s", err)
	}

	// Check missing hidden value is retained
	assertValue(t, sess, &fullCtx, "testhidden", "true")

	// Check loaded value has changed
	assertValue(t, sess, &fullCtx, "testavailable", "false")
}
