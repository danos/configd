// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	client "github.com/danos/configd/client"
	"github.com/danos/utils/pathutil"
)

type cmdLineParams struct {
	action     string
	pfx        string
	cword      string
	cidx       int
	lastcomp   string
	dohelp     bool
	socketpath string
	printcmd   bool
	argsInEnv  bool
}

var cliParams cmdLineParams

func init() {
	flag.StringVar(&cliParams.action, "action", "run",
		"Action to perform [ run | complete | expand | init ]")
	flag.StringVar(&cliParams.pfx, "prefix", "", "Prefix to filter")
	flag.StringVar(&cliParams.cword, "curword", "", "Current word")
	flag.IntVar(&cliParams.cidx, "curidx", 0, "Current word index")
	flag.StringVar(&cliParams.lastcomp, "lastcomp", "", "Last completion")
	flag.BoolVar(&cliParams.dohelp, "dohelp", true,
		"Whether to print help text or complete normally")
	flag.StringVar(&cliParams.socketpath, "socket",
		"/run/vyatta/configd/main.sock",
		"Path to the socket we should write to")
	flag.BoolVar(&cliParams.printcmd, "print", false,
		"Print the command that would be executed")
	flag.BoolVar(&cliParams.argsInEnv, "args-in-env", false,
		"Arguments to this tool are provided in the CFGCLI_ARGS environment variable")
}

func expand(e expander, path []string) {
	pstr, err := e.Expand(pathutil.Pathstr(path))
	handleError(err)
	fmt.Println(strings.Join(pathutil.Makepath(pstr), " "))
	os.Exit(0)
}

func run_handler(c cfgManager, args []string, params cmdLineParams) {
	err := run(c, args, params)
	if err != nil {
		handleError(err)
	}
}

func run(c cfgManager, args []string, params cmdLineParams) error {
	if len(args) == 0 {
		handleError(fmt.Errorf("%s", "Must supply command to run"))
	}
	ctx := &Ctx{
		Args:               args,
		Prefix:             "__noncomp__", //this is important because valiation funcs expect prefix to be populated
		Client:             c,
		Print:              params.printcmd,
		HasLoadKey:         checkLoadKey(c),
		HasConfigMgmt:      checkConfigMgmt(c),
		HasRoutingInstance: checkRoutingInstance(c),
	}
	cmd, err := GetCommand(args[0])
	if err != nil {
		return err
	}
	ctx.Args[0] = cmd.Name
	if cmd.Name == "show" {
		ctx.Args, ctx.All = parseShowAll(ctx)
	}

	if cmd.ValidFn != nil {
		err = cmd.ValidFn(ctx)
		if err != nil {
			return err
		}
	}

	cmd.RunFn(ctx)
	return nil
}

func complete_handler(c cfgManager, args []string, params cmdLineParams) {
	completionText, err := complete(c, args, params)
	if err != nil {
		handleCompError(err, printError)
	}
	fmt.Print(completionText)
}

// Allows test code to create a context the same way as production code.
func createCompleteCtx(
	c cfgManager,
	args []string,
	params cmdLineParams,
) *Ctx {
	return &Ctx{
		Args:               args,
		CompCurIdx:         params.cidx,
		CompCurWord:        params.cword,
		Prefix:             params.pfx,
		LastComp:           params.lastcomp,
		DoHelp:             params.dohelp,
		Client:             c,
		Print:              params.printcmd,
		HasLoadKey:         checkLoadKey(c),
		HasConfigMgmt:      checkConfigMgmt(c),
		HasRoutingInstance: checkRoutingInstance(c),
	}
}

func complete(c cfgManager, args []string, params cmdLineParams) (
	completionText string,
	err error,
) {
	ctx := createCompleteCtx(c, args, params)

	if ctx.CompCurIdx == 0 || len(ctx.Args) == 0 {
		return firstWordComp(ctx), nil
	} else {
		cmd, err := GetCommand(args[0])
		if err != nil {
			return "", err
		}
		ctx.Args[0] = cmd.Name
		if cmd.Name == "show" {
			ctx.Args, ctx.All = parseShowAll(ctx)
			if ctx.All && ctx.CompCurIdx > 1 {
				ctx.CompCurIdx = ctx.CompCurIdx - 1
			}
		}

		if cmd.ValidFn != nil {
			err := cmd.ValidFn(ctx)
			if err != nil {
				return "", err
			}
		}
		completionText = cmd.CompFn(ctx)
	}

	return completionText, nil
}

func parseShowAll(ctx *Ctx) ([]string, bool) {
	var showFlags *flag.FlagSet
	var all bool
	showFlags = flag.NewFlagSet("show", flag.ContinueOnError)
	showFlags.BoolVar(&all, "all", false, "Show defaults")

	showFlags.Parse(ctx.Args[1:])

	return append(ctx.Args[0:1], showFlags.Args()...), all
}

func isSecret(c getSetter, path string) bool {
	tmpl, err := c.TmplGet(path)
	handleError(err)
	return tmpl["secret"] == "1"
}

func setSecret(c getSetter, args []string) {
	var passwd, passwd2 string
	if len(args) == 0 {
		handleError(errors.New("Must supply path to set"))
	}
	pathstr := args[0]
	path := pathutil.Makepath(pathstr)
	if !isSecret(c, pathstr) {
		handleError(errors.New("Path doesn't require a secret"))
	}
	for {
		p, err := GetPass("Secret: ", 1, 0)
		handleError(err)
		p2, err := GetPass("Retype secret: ", 1, 0)
		handleError(err)
		passwd, passwd2 = string(p), string(p2)
		if passwd == passwd2 {
			break
		}
		fmt.Fprintln(os.Stderr, "Secrets do not match")
	}
	path = append(path, passwd)
	out, err := c.Set(pathutil.Pathstr(path))
	handleError(err)
	if out != "" {
		printOutput(out)
	}
}

func argsFromEnv(data string) []string {
	type argData struct {
		Args []string `json:"args"`
	}
	var args argData
	dec := json.NewDecoder(strings.NewReader(data))
	err := dec.Decode(&args)
	if err != nil {
		return []string{}
	}
	return args.Args
}

func main() {
	flag.Parse()
	c, err := client.Dial("unix", cliParams.socketpath,
		os.ExpandEnv("$VYATTA_CONFIG_SID"))
	defer c.Close()
	handleError(err)
	err = insertDynamicCommands(c)
	handleError(err)
	args := flag.Args()
	if cliParams.argsInEnv {
		args = argsFromEnv(os.Getenv("CFGCLI_ARGS"))
	}
	switch cliParams.action {
	case "complete":
		complete_handler(c, args, cliParams)
	case "expand":
		expand(c, args)
	case "run":
		run_handler(c, args, cliParams)
	case "setSecret":
		setSecret(c, args)
	case "init":
		initShell()
	}
}
