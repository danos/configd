// Copyright (c) 2018-2020, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/danos/configd/common"
	"github.com/danos/configd/rpc"
	"github.com/danos/utils/natsort"
	"github.com/danos/utils/pathutil"
)

const (
	CompHeader = "\nPossible Completions:"
)

var defaultcomps = map[string]string{
	"<Enter>": "Execute the current command",
}

type Ctx struct {
	//Args is the current line COMP_WORDS in bash
	Args []string
	//CompCurIdx is COMP_CWORD from bash completion
	//it specifies what the index in the Args array where the cursor is located
	CompCurIdx int
	//CompCurWord is the text string of the COMP_CWORD variable from bash
	CompCurWord string
	//Prefix is the text up to the absolute location of the cursor from the start of CompCurWord
	Prefix string
	//LastComp is another arguement that bash provides to completion functions, currently not used.
	LastComp string
	DoHelp   bool
	Path     []string
	Print    bool
	Client   cfgManager
	All      bool

	HasLoadKey         bool
	HasConfigMgmt      bool
	HasRoutingInstance bool
}

func checkRoutingInstance(c cfgManager) bool {
	feats, err := c.GetConfigSystemFeatures()
	if err != nil {
		return false
	}
	_, exists := feats[common.RoutingInstanceFeature]
	return exists
}

func getcompletions(c completer, args []string) map[string]string {
	cmd, path := args[0], args[1:]
	pstr := pathutil.Pathstr(path)
	comps, err := c.GetCompletions(fromschema(cmd), pstr)
	handleCompError(err, printError)
	return comps
}

func mapkeys(prefix string, m map[string]string) ([]string, []string) {
	keys := make([]string, 0)
	nckeys := make([]string, 0)
	for k, _ := range m {
		if strings.HasPrefix(k, prefix) {
			if strings.HasPrefix(k, "<") && strings.HasSuffix(k, ">") {
				nckeys = append(nckeys, k)
				continue
			}
			keys = append(keys, k)
		}
	}
	natsort.Sort(keys)
	natsort.Sort(nckeys)
	return keys, nckeys
}

func gettypeprefix(c typeGetter, args []string) string {
	path := pathutil.Pathstr(args)
	if v, _ := c.TmplValidatePath(path); !v {
		return "  "
	}
	t, e := c.NodeGetType(path)
	if e != nil {
		return "  "
	}
	switch rpc.NodeType(t) {
	case rpc.LEAF:
		return "  "
	case rpc.LEAF_LIST:
		return "+ "
	case rpc.CONTAINER:
		return " >"
	case rpc.LIST:
		return "+>"
	}
	return "  "
}

func fromschema(cmd string) bool {
	switch cmd {
	case "delete", "show", "comment", "activate", "deactivate":
		return false
	default:
		return true
	}
}

func handleCompError(err error, errHandler func(error)) {
	if err == nil {
		return
	}
	//Print the error to stderr
	errHandler(err)
	//Print the comprreply array to stdout to be evaled
	printOutput("COMPREPLY=( \"\" \" \" )")
	//Exit failure so the script knows to return immediately
	os.Exit(1)
}

func getCurrentPath(ctx *Ctx) []string {
	if ctx.CompCurIdx == 0 {
		return ctx.Args[1:]
	}
	return ctx.Args[1:ctx.CompCurIdx]
}

func expandPathString(e expander, path []string, errHandler func(error),
) string {
	pstr, err := e.Expand(pathutil.Pathstr(path))
	handleCompError(err, errHandler)
	return pstr
}

func ExpandPath(e expander, path []string) []string {
	return pathutil.Makepath(expandPathString(e, path, printError))
}

func expandPathStringWithPrefix(
	e expander,
	path []string,
	errHandler func(error),
	prefix string,
	pos int,
) string {
	pstr, err := e.ExpandWithPrefix(pathutil.Pathstr(path), prefix, pos)
	handleCompError(err, errHandler)
	return pstr
}

func ExpandPathWithPrefix(
	e expander,
	path []string,
	prefix string,
	pos int,
) []string {
	return pathutil.Makepath(expandPathStringWithPrefix(
		e, path, printError, prefix, pos))
}

func singleCommandComp(ctx *Ctx) (completionText string) {
	return doComplete(ctx, true, defaultcomps, printHelp)
}

func validSingleCommand(ctx *Ctx) error {
	if ctx.Prefix != "" && len(ctx.Args) == 2 || len(ctx.Args) > 2 {
		return fmt.Errorf("Invalid command: %s [%s]", ctx.Args[0], ctx.Args[1])
	}

	return nil
}

func checkValidPath(ctx *Ctx) error {
	path := editPath(ctx.Args[1:])
	cl := ctx.Client
	if len(path) == 0 {
		return nil
	}

	// If user has typed 'space' then TAB after previous keyword, we have no
	// need to look at prefixes.
	if ctx.Prefix == "" && path[len(path)-1] == "" {
		if v, _ := cl.TmplValidatePath(pathutil.Pathstr(
			ExpandPath(ctx.Client, path[:len(path)-1]))); v {
			return nil
		}
	}

	// If prefix is not zero length, we need to take it into account when
	// expanding what the user has typed to allow for mid-word tab completion.
	_, err := cl.ExpandWithPrefix(pathutil.Pathstr(path), ctx.Prefix,
		ctx.CompCurIdx-1)
	//BUG(jhs): String comparison, yuck, we need to figure out how to
	//          get better error information out of configd.
	if err != nil && strings.Contains(err.Error(), "is not valid") {
		return err
	}

	return nil
}

func prefixFilterMap(m map[string]string, pfx string) map[string]string {
	out := make(map[string]string)
	for k, v := range m {
		if strings.HasPrefix(k, pfx) {
			out[k] = v
		}
	}
	return out
}

func prefix(ctx *Ctx) string {
	var pfx string
	if ctx.CompCurWord == "" {
		if len(ctx.Args) == 0 {
			pfx = ""
		} else {
			pfx = ctx.Args[0]
		}
	} else {
		pfx = ctx.Prefix
	}
	return pfx
}

//Used as part of bash hack #1 in doComplete
func makeAmbiguous(compreply []string) bool {
	if len(compreply) == 1 {
		return true
	}
	uniq := make(map[rune]bool)
	for _, rep := range compreply {
		r, _ := utf8.DecodeRuneInString(rep)
		uniq[r] = true
	}
	if len(uniq) == 1 {
		return true
	}
	return false
}

type PrintFn func(*Ctx, map[string]string) string

func printPathHelp(ctx *Ctx, comps map[string]string) string {
	buf := new(bytes.Buffer)
	args := ctx.Args
	path := ExpandPath(ctx.Client, args[1:])
	keys, nckeys := mapkeys("", comps)
	twrite := tabwriter.NewWriter(buf, 8, 0, 1, ' ', 0)
	fmt.Fprintln(twrite, CompHeader)
	for _, name := range nckeys {
		typfx := gettypeprefix(ctx.Client, pathutil.CopyAppend(path, name))
		fmt.Fprintf(twrite, "%s %s\t%s\n", typfx, name, comps[name])
	}
	for i, name := range keys {
		typfx := gettypeprefix(ctx.Client, pathutil.CopyAppend(path, name))
		if i == len(keys)-1 {
			fmt.Fprintf(twrite, "%s %s\t%s", typfx, name, comps[name])
		} else {
			fmt.Fprintf(twrite, "%s %s\t%s\n", typfx, name, comps[name])
		}
	}
	twrite.Flush()
	return buf.String()
}

func printHelp(ctx *Ctx, comps map[string]string) string {
	buf := new(bytes.Buffer)
	keys, nckeys := mapkeys("", comps)
	twrite := tabwriter.NewWriter(buf, 8, 0, 1, ' ', 0)
	fmt.Fprintln(twrite, CompHeader)
	for _, name := range nckeys {
		fmt.Fprintf(twrite, "  %s\t%s\n", name, comps[name])
	}
	for i, name := range keys {
		if i == len(keys)-1 {
			fmt.Fprintf(twrite, "  %s\t%s", name, comps[name])
		} else {
			fmt.Fprintf(twrite, "  %s\t%s\n", name, comps[name])
		}
	}
	twrite.Flush()
	return buf.String()
}

func getCompReply(
	dohelp bool,
	helptext string,
	compreply []string,
) (completionText string) {

	buf := new(bytes.Buffer)
	if dohelp && helptext != "" {
		// If the following Replacer is modified, you may need to give
		// some attention to the one in:
		//   configd/schema/validation_errors.go
		//
		// Escape backslashes and double quotes in help text to
		// ensure they appear correctly post bash processing
		escapedHelp := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(helptext)
		fmt.Fprintf(buf, "echo \"%s\" | %s;", escapedHelp, pager)
	}
	fmt.Fprintf(buf, "COMPREPLY=( %s )", strings.Join(compreply, " "))
	return buf.String()
}

func doComplete(
	ctx *Ctx,
	appendSpace bool,
	comps map[string]string,
	prntFn PrintFn,
) (completionText string) {
	var cur string
	var helptext string
	compsin := comps

	if len := len(ctx.Args); len > 0 && len > ctx.CompCurIdx {
		cur = ctx.Args[ctx.CompCurIdx]
	} else {
		cur = ctx.Prefix
	}

	comps = prefixFilterMap(comps, prefix(ctx))
	compreply := make([]string, 0, len(comps))
	for k, _ := range comps {
		if strings.HasPrefix(k, "<") && strings.HasSuffix(k, ">") {
			continue
		}
		compreply = append(compreply, k)
	}

	/*
	 * By default bash will auto complete the only option upon the first tab
	 * In our system we want it to display the option until the user has hit
	 * the first character so that completion will be consistent for all options
	 * in the system. The following code is a hack around bash's behavior.
	 */
	if cur == "" || (len(compreply) > 0 &&
		strings.HasPrefix(compreply[0], cur)) {
		if prefix(ctx) == "" {
			if len(compreply) == 0 {
				compreply = []string{"\" \" ", "\"\""}
			} else if makeAmbiguous(compreply) {
				compreply = append(compreply, " ")
			}
		}
	} //End of bash hack 1.

	//Not ambiguous gotta append a space
	if len(compreply) == 1 {
		//Our system uses compopt nospace so we have to insert them.
		if appendSpace {
			compreply = []string{compreply[0] + " "}
		}
	}

	/*
	 * Hack around another bash compatibility problem.
	 * We want to show long completions first, followed
	 * by short completions (so the engine can actually fill out words)
	 * then long competions again.

	 * This can confuse the completion engine so we have
	 * to provide some help here and not show the help
	 * text again on the same word as last time.
	 */
	if cur == "" {
		ctx.DoHelp = true
	}
	if len(compreply) == 1 && cur != "" &&
		strings.HasPrefix(compreply[0], cur) {
		ctx.DoHelp = false
	} else if len(compreply) == 1 && ctx.Prefix != "" &&
		strings.HasPrefix(compreply[0], ctx.Prefix) {
		ctx.DoHelp = false
	} else if ctx.DoHelp || len(compreply) == 0 {
		if len(compreply) == 0 {
			ctx.DoHelp = true
			helptext = prntFn(ctx, compsin)
		} else {
			helptext = prntFn(ctx, comps)
		}
		compreply = []string{"\"\"", "\" \""}
	} //End of bash hack 2.

	return getCompReply(ctx.DoHelp, helptext, compreply)
}

func filenameComplete(ctx *Ctx) []string {
	//TODO(jhs): some reasonable form of filename completion.
	//           not as fancy as bash_completion scripts but
	//           better than compgen -f
	return []string{}
}

func loadsaveComp(ctx *Ctx, opname, prep string) (map[string]string, bool) {
	var appendSpace bool = true
	var m map[string]string
	m = defaultcomps
	if ctx.Prefix == "" {
		m = make(map[string]string)
		if opname == "Save" {
			m["<Enter>"] = "(deprecated - 'commit' saves system config file)"
		} else {
			m["<Enter>"] =
				fmt.Sprintf("%s %s system config file", opname, prep)
		}
		m["<file>"] =
			fmt.Sprintf("%s %s file on local machine", opname, prep)
		m["scp://<user>:<passwd>@<host>/<file>"] =
			fmt.Sprintf("%s %s file on remote machine", opname, prep)
		m["ftp://<user>:<passwd>@<host>/<file>"] =
			fmt.Sprintf("%s %s file on remote machine", opname, prep)
		m["http://<user>:<passwd>@<host>/<file>"] =
			fmt.Sprintf("%s %s file on remote machine", opname, prep)
		m["tftp://<host>/<file>"] =
			fmt.Sprintf("%s %s file on remote machine", opname, prep)
	} /*else if strings.HasPrefix(ctx.Args[1], "/") {
		m = make(map[string]string)
		comps := filenameComplete(ctx)
		for _, v := range comps {
			m[v] = fmt.Sprintf("%s %s file on local machine", opname, prep)
		}
		if len(comps) == 1 && isFile(comps[0]) {
			appendSpace = true
		}
	} */
	return m, appendSpace
}

func ri_loadsaveComp(ctx *Ctx, opname, prep string) (map[string]string, bool) {
	var appendSpace bool
	var m map[string]string
	m, appendSpace = loadsaveComp(ctx, opname, prep)
	if ctx.Prefix == "" {
		m[routingInstanceArg] = "Use routing instance for remote connection"
	}
	return m, appendSpace
}

func addRINames(ctx *Ctx, m map[string]string) {
	names, err := ctx.Client.Get(rpc.RUNNING, "routing/routing-instance")
	if err != nil {
		return
	}

	for _, name := range names {
		m[name] = ""
	}
}

func riComp(ctx *Ctx, opname, prep string) (map[string]string, bool) {
	var m map[string]string = defaultcomps
	if ctx.Prefix == "" && ctx.Args[ctx.CompCurIdx-1] == routingInstanceArg {
		m = make(map[string]string)
		addRINames(ctx, m)
		return m, true
	}
	return m, true
}

func loadsaveValid(ctx *Ctx) (err error) {
	if ctx.HasRoutingInstance {
		switch ctx.CompCurIdx {
		case 1:
			break
		case 2:
			break
		case 3:
			break
		default:
			if ctx.Prefix != "" && len(ctx.Args) == 5 || len(ctx.Args) > 5 {
				return fmt.Errorf("Invalid command: %s [%s]",
					strings.Join(ctx.Args[0:], " "), ctx.Args[ctx.CompCurIdx])
			}
		}
	} else {
		switch ctx.CompCurIdx {
		case 1:
			break
		default:
			if ctx.Prefix != "" && len(ctx.Args) == 3 || len(ctx.Args) > 3 {
				return fmt.Errorf("Invalid command: %s [%s]",
					strings.Join(ctx.Args[0:2], " "), ctx.Args[2])
			}
		}
	}
	return nil
}

func firstWordComp(ctx *Ctx) (completionText string) {
	return doComplete(ctx, true, CommandHelps(), printHelp)
}

// rollbackValid - check if rollback command is valid
//
// Format of command is 'rollback <ver> [comment <comment>]'
// Thus, if only 2 args (allow for fewer though should never get that here)
// we have no comment.
// Otherwise we check that we have 'comment <comment>' present and correct
// with no trailing text.
// Validation of the revision number is done elsewhere (server/dispatcher.go).
func rollbackValid(ctx *Ctx) error {
	if len(ctx.Args) <= 2 {
		return nil
	}

	args := removeTrailingEmptyArgument(ctx.Args)
	return validateCommentIfAny(args, 2, ctx.Prefix)
}

func rollbackComp(ctx *Ctx) (completionText string) {
	var m map[string]string
	m = defaultcomps
	switch ctx.CompCurIdx {
	case 1: // <revision-number>
		m = map[string]string{
			"<N>": "Rollback to revision N",
		}
		commits, _ := ctx.Client.GetCommitLog()
		for s, v := range commits {
			m[s] = v
		}
	case 2: // optional comment keyword
		m = map[string]string{
			"<Enter>": "Execute the current command",
			"comment": "Comment for commit log",
		}
	case 3: // comment argument
		m = map[string]string{
			"<text>": "Comment for commit log",
		}
	default:
		m = defaultcomps
	}
	return doComplete(ctx, true, m, printHelp)
}

// processCancelCommitCmd takes a Ctx and validates that it is a valid
// cancel-commit command. The command is:
//
//   cancel-commit [{ force | persist-id <text> }] [comment <text>]
//
// It returns a populated cmdDefs, the last keyword matched and an error
// if the command is invalid.
func processCancelCommitCmd(ctx *Ctx) (cmdDefs, string, error) {
	cmds := cmdDefs{
		"force": {needArg: false, atStart: true,
			help: "Force cancelling of pending confirmed commit"},
		"comment": {needArg: true,
			help: "Comment for the commit log"},
		"persist-id": {needArg: true, atStart: true,
			help: "Persist-id of confirmed commit"},
	}

	args := removeTrailingEmptyArgument(ctx.Args)
	lastMatch, err := validateKeywordsAndArguments(args, ctx.CompCurIdx, ctx.Prefix, cmds)
	return cmds, lastMatch, err
}

// cancelcommitValid - check if cancel-commit command is valid
//
// Format of command is
//      'cancel-commit [force | persist-id <persist-id>] [comment <comment>]'
func cancelcommitValid(ctx *Ctx) error {
	if len(ctx.Args) == 1 {
		return nil
	}

	_, _, err := processCancelCommitCmd(ctx)

	return err

}

func cancelcommitComp(ctx *Ctx) (completionText string) {
	m := make(map[string]string, 0)

	commands, match, _ := processCancelCommitCmd(ctx)
	if match != "" {
		// get argument for last matched keyword if required
		def := commands[match]
		if (def.needArg && !def.argpresent &&
			def.pos != len(ctx.Args)-1) ||
			def.pos == len(ctx.Args)-2 {
			m["<text>"] = def.help
		}
	}

	if len(m) == 0 {
		m["<Enter>"] = "Cancel pending confirmed commit"
		for keyword, def := range commands {
			// Include keywords not already included
			// with it's argument.
			// only include an atStart keyword if at
			// begining of command
			if (!def.present || def.pos == len(ctx.Args)-1 || (def.needArg && !def.argpresent)) &&
				(!def.atStart || len(ctx.Args) <= 2) {
				m[keyword] = def.help
			}
		}

	}

	return doComplete(ctx, true, m, printHelp)
}

func confirmValid(ctx *Ctx) error {
	if len(ctx.Args) == 1 {
		return nil
	}

	args := removeTrailingEmptyArgument(ctx.Args)
	return validatePersistIdIfAny(args, 1, ctx.Prefix)
}

func confirmComp(ctx *Ctx) (completionText string) {
	var m map[string]string
	switch ctx.CompCurIdx {
	case 1:
		m = map[string]string{
			"<Enter>":    "Confirm acceptance of running configuration",
			"persist-id": "Persist-id of pending confirmed commit",
		}
	case 2:
		m = map[string]string{
			"<text>": "Persist-id of pending confirmed commit",
		}
	default:
		m = defaultcomps
	}
	return doComplete(ctx, true, m, printHelp)
}

type cmdDef struct {
	needArg    bool
	argpresent bool
	argval     string
	present    bool
	pos        int
	help       string
	atStart    bool
}

type cmdDefs map[string]cmdDef

func (c cmdDefs) assignArgument(keyword, value string) {
	if def, ok := c[keyword]; ok {
		def.argpresent = true
		def.argval = value
		c[keyword] = def
	}
}

func (c cmdDefs) keywordPresent(keyword string, pos int) bool {
	if def, ok := c[keyword]; ok {
		def.pos = pos
		def.present = true
		c[keyword] = def
		return def.needArg
	}
	return false
}

func (c cmdDefs) matchKeyword(keywordPos, idx int, prefix, val string) string {
	match := ""
	for keyword, def := range c {
		// Keyword matches if permitted at current position or
		// if it is a partial match of the current prefix
		if (!def.atStart || idx == 0) && (strings.HasPrefix(keyword, val) ||
			(prefix != "" && keywordPos == idx+1 &&
				strings.HasPrefix(keyword, prefix))) {
			// We have a match
			match = keyword
			break
		}
	}
	return match
}

// validateKeywordsAndArguments takes a CLI args slice,
// and validates that it is a valid command, as defined
// by the cmdDefs.
// args is the current CLI to validate
// keywordPos is the location on the cursor on the CLI
// prefix is the current partial keyword at keywordPos
// cmds is the command definition
//
// Returns are a string, giving the last matched keyword
// and an error if the command is not valid
func validateKeywordsAndArguments(args []string,
	keywordPos int,
	prefix string,
	cmds cmdDefs) (string, error) {

	match := ""
	needarg := false

	for idx, val := range args[1:] {
		if needarg {
			// Get argument for currently matched keyword
			cmds.assignArgument(match, val)
			needarg = false
		} else {
			match = cmds.matchKeyword(keywordPos, idx, prefix, val)

			if match == "" {
				return "", fmt.Errorf("Invalid command: %s [%s]\n",
					strings.Join(args[0:idx+1], " "), args[idx+1])
			}
			needarg = cmds.keywordPresent(match, idx+1)
			if !needarg {
				match = ""
			}
		}
	}

	return match, nil
}

// Check for a keyword, and if present, check there is only a single argument
// following it.
func validateArgumentIfAny(args []string, keywordPos int, prefix, keyword string) error {

	// Command finishes at <keyword> (or at least at point at which that is
	// the only valid command.  For tab completion usage, prefix will be set,
	// and the first HasPrefix() call will be in play.  For 'run' usage,
	// prefix is set to __noncomp__ and so we need to check the full keyword
	// provided.  Note that we cannot use the second check for tab completion
	// as that will fail on mid-word tab completion where we have the likes of
	// 'keyXXX' as the full keyword.
	if len(args) == keywordPos+1 {
		if !strings.HasPrefix(keyword, prefix) &&
			!strings.HasPrefix(keyword, args[keywordPos]) {
			return fmt.Errorf("Invalid command: %s [%s]",
				strings.Join(args[0:keywordPos], " "), args[keywordPos])
		}
		// No argument provided, but the keyword is present and correct.
		return nil
	}

	if len(args) > keywordPos+1 &&
		strings.Index(keyword, args[keywordPos]) != 0 {
		return fmt.Errorf("Invalid command: %s [%s]",
			strings.Join(args[0:keywordPos], " "), args[keywordPos])
	}

	if len(args) >= keywordPos+3 {
		return fmt.Errorf("Invalid command: %s [%s]",
			strings.Join(args[0:keywordPos+2], " "),
			args[keywordPos+2])
	}

	return nil
}

func validateCommentIfAny(args []string, keywordPos int, prefix string) error {
	return validateArgumentIfAny(args, keywordPos, prefix, "comment")
}

func validatePersistIdIfAny(args []string, keywordPos int, prefix string) error {
	return validateArgumentIfAny(args, keywordPos, prefix, "persist-id")
}

func validateForceIfAny(args []string, keywordPos int, prefix string) error {
	return validateArgumentIfAny(args, keywordPos, prefix, "force")
}

// If last argument is a space, remove it rather than constantly having
// to check prefix param to determine if we are interested in last argument.
func removeTrailingEmptyArgument(args []string) []string {
	argLen := len(args)
	if args[argLen-1] == "" {
		return args[:argLen-1]
	}
	return args
}

// Command format is: commit-confirm <timeout> [comment <comment>]
func commitConfValid(ctx *Ctx) error {
	if len(ctx.Args) == 1 {
		return fmt.Errorf("Timeout must be specified for commit-confirm")
	}

	args := removeTrailingEmptyArgument(ctx.Args)

	if len(args) >= 2 {
		timeout, err := strconv.Atoi(args[1])
		if err != nil || timeout <= 0 {
			return fmt.Errorf("Invalid timeout: %s", args[1])
		}
	}

	return validateCommentIfAny(args, 2, ctx.Prefix)
}

func commitConfComp(ctx *Ctx) (completionText string) {
	var m map[string]string
	switch ctx.CompCurIdx {
	case 1:
		m = map[string]string{
			"<value>": "Time (minutes) to issue 'confirm' before automatic rollback",
		}
	case 2:
		m = map[string]string{
			"<Enter>": "Commit working configuration subject to confirmation",
			"comment": "Comment for commit log",
		}
	case 3:
		m = map[string]string{
			"<text>": "Comment for the commit log",
		}
	default:
		m = defaultcomps
	}
	return doComplete(ctx, true, m, printHelp)
}

func commitValid(ctx *Ctx) error {
	if len(ctx.Args) == 1 {
		return nil
	}

	args := removeTrailingEmptyArgument(ctx.Args)
	return validateCommentIfAny(args, 1, ctx.Prefix)
}

func commitComp(ctx *Ctx) (completionText string) {
	var m map[string]string
	switch ctx.CompCurIdx {
	case 1:
		m = map[string]string{
			"<Enter>": "Commit working configuration",
			"comment": "Comment for commit log",
		}
	case 2:
		m = map[string]string{
			"<text>": "Comment for the commit log",
		}
	default:
		m = defaultcomps
	}
	return doComplete(ctx, true, m, printHelp)
}

func compareValid(ctx *Ctx) error {
	//TODO(jhs): there is a pattern to validate functions that I think
	//           we can tease out in the future so they are not so brittle
	//           to change. Since these commands don't change often this
	//           is good enough for now.

	if len(ctx.Args) == 1 {
		return nil
	}
	if !ctx.HasConfigMgmt {
		return validSingleCommand(ctx)
	}
	switch ctx.CompCurIdx {
	case 1:
		if ctx.Prefix == "" {
			break
		} else if strings.HasPrefix("saved", ctx.Prefix) {
			break
		} else if _, err := strconv.Atoi(ctx.Prefix); err == nil {
			break
		} else {
			return fmt.Errorf("Invalid command: %s [%s]",
				ctx.Args[0], ctx.Prefix)
		}
	case 2:
		if strings.HasPrefix("saved", ctx.Args[1]) {
			if ctx.Prefix != "" {
				return fmt.Errorf("Invalid command: %s [%s]",
					strings.Join(ctx.Args[0:2], " "), ctx.Prefix)
			}
			break
		} else if _, err := strconv.Atoi(ctx.Args[1]); err == nil {
			if _, err := strconv.Atoi(ctx.Prefix); ctx.Prefix != "" && err != nil {
				return fmt.Errorf("Invalid command: %s [%s]",
					strings.Join(ctx.Args[0:2], " "), ctx.Prefix)
			}
			break
		} else {
			return fmt.Errorf("Invalid command: %s [%s]",
				ctx.Args[0], ctx.Args[1])
		}
	default:
		if strings.HasPrefix("saved", ctx.Args[1]) {
			if ctx.Prefix != "" && len(ctx.Args) == 3 || len(ctx.Args) > 3 {
				return fmt.Errorf("Invalid command: %s [%s]",
					strings.Join(ctx.Args[0:2], " "), ctx.Args[2])
			}
			break
		}
		if ctx.Prefix != "" && len(ctx.Args) == 4 || len(ctx.Args) > 4 {
			return fmt.Errorf("Invalid command: %s [%s]",
				strings.Join(ctx.Args[0:3], " "), ctx.Args[3])
		}
	}

	return nil
}

func compareComp(ctx *Ctx) (completionText string) {
	var m map[string]string
	m = defaultcomps
	if !ctx.HasConfigMgmt {
		return doComplete(ctx, true, m, printHelp)
	}
	commits, _ := ctx.Client.GetCommitLog()
	if ctx.CompCurIdx == 1 {
		m = map[string]string{
			"<Enter>": "Compare candidate with running",
			"saved":   "Compare candidate with saved",
			"<N>":     "Compare candidate with revision N",
			"<N> <M>": "Compare revision N with revision M",
		}
		for i, v := range commits {
			m[i] = v
		}
	} else if ctx.CompCurIdx == 2 {
		if strings.HasPrefix("saved", ctx.Args[1]) {
			m = map[string]string{
				"<Enter>": "Compare candidate with saved",
			}
		} else {
			m = map[string]string{
				"<Enter>": "Compare candidate with revision N",
				"<M>":     "Compare revision N with revision M",
			}
			for i, v := range commits {
				m[i] = v
			}
		}
	}
	return doComplete(ctx, true, m, printHelp)
}

func editPathLength(args []string) ([]string, int) {
	epath := pathutil.Makepath(os.Getenv("VYATTA_EDIT_LEVEL"))
	return append(epath, args...), len(epath)
}

func editPath(args []string) []string {
	out, _ := editPathLength(args)
	return out
}

func pathComp(ctx *Ctx) (completionText string) {
	epath, elen := editPathLength(ctx.Args[1:ctx.CompCurIdx])
	ctx.Args = append(ctx.Args[0:1], ExpandPath(ctx.Client, epath)...)
	ctx.CompCurIdx = ctx.CompCurIdx + elen
	m := getcompletions(ctx.Client, ctx.Args)
	return doComplete(ctx, true, m, printPathHelp)
}

func exitComp(ctx *Ctx) (completionText string) {
	m := defaultcomps
	if ctx.CompCurIdx == 1 {
		m = map[string]string{
			"<Enter>": defaultcomps["<Enter>"],
			"discard": "Discard any changes",
		}
	}
	return doComplete(ctx, true, m, printHelp)
}

func exitValid(ctx *Ctx) error {
	switch ctx.CompCurIdx {
	case 1:
		if !strings.HasPrefix("discard", ctx.Prefix) {
			return fmt.Errorf("Invalid command: %s [%s]",
				ctx.Args[0], ctx.Prefix)
		}
	case 2:
		if ctx.Prefix != "" {
			return fmt.Errorf("Invalid command: %s [%s]",
				strings.Join(ctx.Args[0:2], " "), ctx.Prefix)
		}
	default:
		if ctx.Prefix != "" && len(ctx.Args) == 3 || len(ctx.Args) > 3 {
			return fmt.Errorf("Invalid command: %s [%s]",
				strings.Join(ctx.Args[0:2], " "), ctx.Args[2])
		}
	}

	return nil
}

func loadComp(ctx *Ctx) (completionText string) {
	var appendSpace bool = true
	m := defaultcomps
	if ctx.HasRoutingInstance {
		switch ctx.CompCurIdx {
		case 1:
			m, appendSpace = ri_loadsaveComp(ctx, "Load", "from")
		case 2:
			m, appendSpace = riComp(ctx, "Load", "from")
		case 3:
			m, appendSpace = loadsaveComp(ctx, "Load", "from")
		}
	} else {
		if ctx.CompCurIdx == 1 {
			m, appendSpace = loadsaveComp(ctx, "Load", "from")
		}
	}
	return doComplete(ctx, appendSpace, m, printHelp)
}
func loadkeyComp(ctx *Ctx) (completionText string) {
	var appendSpace bool = true
	m := defaultcomps
	if ctx.CompCurIdx == 1 {
		us, err := ctx.Client.Get(rpc.CANDIDATE, "/system/login/user")
		handleError(err)
		m = make(map[string]string)
		for _, u := range us {
			m[u] = ""
		}
	}
	if ctx.HasRoutingInstance {
		switch ctx.CompCurIdx {
		case 2:
			m, appendSpace = ri_loadsaveComp(ctx, "Load", "from")
		case 3:
			m, appendSpace = riComp(ctx, "Load", "from")
		case 4:
			m, appendSpace = loadsaveComp(ctx, "Load", "from")
		}
	} else {
		if ctx.CompCurIdx == 2 {
			m, appendSpace = loadsaveComp(ctx, "Load", "from")
		}
	}
	return doComplete(ctx, appendSpace, m, printHelp)
}
func loadKeyValid(ctx *Ctx) error {
	if ctx.HasRoutingInstance {
		switch ctx.CompCurIdx {
		case 1:
			break
		case 2:
			break
		case 3:
			break
		case 4:
			break
		default:
			if ctx.Prefix != "" && len(ctx.Args) == 6 || len(ctx.Args) > 6 {
				return fmt.Errorf("Invalid command: %s [%s]",
					strings.Join(ctx.Args[0:], " "), ctx.Args[ctx.CompCurIdx])
			}
		}
	} else {
		switch ctx.CompCurIdx {
		case 1:
			break
		case 2:
			break
		default:
			if ctx.Prefix != "" && len(ctx.Args) == 4 || len(ctx.Args) > 4 {
				return fmt.Errorf("Invalid command: %s [%s]",
					strings.Join(ctx.Args[0:3], " "), ctx.Args[3])
			}
		}
	}

	return nil
}

func mergeComp(ctx *Ctx) (completionText string) {
	var m map[string]string
	if ctx.CompCurIdx == 1 {
		m = make(map[string]string)
		m["<file>"] = fmt.Sprintf("Load from file on local machine")
	} else {
		m = defaultcomps
	}
	return doComplete(ctx, true, m, printHelp)
}

func mergeValid(ctx *Ctx) (err error) {
	switch ctx.CompCurIdx {
	case 1:
		break
	default:
		if len(ctx.Args) < 2 {
			return fmt.Errorf("Invalid command: merge requires a path argument")
		}

		if len(ctx.Args) >= 3 {
			return fmt.Errorf("Invalid command: %s [%s]",
				strings.Join(ctx.Args[0:2], " "), ctx.Args[2])
		}
	}
	return nil
}

func runComp(ctx *Ctx) (completionText string) {
	//TODO(jhs): Op mode completion needs to be reconciled with the way config completion
	//           works in order for them to be easily composable. Leaving this stub
	//           for documentation.
	return ""
}
func saveComp(ctx *Ctx) (completionText string) {
	var appendSpace bool = true
	m := defaultcomps
	if ctx.HasRoutingInstance {
		switch ctx.CompCurIdx {
		case 1:
			m, appendSpace = ri_loadsaveComp(ctx, "Save", "to")
		case 2:
			m, appendSpace = riComp(ctx, "Save", "to")
		case 3:
			m, appendSpace = loadsaveComp(ctx, "Save", "to")
		}
	} else {
		if ctx.CompCurIdx == 1 {
			m, appendSpace = loadsaveComp(ctx, "Save", "to")
		}
	}
	return doComplete(ctx, appendSpace, m, printHelp)
}
