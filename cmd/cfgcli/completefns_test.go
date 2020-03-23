// Copyright (c) 2018-2020, AT&T Intellectual Property.
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
	"fmt"
	"testing"

	"github.com/danos/configd/common"
)

var testCfgMgr = newTestClient(nil).
	enableFeature(common.ConfigManagementFeature)

type testDefinitions struct {
	name      string
	cmdLine   string
	expOutput []string
	success   bool
	prefix    string
}

// CANCEL-COMMIT
func TestCancelCommitCommand(t *testing.T) {
	testCases := []testDefinitions{
		{
			name:      "CancelCommit",
			cmdLine:   "cancel-commit",
			expOutput: []string{"COMPREPLY=( cancel-commit  )"},
			success:   true,
		},
		{
			name:      "CancelCommit completion",
			cmdLine:   "cancel-co",
			expOutput: []string{"COMPREPLY=( cancel-commit  )"},
			success:   true,
		},
		{
			name:    "Comment completion",
			cmdLine: "cancel-commit com",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
		},
		{
			name:    "persist-id completion",
			cmdLine: "cancel-commit per",
			expOutput: []string{
				"COMPREPLY=( persist-id  )"},
			success: true,
		},
		{
			name:    "cancel-commit force completion",
			cmdLine: "cancel-commit for",
			expOutput: []string{
				"COMPREPLY=( force  )"},
			success: true,
		},
		{
			name:    "Comment completion with trailing text",
			cmdLine: "cancel-commit comTrailingText",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
			prefix:  "com",
		},
		{
			name:    "Wrong comment keyword",
			cmdLine: "cancel-commit not-comment-keyword",
			expOutput: []string{
				"Invalid command: cancel-commit [not-comment-keyword]"},
			success: false,
		},
		{
			name:    "Comment - no comment",
			cmdLine: "cancel-commit comment",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
		},
		{
			name:    "Comment Text",
			cmdLine: "cancel-commit comment text",
			expOutput: []string{
				"<text> Comment for the commit log"},
			success: true,
		},
		{
			name:    "Comment Text - extra text",
			cmdLine: "cancel-commit comment text extra-text",
			expOutput: []string{
				"Invalid command: cancel-commit comment text [extra-text]"},
			success: false,
		},
		{
			name:    "Persist Comment Text",
			cmdLine: "cancel-commit persist-id commit_test comment \"Cancel commit\"",
			expOutput: []string{
				"<text> Comment for the commit log"},
			success: true,
		},
		{
			name:    "Force completion",
			cmdLine: "cancel-commit for",
			expOutput: []string{
				"COMPREPLY=( force  )"},
			success: true,
		},
		{
			name:    "Force with comment text",
			cmdLine: "cancel-commit force comment \"Cancel commit\"",
			expOutput: []string{
				"<text> Comment for the commit log"},
			success: true,
		},
		{
			name:    "lots Text",
			cmdLine: "cancel-commit force comm",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
		},
		{
			name:    "Persist-id only at beginning",
			cmdLine: "cancel-commit comment text persist-",
			expOutput: []string{
				"Invalid command: cancel-commit comment text [persist-]"},
			success: false,
		},
		{
			name:    "Force only at beginning",
			cmdLine: "cancel-commit comment text force",
			expOutput: []string{
				"Invalid command: cancel-commit comment text [force]"},
			success: false,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			completionText, err := getCompletedCmdLine(
				testCfgMgr, test.cmdLine, test.prefix)

			if test.success {
				checkNoError(t, err)
				checkTextContains(t, completionText, test.expOutput)
			} else {
				checkErrorContains(t, err, test.expOutput)
			}
		})
	}
}

// CONFIRM
func TestConfirmCommand(t *testing.T) {
	testCases := []testDefinitions{
		{
			name:      "Confirm",
			cmdLine:   "confirm",
			expOutput: []string{"COMPREPLY=( confirm  )"},
			success:   true,
		},
		{
			name:      "Confirm completion",
			cmdLine:   "conf",
			expOutput: []string{"COMPREPLY=( confirm  )"},
			success:   true,
		},
		{
			name:    "Perisist-id completion",
			cmdLine: "confirm pers",
			expOutput: []string{
				"COMPREPLY=( persist-id  )"},
			success: true,
		},
		{
			name:    "Persist-id completion with trailing text",
			cmdLine: "confirm persisTrailingText",
			expOutput: []string{
				"COMPREPLY=( persist-id  )"},
			success: true,
			prefix:  "per",
		},
		{
			name:    "Wrong persist-id keyword",
			cmdLine: "confirm not-persist-id-keyword",
			expOutput: []string{
				"Invalid command: confirm [not-persist-id-keyword]"},
			success: false,
		},
		{
			name:    "Persist-id - no persist-id",
			cmdLine: "confirm persist-id",
			expOutput: []string{
				"COMPREPLY=( persist-id  )"},
			success: true,
		},
		{
			name:    "Persist-id Text",
			cmdLine: "confirm persist-id text",
			expOutput: []string{
				"<text> Persist-id of pending confirmed commit"},
			success: true,
		},
		{
			name:    "Persist-id Text - extra text",
			cmdLine: "confirm persist-id text extra-text",
			expOutput: []string{
				"Invalid command: confirm persist-id text [extra-text]"},
			success: false,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			completionText, err := getCompletedCmdLine(
				testCfgMgr, test.cmdLine, test.prefix)

			if test.success {
				checkNoError(t, err)
				checkTextContains(t, completionText, test.expOutput)
			} else {
				checkErrorContains(t, err, test.expOutput)
			}
		})
	}
}

// COMMIT
type commitTest struct {
	name      string
	cmdLine   string
	expOutput []string
	success   bool
	prefix    string
}

func TestCommitCommand(t *testing.T) {
	testCases := []commitTest{
		{
			name:      "Commit",
			cmdLine:   "commit",
			expOutput: []string{" commit ", " commit-confirm "},
			success:   true,
		},
		{
			name:      "Commit completion",
			cmdLine:   "comm",
			expOutput: []string{" commit ", " commit-confirm "},
			success:   true,
		},
		{
			name:    "Commit - trailing space",
			cmdLine: "commit ",
			expOutput: []string{
				"<Enter> Commit working configuration",
				"comment Comment for commit log"},
			success: true,
		},
		{
			name:    "Comment completion",
			cmdLine: "commit com",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
		},
		{
			name:    "Comment completion with trailing text",
			cmdLine: "commit comTrailingText",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
			prefix:  "com",
		},
		{
			name:    "Wrong comment keyword",
			cmdLine: "commit not-comment-keyword",
			expOutput: []string{
				"Invalid command: commit [not-comment-keyword]"},
			success: false,
		},
		{
			name:    "Comment - no comment",
			cmdLine: "commit comment",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
		},
		{
			name:    "Comment Text",
			cmdLine: "commit comment text",
			expOutput: []string{
				"<text> Comment for the commit log"},
			success: true,
		},
		{
			name:    "Comment Text - extra text",
			cmdLine: "commit comment text extra-text",
			expOutput: []string{
				"Invalid command: commit comment text [extra-text]"},
			success: false,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			completionText, err := getCompletedCmdLine(
				testCfgMgr, test.cmdLine, test.prefix)

			if test.success {
				checkNoError(t, err)
				checkTextContains(t, completionText, test.expOutput)
			} else {
				checkErrorContains(t, err, test.expOutput)
			}
		})
	}
}

// COMMIT-CONFIRM

// No completion text: we just add a space
func TestCommitConfirmCommandOnly(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-confirm")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( commit-confirm  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCompletion(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( commit-confirm  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmSpace(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-confirm ")

	checkNoError(t, err)

	expText := []string{"<value> Time (minutes) to issue 'confirm'"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmInvalidTimeout(t *testing.T) {
	_, err := completeCmdLineWithCfgMgr(testCfgMgr, "commit-confirm x")

	expectedErrs := []string{"Invalid timeout: x"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmTooLowTimeout(t *testing.T) {
	_, err := completeCmdLineWithCfgMgr(testCfgMgr, "commit-confirm 0")

	expectedErrs := []string{"Invalid timeout: 0"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmValidTimeout(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-confirm 10")

	checkNoError(t, err)

	expText := []string{"<value> Time (minutes) to issue 'confirm'"}
	checkTextContains(t, completionText, expText)
}

// Abbreviated keyword
func TestCommitConfValidTimeout(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 10")

	checkNoError(t, err)

	expText := []string{"<value> Time (minutes) to issue 'confirm'"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmInvalidTimeoutSpace(t *testing.T) {
	_, err := completeCmdLineWithCfgMgr(testCfgMgr, "commit-confirm x ")

	expectedErrs := []string{"Invalid timeout: x"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfInvalidTimeoutSpace(t *testing.T) {
	_, err := completeCmdLineWithCfgMgr(testCfgMgr, "commit-conf x ")

	expectedErrs := []string{"Invalid timeout: x"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmValidTimeoutSpace(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 10 ")

	checkNoError(t, err)

	expText := []string{
		"<Enter> Commit working configuration subject to confirmation",
		"comment Comment for commit log"}
	checkTextContains(t, completionText, expText)
}

//notcomment
func TestCommitConfirmWrongComment(t *testing.T) {
	_, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 notcomment")

	expectedErrs := []string{
		"Invalid command: commit-confirm 999 [notcomment]"}
	checkErrorContains(t, err, expectedErrs)
}

// comx
func TestCommitConfirmPartialCommentInvalid(t *testing.T) {
	_, err := completeCmdLineWithCfgMgr(testCfgMgr, "commit-conf 999 comx")

	expectedErrs := []string{
		"Invalid command: commit-confirm 999 [comx]"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmPartialComment(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 comm")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( comment  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmComment(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 comment")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( comment  )"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentSpace(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 comment ")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentText(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 comment text")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmPartialCommentText(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 co text")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentQuotedText(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(testCfgMgr,
		"commit-conf 999 comment \"quoted text\"")

	checkNoError(t, err)

	expText := []string{
		"<text> Comment for the commit log"}
	checkTextContains(t, completionText, expText)
}

func TestCommitConfirmCommentTextExtraText(t *testing.T) {
	_, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 comment text extra_text")

	expectedErrs := []string{
		"Invalid command: commit-confirm 999 comment text [extra_text]"}
	checkErrorContains(t, err, expectedErrs)
}

func TestCommitConfirmCommentTextSpace(t *testing.T) {
	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "commit-conf 999 comment text ")

	checkNoError(t, err)

	expText := []string{
		"<Enter> Execute the current command"}
	checkTextContains(t, completionText, expText)
}

func testCheckConfigMgmt(c cfgManager) bool { return true }

// Compare
func TestCompareCompletion(t *testing.T) {

	completionText, err := completeCmdLineWithCfgMgr(
		testCfgMgr, "compare sav")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( saved  )"}
	checkTextContains(t, completionText, expText)
}

func TestCompareCompletionWithTrailingText(t *testing.T) {

	completionText, err := completeCmdLineWithPrefixAndCfgMgr(
		testCfgMgr, "compare savTrailingText", "sav")

	checkNoError(t, err)

	expText := []string{"COMPREPLY=( saved  )"}
	checkTextContains(t, completionText, expText)
}

// ROLLBACK
type rollbackTest struct {
	name      string
	cmdLine   string
	expOutput []string
	success   bool
	prefix    string
}

func TestRollbackCommand(t *testing.T) {
	testCases := []rollbackTest{
		{
			name:      "Rollback w/o version",
			cmdLine:   "rollback",
			expOutput: []string{"COMPREPLY=( rollback  )"},
			success:   true,
		},
		{
			name:      "Rollback completion",
			cmdLine:   "roll",
			expOutput: []string{"COMPREPLY=( rollback  )"},
			success:   true,
		},
		{
			name:    "Rollback - trailing space",
			cmdLine: "rollback ",
			expOutput: []string{
				"<N>   Rollback to revision N",
				"1     2019-08-21 09:00:1 vyatta"},
			success: true,
		},
		{
			name:      "Rollback with valid version",
			cmdLine:   "rollback 1",
			expOutput: []string{"COMPREPLY=( 1  )"},
			success:   true,
		},
		{
			name:    "Rollback with valid version - trailing space",
			cmdLine: "rollback 1 ",
			expOutput: []string{
				"<Enter> Execute the current command",
				"comment Comment for commit log"},
			success: true,
		},
		{
			name:      "Rollback - invalid version (numeric)",
			cmdLine:   "rollback 99",
			expOutput: []string{"<N>   Rollback to revision N"},
			success:   true,
		},
		{
			name:      "Rollback - invalid version (non-numeric)",
			cmdLine:   "rollback invalid-version",
			expOutput: []string{"<N>   Rollback to revision N"},
			success:   true,
		},
		{
			name:    "Comment completion",
			cmdLine: "rollback 1 com",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
		},
		{
			name:    "Comment completion with trailing text",
			cmdLine: "rollback 1 comTrailingText",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
			prefix:  "com",
		},
		{
			name:    "Wrong comment keyword",
			cmdLine: "rollback 0 not-comment-keyword",
			expOutput: []string{
				"Invalid command: rollback 0 [not-comment-keyword]"},
			success: false,
		},
		{
			name:    "Comment - no comment",
			cmdLine: "rollback 0 comment",
			expOutput: []string{
				"COMPREPLY=( comment  )"},
			success: true,
		},
		{
			name:    "Comment Text",
			cmdLine: "rollback 3 comment text",
			expOutput: []string{
				"<text> Comment for commit log"},
			success: true,
		},
		{
			name:    "Comment Text - extra text",
			cmdLine: "rollback 1 comment text extra-text",
			expOutput: []string{
				"Invalid command: rollback 1 comment text [extra-text]"},
			success: false,
		},
	}
	for _, test := range testCases {
		// We need to have a commit log history, and enable the rollback cmd.
		// '3' get us 3 log entries, and we use 0, 1 and 3 in tests as valid,
		// and 99 as invalid.
		cfgMgr := newTestClient(nil).
			setCommitLog(3).
			enableFeature(
				common.ConfigManagementFeature)

		t.Run(test.name, func(t *testing.T) {
			completionText, err := getCompletedCmdLine(
				cfgMgr, test.cmdLine, test.prefix)

			if test.success {
				checkNoError(t, err)
				checkTextContains(t, completionText, test.expOutput)
			} else {
				checkErrorContains(t, err, test.expOutput)
			}
		})
	}
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

type compReplyTest struct {
	name   string
	input  string
	output string
}

func formatCompReply(input string) string {
	return fmt.Sprintf("echo \"%s\" | ${VYATTA_PAGER:-cat};COMPREPLY=(  )",
		input)
}

func TestGetCompReplyEscaping(t *testing.T) {
	testStrings := []compReplyTest{
		{
			name:   "Backticks in single quotes",
			input:  "'`echo 001`'",
			output: "'\\`echo 001\\`'",
		},
		{
			name:   "Double dollar in single quotes",
			input:  "'$$'",
			output: "'\\$\\$'",
		},
	}

	for _, test := range testStrings {
		actualOutput := getCompReply(
			true, /* doHelp */
			test.input,
			[]string{""} /* existing compReply*/)
		expOutput := formatCompReply(test.output)
		if actualOutput != expOutput {
			t.Fatalf("\nGot: %s\nExp: %s\n", actualOutput, expOutput)
		}
	}
}
