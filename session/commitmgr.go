// Copyright (c) 2018-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/danos/config/data"
	"github.com/danos/config/diff"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/configd/common"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/exec"
)

type commitmgrreq struct {
	sid     string
	ctx     *configd.Context
	t       *data.Node
	message string
	debug   bool
	resp    chan *commitresp
}

type commitresp struct {
	out []*exec.Output
	err []error
	ok  bool
}

type CommitMgr struct {
	running   *data.AtomicNode
	effective *Session
	schema    schema.ModelSet
	reqch     chan commitmgrreq
	hadcommit bool
}

func NewCommitMgr(running *data.AtomicNode, schema schema.ModelSet) *CommitMgr {
	c := &CommitMgr{
		running: running,
		schema:  schema,
		reqch:   make(chan commitmgrreq),
	}
	go c.run()
	return c
}

func (m *CommitMgr) SetEffective(effective *Session) {
	m.effective = effective
}

func (m *CommitMgr) writeRunning(ctx *configd.Context) error {
	f, err := os.Create(ctx.Config.Runfile)
	if err != nil {
		return err
	}
	defer f.Close()
	// The running file contains the running configuration with secrets, and
	// should definitely NOT be world readable.
	err = f.Chmod(0600)
	if err != nil {
		return err
	}
	//Effective and running are equivalent here use that
	//fact to avoid creating another union tree.
	out, err := m.effective.Show(ctx, []string{}, false, false)
	if err != nil {
		return err
	}
	_, err = f.WriteString(out)
	return err
}

func (m *CommitMgr) commit(sid string, sctx *configd.Context, candidate *data.Node, message string, debug bool) *commitresp {
	//"and now for the subtle bit..."
	//This is important so it deserves an explanation.
	//In order for the defaults to be propagated to the upper layers correctly
	//We need the running tree (with no defaults instanciated) this is m.Running().
	//m.Running() does an atomic dereference of the running tree pointer, so we can only
	//call it once, and must store the returned value on the stack.
	//'run' contains the running tree with the instanciated defaults.
	//'mcan' contains the current candidate tree with the instanciated defaults, in order
	//to properly instanciate the default state in this tree it must be merged with the running
	//tree without the default values in it.
	overallStart := time.Now()
	rtree := m.Running()

	var run *data.Node

	if !m.hadcommit {
		// Very first commit. To ensure that defaults values
		// are configured, set running tree with no defaults
		// instantiated. When compared to candiate tree, the
		// defaults will be seen as a difference.
		m.hadcommit = true
		run = union.NewNode(nil, rtree, m.schema, nil, 0).MergeWithoutDefaults()
	} else {
		run = union.NewNode(nil, rtree, m.schema, nil, 0).Merge()
	}

	ucan := union.NewNode(candidate, rtree, m.schema, nil, 0)
	mcan := ucan.Merge()
	// debug-level logging should be enabled if the debug flag passed in is
	// set OR if configd 'commit' logging is set to debug level.
	debug = debug || common.LoggingIsEnabledAtLevel(
		common.LevelDebug, common.TypeCommit)
	mustThreshold, _ := common.LoggingValueAndStatus(common.TypeMust)
	ctx := newctx(sid, sctx, m.effective, mcan, run, m.schema, message,
		debug, mustThreshold)
	ctx.LogCommitMsg("Starting validation and commit")
	outs, errs, ok := ctx.validate()
	if !ok {
		return &commitresp{out: outs, err: errs, ok: ok}
	}

	// Create environment for hooks
	commitStart := time.Now()
	env := make([]string, 0)
	uid := strconv.FormatUint(uint64(sctx.Uid), 10)
	user, lookuperr := user.LookupId(uid)
	if lookuperr != nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = "Could not lookup UID"
		errs = append(errs, err)
		return &commitresp{out: outs, err: errs, ok: false}
	}
	env = append(env, "COMMIT_USER="+user.Username)
	env = append(env, "PATH=/bin:/usr/bin:/sbin:/usr/sbin:/opt/vyatta/bin:/opt/vyatta/sbin")

	// Run pre-hooks
	hout, herr := ctx.execute_hooks("/etc/commit/pre-hooks.d", env)
	outs = append(outs, hout)
	if herr != nil {
		errs = append(errs, herr)
	}
	ctx.LogCommitTime("Pre-commit hooks", commitStart)

	// Can't use AppendOutput because ctx.commit signature is different
	var couts []*exec.Output
	var cerrs []error
	changedNSMap := diff.CreateChangedNSMap(mcan, run, m.schema, nil)
	couts = sctx.CompMgr.ComponentSetRunningWithLog(
		m.schema, ucan, changedNSMap, ctx.LogCommitTime)
	outs = append(outs, couts...)

	couts, cerrs, _ = ctx.commit(&env)
	outs = append(outs, couts...)
	errs = append(errs, cerrs...)

	writeStart := time.Now()
	effective := m.effective.MergeTreeWithoutDefaults(ctx.ctx)
	m.effective.Discard(ctx.ctx) //we got what we needed
	m.running.Store(effective)
	m.writeRunning(ctx.ctx)
	ctx.LogCommitTime("Write config", writeStart)

	// Run post-hooks after we've written out the running cfg
	postCmtHookStart := time.Now()
	env = append(env, "COMMIT_COMMENT="+ctx.message)
	hout, herr = ctx.execute_hooks("/etc/commit/post-hooks.d", env)
	outs = append(outs, hout)
	if herr != nil {
		errs = append(errs, herr)
	}

	ctx.LogCommitTime("Post-commit hooks", postCmtHookStart)
	ctx.LogCommitTime("Commit OVERALL", commitStart)
	ctx.LogCommitTime("End of validation and commit", overallStart)

	// errs here are warnings, so we return true in all cases as the commit
	// will have been committed if we have got this far.
	return &commitresp{out: outs, err: errs, ok: true}
}

func (m *CommitMgr) run() {
	var done struct{}
	var inCommit bool
	donech := make(chan struct{})
	for {
		select {
		case req := <-m.reqch:
			if inCommit {
				err := mgmterror.NewResourceDeniedProtocolError()
				err.Message = "Commit already in progress"
				req.resp <- MakeCommitError(err)
				break
			}
			inCommit = true
			go func(r commitmgrreq) {
				resp := m.commit(r.sid, r.ctx, r.t, r.message, r.debug)
				donech <- done
				r.resp <- resp
			}(req)
		case <-donech:
			inCommit = false
		}
	}
}

func (m *CommitMgr) Commit(sid string, ctx *configd.Context, candidate *data.Node, message string, debug bool) *commitresp {
	respch := make(chan *commitresp)
	m.reqch <- commitmgrreq{
		sid:     sid,
		ctx:     ctx,
		t:       candidate,
		resp:    respch,
		message: message,
		debug:   debug,
	}
	return <-respch
}

func (m *CommitMgr) Running() *data.Node {
	return m.running.Load()
}

func MakeCommitError(err error) *commitresp {
	return &commitresp{
		err: []error{err},
	}
}
