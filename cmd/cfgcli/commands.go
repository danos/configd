// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"fmt"
	"strings"

	"github.com/danos/configd/common"
	"github.com/danos/mgmterror"
)

//CompFunc takes the current context and returns the completion text,
// inserting a space after the command if required.  Returned text is
// formatted for use by bash completion mechanism.
type CompFunc func(ctx *Ctx) (CompletionText string)

//RunFunc will exit as soon as it can to eliminate the possibility
//of extra output confusing the eval of the results.
type RunFunc func(ctx *Ctx)

//ValidFuncs validate the command's arguements and exit if they are invalid
type ValidFunc func(ctx *Ctx) (err error)

type Command struct {
	Name    string
	Help    string
	CompFn  CompFunc
	RunFn   RunFunc
	ValidFn ValidFunc
}

func NewCommand(name string, help string, cfunc CompFunc, rfunc RunFunc, vfunc ValidFunc) *Command {
	return &Command{
		Name:    name,
		Help:    help,
		CompFn:  cfunc,
		RunFn:   rfunc,
		ValidFn: vfunc,
	}
}

var Commands = populateCommands()

func populateCommands() map[string]*Command {
	cmds := map[string]*Command{
		"commit": NewCommand("commit",
			"Commit the current set of changes",
			commitComp, commitRun, commitValid),
		"compare": NewCommand("compare",
			"Compare configuration revisions",
			compareComp, compareRun, compareValid),
		"delete": NewCommand("delete",
			"Delete a configuration element",
			pathComp, deleteRun, checkValidPath),
		"discard": NewCommand("discard",
			"Discard uncommitted changes",
			singleCommandComp, discardRun, validSingleCommand),
		"edit": NewCommand("edit",
			"Edit a sub-element",
			pathComp, editRun, checkValidPath),
		"exit": NewCommand("exit",
			"Exit from this configuration level",
			exitComp, exitRun, exitValid),
		"load": NewCommand("load",
			"Load configuration from a file and replace candidate configuration",
			loadComp, loadRun, loadsaveValid),
		"merge": NewCommand("merge",
			"Merge configuration from a file into the candidate configuration",
			mergeComp, mergeRun, mergeValid),
		"run": NewCommand("run",
			"Run an operational-mode command",
			runComp, runRun, nil),
		"save": NewCommand("save",
			"Save configuration to a file",
			saveComp, saveRun, loadsaveValid),
		"set": NewCommand("set",
			"Set the value of a parameter or create a new element",
			pathComp, setRun, checkValidPath),
		"show": NewCommand("show",
			"Show the configuration (default values may be suppressed)",
			pathComp, showRun, checkValidPath),
		"top": NewCommand("top",
			"Set the edit level to the root",
			singleCommandComp, topRun, validSingleCommand),
		"up": NewCommand("up",
			"Set the edit level one level up",
			singleCommandComp, upRun, validSingleCommand),
		"validate": NewCommand("validate",
			"Validate the current set of changes",
			singleCommandComp, validateRun, validSingleCommand),
	}

	return cmds
}

func updateDynamicCommands(c cfgManager) error {
	if checkConfigMgmt(c) {
		Commands["confirm"] = NewCommand("confirm",
			"Confirm configuration changes",
			confirmComp, confirmRun, confirmValid)
		Commands["rollback"] = NewCommand("rollback",
			"Rollback to a previous configuration",
			rollbackComp, rollbackRun, rollbackValid)
		Commands["commit-confirm"] = NewCommand("commit-confirm",
			"Commit the current set of changes; rollback if not confirmed",
			commitConfComp, commitConfRun, commitConfValid)
	} else {
		delete(Commands, "confirm")
		delete(Commands, "rollback")
		delete(Commands, "commit-confirm")
	}

	if checkLoadKey(c) {
		Commands["loadkey"] = NewCommand("loadkey",
			"Load user SSH key from a file",
			loadkeyComp, loadkeyRun, loadKeyValid)
	} else {
		delete(Commands, "loadkey")
	}

	return nil
}

func checkLoadKey(c cfgManager) bool {
	feats, err := c.GetConfigSystemFeatures()
	if err != nil {
		return false
	}
	_, exists := feats[common.LoadKeysFeature]
	return exists
}

var cfgMgmtPtr = checkConfigMgmtInternal

func overrideConfigMgmtCheck(fp func(cfgManager) bool) { cfgMgmtPtr = fp }
func resetConfigMgmtCheck() {
	cfgMgmtPtr = checkConfigMgmtInternal
}

func checkConfigMgmt(c cfgManager) bool {
	return cfgMgmtPtr(c)
}

func checkConfigMgmtInternal(c cfgManager) bool {
	feats, err := c.GetConfigSystemFeatures()
	if err != nil {
		return false
	}
	_, exists := feats[common.ConfigManagementFeature]
	return exists
}

func CommandHelps() map[string]string {
	out := make(map[string]string)
	for k, v := range Commands {
		out[k] = v.Help
	}
	return out
}

func ExpandCommand(name string) (string, error) {
	matches := make([]string, 0, 1)
	for k, _ := range Commands {
		if k == name {
			return name, nil
		} else if strings.HasPrefix(k, name) {
			matches = append(matches, k)
		}
	}
	switch len(matches) {
	case 0:
		err := mgmterror.NewUnknownElementApplicationError(name)
		return "", err
	case 1:
		return matches[0], nil
	default:
		matchmap := make(map[string]string)
		for _, v := range matches {
			matchmap[v] = Commands[v].Help
		}
		return "", mgmterror.NewPathAmbiguousError([]string{}, matchmap)
	}
}

func GetCommand(name string) (cmd *Command, err error) {
	name, err = ExpandCommand(name)
	if err != nil {
		return nil, err
	}
	cmd, ok := Commands[name]
	if !ok {
		return nil, fmt.Errorf("Invalid command %s", name)
	}
	return cmd, nil
}
