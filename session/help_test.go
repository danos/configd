// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// Tests for help text.
package session_test

import (
	"testing"

	"github.com/danos/configd"
	. "github.com/danos/configd/session"
	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
	"reflect"
)

type validateHelpTextTbl struct {
	path   []string
	schema bool
	exp    map[string]string
}

func validateHelpText(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	tst validateHelpTextTbl,
) {
	m, err := sess.GetHelp(ctx, tst.schema, tst.path)
	if err != nil {
		t.Errorf("Unable to get help for path [%s]", pathutil.Pathstr(tst.path))
	}
	if !reflect.DeepEqual(m, tst.exp) {
		t.Errorf("Unexpected help text from path [%s]",
			pathutil.Pathstr(tst.path))
		t.Logf("Received: %s", m)
		t.Logf("Expected: %s", tst.exp)
	}
}

func validateHelpTextTable(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	tbl []validateHelpTextTbl,
) {
	for key, _ := range tbl {
		validateHelpText(t, sess, ctx, tbl[key])
	}
}

// TODO: test help with from_schema set to false
func TestGetHelp(t *testing.T) {
	const schema = `
container testcontainer {
	configd:help "testcontainerhelp";
	leaf testempty {
		type empty;
		configd:help "testemptyhelp";
	}
	leaf testDeprecated {
		type string;
		status deprecated;
		configd:help "testDeprecatedHelp";
	}
	leaf testObsolete {
		type string;
		status obsolete;
		configd:help "testObsoleteHelp";
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		configd:help "testleaflisthelp";
	}
}
`
	const config = `
testcontainer {
	testempty
}
`
	const testcontainerhelp = "testcontainerhelp"
	const testemptyhelp = "testemptyhelp"
	const testDeprecatedhelp = "testDeprecatedHelp [Deprecated]"
	const testleaflisthelp = "testleaflisthelp"
	testleaflistvaluepath := pathutil.CopyAppend(testleaflistuserpath, "foo")
	emptyhelptext := map[string]string{}
	emptypathhelptext := map[string]string{testcontainer: testcontainerhelp}
	leaflistuserhelptext := map[string]string{"<text>": testleaflisthelp}
	invalidpathhelptext := map[string]string{}
	enterhelptext := map[string]string{"<Enter>": "Execute the current command"}
	testcontainerhelpschematext := map[string]string{
		testempty:          testemptyhelp,
		"testDeprecated":   testDeprecatedhelp,
		"testleaflistuser": testleaflisthelp,
	}
	testcontainerhelptext := map[string]string{testempty: testemptyhelp}
	tbl := []validateHelpTextTbl{
		{emptypath, true, emptypathhelptext},
		{invalidpath, true, invalidpathhelptext},
		{rootpath, true, emptyhelptext},
		{testcontainerpath, true, testcontainerhelpschematext},
		{testemptypath, true, enterhelptext},
		{testleaflistuserpath, true, leaflistuserhelptext},
		{testleaflistvaluepath, true, enterhelptext},
		{testcontainerpath, false, testcontainerhelptext},
	}
	srv, sess := TstStartup(t, schema, config)
	validateHelpTextTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

// Verify that help text for a 'configd:secret "true";' field only shows
// the generic schema help text.  Check that other help text does show
// full set of currently configured options as well.
func TestGetSecretHelp(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testSecret {
		type string;
		configd:help "testSecret help";
		configd:secret "true";
	}
	leaf testOpen {
		type string;
		configd:help "testOpen help";
	}
}
`
	const config = `
testcontainer {
	testSecret topsecret;
	testOpen openPassword;

}
`
	testSecret := []string{"testcontainer", "testSecret"}
	testOpen := []string{"testcontainer", "testOpen"}

	const testSecretHelp = "testSecret help"
	const testOpenHelp = "testOpen help"

	testSecretHelpText := map[string]string{
		"<text>": testSecretHelp}
	testOpenHelpText := map[string]string{
		"<text>":       testOpenHelp,
		"openPassword": testOpenHelp,
	}

	tbl := []validateHelpTextTbl{
		{testSecret, true, testSecretHelpText},
		{testOpen, true, testOpenHelpText},
	}

	// CustomAuth: not configd user
	srv, sess := TstStartupWithCustomAuth(t, schema, config,
		nil, false, false)

	validateHelpTextTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}
