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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/danos/configd/rpc"
	"github.com/danos/utils/pathutil"
)

const notspec = "Must specify a path to %s"
const pager = "${VYATTA_PAGER:-cat}"
const editenv = "VYATTA_EDIT_LEVEL"
const configDir = "/config"
const configBootPath = configDir + "/config.boot"
const routingInstanceArg = "routing-instance"

func writeOutput(w io.Writer, out interface{}) {
	fmt.Fprintf(w, "\n\n  %v\n", out)
}

func printOutput(out string) {
	writeOutput(os.Stdout, strings.Replace(out, "\n", "  \n", -1))
}

func printError(err error) {
	writeOutput(os.Stderr, err)
}

func sessionChanged(ctx *Ctx) bool {
	ret, err := ctx.Client.SessionChanged()
	handleError(err)
	return ret
}

func handleError(err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "\n  %s\n\n", strings.Replace(err.Error(), "\n", "\n  ", -1))
	os.Exit(1)
}

func handleNoError(msg string) {
	fmt.Fprintf(os.Stderr, "\n  %s\n\n", strings.Replace(msg, "\n", "\n  ", -1))
}

func handleErrorNoIndent(err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "\n%s\n\n", err)
	os.Exit(1)
}

func handleRollbackError(err error) {
	if err == nil {
		return
	}
	logRollbackEvent(fmt.Sprintf("Failed with error: %s", err))
	handleError(err)
}

func shellOutExecute(snippit string, env ...string) {
	bashCmd := exec.Command("bash", "-c", snippit)
	bashCmd.Env = append(os.Environ(), env...)
	bashOut, err := bashCmd.Output()
	if err != nil {
		handleError(err)
	} else if len(bashOut) > 0 {
		printOutput(string(bashOut))
	}
}

func doSnippit(ctx *Ctx, snippit string, env ...string) {
	doSnippitAndContinue(ctx, snippit, env...)
	os.Exit(0)
}

func doSnippitAndContinue(ctx *Ctx, snippit string, env ...string) {
	if ctx.Print {
		var buf bytes.Buffer
		for _, ent := range env {
			parts := strings.SplitN(ent, "=", 2)
			fmt.Fprintf(&buf, "%s=\"%s\" ", parts[0], parts[1])
		}
		fmt.Fprint(&buf, snippit)
		fmt.Print(buf.String())
	} else {
		shellOutExecute(snippit, env...)
	}
}

func validateConfirmPersistIdIfAny(
	ctx *Ctx,
	persistidKeywordPos int,
) (persistid string) {

	if len(ctx.Args) == persistidKeywordPos+1 {
		handleError(fmt.Errorf("Please provide persisit-id."))
	}

	if len(ctx.Args) > persistidKeywordPos+1 {
		persistid = ctx.Args[persistidKeywordPos+1]
	}

	return persistid
}

func confirmRun(ctx *Ctx) {
	var out string
	var err error

	persistid := validateConfirmPersistIdIfAny(ctx, 1)

	switch persistid {
	case "":
		out, err = ctx.Client.Confirm()
	default:
		out, err = ctx.Client.ConfirmPersistId(persistid)
	}
	handleErrorNoIndent(err)
	logRollbackEvent("Commit confirmed.")
	if out != "" {
		doSnippit(ctx, fmt.Sprintf("echo \"%s\"\n", out))
	}
}

// A commit or rollback after a commit-confirm, within the timeout period,
// cancels the pending rollback.  Ignore errors - we want commit / rollback
// to happen regardless.
func confirmSilentRun(ctx *Ctx) {
	ctx.Client.ConfirmSilent()
	logRollbackEvent(
		"Commit/Rollback operation - any pending rollback cancelled.")
}

// Need to validate we got comment text, if comment keyword was
// specified.
func validateCommitCommentIfAny(
	ctx *Ctx,
	commentKeywordPos int,
) (comment string) {

	if len(ctx.Args) == commentKeywordPos+1 {
		handleError(fmt.Errorf("Please provide comment."))
	}

	if len(ctx.Args) > commentKeywordPos+1 {
		comment = ctx.Args[commentKeywordPos+1]
	}

	return comment
}

func commitConfRun(ctx *Ctx) {
	comment := validateCommitCommentIfAny(ctx, 2)

	// Find timeout.  Params have been validated already.
	mins, _ := strconv.Atoi(ctx.Args[1])
	commitRunInternal(ctx, comment, mins)
}

func commitRun(ctx *Ctx) {
	comment := validateCommitCommentIfAny(ctx, 1)

	confirmSilentRun(ctx)

	commitRunInternal(ctx, comment, 0 /* no timeout */)
	os.Exit(0)
}

func isCommitDebugOn() bool {
	return os.ExpandEnv("$COMMIT_DEBUG") != ""
}

func commitRunInternal(ctx *Ctx, comment string, confirmTimeout int) {
	if !sessionChanged(ctx) {
		handleError(errors.New("No configuration changes to commit"))
	}
	debug := isCommitDebugOn()
	var out string
	var err error
	if confirmTimeout != 0 {
		out, err = ctx.Client.CommitConfirm(comment, debug, confirmTimeout)
		handleErrorNoIndent(err)
		// Only log once timer set via RPC, and no error returned.
		logRollbackEvent(
			fmt.Sprintf("Commit will rollback in %d minutes unless confirmed.",
				confirmTimeout))
	} else {
		out, err = ctx.Client.Commit(comment, debug)
		handleErrorNoIndent(err)
	}
	if out != "" {
		doSnippitAndContinue(ctx, fmt.Sprintf("echo \"%s\"\n", out))
	}

	// commit = save ...
	saveRunInternal(ctx, []string{})
}

var slog *log.Logger

func logRollbackEvent(msg string) {
	if slog == nil {
		var err error
		slog, err = syslog.NewLogger(syslog.LOG_WARNING, 0)
		if err != nil {
			return
		}
	}

	// Log only the first non-blank line
	for _, s := range strings.Split(msg, "\n") {
		if s != "" {
			slog.Println("Rollback: " + s)
			break
		}
	}
}

func getCmdArg(cmds cmdDefs, arg string) (bool, string) {

	if def, ok := cmds[arg]; ok {
		present := def.present
		val := def.argval
		return present, val
	}

	return false, ""
}

func cancelcommitRun(ctx *Ctx) {
	cmds, _, _ := processCancelCommitCmd(ctx)

	_, persistid := getCmdArg(cmds, "persist-id")
	_, comment := getCmdArg(cmds, "comment")
	force, _ := getCmdArg(cmds, "force")

	out, err := ctx.Client.CancelCommit(comment, persistid, force, isCommitDebugOn())

	if out != "" {
		doSnippit(ctx, fmt.Sprintf("echo \"%s\"\n", out))
	}
	handleError(err)
	os.Exit(0)
}

func rollbackRun(ctx *Ctx) {
	var comment string
	if len(ctx.Args[1:]) == 0 {
		handleError(errors.New("Missing argument, usage is:\n  rollback <revision> [comment <text>]"))
	}

	if len(ctx.Args[1:]) >= 3 {
		if ctx.Args[2] == "comment" {
			comment = ctx.Args[3]
		}
	}

	out, err := ctx.Client.Rollback(ctx.Args[1], comment, isCommitDebugOn())
	if out != "" {
		doSnippit(ctx, fmt.Sprintf("echo \"%s\"\n", out))
	}
	handleError(err)
	os.Exit(0)
}

func compareRun(ctx *Ctx) {
	var diff string
	var msg string
	var err error

	if len(ctx.Args) == 1 {
		diff, err = ctx.Client.CompareSessionChanges()
	} else if len(ctx.Args) > 2 {
		diff, err = ctx.Client.CompareConfigRevisions(ctx.Args[1], ctx.Args[2])
		if diff == "" {
			msg = fmt.Sprintf("No changes between revision %v and revision "+
				"%v configurations", ctx.Args[1], ctx.Args[2])
		}
	} else {
		diff, err = ctx.Client.CompareConfigRevisions("session", ctx.Args[1])
		if diff == "" && ctx.Args[1] != "saved" {
			msg = fmt.Sprintf("No changes between working and revision "+
				"%v configurations", ctx.Args[1])
		}
	}
	handleError(err)

	if diff != "" {
		doSnippit(ctx, fmt.Sprintf("echo -n \"%s\" | %s", escapeConfig(diff), pager))
	} else if msg != "" {
		doSnippit(ctx, fmt.Sprintf("echo \"%s\"\n", msg))
	}
	os.Exit(0)
}

func deleteRun(ctx *Ctx) {
	if len(ctx.Args[1:]) == 0 {
		handleError(fmt.Errorf(notspec, "delete"))
	}
	handleError(ctx.Client.Delete(expandPathString(ctx.Client, editPath(ctx.Args[1:]), handleError)))
	os.Exit(0)
}

func discardRun(ctx *Ctx) {
	handleError(ctx.Client.Discard())
	os.Exit(0)
}

func doEditSnippit(ctx *Ctx, path []string) {
	const editFmt = "export %s='%s'; export PS1='[%s]\\n\\u@\\h# ';"

	var promptstr = func(path []string) string {
		var buf = new(bytes.Buffer)
		fmt.Fprint(buf, "edit")
		if len(path) == 0 {
			return buf.String()
		}
		for _, elem := range path {
			fmt.Fprintf(buf, " %s", elem)
		}
		return buf.String()
	}

	pathstr := "/"
	if len(path) > 0 {
		pathstr = pathutil.Pathstr(path)
	}

	doSnippit(ctx,
		fmt.Sprintf(editFmt, editenv, pathstr, promptstr(path)))
}

func isListKey(tmpl map[string]string) bool {
	//TODO: we need to figure out a better api for retreiving this information
	return tmpl["is_value"] == "1" && tmpl["tag"] == "1"
}

func isValue(tmpl map[string]string) bool {
	return tmpl["is_value"] == "1"
}

func editRun(ctx *Ctx) {
	var isTypeless = func(tmpl map[string]string) bool {
		return tmpl["type"] == ""
	}

	client := ctx.Client
	path := ExpandPath(client, editPath(ctx.Args[1:]))
	tmpl, err := client.TmplGet(pathutil.Pathstr(path))
	handleError(err)
	if !isListKey(tmpl) && !isTypeless(tmpl) {
		handleError(errors.New(
			"The \"edit\" command cannot be issued at the specified level",
		))
	}
	ok, err := client.Exists(rpc.CANDIDATE, pathutil.Pathstr(path))
	handleError(err)
	if !ok {
		client.Set(pathutil.Pathstr(path))
	}
	doEditSnippit(ctx, path)
}

func exitRun(ctx *Ctx) {
	var discard, changed bool
	if len(ctx.Args) > 1 {
		if strings.HasPrefix("discard", ctx.Args[1]) {
			if sessionChanged(ctx) {
				discard = true
			}
		}
	}
	changed = sessionChanged(ctx)
	snippit := `
		if ! cli-shell-api editLevelAtRoot; then
			reset_edit_level
			return 0
		fi
		if %t; then
			if %t; then
				echo "Cannot exit: configuration modified."
       			echo "Use 'exit discard' to discard the changes and exit."
				return 1
			fi
		fi
		builtin exit
	`
	doSnippit(ctx, fmt.Sprintf(snippit, changed, !discard))
}

// Parse arguments for load and save (and loadkey) commands
// args is expected to be one of:
//   {"<uri>"}
//   {"routing-instance", "<name>", "<uri>"}
func parseCfgMgmtCmdArgs(args []string, usage string) (string, string) {
	var uri, routingInstance string

	if len(args) == 3 && args[0] == routingInstanceArg {
		routingInstance = args[1]
		uri = args[2]
	} else if len(args) == 1 {
		uri = args[0]
	} else {
		handleError(fmt.Errorf("Invalid arguments: %v\nUsage: %v",
			strings.Join(args, " "), usage))
	}

	// Determine if URI is local or remote by looking for scheme eg. scp://
	remote, err := regexp.MatchString(`^\w+://.+`, uri)
	handleError(err)
	if !remote {
		// Expand relative path
		if !strings.HasPrefix(uri, "/") {
			uri = configDir + "/" + uri
		}
	}
	return uri, routingInstance
}

func loadRun(ctx *Ctx) {
	const usage = "load [routing-instance <name>] <source>"

	if sessionChanged(ctx) {
		handleError(fmt.Errorf("%s\n%s",
			"Cannot load: configuration modified.",
			"Commit or discard the changes before loading a config file.",
		))
	}

	var source, routingInstance string
	if len(ctx.Args) < 2 {
		source = configBootPath
	} else {
		source, routingInstance = parseCfgMgmtCmdArgs(ctx.Args[1:], usage)
	}

	buf := new(bytes.Buffer)
	err := ctx.Client.LoadFrom(source, routingInstance)
	if err != nil {
		// End errors with a consistent double newline
		// This ensures a single blank line between the error message and
		// any subsequent messages printed below.
		fmt.Fprint(buf, strings.TrimRight(err.Error(), "\n")+"\n\n")
	}

	sessionHasChanged := sessionChanged(ctx)

	// Print a confirmation message if there was no error, or if there was an
	// error but changes were made to the candidate configuration. This mimics
	// the behaviour of the old load script.
	if err == nil || sessionHasChanged {
		fmt.Fprint(buf, "Configuration loaded from '"+source+"'")
	}

	if sessionHasChanged {
		fmt.Fprintln(buf, " will replace the existing configuration")
		fmt.Fprint(buf, "Use 'compare' to view the changes, or 'commit' to make them active")
	} else {
		fmt.Fprint(buf, "\nNo configuration changes to commit")
	}

	if err != nil {
		handleError(errors.New(buf.String()))
	}

	doSnippit(ctx, "echo -e \""+buf.String()+"\"")
}
func loadkeyRun(ctx *Ctx) {
	const usage = "loadkey <user> [routing-instance <name>] <source>"

	if len(ctx.Args) < 3 {
		handleError(fmt.Errorf("Invalid command, usage is:\n  " + usage))
	}

	if sessionChanged(ctx) {
		handleError(fmt.Errorf("%s\n%s",
			"Cannot load key: configuration modified.",
			"Commit or discard the changes before loading a key.",
		))
	}

	user := ctx.Args[1]
	source, routingInstance := parseCfgMgmtCmdArgs(ctx.Args[2:], usage)

	out, err := ctx.Client.LoadKeys(user, source, routingInstance)
	handleError(err)
	if out != "" {
		doSnippit(ctx, fmt.Sprintf("echo \"%s\"", out))
	}
	os.Exit(0)
}

func mergeRun(ctx *Ctx) {
	os.Setenv(editenv, "")
	ok, errOrWarn := ctx.Client.MergeReportWarnings(
		strings.Join(ctx.Args[1:], " "))
	if !ok {
		handleError(errOrWarn)
		return
	}
	if errOrWarn != nil {
		handleNoError(errOrWarn.Error())
	}
	os.Exit(0)
}

func encodeOpcArgs(ctx *Ctx, args []string) string {
	encArgs := new(bytes.Buffer)
	type opcArgs struct {
		Args []string `json:"args"`
	}
	enc := json.NewEncoder(encArgs)
	err := enc.Encode(&opcArgs{Args: args})
	if err != nil {
		return ""
	}
	out := strings.TrimSpace(encArgs.String())
	if ctx.Print {
		out = strings.Replace(out, "\"", "\\\"", -1)
	}
	return out
}

func runRun(ctx *Ctx) {
	args := ctx.Args[1:]
	if len(args) == 0 {
		handleError(errors.New("Incomplete command: run"))
	}
	if strings.HasPrefix(args[0], "/") {
		doSnippit(ctx, strings.Join(args, " "))
	} else if strings.HasPrefix("set", args[0]) {
		doSnippit(ctx, fmt.Sprint("_vyatta_op_run \"${@:2}\""))
	} else if strings.HasPrefix("show", args[0]) {
		doSnippit(ctx, fmt.Sprintf("/opt/vyatta/bin/opc -op run-from-env | %s", pager),
			fmt.Sprintf("OPC_ARGS=%s", encodeOpcArgs(ctx, args)))
	} else {
		doSnippit(ctx, "/opt/vyatta/bin/opc -op run-from-env",
			fmt.Sprintf("OPC_ARGS=%s", encodeOpcArgs(ctx, args)))
	}
}

func saveRun(ctx *Ctx) {
	if len(ctx.Args) == 1 {
		handleNoError(
			"'commit' saves configuration.  This command has no effect")
		os.Exit(0)
	}
	saveRunInternal(ctx, ctx.Args[1:])
	os.Exit(0)
}

func saveRunInternal(ctx *Ctx, args []string) {
	buf := new(bytes.Buffer)
	if sessionChanged(ctx) {
		fmt.Fprintln(buf, "echo \"Warning: you have uncommitted changes that will not be saved.\"")
	}
	if len(args) == 0 {
		handleError(ctx.Client.Save(configBootPath))
	} else {
		const usage = "save [routing-instance <name>] <destination>"
		dest, routingInstance := parseCfgMgmtCmdArgs(args, usage)
		handleError(ctx.Client.SaveTo(dest, routingInstance))
		fmt.Fprintln(buf, "echo \"Configuration saved to '"+dest+"'\"")
	}
	handleError(ctx.Client.SessionMarkSaved())
	doSnippitAndContinue(ctx, buf.String())
}

func setRun(ctx *Ctx) {
	if len(ctx.Args[1:]) == 0 {
		handleError(fmt.Errorf(notspec, "set"))
	}
	path := expandPathString(ctx.Client, editPath(ctx.Args[1:]), handleError)
	tmpl, err := ctx.Client.TmplGet(path)
	handleError(err)
	if !isValue(tmpl) && isSecret(ctx.Client, path) {
		doSnippit(ctx, fmt.Sprintf("cfgcli -action setSecret %s", path))
	}
	out, err := ctx.Client.Set(path)
	handleError(err)
	if out != "" {
		doSnippit(ctx, fmt.Sprintf("echo \"%s\"\n", out))
	} else {
		os.Exit(0)
	}
}

// escapeConfig - escape config for show (and other) commands.
// In some cases, config destined for the console needs to be (re)escaped
// so it appears correctly.  Specifically, '\', '"' and '$' are escaped here.
//
// Note that the order of replacement matters - escape '\' first so we don't
// re-escape backslashes added to escape other characters!
func escapeConfig(in string) string {
	return strings.NewReplacer(
		`\`, `\\`, `"`, `\"`, `$`, `\$`).
		Replace(in)
}

func showRun(ctx *Ctx) {
	if err := checkValidPath(ctx); err != nil {
		handleError(err)
	}
	path := expandPathString(ctx.Client, editPath(ctx.Args[1:]), printError)
	out, err := ctx.Client.ShowConfigWithContextDiffs(path, ctx.All)
	handleError(err)
	if out != "" {
		// Output from ShowConfigWithContextDiffs() would look correct if
		// printed as-is.  However, by the time it has all gone through
		// doSnippit() the escaping is wrong.
		doSnippit(ctx, fmt.Sprintf("echo -n \"%s\" | %s",
			escapeConfig(out), pager))
	}
}

func topRun(ctx *Ctx) {
	editlvl := os.Getenv(editenv)
	if editlvl == "/" {
		handleError(errors.New("Already at the top level"))
	}

	doEditSnippit(ctx, []string{})
}

func upRun(ctx *Ctx) {
	popnum := 1
	editlvl := os.Getenv(editenv)
	if editlvl == "/" {
		handleError(errors.New("Already at the top level"))
	}

	path := pathutil.Makepath(editlvl)

	tmpl, err := ctx.Client.TmplGet(pathutil.Pathstr(path))
	handleError(err)

	if isListKey(tmpl) {
		popnum = 2
	}

	path = path[:len(path)-popnum]

	doEditSnippit(ctx, path)
}

func validateRun(ctx *Ctx) {
	if !sessionChanged(ctx) {
		handleError(errors.New("No configuration changes to validate"))
	}
	out, err := ctx.Client.Validate()
	handleErrorNoIndent(err)
	if out != "" {
		doSnippit(ctx, fmt.Sprintf("echo \"%s\"\n", out))
	} else {
		os.Exit(0)
	}
}
