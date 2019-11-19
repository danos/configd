// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// Provides test infra for completion functions

package main

import (
	"bytes"
	"strings"
	"testing"
)

// When not using a prefix that differs from the full word 'under' the cursor,
// we need a couple of dummy values to indicate this, both for prefix itself,
// and for the cursor position.
const (
	FULL_WORD   = "__DUMMY_VALUE__"
	POS_NOT_SET = -1
)

func checkErrorContains(t *testing.T, err error, expErrs []string) {
	if err == nil {
		t.Fatalf("No error detected.")
	}
	checkTextContains(t, err.Error(), expErrs)
}

func checkTextContains(t *testing.T, text string, expTexts []string) {
	for _, expText := range expTexts {
		if !strings.Contains(text, expText) {
			t.Fatalf("'%s' not found in:\n'%s'", expText, text)
		}
	}
}

func checkNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %s\n", err.Error())
	}
}

func parseCmdLine(
	cmdLine, prefix string,
	pos int,
) (args []string, params cmdLineParams) {

	if cmdLine == "" {
		return nil, cmdLineParams{}
	}

	args = splitCmdLineIntoArgs(cmdLine)
	params = assignParamsFromArgs(args, prefix, pos)

	return args, params
}

// 'args' is a slice of each space-separated item typed by the
// user, noting the following points:
//
// - Leading spaces are ignored
// - Consecutive spaces are squashed to a single space
// - A trailing space is treated as an extra, zero length, argument
//   (strings.Split() handles this implicitly for us)
// - Quotes may be single or double, and contain the other type of quotes.
//
func splitCmdLineIntoArgs(cmdLine string) (args []string) {
	cmdLine = strings.TrimLeft(cmdLine, " ")

	for strings.Contains(cmdLine, "  ") {
		cmdLine = strings.Replace(cmdLine, "  ", " ", -1)
	}

	replacement := "@SPACE@"
	modCmdLine := replaceSpacesInQuotes(cmdLine, replacement)
	args = strings.Split(modCmdLine, " ")
	for index, arg := range args {
		args[index] = strings.Replace(arg, replacement, " ", -1)
	}

	return args
}

// Deal with quoted text by temporarily replacing spaces in quoted
// items with <replacement>.  Handle single and double quotes
func replaceSpacesInQuotes(cmdLine, replacement string) string {
	var modCmdLine bytes.Buffer
	insideQuotes := false
	var quoteChar rune

	for _, char := range cmdLine {
		if insideQuotes {
			if char == ' ' {
				modCmdLine.Write([]byte(replacement))
			} else {
				modCmdLine.WriteRune(char)
				if char == quoteChar {
					insideQuotes = false
				}
			}
		} else {
			modCmdLine.WriteRune(char)
			if char == '"' || char == '\'' {
				insideQuotes = true
				quoteChar = char
			}
		}
	}
	return modCmdLine.String()
}

// 'params' represents parameters set by bash autocompletion (see CWORD),
//          as follows
//   - cidx: index of last argument in args (= len(args) - 1)
//   - cword: last complete argument (so last bar one member of args as
//            last element of argument is either empty (space), or incomplete)
//            Special case: if only one arg, cword is set to this arg
//   - prefix: last element of argument (so either empty or incomplete)
//
func assignParamsFromArgs(
	args []string,
	prefix string,
	pos int,
) (params cmdLineParams) {

	argLen := len(args)
	params.cidx = argLen - 1

	if prefix != FULL_WORD {
		params.pfx = prefix
	} else if args[argLen-1] == "" {
		params.pfx = ""
	} else {
		params.pfx = args[argLen-1]
	}
	if pos != POS_NOT_SET {
		params.cword = args[pos]
		params.cidx = pos
	} else if argLen == 1 {
		params.cword = args[0]
	} else {
		params.cword = args[argLen-2]
	}

	return params
}

func completeCmdLineWithTestClient(
	cfgMgr cfgManager,
	cmdLine, prefix string,
	pos int,
) (completionText string, err error) {

	args, params := parseCmdLine(cmdLine, prefix, pos)
	return complete(cfgMgr, args, params)
}

func testValidFn(
	cfgMgr cfgManager,
	valFn ValidFunc,
	cmdLine, prefix string,
	pos int,
) error {

	args, params := parseCmdLine(cmdLine, prefix, pos)
	ctx := createCompleteCtx(cfgMgr, args, params)
	return valFn(ctx)
}

func completeCmdLineWithPrefixAndCfgMgr(
	c cfgManager,
	cmdLine, prefix string,
) (completionText string, err error) {

	updateDynamicCommands(c)
	defer updateDynamicCommands(newTestClient(nil))

	args, params := parseCmdLine(cmdLine, prefix, POS_NOT_SET)
	return complete(c, args, params)
}

func completeCmdLineWithPrefix(cmdLine, prefix string) (
	completionText string, err error) {

	return completeCmdLineWithPrefixAndCfgMgr(
		newTestClient(nil), cmdLine, prefix)
}

func completeCmdLineWithCfgMgr(
	c cfgManager,
	cmdLine string,
) (completionText string, err error) {

	return completeCmdLineWithPrefixAndCfgMgr(c, cmdLine, FULL_WORD)
}

func completeCmdLine(cmdLine string) (completionText string, err error) {

	return completeCmdLineWithPrefixAndCfgMgr(
		newTestClient(nil), cmdLine, FULL_WORD)
}

func getCompletedCmdLine(
	cfgMgr cfgManager,
	inputCmdLine, prefix string,
) (string, error) {

	if prefix == "" {
		return completeCmdLineWithCfgMgr(cfgMgr, inputCmdLine)
	} else {
		return completeCmdLineWithPrefixAndCfgMgr(
			cfgMgr, inputCmdLine, prefix)
	}
}

func checkArgs(t *testing.T, args []string, numArgs int, expArgs []string) {
	if len(args) != len(expArgs) {
		t.Fatalf("Got %d args (%s); expected %d (%s)",
			len(args), args, len(expArgs), expArgs)
	}
	for index, arg := range args {
		if arg != expArgs[index] {
			t.Fatalf("Arg[%d]: got '%s', exp '%s'",
				index, arg, expArgs[index])
		}
	}
}

func checkParams(
	t *testing.T,
	params cmdLineParams,
	expCidx int,
	expCword, expPrefix string,
) {
	if params.cidx != expCidx {
		t.Fatalf("Params cidx: got %d, exp %d", params.cidx, expCidx)
	}
	if params.cword != expCword {
		t.Fatalf("Params cword: got '%s', exp '%s'", params.cword, expCword)
	}
	if params.pfx != expPrefix {
		t.Fatalf("Params pfx: got '%s', exp '%s'", params.pfx, expPrefix)
	}
}

// Who tests the testers?
func TestParseCmdLine(t *testing.T) {
	args, params := parseCmdLine("single-param",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 1, []string{"single-param"})
	checkParams(t, params, 0, "single-param", "single-param")

	args, params = parseCmdLine("single-param-then-space ",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 2, []string{"single-param-then-space", ""})
	checkParams(t, params, 1, "single-param-then-space", "")

	args, params = parseCmdLine("first-param second-param",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 1, []string{"first-param", "second-param"})
	checkParams(t, params, 1, "first-param", "second-param")

	args, params = parseCmdLine("first-param second-param-then-space ",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 1,
		[]string{"first-param", "second-param-then-space", ""})
	checkParams(t, params, 2, "second-param-then-space", "")

	args, params = parseCmdLine("param       param-after-many-spaces  ",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 1,
		[]string{"param", "param-after-many-spaces", ""})
	checkParams(t, params, 2, "param-after-many-spaces", "")

	args, params = parseCmdLine("param \"dbl-quoted param\" ",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 1, []string{"param", "\"dbl-quoted param\"", ""})
	checkParams(t, params, 2, "\"dbl-quoted param\"", "")

	args, params = parseCmdLine("param 'sgl-quoted param' ",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 1, []string{"param", "'sgl-quoted param'", ""})
	checkParams(t, params, 2, "'sgl-quoted param'", "")

	args, params = parseCmdLine("param \"quoted 'x' param\" ",
		FULL_WORD, POS_NOT_SET)
	checkArgs(t, args, 1, []string{"param", "\"quoted 'x' param\"", ""})
	checkParams(t, params, 2, "\"quoted 'x' param\"", "")

	// Test cases for first and subsequent word where prefix is not whole word
	args, params = parseCmdLine("prefixSuffix", "prefix",
		POS_NOT_SET)
	checkArgs(t, args, 1, []string{"prefixSuffix"})
	checkParams(t, params, 0, "prefixSuffix", "prefix")

	args, params = parseCmdLine("first prefixSuffix", "prefix",
		POS_NOT_SET)
	checkArgs(t, args, 1, []string{"first", "prefixSuffix"})
	checkParams(t, params, 1, "first", "prefix")

	args, params = parseCmdLine("first prefixSuffixAndSpace ", "prefix",
		POS_NOT_SET)
	checkArgs(t, args, 1, []string{"first", "prefixSuffixAndSpace", ""})
	checkParams(t, params, 2, "prefixSuffixAndSpace", "prefix")
}
