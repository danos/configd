// Copyright (c) 2017-2019,2021, AT&T Intellectual Property.
// All rights reserved.
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	spawn "os/exec"

	"github.com/danos/config/commit"
	"github.com/danos/config/data"
	"github.com/danos/config/schema"
	"github.com/danos/configd"
	"github.com/danos/configd/common"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/exec"
)

type commitctx struct {
	debug              bool
	mustDebugThreshold int // Time in ms above which we log time taken. 0 = off
	effective          *Session
	candidate          *data.Node
	running            *data.Node
	schema             schema.Node
	sid                string
	sctx               *configd.Context
	ctx                *configd.Context
	message            string
}

func newctx(
	sid string,
	sctx *configd.Context,
	effective *Session,
	candidate, running *data.Node,
	sch schema.Node,
	message string,
	debug bool,
	mustDebugThreshold int,
) *commitctx {
	return &commitctx{
		debug:              debug,
		mustDebugThreshold: mustDebugThreshold,
		effective:          effective,
		candidate:          candidate,
		running:            running,
		schema:             sch,
		sid:                sid,
		sctx:               sctx,
		ctx: &configd.Context{
			Configd: true,
			Config:  sctx.Config,
			Pid:     int32(configd.SYSTEM),
			Auth:    sctx.Auth,
			Dlog:    sctx.Dlog,
			Elog:    sctx.Elog,
			CompMgr: sctx.CompMgr,
			Noexec:  true,
		},
		message: message,
	}
}

const (
	commitLogMsgPrefix = "COMMIT"
	padToLength        = 50
	// 50 + 3 extra just in case
	padding = "                                                     "
)

func pad(msg string) string {
	msgLen := len(msg)
	padLen := 0
	if msgLen < padToLength {
		padLen = padToLength - msgLen
	}
	return msg + ": " + padding[:padLen]
}

// We log here if COMMIT_DEBUG is set (c.debug) OR if cfgdbg settings
// are error / debug level.  Per-script logging is done elsewhere, and only
// if debug level logging is enabled.
func (c *commitctx) loggingEnabled() bool {
	if c.debug {
		return true
	}
	if common.LoggingIsEnabledAtLevel(common.LevelError, common.TypeCommit) {
		return true
	}
	return false
}

func (c *commitctx) LogCommitMsg(msg string) {
	if c.loggingEnabled() {
		c.sctx.Elog.Println(fmt.Sprintf("%s: %s", commitLogMsgPrefix, msg))
	}
}

func (c *commitctx) LogCommitTime(msg string, startTime time.Time) {
	if c.loggingEnabled() {
		c.sctx.Elog.Println(
			fmt.Sprintf("%s: %s%s", commitLogMsgPrefix, pad(msg),
				time.Since(startTime).Round(time.Millisecond)))
	}
}

func (c *commitctx) MustDebugThreshold() int {
	return c.mustDebugThreshold
}

func (c *commitctx) Log(msgs ...interface{}) {
	c.sctx.Dlog.Println(msgs...)
}

func (c *commitctx) LogError(msgs ...interface{}) {
	c.sctx.Elog.Println(msgs...)
}

func (c *commitctx) LogAudit(msg string) {
	c.ctx.Auth.AuditLog(msg)
}

func (c *commitctx) Debug() bool {
	return c.debug
}

func (c *commitctx) Sid() string {
	return c.sid
}

func (c *commitctx) Uid() uint32 {
	return c.sctx.Uid
}

func (c *commitctx) CompMgr() schema.ComponentManager {
	return c.ctx.CompMgr
}

func (c *commitctx) Running() *data.Node {
	return c.running
}

func (c *commitctx) Candidate() *data.Node {
	return c.candidate
}

func (c *commitctx) Schema() schema.Node {
	return c.schema
}

func (c *commitctx) RunDeferred() bool {
	return false
}

func (c *commitctx) Effective() commit.EffectiveDatabase {
	return c
}

func (c *commitctx) Set(path []string) error {
	return c.effective.Set(c.ctx, path)
}

func (c *commitctx) Delete(path []string) error {
	return c.effective.Delete(c.ctx, path)
}

func (c *commitctx) validate() ([]*exec.Output, []error, bool) {
	return commit.Validate(c)
}

// Original implementation ignores the result of the hooks
func (c *commitctx) execute_hooks(hookdir string, env []string) (*exec.Output, error) {
	out := new(bytes.Buffer)
	err := new(bytes.Buffer)
	cmd := spawn.Command("/bin/run-parts", "--regex=^[a-zA-Z0-9._-]+$", "--", hookdir)
	cmd.Stdout = out
	cmd.Stderr = err
	if env != nil {
		cmd.Env = append(cmd.Env, env...)
	}

	c.sctx.Dlog.Printf("Executing %s hooks\n", hookdir)
	if cmd.Run() != nil {
		cerr := mgmterror.NewOperationFailedApplicationError()
		cerr.Message = err.String()
		return &exec.Output{Output: out.String()}, cerr
	}
	return &exec.Output{Output: out.String()}, nil
}

func (c *commitctx) send_notify() {
	pid := strconv.FormatInt(int64(c.sctx.Pid), 10)
	uid := strconv.FormatUint(uint64(c.sctx.Uid), 10)
	spawn.Command("/opt/vyatta/sbin/vyatta-cfg-notify", uid, pid).Run()
}

func (c *commitctx) commit(env *[]string) ([]*exec.Output, []error, bool) {
	outs, errs, successes, failures := commit.Commit(c)

	if successes > 0 {
		c.send_notify()
	}

	status := "COMMIT_STATUS="
	if failures > 0 {
		if successes > 0 {
			status += "PARTIAL"
		} else {
			status += "FAILURE"
		}
	} else {
		status += "SUCCESS"
	}
	*env = append(*env, status)
	return outs, errs, failures == 0
}
