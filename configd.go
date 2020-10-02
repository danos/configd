// Copyright (c) 2018-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package configd

import (
	"log"
	"log/syslog"
	"os"
	"path/filepath"

	"github.com/danos/config/auth"
)

type LockId int32

const (
	COMMIT LockId = -1
	SYSTEM LockId = -2
)

func (l LockId) String() string {
	switch l {
	case COMMIT:
		return "commit"
	case SYSTEM:
		return "system"
	}
	return "unknown"
}

type Context struct {
	Configd   bool
	Auth      auth.Auther
	Pid       int32
	Uid       uint32
	User      string
	UserHome  string
	Groups    []string
	Superuser bool
	Config    *Config
	Dlog      *log.Logger
	Elog      *log.Logger
	Wlog      *log.Logger
	Noexec    bool
}

// Raising privileges should be done sparingly as it bypasses things like
// ACM, secret redaction etc. however it is occasionally necessary.
func (c *Context) RaisePrivileges() {
	c.Configd = true
}

func (c *Context) DropPrivileges() {
	c.Configd = false
}

type Config struct {
	User         string
	Runfile      string
	Logfile      string
	Pidfile      string
	Yangdir      string
	Socket       string
	SecretsGroup string
	SuperGroup   string
	Capabilities string
}

//version of syslog.NewLogger which uses base program name as logging tag
func NewLogger(p syslog.Priority, logFlag int) (*log.Logger, error) {
	var tag string

	tag = filepath.Base(os.Args[0])
	s, err := syslog.New(p, tag)
	if err != nil {
		return nil, err
	}
	return log.New(s, "", logFlag), nil
}

func InSecretsGroup(ctx *Context) bool {
	if ctx.Configd {
		return true
	}
	for _, g := range ctx.Groups {
		if g == ctx.Config.SecretsGroup {
			return true
		}
	}
	return false
}
