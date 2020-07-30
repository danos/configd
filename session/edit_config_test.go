// Copyright (c) 2019-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"testing"

	"github.com/danos/config/testutils"
	"github.com/danos/configd"
	. "github.com/danos/configd/session"
	. "github.com/danos/configd/session/sessiontest"
)

const (
	target_candidate = "candidate"
	target_running   = "running"
)

const (
	defop_merge   = "merge"
	defop_replace = "replace"
	defop_none    = "none"
)

const (
	testopt_testset  = "test-then-set"
	testopt_set      = "set"
	testopt_testonly = "test-only"
)

const (
	erropt_stop     = "stop-on-error"
	erropt_cont     = "continue-on-error"
	erropt_rollback = "rollback-on-error"
)

const (
	op_merge   = "merge"
	op_replace = "replace"
	op_create  = "create"
	op_delete  = "delete"
	op_remove  = "remove"
	op_notset  = "" // Special case - see NewTestEditOp()
)

const schemaProtocols = `
container protocols {
}
`
const schemaOspf = `
augment "/protocols:protocols" {
	container ospf {
		presence "true";
	        list area {
			key "tagnode";
			leaf tagnode {
				type string;
			}
			leaf-list network {
				ordered-by "user";
				type string;
			}
		}
		container parameters {
			presence "true";
			leaf opaque-lsa {
				type empty;
			}
			leaf abr-type {
				type enumeration {
					enum "cisco";
					enum "ibm";
					enum "shortcut";
					enum "standard";
				}
				default "cisco";
			}
			leaf router-id {
				type string {
					configd:normalize "echo FOO";
				}
			}
			leaf password {
				configd:secret true;
				type string;
			}
		}
	}
}
`

var edit_config_schema = []TestSchema{
	{
		Name:          NameDef{"vyatta-protocols", "vyatta-protocols"},
		Prefix:        "vyatta-protocols",
		SchemaSnippet: schemaProtocols,
	},
	{
		Name:          NameDef{"vyatta-protocols-ospf", "vyatta-protocols-ospf"},
		Prefix:        "vyatta-protocols-ospf",
		Imports:       []NameDef{{"vyatta-protocols", "protocols"}},
		SchemaSnippet: schemaOspf,
	},
}

func validateEditConfig(t *testing.T, experr bool, sess *Session, ctx *configd.Context,
	config_target, default_operation, test_option, error_option, config string,
) {
	err := sess.EditConfigXML(ctx, config_target, default_operation, test_option, error_option, config)
	if (err != nil) != experr {
		t.Log("validateEditConfig")
		if err == nil {
			t.Error("Unexpected edit-config success")
		} else {
			t.Error("Unexpected edit-config failure")
		}
		t.Fatal(err)
		testutils.LogStack(t)
	}
}

func TestEditConfigBadNamespace(t *testing.T) {
	// A bad namespace causes the XML sub-tree (ospf) to be ignored
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols" xc:operation="create">
    <area>
      <tagnode>0</tagnode>
      <network>10.1.1.0/24</network>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, emptyconfig, true)
}

func TestEditConfigMissingNamespace(t *testing.T) {
	// A missing namespace causes the XML sub-tree (ospf) to be ignored
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xc:operation="create">
    <area>
      <tagnode>0</tagnode>
      <network>10.1.1.0/24</network>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, emptyconfig, true)
}

func TestEditConfigCreateContainer(t *testing.T) {
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="create">
    <area>
      <tagnode>0</tagnode>
      <network>10.1.1.0/24</network>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigCreateContainerError(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 1.1.1.1/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="create">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}

func TestEditConfigCreateList(t *testing.T) {
	const expconfig = `protocols {
	ospf {
		area 0
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area  xc:operation="create">
      <tagnode>0</tagnode>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigCreateListChild(t *testing.T) {
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 1.1.1.1/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area  xc:operation="create">
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigCreateEmptyLeaf(t *testing.T) {
	const expconfig = `protocols {
	ospf {
		parameters {
			opaque-lsa
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <opaque-lsa xc:operation="create"/>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDeleteContainerNotExistError(t *testing.T) {
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="delete"/>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, emptyconfig, true)
}

func TestEditConfigDeleteContainer(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="delete"/>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, emptyconfig, true)
}

func TestEditConfigDeleteList(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="delete"/>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDeleteListEntry(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="delete">
      <tagnode>0</tagnode>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDeleteListKey(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	// This is not a valid edit-config so make sure it is an error
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode xc:operation="delete">0</tagnode>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}

func TestEditConfigDeleteListChild(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network xc:operation="delete">10.1.1.0/24</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDeleteEmptyLeaf(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters {
			opaque-lsa
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		parameters
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <opaque-lsa xc:operation="delete"/>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigMergeList(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
		area 2 {
			network 3.3.3.3/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="merge">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area>
      <tagnode>2</tagnode>
      <network>3.3.3.3/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigMergeListEntry(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="merge">
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigReplaceList(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 3 {
			network 3.3.3.3/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="replace">
    <area>
      <tagnode>3</tagnode>
      <network>3.3.3.3/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigReplaceListEntry(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 3.3.3.3/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="replace">
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

// No failure removing non-existing paths
func TestEditConfigRemoveContainerNotExist(t *testing.T) {
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="remove"/>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, emptyconfig, true)
}

func TestEditConfigRemoveContainer(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="remove"/>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, emptyconfig, true)
}

func TestEditConfigRemoveList(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="remove"/>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigRemoveListEntry(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="remove">
      <tagnode>0</tagnode>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigRemoveListKey(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	// This is not a valid edit-config so make sure it is an error
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode xc:operation="remove">0</tagnode>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}

func TestEditConfigRemoveListChild(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network xc:operation="remove">10.1.1.0/24</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigRemoveEmptyLeaf(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters {
			opaque-lsa
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		parameters
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <opaque-lsa xc:operation="remove"/>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDefOpNone(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`

	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area>
      <tagnode>2</tagnode>
      <network>3.3.3.3/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}

func TestEditConfigDefOpNoneWithOps(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
		area 2 {
			network 10.3.3.0/24
		}
		area 3 {
			network 10.4.4.0/24
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
		area 3 {
			network 4.4.4.4/32
		}
		area 5 {
			network 6.6.6.6/32
		}
	}
}
`

	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="merge">
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area xc:operation="delete">
      <tagnode>2</tagnode>
    </area>
    <area xc:operation="replace">
      <tagnode>3</tagnode>
      <network>4.4.4.4/32</network>
    </area>
    <area>
      <tagnode>4</tagnode>
      <network>5.5.5.5/32</network>
    </area>
    <area xc:operation="create">
      <tagnode>5</tagnode>
      <network>6.6.6.6/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDefOpMerge(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
		area 2 {
			network 3.3.3.3/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area>
      <tagnode>2</tagnode>
      <network>3.3.3.3/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_merge, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDefOpMergeWithOps(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
		area 2 {
			network 10.3.3.0/24
		}
		area 3 {
			network 10.4.4.0/24
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 1 {
			network 2.2.2.2/32
		}
		area 3 {
			network 4.4.4.4/32
		}
		area 4 {
			network 5.5.5.5/32
		}
		area 5 {
			network 6.6.6.6/32
		}
	}
}
`

	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area xc:operation="delete">
      <tagnode>2</tagnode>
    </area>
    <area xc:operation="replace">
      <tagnode>3</tagnode>
      <network>4.4.4.4/32</network>
    </area>
    <area xc:operation="merge">
      <tagnode>4</tagnode>
      <network>5.5.5.5/32</network>
    </area>
    <area xc:operation="create">
      <tagnode>5</tagnode>
      <network>6.6.6.6/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_merge, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDefOpReplace(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 1.1.1.1/32
		}
		area 2 {
			network 3.3.3.3/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area>
      <tagnode>2</tagnode>
      <network>3.3.3.3/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_replace, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigDefOpReplaceWithOps(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
		area 2 {
			network 10.3.3.0/24
		}
		area 3 {
			network 10.4.4.0/24
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
		area 3 {
			network 4.4.4.4/32
		}
		area 4 {
			network 5.5.5.5/32
		}
		area 5 {
			network 6.6.6.6/32
		}
	}
}
`

	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area xc:operation="merge">
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area xc:operation="replace">
      <tagnode>3</tagnode>
      <network>4.4.4.4/32</network>
    </area>
    <area>
      <tagnode>4</tagnode>
      <network>5.5.5.5/32</network>
    </area>
    <area xc:operation="create">
      <tagnode>5</tagnode>
      <network>6.6.6.6/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	t.Skip("area 0 config is replaced instead of being merged")
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_replace, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigTestOptTestOnly(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area>
      <tagnode>2</tagnode>
      <network>3.3.3.3/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_replace, testopt_testonly, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}

func TestEditConfigTestOptTestOnlyError(t *testing.T) {
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="delete"/>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_testonly, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, emptyconfig, true)
}

func TestEditConfigTestOptSet(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
		area 1 {
			network 2.2.2.2/32
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 1.1.1.1/32
		}
		area 2 {
			network 3.3.3.3/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <area>
      <tagnode>0</tagnode>
      <network>1.1.1.1/32</network>
    </area>
    <area>
      <tagnode>2</tagnode>
      <network>3.3.3.3/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_replace, testopt_set, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigTestOptSetError(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters {
			opaque-lsa
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
      <parameters>
        <opaque-lsa xc:operation="create"/>
      </parameters>
    </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_set, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}

func TestEditConfigTestOptSetErrOptCont(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
			network 1.1.1.1/32
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="create">
    <area>
      <tagnode>0</tagnode>
      <network>10.1.1.0/24</network>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_set, erropt_cont, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigTestOptSetErrOptRollback(t *testing.T) {
	const config = `protocols {
	ospf {
		area 0 {
			network 10.1.1.0/24
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf" xc:operation="create">
    <area>
      <tagnode>0</tagnode>
      <network>10.1.1.0/24</network>
      <network>1.1.1.1/32</network>
    </area>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_set, erropt_rollback, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}

func TestEditConfigNormilization(t *testing.T) {
	const expconfig = `protocols {
	ospf {
		parameters {
			router-id FOO
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <router-id xc:operation="create">bar</router-id>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_set, erropt_rollback, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigEnumLeafCreate(t *testing.T) {
	const expconfig = `protocols {
	ospf {
		parameters {
			abr-type standard
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <abr-type xc:operation="create">standard</abr-type>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, emptyconfig)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigEnumLeafReplace(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters {
			abr-type standard
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		parameters {
			abr-type shortcut
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <abr-type xc:operation="replace">shortcut</abr-type>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigEnumLeafDelete(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters {
			abr-type standard
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		parameters
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <abr-type xc:operation="delete"/>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigEnumLeafCreateDefault(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters
	}
}
`
	const expconfig = `protocols {
	ospf {
		parameters {
			abr-type cisco
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <abr-type xc:operation="create">cisco</abr-type>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigEnumLeafReplaceFromDefault(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters
	}
}
`
	const expconfig = `protocols {
	ospf {
		parameters {
			abr-type cisco
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <abr-type xc:operation="replace">cisco</abr-type>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigEnumLeafReplaceToDefault(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters {
			abr-type standard
		}
	}
}
`
	const expconfig = `protocols {
	ospf {
		parameters {
			abr-type cisco
		}
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <abr-type xc:operation="replace">cisco</abr-type>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	validateEditConfig(t, false, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, expconfig, true)
}

func TestEditConfigEnumLeafDeleteDefault(t *testing.T) {
	const config = `protocols {
	ospf {
		parameters
	}
}
`
	const edit_config = `
<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:xc="urn:ietf:params:xml:ns:netconf:base:1.0">
<protocols xmlns="urn:vyatta.com:test:vyatta-protocols">
  <ospf xmlns="urn:vyatta.com:test:vyatta-protocols-ospf">
    <parameters>
      <abr-type xc:operation="delete"/>
    </parameters>
  </ospf>
</protocols>
</config>
`
	srv, sess := TstStartupMultipleSchemas(t, edit_config_schema, config)
	defer sess.Kill()
	// Deleting a non-existent but default value should fail
	validateEditConfig(t, true, sess, srv.Ctx, target_candidate, defop_none, testopt_testset, erropt_stop, edit_config)
	ValidateShow(t, sess, srv.Ctx, emptypath, true, config, true)
}
