// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.

package server

import (
	"github.com/danos/config/schema"
	"github.com/danos/utils/pathutil"
)

type commandArgs struct {
	cmd   []string
	attrs *pathutil.PathAttrs
}

// Generate a commandArgs instance for a given command and arguments
// pathArgs represents any schema path arguments (which may be subject to redaction)
// args represents any non-path arguments (if required these must be redacted by the caller)
// Schema path arguments are assumed to always come after any command arguments
//   ie. the command will be built as "cmd args pathArgs"
func (d *Disp) newCommandArgs(cmd string, args []string, pathArgs []string) *commandArgs {
	if args == nil {
		args = []string{}
	}
	if pathArgs == nil {
		pathArgs = []string{}
	}

	// Ensure any path arguments are expanded
	// The arguments should already have been normalized (if required for the command)
	pathArgs, err := d.expandPath(pathArgs, NoPrefix, InvalidPos)
	if err != nil {
		return nil
	}

	// Attempt to generate attributes for any path arguments to the command.
	// If we failed to generate attributes for the path we can still attempt
	// to authorize the command since the attributes may not always be required.
	attrs := schema.AttrsForPath(d.msFull, pathArgs)
	if attrs == nil {
		newAttrs := pathutil.NewPathAttrs()
		attrs = &newAttrs
	}

	// We also need to generate attributes for the command and any arguments which
	// are not a "path". These are always deemed to be non-sensitive.
	cmdArgs := append([]string{cmd}, args...)
	cmdAttrs := pathutil.NewPathAttrs()
	for _, _ = range cmdArgs {
		elemAttrs := pathutil.NewPathElementAttrs()
		elemAttrs.Secret = false
		cmdAttrs.Attrs = append(cmdAttrs.Attrs, elemAttrs)
	}

	// Finally join the "command" and "path" attributes
	attrs.Attrs = append(cmdAttrs.Attrs, attrs.Attrs...)

	return &commandArgs{cmd: append(cmdArgs, pathArgs...), attrs: attrs}
}

func (d *Disp) newCommandArgsForAaa(cmd string, args []string, pathArgs []string) *commandArgs {
	// Shortcut - since AAA does not happen with elevated privileges
	if d.ctx.Configd {
		return nil
	}
	return d.newCommandArgs(cmd, args, pathArgs)
}

func (d *Disp) accountCommand(args *commandArgs) {
	if args != nil {
		d.ctx.Auth.AccountCommand(d.ctx.Uid, d.ctx.Groups, args.cmd, args.attrs)
	}
}

// Perform "command authorization" for a given command and args
func (d *Disp) authCommand(args *commandArgs) bool {
	if d.ctx.Configd {
		return true
	}
	if args == nil {
		return false
	}

	return d.ctx.Auth.AuthorizeCommand(d.ctx.Uid, d.ctx.Groups, args.cmd, args.attrs)
}
