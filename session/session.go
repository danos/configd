// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"io"

	"github.com/danos/config/data"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/configd/rpc"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/exec"
)

func init() {
	exec.NewExecError = func(path []string, err string) error {
		return mgmterror.NewExecError(path, err)
	}
}

// Defaults - return defaults
// Secrets - return secrets in plain text
// CouldExist - path is valid if it *could* exist, but currently doesn't
type TreeOpts struct {
	Defaults, Secrets, CouldExistIsAllowed bool
}

func NewTreeOpts(flags map[string]interface{}) *TreeOpts {
	opts := &TreeOpts{}
	for flag, val := range flags {
		v, ok := val.(bool)
		if !ok {
			continue
		}
		switch flag {
		case "Defaults":
			opts.Defaults = v
		case "Secrets":
			opts.Secrets = v
		case "CouldExist":
			opts.CouldExistIsAllowed = v
		}
	}
	return opts
}

func (t *TreeOpts) AllowCouldExist() {
	t.CouldExistIsAllowed = true
}

func (t *TreeOpts) ToUnionOptions() []union.UnionOption {
	var options []union.UnionOption
	if t.Defaults {
		options = append(options, union.IncludeDefaults)
	}
	if !t.Secrets {
		options = append(options, union.HideSecrets)
	}
	// CouldExist is not relevant in UnionOptions
	return options
}

type Session struct {
	s session
}

type SessionOption func(*session)

func NewSession(
	sid string,
	cmgr *CommitMgr,
	st,
	stFull schema.ModelSet,
	options ...SessionOption,
) *Session {
	s := &Session{
		s: session{
			sid:        sid,
			candidate:  data.New("root"),
			cmgr:       cmgr,
			schema:     st,
			schemaFull: stFull,
			reqch:      make(chan request),
			commitch:   make(chan *data.Node),
			kill:       make(chan struct{}),
			term:       make(chan struct{}),
		},
	}

	for _, option := range options {
		option(&s.s)
	}

	go s.s.run()
	return s
}

func (s *Session) NewAuther(ctx *configd.Context) union.Auther {
	return s.s.newAuther(ctx)
}

func (s *Session) MergeTree(ctx *configd.Context) *data.Node {
	respch := make(chan *data.Node)
	req := &mergetreereq{
		ctx:      ctx,
		defaults: false,
		resp:     respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return nil
}

func (s *Session) MergeTreeWithoutDefaults(ctx *configd.Context) *data.Node {
	respch := make(chan *data.Node)
	req := &mergetreereq{
		ctx:      ctx,
		defaults: false,
		resp:     respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return nil
}

func (s *Session) Exists(ctx *configd.Context, path []string) bool {
	respch := make(chan bool)
	req := &existsreq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return false
}

func (s *Session) Get(ctx *configd.Context, path []string) ([]string, error) {
	respch := make(chan getresp)
	req := &getreq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.vals, resp.err
	case <-s.s.term:
	}
	return nil, sessTermError()
}

func (s *Session) GetType(ctx *configd.Context, path []string) (rpc.NodeType, error) {
	respch := make(chan typeresp)
	req := &typereq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.val, resp.err
	case <-s.s.term:
	}
	return rpc.CONTAINER, sessTermError()
}

func (s *Session) GetStatus(ctx *configd.Context, path []string) (rpc.NodeStatus, error) {
	respch := make(chan statusresp)
	req := &statusreq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.val, resp.err
	case <-s.s.term:
	}
	return rpc.UNCHANGED, sessTermError()
}

func (s *Session) IsDefault(ctx *configd.Context, path []string) (bool, error) {
	respch := make(chan defaultresp)
	req := &defaultreq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.val, resp.err
	case <-s.s.term:
	}
	return false, sessTermError()
}

func (s *Session) GetTree(ctx *configd.Context, path []string, opts *TreeOpts) (union.Node, error) {
	respch := make(chan gettreeresp)
	req := &gettreereq{
		ctx:  ctx,
		path: path,
		opts: opts,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.val, resp.err
	case <-s.s.term:
	}
	return nil, sessTermError()
}

// GetFullTree - return state and config nodes, plus any error and warnings.
// error is fatal; warnings relate to specific parts of the tree not returning
// valid data.
func (s *Session) GetFullTree(ctx *configd.Context, path []string, opts *TreeOpts) (union.Node, error, []error) {
	respch := make(chan getfulltreeresp)
	req := &getfulltreereq{
		ctx:  ctx,
		path: path,
		opts: opts,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.val, resp.err, resp.warns
	case <-s.s.term:
	}
	return nil, sessTermError(), nil
}

func (s *Session) Set(ctx *configd.Context, path []string) error {
	respch := make(chan error)
	req := &setreq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}

	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return sessTermError()
}

func (s *Session) ValidateSet(ctx *configd.Context, path []string) error {
	respch := make(chan error)
	req := &validatesetreq{
		setreq: setreq{
			ctx:  ctx,
			path: path,
			resp: respch,
		},
	}

	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return sessTermError()
}

func (s *Session) Delete(ctx *configd.Context, path []string) error {
	respch := make(chan error)
	req := &delreq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}

	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return sessTermError()
}
func (s *Session) Validate(ctx *configd.Context) ([]*exec.Output, []error, bool) {
	respch := make(chan *commitresp)
	req := &validatereq{
		ctx:  ctx,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.out, resp.err, resp.ok
	case <-s.s.term:
	}
	ret := MakeCommitError(sessTermError())
	return ret.out, ret.err, ret.ok
}

func (s *Session) Lock(ctx *configd.Context) (int32, error) {
	respch := make(chan lockresp)
	req := &lockreq{
		ctx:  ctx,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.pid, resp.err
	case <-s.s.term:
	}
	return -1, sessTermError()
}

func (s *Session) Unlock(ctx *configd.Context) (int32, error) {
	respch := make(chan lockresp)
	req := &unlockreq{
		ctx:  ctx,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.pid, resp.err
	case <-s.s.term:
	}
	return -1, sessTermError()
}

func (s *Session) Locked(ctx *configd.Context) (int32, error) {
	respch := make(chan lockresp)
	req := &lockedreq{
		ctx:  ctx,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.pid, resp.err
	case <-s.s.term:
	}
	return -1, sessTermError()
}

func (s *Session) Comment(ctx *configd.Context, path []string) error {
	respch := make(chan error)
	req := &commentreq{
		ctx:  ctx,
		path: path,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return sessTermError()
}

func (s *Session) Changed(ctx *configd.Context) bool {
	respch := make(chan bool)
	req := &changedreq{
		ctx:  ctx,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return false
}

func (s *Session) Saved(ctx *configd.Context) bool {
	respch := make(chan bool)
	req := &savedreq{
		ctx:  ctx,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return false
}

func (s *Session) MarkSaved(ctx *configd.Context, saved bool) {
	respch := make(chan error)
	req := &marksavedreq{
		ctx:   ctx,
		saved: saved,
		resp:  respch,
	}
	select {
	case s.s.reqch <- req:
		<-respch
		return
	case <-s.s.term:
	}
}

func (s *Session) showInternal(ctx *configd.Context, path []string, hideSecrets, showDefaults, forceShowSecrets bool) (string, error) {
	respch := make(chan showresp)
	req := &showreq{
		ctx:              ctx,
		path:             path,
		hideSecrets:      hideSecrets,
		showDefaults:     showDefaults,
		forceShowSecrets: forceShowSecrets,
		resp:             respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.data, resp.err
	case <-s.s.term:
	}
	return "", sessTermError()
}
func (s *Session) Show(ctx *configd.Context, path []string, hideSecrets, showDefaults bool) (string, error) {
	return s.showInternal(ctx, path, hideSecrets, showDefaults, false)
}

func (s *Session) ShowForceSecrets(ctx *configd.Context, path []string, hideSecrets, showDefaults bool) (string, error) {
	return s.showInternal(ctx, path, hideSecrets, showDefaults, true)
}

func (s *Session) Discard(ctx *configd.Context) error {
	respch := make(chan error)
	req := &discardreq{
		ctx:  ctx,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return sessTermError()
}

func (s *Session) Load(ctx *configd.Context, file string, r io.Reader) (error, []error) {
	respch := make(chan loadresp)
	req := &loadreq{
		ctx:    ctx,
		file:   file,
		reader: r,
		resp:   respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.err, resp.invalidPaths
	case <-s.s.term:
	}
	return sessTermError(), nil
}

func (s *Session) Merge(ctx *configd.Context, file string) (error, []error) {
	respch := make(chan mergeresp)
	req := &mergereq{
		ctx:  ctx,
		file: file,
		resp: respch,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.err, resp.invalidPaths
	case <-s.s.term:
	}
	return sessTermError(), nil
}

func (s *Session) Commit(ctx *configd.Context, message string, debug bool) ([]*exec.Output, []error, bool) {
	respch := make(chan *commitresp)
	req := &commitreq{
		ctx:     ctx,
		message: message,
		resp:    respch,
		debug:   debug,
	}
	select {
	case s.s.reqch <- req:
		resp := <-respch
		return resp.out, resp.err, resp.ok
	case <-s.s.term:
	}
	ret := MakeCommitError(sessTermError())
	return ret.out, ret.err, ret.ok
}

func (s *Session) GetHelp(ctx *configd.Context, schema bool, path []string) (map[string]string, error) {
	respch := make(chan map[string]string)
	req := &gethelpreq{
		ctx:    ctx,
		schema: schema,
		path:   path,
		resp:   respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch, nil
	case <-s.s.term:
	}
	return nil, sessTermError()
}

func (s *Session) Kill() {
	s.s.kill <- struct{}{}
}

func (s *Session) EditConfigXML(ctx *configd.Context, config_target, default_operation, test_option, error_option, config string) error {
	respch := make(chan error)
	req := &editconfigreq{
		ctx:     ctx,
		target:  config_target,
		defop:   default_operation,
		testopt: test_option,
		erropt:  error_option,
		config:  config,
		resp:    respch,
	}
	select {
	case s.s.reqch <- req:
		return <-respch
	case <-s.s.term:
	}
	return sessTermError()
}
