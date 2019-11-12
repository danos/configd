// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// Provides tests for cfgcli command completion
//
// General method is to start with shortest variant of command (ie just
// keyword) and build up with correct / incorrect parameters to verify
// that completion options and validation are correct.

package main

import (
	"testing"
)

// CONFIRM
func TestConfirm(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("confirm")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( confirm  )"}
	checkTextContains(t, completionText, expText)
}

func TestConfirmCompletion(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("confirm")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( confirm  )"}
	checkTextContains(t, completionText, expText)
}

func TestConfirmSpace(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("confirm ")

	checkNoError(t, err)

	expText := []string{
		"<Enter> Confirm acceptance of running configuration"}
	checkTextContains(t, completionText, expText)
}

// COMMIT
func TestCommit(t *testing.T) {
	completionText, err := completeCmdLine("commit")

	checkNoError(t, err)

	// May be returned in either order ...
	expText := []string{" commit "}
	if checkConfigMgmt(newTestClient(nil)) {
		expText = append(expText, " commit-confirm ")
	}
	checkTextContains(t, completionText, expText)
}

func TestCommitCompletion(t *testing.T) {
	completionText, err := completeCmdLine("comm")

	checkNoError(t, err)

	// May be returned in either order ...
	expText := []string{" commit "}
	if checkConfigMgmt(newTestClient(nil)) {
		expText = append(expText, " commit-confirm ")
	}
	checkTextContains(t, completionText, expText)
}

func TestCommitSpace(t *testing.T) {
	completionText, err := completeCmdLine("commit ")

	checkNoError(t, err)

	expText := []string{
		"<Enter> Commit working configuration",
		"comment Comment for commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitCommentCompletion(t *testing.T) {
	completionText, err := completeCmdLine("commit com")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( comment  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitCommentCompletionWithTrailingText(t *testing.T) {
	completionText, err := completeCmdLineWithPrefix(
		"commit comTrailingText", "com")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( comment  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitWrongCommentKeyword(t *testing.T) {
	_, err := completeCmdLine("commit not-comment-keyword")

	expectedErrs := []string{"Invalid command: commit [not-comment-keyword]"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitCommentNoComment(t *testing.T) {
	completionText, err := completeCmdLine("commit comment")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( comment  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitCommentText(t *testing.T) {
	completionText, err := completeCmdLine("commit comment text")

	checkNoError(t, err)

	expText := []string{"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitCommentTextExtraText(t *testing.T) {
	_, err := completeCmdLine("commit comment text extra-text")

	expectedErrs := []string{
		"Invalid command: commit comment text [extra-text]"}
	checkErrorContains(t, err, expectedErrs)
}

// COMMIT-CONFIRM

// No completion text: we just add a space
func TestCommitConfirmCommandOnly(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-confirm")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( commit-confirm  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCompletion(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( commit-confirm  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmSpace(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-confirm ")

	checkNoError(t, err)

	expText := []string{"<value> Time (minutes) to issue 'confirm'"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmInvalidTimeout(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	_, err := completeCmdLine("commit-confirm x")

	expectedErrs := []string{"Invalid timeout: x"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmTooLowTimeout(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	_, err := completeCmdLine("commit-confirm 0")

	expectedErrs := []string{"Invalid timeout: 0"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmValidTimeout(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-confirm 10")

	checkNoError(t, err)

	expText := []string{"<value> Time (minutes) to issue 'confirm'"}
	checkTextContains(t, completionText, expText)
}

// Abbreviated keyword
func TestCommitConfValidTimeout(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 10")

	checkNoError(t, err)

	expText := []string{"<value> Time (minutes) to issue 'confirm'"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmInvalidTimeoutSpace(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	_, err := completeCmdLine("commit-confirm x ")

	expectedErrs := []string{"Invalid timeout: x"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfInvalidTimeoutSpace(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	_, err := completeCmdLine("commit-conf x ")

	expectedErrs := []string{"Invalid timeout: x"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmValidTimeoutSpace(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 10 ")

	checkNoError(t, err)

	expText := []string{
		"<Enter> Commit working configuration subject to confirmation",
		"comment Comment for commit log"}
	checkTextContains(t, completionText, expText)
}

//notcomment
func TestCommitConfirmWrongComment(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	_, err := completeCmdLine("commit-conf 999 notcomment")

	expectedErrs := []string{
		"Invalid command: commit-confirm 999 [notcomment]"}
	checkErrorContains(t, err, expectedErrs)
}

// comx
func TestCommitConfirmPartialCommentInvalid(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	_, err := completeCmdLine("commit-conf 999 comx")

	expectedErrs := []string{
		"Invalid command: commit-confirm 999 [comx]"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmPartialComment(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 999 comm")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( comment  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmComment(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 999 comment")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( comment  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentSpace(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 999 comment ")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentText(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 999 comment text")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmPartialCommentText(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 999 co text")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentQuotedText(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine(
		"commit-conf 999 comment \"quoted text\"")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentTextExtraText(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	_, err := completeCmdLine("commit-conf 999 comment text extra_text")

	expectedErrs := []string{
		"Invalid command: commit-confirm 999 comment text [extra_text]"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmCommentTextSpace(t *testing.T) {
	if !checkConfigMgmt(newTestClient(nil)) {
		t.Skip("config managment isn't available")
	}
	completionText, err := completeCmdLine("commit-conf 999 comment text ")

	checkNoError(t, err)

	expText := []string{
		"<Enter> Execute the current command"}
	checkTextContains(t, completionText, expText)
}

func testCheckConfigMgmt(c cfgManager) bool { return true }

// Compare
func TestCompareCompletion(t *testing.T) {
	overrideConfigMgmtCheck(testCheckConfigMgmt)

	completionText, err := completeCmdLine("compare sav")

	resetConfigMgmtCheck()

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( saved  )"}
	checkTextContains(t, completionText, expText)
}

func TestCompareCompletionWithTrailingText(t *testing.T) {
	overrideConfigMgmtCheck(testCheckConfigMgmt)

	completionText, err := completeCmdLineWithPrefix(
		"compare savTrailingText", "sav")

	resetConfigMgmtCheck()

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( saved  )"}
	checkTextContains(t, completionText, expText)
}

// SAVE
func TestSave(t *testing.T) {
	completionText, err := completeCmdLine("save")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( save  )"}
	checkTextContains(t, completionText, expText)
}

func TestSaveSpace(t *testing.T) {
	completionText, err := completeCmdLine("save ")

	checkNoError(t, err)

	expText := []string{
		"<Enter>                              " +
			"(deprecated - 'commit' saves system config file)",
		"<file>                               Save to file on local machine",
		"ftp://<user>:<passwd>@<host>/<file>  Save to file on remote machine",
		"http://<user>:<passwd>@<host>/<file> Save to file on remote machine",
		"scp://<user>:<passwd>@<host>/<file>  Save to file on remote machine",
		"tftp://<host>/<file>                 Save to file on remote machine",
	}
	checkTextContains(t, completionText, expText)
}

// SET
func TestSet(t *testing.T) {
	completionText, err := completeCmdLine("set")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( set  )"}
	checkTextContains(t, completionText, expText)
}

func TestSetCompletion(t *testing.T) {
	completionText, err := completeCmdLine("se")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( set  )"}
	checkTextContains(t, completionText, expText)
}

func TestSetCompletionWithTrailingText(t *testing.T) {
	completionText, err := completeCmdLineWithPrefix("seTrailingText", "se")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( set  )"}
	checkTextContains(t, completionText, expText)
}

// Tests on code that deals with configuration (delete / edit / set / show)
// via the client API.
//
// delete / edit / set / show:
//  -> ValidFn: checkValidPath()
//  -> CompFn: pathComp()
//
// checkValidPath      -> ExpandPath() -> expandPathString() -> cl.Expand()
//                        -> TmplValidatePath()
//                     -> cl.Expand()
//
// pathComp            -> ExpandPath() -> expandPathString() -> cl.Expand()
//   -> printPathHelp  -> ExpandPath() -> expandPathString() -> cl.Expand()

type completionTestCase struct {
	name     string
	cliInput string
	prefix   string
	expErr   string
	pos      int // 0 = set/delete.  0 = next word in call to Expand etc.
	expCalls []MockExpectation
}

func TestCheckValidPathPass(t *testing.T) {

	testCases := []completionTestCase{
		{
			name:     "Empty last word",
			cliInput: "set int data ",
			prefix:   "",
			pos:      3,
			expCalls: []MockExpectation{
				{
					fnName:     "Expand",
					callParams: []string{"/int/data"},
					retParams: &MockReturnParams{
						retStr: "/interfaces/dataplane", retErr: nil},
				},
				{
					fnName:     "TmplValidatePath",
					callParams: []string{"/interfaces/dataplane"},
					retParams:  &MockReturnParams{retBool: true, retErr: nil},
				},
			},
		},
		{
			name:     "Cursor at end of last word",
			cliInput: "set int data",
			prefix:   "data",
			pos:      2,
			expCalls: []MockExpectation{
				{
					fnName:     "ExpandWithPrefix",
					callParams: []string{"/int/data", "data", "1"},
					retParams: &MockReturnParams{
						retStr: "/interfaces/dataplane", retErr: nil},
				},
			},
		},
		{
			name:     "Cursor in last word, full word valid",
			cliInput: "set int data",
			prefix:   "dat",
			pos:      2,
			expCalls: []MockExpectation{
				{
					fnName:     "ExpandWithPrefix",
					callParams: []string{"/int/data", "dat", "1"},
					retParams: &MockReturnParams{
						retStr: "/interfaces/dataplane", retErr: nil},
				},
			},
		},
		{
			name:     "Cursor in last word, valid only to prefix",
			cliInput: "set int datax",
			prefix:   "data",
			pos:      2,
			expCalls: []MockExpectation{
				{
					fnName:     "ExpandWithPrefix",
					callParams: []string{"/int/datax", "data", "1"},
					retParams: &MockReturnParams{
						retStr: "/interfaces/dataplanex", retErr: nil},
				},
			},
		},
		{
			name:     "Cursor at end of middle word",
			cliInput: "set int data",
			prefix:   "int",
			pos:      1,
			expCalls: []MockExpectation{
				{
					fnName:     "ExpandWithPrefix",
					callParams: []string{"/int/data", "int", "0"},
					retParams: &MockReturnParams{
						retStr: "/interfaces/dataplane", retErr: nil},
				},
			},
		},
		{
			name:     "Cursor in middle word, full word valid",
			cliInput: "set int data",
			prefix:   "in",
			pos:      1,
			expCalls: []MockExpectation{
				{
					fnName:     "ExpandWithPrefix",
					callParams: []string{"/int/data", "in", "0"},
					retParams: &MockReturnParams{
						retStr: "/interfaces/dataplane", retErr: nil},
				},
			},
		},
		{
			name:     "Cursor in middle word, valid only to prefix",
			cliInput: "set intx data",
			prefix:   "int",
			pos:      1,
			expCalls: []MockExpectation{
				{
					fnName:     "ExpandWithPrefix",
					callParams: []string{"/intx/data", "int", "0"},
					retParams: &MockReturnParams{
						retStr: "/interfacesx/dataplane", retErr: nil},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			tc := newTestClient(t)
			for _, expCall := range test.expCalls {
				tc.AddExpectedCall(expCall)
			}

			err := testValidFn(
				tc, checkValidPath, test.cliInput, test.prefix, test.pos)

			checkNoError(t, err)
			tc.CheckAllCallsMade(t)
		})
	}
}
