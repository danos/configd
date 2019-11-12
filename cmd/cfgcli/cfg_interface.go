// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"github.com/danos/configd/rpc"
)

type expander interface {
	Expand(path string) (string, error)
	ExpandWithPrefix(path, prefix string, pos int) (string, error)
}

type getSetter interface {
	Set(path string) (string, error)
	TmplGet(path string) (map[string]string, error)
}

// These represent implementations of cfgcli's keywords, so make a logical
// grouping.
type commander interface {
	Commit(message string, debug bool) (string, error)
	CommitConfirm(message string, debug bool, mins int) (string, error)
	CompareConfigRevisions(revOne, revTwo string) (string, error)
	CompareSessionChanges() (string, error)
	Confirm() (string, error)
	ConfirmSilent() (string, error)
	Delete(path string) error
	Discard() error
	getSetter
	Load(file string) error
	LoadFrom(source, routingInstance string) error
	LoadKeys(user, source, routingInstance string) (string, error)
	MergeReportWarnings(file string) (bool, error)
	Rollback(string, string, bool) (string, error)
	Save(file string) error
	SaveTo(dest, routingInstance string) error
	ShowConfigWithContextDiffs(path string, showDefaults bool) (string, error)
	Validate() (string, error)
}

type completer interface {
	GetCompletions(schema bool, path string) (map[string]string, error)
}

type typeGetter interface {
	NodeGetType(path string) (rpc.NodeType, error)
	TmplValidatePath(path string) (bool, error)
}

// Essentially a cut-down version of the configd/Client API, converted into
// an interface so we can insert a test version when required.  Some APIs
// are extracted into their own interfaces, with 'utility' functions left
// here.
type cfgManager interface {
	commander
	completer
	Exists(db rpc.DB, path string) (bool, error)
	expander
	ExtractArchive(file, destination string) (string, error)
	Get(db rpc.DB, path string) ([]string, error)
	GetCommitLog() (map[string]string, error)
	GetConfigSystemFeatures() (map[string]struct{}, error)
	SessionChanged() (bool, error)
	SessionMarkSaved() error
	typeGetter
}
