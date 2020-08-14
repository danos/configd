// Copyright (c) 2017-2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains functions that format the standard / internal / 'raw'
// mgmterrors into the format we know and love on the CLI.  Many tests (QA,
// regression, individual features) rely on the content of these messages,
// so change them at your peril.
//
// Changes to the number of newlines should be fine, but adding / removing
// specific '<foo> failed' strings is highly likely to break tests.
//
// Additionally, changing the path format from 'spaced' to 'slashed' will
// also almost certainly break things.
//
// The 'set_and_delete_test' and 'validate_error_test' files in this
// directory check content (path format, presence / absence of specific
// strings), but don't check precise ordering and newlines, to keep them
// slightly less brittle.

package common

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/danos/mgmterror"
	"github.com/danos/utils/pathutil"
)

const (
	configPath        = "Configuration path: "
	isntValid         = " is not valid"
	pathIsInvalid     = "Path is invalid"
	setFailed         = "Set failed"
	validationFailed  = "Value validation failed"
	warningsGenerated = "Warnings were generated when applying " +
		"the configuration:"
)

// errpath - pretty print path for the CLI
//
// Format here is to put the last element of the path inside [].
//
// We need to unescape various characters (eg ':' which otherwise appears as
// '%2F'.  Note that QueryUnescape would convert a space to a plus in a path,
// whereas PathUnescape does not (but doesn't exist in Go1.7).  However, as we
// don't escape either of these it doesn't matter.
//
func errpath(fwdSlashPath string) string {
	if len(fwdSlashPath) == 0 || fwdSlashPath == "/" {
		return "[]"
	}
	if fwdSlashPath[0] == '/' {
		fwdSlashPath = fwdSlashPath[1:]
	}
	path := strings.Split(fwdSlashPath, "/")
	if len(path) < 2 {
		retStr, _ := url.QueryUnescape(fmt.Sprintf("%s", path))
		return retStr
	}
	path, val := path[:len(path)-1], path[len(path)-1]
	retStr, _ := url.QueryUnescape(
		fmt.Sprintf("%s [%s]", strings.Join(path, " "), val))
	return retStr
}

// FormatConfigPathError - pretty print Exists() / Expands() errors for CLI
func FormatConfigPathError(err error) error {
	var b bytes.Buffer

	if err == nil {
		return nil
	}

	if me, ok := err.(mgmterror.Formattable); ok {
		// 'is not valid' is incorporated in the message.
		b.WriteString(configPath)
		b.WriteString(me.GetMessage())
	} else {
		b.WriteString(configPath)
		b.WriteString(err.Error())
	}
	return fmt.Errorf(b.String())
}

// FormatRpcPathError - pretty print RPC errors for the CLI
func FormatRpcPathError(err error) error {
	var b bytes.Buffer

	if err == nil {
		return nil
	}

	if me, ok := err.(mgmterror.Formattable); ok {
		switch err.(type) {
		case *mgmterror.UnknownElementApplicationError:
			// 'is not valid' is incorporated in the message.
			// We can't use me.GetPath() as that either only includes the final
			// element, or all bar the final element.
			b.WriteString(me.GetMessage())
		default:
			if me.GetPath() != "" {
				b.WriteString(errpath(me.GetPath()))
				b.WriteString(isntValid)
				b.WriteString("\n\n")
			}
			b.WriteString(me.GetMessage())
		}
	} else {
		b.WriteString(err.Error())
	}
	return fmt.Errorf(b.String())
}

// FormatCommitOrValErrors - pretty print commit / validation errors
//
// These are somewhat verbose, and in the case of multiple errors for a single
// node, very repetitive.  We print the path, then the error, then repeat the
// path (CLI format) for good measure.
func FormatCommitOrValErrors(err error) string {
	var b bytes.Buffer

	if me, ok := err.(mgmterror.Formattable); ok {
		pathStr := strings.Join(pathutil.Makepath(me.GetPath()), " ")
		b.WriteString("[")
		b.WriteString(pathStr)
		b.WriteString("]\n\n")
		b.WriteString(me.GetMessage())
		b.WriteString("\n\n[[")
		b.WriteString(pathStr)
		b.WriteString("]] failed.")
	} else {
		b.WriteString(err.Error())
	}
	return b.String()
}

func FormatWarnings(warns []error) error {
	if len(warns) == 0 {
		return nil
	}

	var b bytes.Buffer
	b.WriteString(warningsGenerated)
	b.WriteString("\n\n")
	for _, warn := range warns {
		b.WriteString(formatLoadOrMergeWarningMultiline(warn))
		b.WriteString("\n\n")
	}

	return fmt.Errorf(b.String())
}

const (
	withPathPrefix  = true
	noPathPrefix    = false
	withSetFailed   = true
	noSetFailed     = false
	withPathInvalid = true
	noPathInvalid   = false
)

// FormatConfigPathErrorMultiline - pretty print multiline config path errors
//
// Deal with various formats of set/delete 'configuration path' errors, which
// may or may not include 'Set failed' or 'Value validation failed'.
func FormatConfigPathErrorMultiline(err error) error {
	return fmt.Errorf(formatMultilineSetWarnings(err,
		noPathPrefix, withSetFailed, noPathInvalid))
}

// formatLoadOrMergeMultiline - pretty print for load/merge errors
//
// Subtle differences from single set commands, but close enough that bools
// can be used to tweak the output in the common function.
func formatLoadOrMergeWarningMultiline(err error) string {
	return formatMultilineSetWarnings(err,
		withPathPrefix, noSetFailed, withPathInvalid)
}

func formatMultilineSetWarnings(
	err error,
	printPathPrefix bool,
	printSetFailed bool,
	printPathInvalid bool,
) string {
	var b bytes.Buffer

	if err == nil {
		return ""
	}

	if me, ok := err.(mgmterror.Formattable); ok {

		switch err.(type) {
		case *mgmterror.UnknownElementApplicationError:
			return formatUnknownElemAppError(me,
				printPathPrefix, printPathInvalid)
		}

		if printPathPrefix {
			pathStr := strings.Join(pathutil.Makepath(me.GetPath()), " ")
			b.WriteString("[")
			b.WriteString(pathStr)
			b.WriteString("]: ")
		}

		b.WriteString(configPath)
		b.WriteString(errpath(me.GetPath()))
		b.WriteString(isntValid)

		switch err.(type) {
		case *mgmterror.UnknownElementApplicationError:
			// Shouldn't get here, but just in case ...
		case *mgmterror.DataExistsError, *mgmterror.DataMissingError:
			b.WriteString("\n\n")
			b.WriteString(me.GetMessage())
		case *mgmterror.InvalidValueApplicationError:
			b.WriteString("\n\n")
			b.WriteString(me.GetMessage())
			b.WriteString("\n")
			b.WriteString(validationFailed)
			if printSetFailed {
				b.WriteString("\n\n")
				b.WriteString(setFailed)
			}
		default:
			b.WriteString("\n\n")
			b.WriteString(me.GetMessage())
			if printSetFailed {
				b.WriteString("\n\n")
				b.WriteString(setFailed)
			}
		}

	} else {
		b.WriteString(configPath)
		b.WriteString(err.Error())
	}

	return b.String()
}

func formatUnknownElemAppError(
	me mgmterror.Formattable,
	printPathPrefix,
	printPathInvalid bool,
) string {

	var b bytes.Buffer

	if printPathPrefix {
		path := pathutil.Makepath(me.GetPath())
		if len(me.GetInfo()) > 0 {
			path = append(path, me.GetInfo()[0].Value)
		}
		pathStr := strings.Join(path, " ")
		b.WriteString("[")
		b.WriteString(pathStr)
		b.WriteString("]: ")
	}

	// 'is not valid' is incorporated in the message.
	// We can't use me.GetPath() as that either only includes the final
	// element, or all bar the final element.
	b.WriteString(configPath)
	b.WriteString(me.GetMessage())
	if printPathInvalid {
		b.WriteString("\n\n")
		b.WriteString(pathIsInvalid)
	}
	return b.String()
}
