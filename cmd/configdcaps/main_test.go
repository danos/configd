// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"testing"

	"github.com/danos/config/testutils"
)

func verifyString(t *testing.T, out, exp_out string) {
	if out != exp_out {
		t.Logf("Received: %s", out)
		t.Logf("Expected: %s", exp_out)
		t.Error("Schema string incorrect")
		testutils.LogStack(t)
	}
}

func TestSchemaString(t *testing.T) {
	const exp_out = "urn:vyatta.com:mgmt:configd:1?module=configd-v1&amp;revision=2015-07-28&amp;features=feat1,feat2"
	s := Schema{
		Id:       "configd-v1",
		Ns:       "urn:vyatta.com:mgmt:configd:1",
		Ver:      "2015-07-28",
		Features: []string{"feat1", "feat2"},
	}
	verifyString(t, s.String(), exp_out)
}

func TestSchemaStringMissingVer(t *testing.T) {
	const exp_out = "urn:vyatta.com:mgmt:configd:1?module=configd-v1&amp;features=feat1,feat2"
	s := Schema{
		Id:       "configd-v1",
		Ns:       "urn:vyatta.com:mgmt:configd:1",
		Features: []string{"feat1", "feat2"},
	}
	verifyString(t, s.String(), exp_out)
}
