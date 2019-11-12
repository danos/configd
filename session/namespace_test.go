// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"strings"
	"testing"

	. "github.com/danos/configd/session/sessiontest"
)

func TestNamespace(t *testing.T) {
	const config = `
testcontainer {
	embedded-container {
		leaf-group-one namespace-one
	}
	one {
		aug-one namespace-one
		leaf-group-one namespace-one
	}
	two {
		aug-two namespace-two
		leaf-group-one namespace-two
		leaf-group-two namespace-two
	}
	three {
		aug-three namespace-three
		leaf-group-one namespace-three
		leaf-group-two namespace-three
		leaf-group-three namespace-three
	}
}
test-remote {
	test-remote-leaf namespace-two
	leaf-group-one namespace-two
	leaf-group-two namespace-two
	three-aug-two {
		leaf-group-one namespace-three
		leaf-group-two namespace-three
		leaf-group-three namespace-three
	}
}
test-three {
	three-leaf namespace-three
	leaf-group-one namespace-three
	leaf-group-two namespace-three
	leaf-group-three namespace-three
}
`
	// In a more human readable format, be sure to strip out
	// newlines for results comparison
	const cfg_xml = `<data>
<test-remote xmlns="urn:vyatta.com:mgmt:namespace-two">
<leaf-group-one xmlns="urn:vyatta.com:mgmt:namespace-two">namespace-two</leaf-group-one>
<leaf-group-two xmlns="urn:vyatta.com:mgmt:namespace-two">namespace-two</leaf-group-two>
<test-remote-leaf xmlns="urn:vyatta.com:mgmt:namespace-two">namespace-two</test-remote-leaf>
<three-aug-two xmlns="urn:vyatta.com:mgmt:namespace-three">
<leaf-group-one xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-one>
<leaf-group-three xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-three>
<leaf-group-two xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-two>
</three-aug-two>
</test-remote>
<test-three xmlns="urn:vyatta.com:mgmt:namespace-three">
<leaf-group-one xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-one>
<leaf-group-three xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-three>
<leaf-group-two xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-two>
<three-leaf xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</three-leaf>
</test-three>
<testcontainer xmlns="urn:vyatta.com:mgmt:namespace-one">
<embedded-container xmlns="urn:vyatta.com:mgmt:namespace-one">
<leaf-group-one xmlns="urn:vyatta.com:mgmt:namespace-one">namespace-one</leaf-group-one>
</embedded-container>
<one xmlns="urn:vyatta.com:mgmt:namespace-one">
<aug-one xmlns="urn:vyatta.com:mgmt:namespace-one">namespace-one</aug-one>
<leaf-group-one xmlns="urn:vyatta.com:mgmt:namespace-one">namespace-one</leaf-group-one>
</one>
<three xmlns="urn:vyatta.com:mgmt:namespace-three">
<aug-three xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</aug-three>
<leaf-group-one xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-one>
<leaf-group-three xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-three>
<leaf-group-two xmlns="urn:vyatta.com:mgmt:namespace-three">namespace-three</leaf-group-two>
</three>
<two xmlns="urn:vyatta.com:mgmt:namespace-two">
<aug-two xmlns="urn:vyatta.com:mgmt:namespace-two">namespace-two</aug-two>
<leaf-group-one xmlns="urn:vyatta.com:mgmt:namespace-two">namespace-two</leaf-group-one>
<leaf-group-two xmlns="urn:vyatta.com:mgmt:namespace-two">namespace-two</leaf-group-two>
</two>
</testcontainer>
</data>`
	const enc_xml = "xml"
	tbl := []validateGetTreeTbl{
		{emptypath, enc_xml, strings.Replace(cfg_xml, "\n", "", -1), false},
	}

	srv, sess := TstStartupSchemaDir(t, "testdata/namespaceValid", config, "")
	for key, _ := range tbl {
		validateGetTree(t, sess, srv.Ctx, tbl[key])
	}
	sess.Kill()
}
