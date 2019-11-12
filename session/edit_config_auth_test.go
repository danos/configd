// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"fmt"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils"
	. "github.com/danos/configd/session"
	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
)

type editConfigCmdAuthTestCmd struct {
	cmd   []string
	attrs *pathutil.PathAttrs
}

type editConfigCmdAuthTest struct {
	desc       string
	auther     auth.TestAuther
	initConfig string
	operation  string
	path       []string
	expCmds    []editConfigCmdAuthTestCmd
	expAuthd   bool
}

const ospfAreaZeroConfig = `protocols {
	ospf {
		area 0
	}
}
`

var editConfigCmdAuthTests = []editConfigCmdAuthTest{
	{
		desc:       "Check path attributes are generated correctly",
		initConfig: emptyConfig,
		operation:  op_merge,
		path:       []string{"protocols", "ospf", "parameters", "password", testutils.POISON_SECRETS[0]},
		expCmds: []editConfigCmdAuthTestCmd{
			{
				[]string{"set", "protocols", "ospf", "parameters", "password", testutils.POISON_SECRETS[0]},
				&pathutil.PathAttrs{Attrs: []pathutil.PathElementAttrs{
					pathutil.PathElementAttrs{Secret: false},
					pathutil.PathElementAttrs{Secret: false},
					pathutil.PathElementAttrs{Secret: false},
					pathutil.PathElementAttrs{Secret: false},
					pathutil.PathElementAttrs{Secret: false},
					pathutil.PathElementAttrs{Secret: true}},
				},
			},
		},
		expAuthd: true,
	},
	{
		desc:       "Check merge with initial empty config",
		initConfig: emptyConfig,
		operation:  op_merge,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check merge with existing config",
		initConfig: ospfAreaZeroConfig,
		operation:  op_merge,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc: "Check merge with existing config with an auther which denies updates",
		auther: auth.NewTestAuther(auth.NewTestRule(
			auth.Allow, auth.P_READ|auth.P_CREATE, "/protocols")),
		initConfig: ospfAreaZeroConfig,
		operation:  op_merge,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: false,
	},
	{
		desc: "Check merge with initial empty config with an auther which denies creates",
		auther: auth.NewTestAuther(auth.NewTestRule(
			auth.Allow, auth.P_READ|auth.P_UPDATE, "/protocols")),
		initConfig: emptyConfig,
		operation:  op_merge,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: false,
	},
	{
		desc:       "Check replace with empty initial config",
		initConfig: emptyConfig,
		operation:  op_replace,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check replace with existing config",
		initConfig: ospfAreaZeroConfig,
		operation:  op_replace,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"delete", "protocols", "ospf", "area", "0"}},
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc: "Check replace with existing config, the delete of which is not authorized",
		auther: auth.NewTestAuther(auth.NewTestRule(
			auth.Allow, auth.P_READ|auth.P_CREATE|auth.P_UPDATE, "/protocols")),
		initConfig: ospfAreaZeroConfig,
		operation:  op_replace,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"delete", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: false,
	},
	{
		desc:       "Check create with empty initial config",
		initConfig: emptyConfig,
		operation:  op_create,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check create with existing config",
		initConfig: ospfAreaZeroConfig,
		operation:  op_create,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"set", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check delete with empty initial config",
		initConfig: emptyConfig,
		operation:  op_delete,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"delete", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check delete with existing config",
		initConfig: ospfAreaZeroConfig,
		operation:  op_delete,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"delete", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check remove with empty initial config",
		initConfig: emptyConfig,
		operation:  op_remove,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"delete", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check remove with existing config",
		initConfig: ospfAreaZeroConfig,
		operation:  op_remove,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds: []editConfigCmdAuthTestCmd{
			{cmd: []string{"delete", "protocols", "ospf", "area", "0"}},
		},
		expAuthd: true,
	},
	{
		desc:       "Check notset operation is an auth no-op with empty initial config",
		initConfig: emptyConfig,
		operation:  op_notset,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds:    []editConfigCmdAuthTestCmd{},
		expAuthd:   true,
	},
	{
		desc:       "Check notset operation is an auth no-op with existing config",
		initConfig: ospfAreaZeroConfig,
		operation:  op_notset,
		path:       []string{"protocols", "ospf", "area", "0"},
		expCmds:    []editConfigCmdAuthTestCmd{},
		expAuthd:   true,
	},
}

func TestEditConfigAuth(t *testing.T) {
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
</config>
`
	for i, tc := range editConfigCmdAuthTests {
		tcDesc := fmt.Sprintf("Test case \"%v\" (%d):\n", tc.desc, i)

		a := tc.auther
		if a == nil {
			a = auth.TestAutherAllowAll()
		}

		srv, sess := TstStartupMultipleSchemasWithCustomAuth(
			t, edit_config_schema, tc.initConfig, a, false, false)

		ec, err := NewTestEditConfig(sess, srv.Ctx, target_candidate,
			defop_none, testopt_testset, erropt_stop, edit_config)
		if err != nil {
			t.Fatalf(tcDesc+"%v", err)
		}

		op, err := NewTestEditOp(tc.path, tc.operation)
		if err != nil {
			t.Fatal(err)
		}

		res := op.Auth(*ec)
		if res != tc.expAuthd {
			t.Fatalf(tcDesc+"Auth() result was %v but expected %v",
				res, tc.expAuthd)
		}

		expReqs := auth.NewTestAutherRequests()

		for _, cmd := range tc.expCmds {
			attrs := cmd.attrs

			// If no path attributes are specified by the test case
			// then generate some, assuming all elements are non-secret
			if attrs == nil {
				ret := pathutil.NewPathAttrs()
				attrs = &ret
				for _, _ = range cmd.cmd {
					attrs.Attrs = append(attrs.Attrs,
						pathutil.PathElementAttrs{Secret: false})
				}
			} else if len(attrs.Attrs) != len(cmd.cmd) {
				t.Fatalf(tcDesc+"test case expected command and attribute "+
					"length mismatch: %d != %d", len(attrs.Attrs), len(cmd.cmd))
			}

			// Build expected auth/acct request list
			expReqs.Reqs = append(expReqs.Reqs,
				auth.NewTestAutherCommandRequest(cmd.cmd, attrs))
		}

		// Verify expected command authorization and accounting were seen
		err = auth.CheckRequests(a.GetCmdRequests(), expReqs)
		if err != nil {
			t.Fatalf(tcDesc+"%v", err)
		}

		err = auth.CheckRequests(a.GetCmdAcctRequests(), expReqs)
		if err != nil {
			t.Fatalf(tcDesc+"%v", err)
		}

		a.ClearCmdRequests()
		a.ClearCmdAcctRequests()
		sess.Kill()
	}
}
