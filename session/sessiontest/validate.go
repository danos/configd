// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package sessiontest

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/danos/configd"
	"github.com/danos/configd/rpc"
	. "github.com/danos/configd/session"
	"github.com/danos/utils/exec"
	"github.com/danos/utils/pathutil"
)

// Tests run in the order they are defined

func ValidateExists(t *testing.T, sess *Session, ctx *configd.Context, path []string, exp bool) {
	//	t.Log(sess.Show(ctx, nil, false))
	exist := sess.Exists(ctx, path)
	if exist != exp {
		t.Errorf("Path [%s] existence not as expected; expected %v",
			pathutil.Pathstr(path), exp)
		logStack(t)
	}
}

// exp is whether we expect an error or not (i.e., true means expect error)
func ValidateSetPath(t *testing.T, sess *Session, ctx *configd.Context, path []string, exp bool) {
	err := sess.ValidateSet(ctx, path)
	if (err != nil) != exp {
		if err == nil {
			t.Errorf("Unexpected validate set path result for path [%s]",
				pathutil.Pathstr(path))
		} else {
			t.Errorf("Unexpected validate set path result for path [%s]; %s",
				pathutil.Pathstr(path), err)
		}
		logStack(t)
	}
}

type Operation int

const (
	SET Operation = iota
	SET_PATH
	DELETE
	SET_AND_COMMIT
	DELETE_AND_COMMIT
)

type ValidateOpTbl struct {
	Description string
	Path        []string
	Value       string
	Result      bool // result of commit if requested, otherwise, set/delete
}

type SessOp int

const (
	COMMIT SessOp = iota
	VALIDATE
)

func (s SessOp) String() string {
	switch s {
	case COMMIT:
		return "commit"
	case VALIDATE:
		return "validate"
	default:
		return "undefined"
	}
}

func ValidateSetPathTable(t *testing.T, sess *Session, ctx *configd.Context, tbl []ValidateOpTbl) {
	ValidateOperationTable(t, sess, ctx, tbl, SET_PATH)
}

func ValidateSet(t *testing.T, sess *Session, ctx *configd.Context, path []string, exp bool) {
	//	t.Log("ValidateSet", path)
	err := sess.Set(ctx, path)
	if (err != nil) != exp {
		if err == nil {
			t.Errorf("Unexpected success from set path [%s]",
				pathutil.Pathstr(path))
		} else {
			t.Errorf("Unexpected error from set path [%s]; %s",
				pathutil.Pathstr(path), err)
		}
		logStack(t)
		return
	}
	if !exp {
		ValidateExists(t, sess, ctx, path, true)
	}
}

func ValidateSetTable(t *testing.T, sess *Session, ctx *configd.Context, tbl []ValidateOpTbl) {
	ValidateOperationTable(t, sess, ctx, tbl, SET)
}

const (
	CommitDebugOff = false
	CommitDebugOn  = true
	NoCfgCheck     = false
	CfgCheck       = true
)

func ValidateCommitWithDebug(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	expPass bool,
	expCfg string,
	expOut []string,
	expDebug string,
) {
	validateCommitInternal(t, sess, ctx, CommitDebugOn, CfgCheck,
		expPass, expCfg, expOut)
}

func ValidateCommitMultipleOutput(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	expPass bool,
	expCfg string,
	expOut []string,
) {
	validateCommitInternal(t, sess, ctx, CommitDebugOff, CfgCheck,
		expPass, expCfg, expOut)
}

func ValidateCommit(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	expPass bool,
	expOut ...string,
) {
	switch len(expOut) {
	case 0:
		validateCommitInternal(t, sess, ctx, CommitDebugOff, NoCfgCheck,
			expPass, "", nil)
	case 1:
		validateCommitInternal(t, sess, ctx, CommitDebugOff, CfgCheck,
			expPass, expOut[0], nil)
	default:
		validateCommitInternal(t, sess, ctx, CommitDebugOff, CfgCheck,
			expPass, expOut[0], expOut[1:])
	}
}

// expout is an optional parameter to this function, implemented as a variadic.
//
// First expout string validated the resulting config
// Second expout string validates the output if we expect success, and
// error if we expect failure.
func validateCommitInternal(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	debug bool,
	cfgCheck bool,
	expPass bool,
	expCfg string,
	expOutOrErr []string,
) {
	t.Log("ValidateCommit")
	out, err, result := sess.Commit(ctx, "", debug)
	if result != expPass {
		if !result {
			t.Error("Unexpected commit failure")
			t.Fatal(err)
		} else {
			t.Error("Unexpected commit success")
			t.Fatal(out)
		}
		return
	}

	// Check config, if provided.  Assume zero length config means don't check.
	if expPass && cfgCheck {
		ValidateShow(t, sess, ctx, emptypath, false, expCfg, true)
	}

	// Check each expected output string is found in the output.
	for _, o := range expOutOrErr {
		var outStr, descStr string
		if expPass {
			for _, s := range out {
				outStr = outStr + s.String()
			}
			descStr = "output"
		} else {
			for _, s := range err {
				outStr = outStr + s.Error() + "\n"
			}
			descStr = "error"
		}
		if !strings.Contains(outStr, o) {
			t.Errorf("Unexpected %s from commit", descStr)
			t.Logf("Received: \n%s", outStr)
			t.Logf("Expected to contain: \n%s", o)
			logStack(t)
		}
	}
}

func ValidateShowWithDefaults(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	path []string,
	hideSecrets bool,
	expectedCfgOrError string,
	expectCfgPresent bool, // Ignored in error case
) {
	validateShowInternal(t, sess, ctx, path, hideSecrets, expectedCfgOrError,
		true, expectCfgPresent)
}

func ValidateShow(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	path []string,
	hideSecrets bool,
	expectedCfgOrError string,
	expectCfgPresent bool, // Ignored in error case
) {
	validateShowInternal(t, sess, ctx, path, hideSecrets, expectedCfgOrError,
		false, expectCfgPresent)
}

func ValidateShowContains(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	path []string,
	hideSecrets bool,
	expectCfgPresent bool, // Ignored in error case
	expectedToContain ...string,
) {
	validateShowInternal(t, sess, ctx, path, hideSecrets, "",
		false, expectCfgPresent, expectedToContain...)
}

// This function performs one of 3 related types of check:
//
// - verify expected and actual config match EXACTLY
// - verify error generating configuration matches expected error EXACTLY
// - verify given config is NOT present (given config is not substring of
//     full config)
//
func validateShowInternal(
	t *testing.T,
	sess *Session,
	ctx *configd.Context,
	path []string,
	hideSecrets bool,
	expectedExactCfgOrError string,
	showDefaults bool,
	expectCfgPresent bool, // Ignored in error case
	expToContain ...string,
) {
	cfg, err := sess.Show(ctx, path, hideSecrets, showDefaults)
	if err != nil {
		// The common code that generates formatted error messages for show
		// and config means we get an extra newline at the end of the
		// expected error here.  Simplest just to add one to the actual
		// message before comparing.  It's only a newline ... (-:
		actualErr := err.Error() + "\n"
		if expectedExactCfgOrError != "" {
			if actualErr != expectedExactCfgOrError {
				t.Error("Unexpected error showing exact error")
				t.Logf("Received: \n---\n%sXXX\n", actualErr)
				t.Logf("Expected: \n---\n%sXXX\n", expectedExactCfgOrError)
				logStack(t)
			}
		}
		for _, exp := range expToContain {
			if !strings.Contains(actualErr, exp) {
				t.Error("Unexpected error checking output content")
				t.Logf("Full error:\n---\n%sXXX\n", actualErr)
				t.Logf("Expected to contain:\n---\n%sXXX\n", exp)
				logStack(t)
			}
		}
		return
	}
	if expectCfgPresent {
		if cfg != expectedExactCfgOrError {
			t.Errorf("Config mismatch")
			t.Logf("Received:\n%s", cfg)
			t.Logf("Expected:\n%s", expectedExactCfgOrError)
			logStack(t)
		}
	} else {
		if strings.Contains(cfg, expectedExactCfgOrError) {
			t.Errorf("Config present unexpectedly")
			t.Logf("Received      :\n%s", cfg)
			t.Logf("Did not expect:\n%s", expectedExactCfgOrError)
			logStack(t)
		}
	}
}

// exp is whether we expect an error or not (i.e., true means expect error)
func ValidateDelete(t *testing.T, sess *Session, ctx *configd.Context, path []string, exp bool) {
	t.Log("ValidateDelete", path)
	err := sess.Delete(ctx, path)
	if (err != nil) != exp {
		if err == nil {
			t.Errorf("Unexpected result from delete path [%s]",
				pathutil.Pathstr(path))
		} else {
			t.Errorf("Unexpedted result from delete path [%s]; %s",
				pathutil.Pathstr(path), err)
		}
		logStack(t)
		return
	}
	if !exp {
		def, err := sess.IsDefault(ctx, path)
		if err != nil {
			t.Errorf("Unable to get default for path [%s]; %s",
				pathutil.Pathstr(path), err)
			logStack(t)
			return
		}
		if !def {
			ValidateExists(t, sess, ctx, path, false)
		}
	}
}

func ValidateOperationTable(t *testing.T, sess *Session, ctx *configd.Context, tbl []ValidateOpTbl, op Operation) {
	var path []string
	for key, _ := range tbl {
		desc := "Table"
		if len(tbl[key].Description) != 0 {
			desc = tbl[key].Description
		}
		t.Logf("%s: index %d", desc, key)

		if len(tbl[key].Value) != 0 {
			path = pathutil.CopyAppend(tbl[key].Path, tbl[key].Value)
		} else {
			path = tbl[key].Path
		}

		switch op {
		case SET:
			ValidateSet(t, sess, ctx, path, tbl[key].Result)
		case SET_PATH:
			ValidateSetPath(t, sess, ctx, path, tbl[key].Result)
		case SET_AND_COMMIT:
			// Assumes the requested SET will be successful
			ValidateSet(t, sess, ctx, path, false)
			ValidateCommit(t, sess, ctx, tbl[key].Result)

		case DELETE:
			ValidateDelete(t, sess, ctx, path, tbl[key].Result)

		case DELETE_AND_COMMIT:
			// Assumes the requested DELETE will be successful
			ValidateDelete(t, sess, ctx, path, false)
			ValidateCommit(t, sess, ctx, tbl[key].Result)

		}
	}
}

func ValidateDeleteTable(t *testing.T, sess *Session, ctx *configd.Context, tbl []ValidateOpTbl) {
	ValidateOperationTable(t, sess, ctx, tbl, DELETE)
}

func ValidateChanged(t *testing.T, sess *Session, ctx *configd.Context, exp bool) {
	if sess.Changed(ctx) != exp {
		t.Errorf("Session marked with incorrect changed state; expected %v", exp)
	}
}

type ValidateStatusTbl struct {
	Path   []string
	Status rpc.NodeStatus
	Err    bool
}

func ValidateStatus(t *testing.T, sess *Session, ctx *configd.Context, exp ValidateStatusTbl) {
	status, err := sess.GetStatus(ctx, exp.Path)
	if (err != nil) != exp.Err {
		if err == nil {
			t.Errorf("Unexpected error from get status of  path [%s]",
				pathutil.Pathstr(exp.Path))
		} else {
			t.Errorf("Unexpeced error from to get status of path [%s]; %s",
				pathutil.Pathstr(exp.Path), err)
		}
		logStack(t)
		return
	}
	if status != exp.Status {
		t.Errorf("Unexpected status from path [%s]", pathutil.Pathstr(exp.Path))
		t.Logf("Received: %s(%d)", status, status)
		t.Logf("Expected: %s(%d)", exp.Status, exp.Status)
		logStack(t)
	}
}

func mkLoadFile(t *testing.T, config string) string {
	f, err := ioutil.TempFile("/tmp", "tmpconfig")
	if err != nil {
		t.Fatal("Unable to create test config file")
		logStack(t)
		return ""
	}
	name := f.Name()
	f.WriteString(config)
	f.Close()
	return name
}

// Do a session operation and validate the output received againt the expected output
func ValidateSessOpOutput(t *testing.T, sess *Session, ctx *configd.Context, exp bool, expOut string, op SessOp) {
	var out []*exec.Output
	var err []error
	var result bool

	switch op {
	case COMMIT:
		out, err, result = sess.Commit(ctx, "", false)
	case VALIDATE:
		out, err, result = sess.Validate(ctx)
	}

	if result != exp {
		if !result {
			t.Errorf("Unexpected %s failure", op)
			t.Log(err)
		} else {
			t.Errorf("Unexpected %s success", op)
			t.Log(out)
		}
		logStack(t)
		return
	}

	if exp {
		var b bytes.Buffer
		for _, o := range out {
			b.WriteString(o.String())
		}
		outString := b.String()
		if outString != expOut {
			t.Errorf("Unexpected %s output", op)
			t.Logf("Expected: %s", expOut)
			t.Fatalf("Received: %s", outString)
		}
	}
}
