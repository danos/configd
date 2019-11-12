// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests on XPATH functionality that require an
// active session to be running.

// Test procedure is the same for each node type.  We create an entry, and
// ensure 'when' fails, and that a subsequent 'must' statement that would
// fail is not checked (not necessary).  We then change config so that the
// second 'must' will fail (so we verify that multiple musts are checked).
// Finally we get all 3 checks to pass.

package session_test

import (
	"strings"
	"testing"

	"github.com/danos/configd/session"
	. "github.com/danos/configd/session/sessiontest"
)

var expOutAllOK = []string{"[]\n\n[]\n\n"}

const NoFilter = ""

type xpathTestEntry struct {
	set_cmds        []ValidateOpTbl
	del_cmds        []ValidateOpTbl
	expCommitResult bool
	expConfig       string
	expOutput       []string
	expDebug        string
}

func newXpathTestEntry(
	set_cmds []ValidateOpTbl,
	del_cmds []ValidateOpTbl,
	expCommitResult bool,
	expConfig string,
	expOutput []string,
) xpathTestEntry {
	return xpathTestEntry{
		set_cmds:        set_cmds,
		del_cmds:        del_cmds,
		expCommitResult: expCommitResult,
		expConfig:       expConfig,
		expOutput:       expOutput,
	}
}

func newXpathTestEntryWithDebug(
	set_cmds []ValidateOpTbl,
	del_cmds []ValidateOpTbl,
	expCommitResult bool,
	expConfig string,
	expOutput []string,
	expDebug string,
) xpathTestEntry {
	return xpathTestEntry{
		set_cmds:        set_cmds,
		del_cmds:        del_cmds,
		expCommitResult: expCommitResult,
		expConfig:       expConfig,
		expOutput:       expOutput,
		expDebug:        expDebug,
	}
}

func runXpathTestsCheckOutputMultipleSchemas(
	t *testing.T,
	schemaDefs []TestSchema,
	config string,
	tests []xpathTestEntry,
) {
	srv, sess := NewTestSpec(t).
		SetSchemaDefs(schemaDefs).
		SetConfig(config).
		Init()
	runXpathTestsInternal(t, srv, sess, tests, true, NoFilter)
}

func runXpathTestsCheckOutput(
	t *testing.T,
	schema string,
	config string,
	tests []xpathTestEntry,
) {
	srv, sess := NewTestSpec(t).
		SetSingleSchema(schema).
		SetConfig(config).
		Init()
	runXpathTestsInternal(t, srv, sess, tests, true, NoFilter)
}

type captureStdOutFromTestFn func()

func runXpathTestsCheckDebugOutput(
	t *testing.T,
	schema string,
	config string,
	tests []xpathTestEntry,
	filter string,
) {
	srv, sess := NewTestSpec(t).
		SetSingleSchema(schema).
		SetConfig(config).
		Init()
	runXpathTestsInternal(t, srv, sess, tests, true, filter)
}

func runXpathTests(
	t *testing.T,
	schema string,
	config string,
	tests []xpathTestEntry,
) {
	srv, sess := TstStartup(t, schema, config)
	runXpathTestsInternal(t, srv, sess, tests, false, NoFilter)
}

func runXpathTestsInternal(
	t *testing.T,
	srv *TstSrv,
	sess *session.Session,
	tests []xpathTestEntry,
	checkOutput bool,
	filter string,
) {
	// For each test case, enter set and delete commands then commit as a
	// single transaction.  If commit is expected to fail, we need to reverse
	// the changes.
	for _, test := range tests {
		ValidateOperationTable(t, sess, srv.Ctx, test.set_cmds, SET)
		ValidateOperationTable(t, sess, srv.Ctx, test.del_cmds, DELETE)

		if checkOutput {
			if len(filter) > 0 {
				ValidateCommitWithDebug(t, sess, srv.Ctx,
					test.expCommitResult, test.expConfig, test.expOutput,
					test.expDebug)
			} else {
				ValidateCommitMultipleOutput(t, sess, srv.Ctx,
					test.expCommitResult, test.expConfig, test.expOutput)
			}
		}

		if test.expCommitResult == CommitFail {
			// If commit is expected to fail, we need to remove changes
			// so we have clean slate for next test.
			ValidateOperationTable(t, sess, srv.Ctx, test.set_cmds, DELETE)
			ValidateOperationTable(t, sess, srv.Ctx, test.del_cmds, SET)
		}
	}

	sess.Kill()
}

// Helper function for creating ValidateOpTbl items we can pass into
// ValidateOperationTable.
//
// Path and Value could be merged as all the underlying code does
// with the two is to stick them together.  Here we just ignore value.
func createValOpTbl(desc, path string, expResult bool) ValidateOpTbl {
	opTbl := ValidateOpTbl{
		Description: desc,
		Result:      expResult,
		Path:        strings.Split(path, "/"),
		Value:       "",
	}

	return opTbl
}
