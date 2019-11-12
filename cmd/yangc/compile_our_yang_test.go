// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main_test

import (
	"testing"

	"github.com/danos/config/schema"
	"github.com/danos/yang/compile"
)

func TestOurYang(t *testing.T) {

	cfg := &compile.Config{
		YangDir:     "../../yang",
		SkipUnknown: true,
		Filter:      compile.IsConfigOrState(),
	}

	_, err := schema.CompileDir(cfg, nil)
	if err != nil {
		t.Fatalf("Unexpected error compiling our schema\n  %s", err.Error())
	}
}
