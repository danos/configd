// Copyright (c) 2017-2021, AT&T Intellectual Property.
// All rights reserved.
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"fmt"
	"strconv"
	"syscall"
	"time"

	"github.com/danos/config/commit"
	"github.com/danos/config/data"
	"github.com/danos/config/diff"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/configd/common"
	"github.com/danos/configd/rpc"
	rfc7951utils "github.com/danos/configd/session/internal/rfc7951"
	"github.com/danos/encoding/rfc7951"
	rfc7951data "github.com/danos/encoding/rfc7951/data"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/exec"
	"github.com/danos/utils/pathutil"
	"github.com/danos/vci"
	"github.com/danos/yang/data/encoding"
	yang "github.com/danos/yang/schema"
)

const (
	includeDefault = true
	excludeDefault = false

	Shared   = true
	Unshared = false
)

//Implements the Auther interface from union tree
type Auther struct {
	s           *session
	ctx         *configd.Context
	showSecrets bool
}

func (s *session) newAuther(ctx *configd.Context) union.Auther {
	return &Auther{s: s, ctx: ctx}
}

func (s *session) newShowSecAuther(ctx *configd.Context) union.Auther {
	return &Auther{s: s, ctx: ctx, showSecrets: true}
}

func (s *Auther) AuthRead(path []string) bool {
	attrs := schema.AttrsForPath(s.s.schemaFull, path)
	return s.ctx.Configd || s.ctx.Auth.AuthorizeRead(s.ctx.Uid, s.ctx.Groups, path, attrs)
}

func (s *Auther) AuthCreate(path []string) bool {
	attrs := schema.AttrsForPath(s.s.schemaFull, path)
	return s.ctx.Configd || s.ctx.Auth.AuthorizeCreate(s.ctx.Uid, s.ctx.Groups, path, attrs)
}

func (s *Auther) AuthUpdate(path []string) bool {
	attrs := schema.AttrsForPath(s.s.schemaFull, path)
	return s.ctx.Configd || s.ctx.Auth.AuthorizeUpdate(s.ctx.Uid, s.ctx.Groups, path, attrs)
}

func (s *Auther) AuthDelete(path []string) bool {
	attrs := schema.AttrsForPath(s.s.schemaFull, path)
	return s.ctx.Configd || s.ctx.Auth.AuthorizeDelete(s.ctx.Uid, s.ctx.Groups, path, attrs)
}

func (s *Auther) AuthReadSecrets(path []string) bool {
	return s.showSecrets || s.ctx.Configd || configd.InSecretsGroup(s.ctx)
}

type session struct {
	sid   string
	owner *uint32
	lpid  int32
	saved bool

	candidate  *data.Node
	cmgr       *CommitMgr
	schema     schema.ModelSet
	schemaFull schema.ModelSet

	reqch    chan request
	commitch chan *data.Node

	kill chan struct{}
	term chan struct{}
}

func (s *session) getUnionFull() union.Node {
	return union.NewNode(s.getUnion().Merge(), data.New("state"), s.schemaFull, nil, 0)
}

func (s *session) getUnion() union.Node {
	return union.NewNode(s.candidate, s.cmgr.Running(), s.schema, nil, 0)
}

func (s *session) getRunning() *data.Node {
	//since trees are stored without the defaults, we need to run with the merge operation
	//to get the actual running tree.
	return union.NewNode(nil, s.cmgr.Running(), s.schema, nil, 0).Merge()
}

func (s *session) mergetree(ctx *configd.Context, defaults bool) *data.Node {
	if !defaults {
		return s.getUnion().MergeWithoutDefaults()
	}
	return s.getUnion().Merge()
}

const (
	incompletePathIsValid   = true
	incompletePathIsInvalid = false
	fullSchema              = true
	cfgSchemaOnly           = false
)

// validateSetPath - used for 'set' command validation, and testing paths
// provided by NETCONF for GetTree / GetTreeFull.  In the NETCONF case, we
// allow incomplete paths (eg path ends at NP container, or list name, or
// leaf name (no value), and may need to use the full schema to include
// config:false nodes.
func (s *session) validateSetPath(
	ctx *configd.Context,
	path []string,
	allowIncompletePaths, useFullSchema bool) error {
	vctx := schema.ValidateCtx{
		CurPath:               path,
		Path:                  pathutil.Pathstr(path),
		Sid:                   s.sid,
		Noexec:                ctx.Noexec,
		St:                    s.schema,
		IncompletePathIsValid: allowIncompletePaths,
	}
	errch := make(chan error)
	go func() {
		if useFullSchema {
			vctx.St = s.schemaFull
			errch <- s.schemaFull.Validate(vctx, []string{}, path)
		} else {
			errch <- s.schema.Validate(vctx, []string{}, path)
		}
		return
	}()
	for {
		select {
		case err := <-errch:
			if err != nil {
				return err
			}
			return nil
		case req := <-s.reqch:
			s.processreq(req, nil)
		}
	}
}

func (s *session) _set(ctx *configd.Context, path []string) error {
	//we have to do syntax checking and substitutions at this
	//level because the scripts may need to access the session process
	//so we have to be able to go asynchronous selectively.

	//Check syntax
	err := s.validateSetPath(ctx, path, incompletePathIsInvalid, cfgSchemaOnly)
	if err != nil {
		return err
	}

	sch := schema.Descendant(s.schema, path)
	if sch == nil {
		// Should not happen, but let's be safe
		cerr := mgmterror.NewUnknownElementApplicationError(path[len(path)-1])
		cerr.Path = pathutil.Pathstr(path[:len(path)-1])
		return cerr
	}

	//do substitution
	//if subst then run that and exit
	if subst := sch.ConfigdExt().Subst; len(subst) > 0 {
		errch := make(chan error)
		go func() {
			var err error
			for _, sub := range subst {
				_, err = exec.Exec(exec.Env(s.sid, path, "sub", ""), path, sub)
				if err != nil {
					break
				}
			}
			errch <- err
		}()
		for {
			select {
			case err := <-errch:
				//Exec returned we're done
				if err != nil {
					return err
				}
				return nil
			case req := <-s.reqch:
				s.processreq(req, nil)
			}
		}
	}

	return s.getUnion().Set(s.newAuther(ctx), path)
}

func (s *session) set(ctx *configd.Context, path []string) error {
	var err error
	err = s.trylock(ctx.Pid)
	if err != nil {
		return err
	}

	sauth := s.newAuther(ctx)

	// Need to check authorization BEFORE we do any substitutions in
	// _set() as anything substituted is run with superuser privileges.
	//
	// Cannot do this at a higher level as this function is the first
	// common point for load() and set() operations.
	//
	// edit_config operations come in directly to _set() but check
	// authorization first, so we don't want to repeat that.
	//
	// NB: ctx.AuthUpdate/Create() allow for the 'configd' superuser to set
	//     the likes of the ACM ruleset at bootup.  If you use the underlying
	//     functions (ctx.Auth.AuthorizeUpdate/Create) then you lose the
	//     superuser 'get out of jail free' card and things go pear-shaped.
	//
	if s.existsInTree(s.getUnion(), ctx, path,
		false /* Don't count implicitly set defaults */) {
		if !sauth.AuthUpdate(path) {
			return mgmterror.NewAccessDeniedApplicationError()
		}
	} else {
		if !sauth.AuthCreate(path) {
			return mgmterror.NewAccessDeniedApplicationError()
		}
	}

	return s._set(ctx, path)
}

func (s *session) del(ctx *configd.Context, path []string) error {
	if err := s.trylock(ctx.Pid); err != nil {
		return err
	}
	return s.getUnion().Delete(s.newAuther(ctx), path, union.DontCheckAuth)
}

// When 'def' is true, a node is deemed to exist when it is implicitly set
// to the default value (ie no one has explicitly set it to the default value,
// or to any other value).
//
// When 'def' is false, a node is deemed to exist only when it has an
// explicitly set value (which may or may not match the default value).
//
func (s *session) existsInTree(ut union.Node, ctx *configd.Context, path []string, def bool) bool {
	sauth := s.newAuther(ctx)
	exists := ut.Exists(sauth, path)
	if def {
		return exists == nil
	} else {
		if ok, _ := ut.IsDefault(sauth, path); !ok {
			return exists == nil
		} else {
			return false
		}
	}
}

func (s *session) get(ctx *configd.Context, path []string) ([]string, error) {
	return s.getUnion().Get(s.newAuther(ctx), path)
}

func (s *session) gettype(ctx *configd.Context, path []string) (rpc.NodeType, error) {
	//This is a mapping to the old type returns.
	//It must stay the same for the external API.
	sch := schema.Descendant(s.schema, path)
	switch sch.(type) {
	case schema.Tree:
		return rpc.CONTAINER, nil
	case schema.Container:
		return rpc.CONTAINER, nil
	case schema.List:
		return rpc.LIST, nil
	case schema.ListEntry:
		return rpc.CONTAINER, nil
	case schema.Leaf:
		return rpc.LEAF, nil
	case schema.LeafList:
		return rpc.LEAF_LIST, nil
	case schema.LeafValue:
		return rpc.LEAF, nil
	}
	return rpc.CONTAINER, nil
}

func (s *session) getstatus(ctx *configd.Context, path []string, diffCache *diff.Node) (rpc.NodeStatus, error) {
	if len(path) == 0 {
		// Root is always unchanged
		return rpc.UNCHANGED, nil
	}

	var candidate, running *data.Node

	sauth := s.newAuther(ctx)
	diffTree := diffCache
	if diffTree == nil {
		//if we don't have a diffCache i.e, not in commit
		//do a faster lookup by only processing the required
		//part of the tree.
		ppath := path[:len(path)-1]
		path = path[len(path)-1:]
		ut := s.getUnion()
		un, _ := ut.Descendant(sauth, ppath)
		if un != nil {
			candidate = un.Merge()
		}

		rt := union.NewNode(nil, s.cmgr.Running(), s.schema, nil, 0)
		rn, _ := rt.Descendant(sauth, ppath)
		if rn != nil {
			running = rn.Merge()
		}

		sn := schema.Descendant(s.schema, ppath)
		diffTree = diff.NewNode(candidate, running, sn, nil)
		if diffTree == nil {
			return rpc.UNCHANGED, yang.NewNodeNotExistsError(ppath)
		}
	}
	diffNode := diffTree.Descendant(path)
	if diffNode == nil {
		//TODO: I'd rather we not return an error at all for unknown nodes,
		//      IIRC the upper layer throws away the information anyway
		return rpc.UNCHANGED, yang.NewNodeNotExistsError(path)
	}

	//This is gross, but the old API clients expects exactly this behavior
	//ideally we could use the simple diff output as it actually reflects
	//the useful state of the node.
	_, isLeafVal := diffNode.Schema().(schema.LeafValue)
	parent := diffNode.Parent()
	var parentIsLeaf, parentIsLeafList bool
	if parent != nil {
		_, parentIsLeaf = parent.Schema().(schema.Leaf)
		_, parentIsLeafList = parent.Schema().(schema.LeafList)
	}
	switch {
	case diffNode.Deleted():
		return rpc.DELETED, nil
	case isLeafVal && parentIsLeaf:
		return rpc.CHANGED, nil
	case diffNode.Added():
		return rpc.ADDED, nil
	case diffNode.Changed():
		return rpc.CHANGED, nil
	case isLeafVal && parentIsLeafList && diffNode.Parent().Changed():
		return rpc.CHANGED, nil
	default:
		return rpc.UNCHANGED, nil
	}
}

func (s *session) isdefault(ctx *configd.Context, path []string) (bool, error) {
	if len(path) == 0 {
		return false, nil
	}
	def, err := s.getUnion().IsDefault(s.newAuther(ctx), path)
	return def, err
}

func nodeFinder(
	targetNode yang.Node,
	parentNode *yang.XNode,
	nodeToFind yang.NodeSpec,
	path []string,
	param interface{},
) (bool, bool, []interface{}) {
	tmp_path := append(path, targetNode.Name())
	if !pathsEqual(tmp_path, nodeToFind.Path) {
		return false, true, nil
	}
	return true, true, nil
}

func pathsEqual(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}

	for (len(first) > 0) && (len(second) > 0) {
		if first[0] != second[0] {
			return false
		}
		first = first[1:]
		second = second[1:]
	}

	return true
}

func pathCouldExist(ms schema.ModelSet, path []string) bool {
	nspec := yang.NodeSpec{Path: path}
	sn, _, _ := ms.FindOrWalk(nspec, nodeFinder, nil)
	return (sn != nil)
}

func (s *session) gettree(ctx *configd.Context, path []string, opts *TreeOpts) (union.Node, error) {
	ut := s.getUnion()

	if !s.existsInTree(ut, ctx, path, opts.Defaults) {
		if opts.CouldExistIsAllowed {
			err := s.validateSetPath(
				ctx, path, incompletePathIsValid, cfgSchemaOnly)
			if err == nil {
				return nil, nil
			}
		}
		err := mgmterror.NewUnknownElementApplicationError(path[len(path)-1])
		err.Path = pathutil.Pathstr(path[:len(path)-1])
		return nil, err
	}

	return ut.Descendant(s.newAuther(ctx), path)
}

const (
	stateLogMsgPrefix = "STATE"
	msgPadToLength    = 40
	// 40 + 3 extra just in case
	msgPadding = "                                                  "
)

func msgPad(msg string) string {
	msgLen := len(msg)
	padLen := 0
	if msgLen < msgPadToLength {
		padLen = msgPadToLength - msgLen
	}
	return msg + ": " + msgPadding[:padLen]
}

func logStateTime(logger schema.StateLogger, msg string, startTime time.Time) {
	if logger == nil {
		return
	}
	logger.Printf("%s: %s%s", stateLogMsgPrefix, msgPad(msg),
		time.Since(startTime).Round(time.Millisecond))
}

func logStateEvent(logger schema.StateLogger, msg string) {
	if logger == nil {
		return
	}
	logger.Printf("%s: %s", stateLogMsgPrefix, msg)
}

func stateErrLogEnabled() bool {
	return common.LoggingIsEnabledAtLevel(common.LevelError, common.TypeState)
}

func stateDbgLogEnabled() bool {
	return common.LoggingIsEnabledAtLevel(common.LevelDebug, common.TypeState)
}

func (s *session) getfulltree(ctx *configd.Context, path []string, opts *TreeOpts) (union.Node, error, []error) {

	var errLogger schema.StateLogger
	if stateErrLogEnabled() {
		errLogger = ctx.Elog
	}

	var dbgLogger schema.StateLogger
	if stateDbgLogEnabled() {
		dbgLogger = ctx.Dlog
	}

	logStateEvent(errLogger,
		fmt.Sprintf("Start getfulltree '%v' operation.", path))
	stateStart := time.Now()

	// 1. Handle legacy state scripts
	// This should move to provisiond.
	ut := s.getUnionFull()
	// Asynchronously process state scripts.  These will be inserted with
	// syntax validation performed, but constraint validation will be
	// delayed until we have the full tree present to avoid false errors.
	var errAndWarns errorAndWarnings
	respch := make(chan errorAndWarnings)
	go func() {
		respch <- addStateToTree(ut, path, dbgLogger)
	}()

	//Process requests that don't modify the session during commit
Loop:
	for {
		select {
		case errAndWarns = <-respch:
			break Loop
		case req := <-s.reqch:
			s.processreq(req, nil)
		}
	}
	if errAndWarns.err != nil {
		// We may get an error if a node doesn't exist, but could exist.
		// That might be a config node that hasn't been configured, or a
		// state node that is populated by a VCI component instead of a
		// configd:state script.
		//
		// In such cases, we now check to see if the node could have existed
		// and if so, continue.
		if opts.CouldExistIsAllowed {
			err := s.validateSetPath(
				ctx, path, incompletePathIsValid, fullSchema)
			if err != nil {
				return nil, errAndWarns.err, errAndWarns.warns
			}
		}
	}
	logStateTime(errLogger, "Legacy scripts", stateStart)

	// 2. Convert union tree to an rfc7951 data tree
	// For now marshal to and from rfc7951, eventually this will be more
	// efficient since the session tree will be stored as a data tree,
	/// but this solves the problem in the meantime.
	convertToRFCStart := time.Now()
	ft := rfc7951data.TreeNew()
	err := rfc7951.Unmarshal(
		ut.ToRFC7951(union.Authorizer(s.newAuther(ctx)),
			union.ForceShowSecrets),
		ft,
	)
	if err != nil {
		return nil, err, nil
	}
	logStateTime(errLogger, "Convert to RFC7951 data tree", convertToRFCStart)

	// 3. Merge in component state
	// This should be one Client for the whole sessiond daemon (whenever that
	// is built)
	vciStart := time.Now()
	logStateEvent(errLogger, "Start VCI scripts")
	mrgr := rfc7951utils.NewRFC7951Merger(s.schemaFull, ft)
	client, vciErr := vci.Dial()
	if vciErr == nil {
		defer client.Close()
		// Only do model access when VCI is available. In some testing
		// scenarios VCI is not available. We can rework this after the
		// provisiond split, which will move the test cases
		// for legacy operational state to provisiond.
		modelNames := s.schemaFull.ListActiveModels(ut)
		for _, model := range modelNames {
			compStartTime := time.Now()
			state := rfc7951data.TreeNew()
			err := client.StoreStateByModelInto(model, state)
			if err != nil {
				// No error if component doesn't implement state.
				_, ok :=
					err.(*mgmterror.OperationNotSupportedApplicationError)
				if ok {
					continue
				}
				ctx.Elog.Printf("%s state retrieval failed: %s\n",
					model, err)
				continue
			}
			mrgr.Merge(state)
			logStateTime(errLogger, fmt.Sprintf("  %s", model),
				compStartTime)
		}
	}
	ft = mrgr.Tree()
	logStateTime(errLogger, "End VCI scripts", vciStart)

	// 4. Convert back to a union tree.
	marshalStart := time.Now()
	d, err := rfc7951.Marshal(ft)
	if err != nil {
		return nil, err, nil
	}
	logStateTime(dbgLogger, "Marshal RFC7951 data", marshalStart)

	// Unmarshal will perform validation of the merged tree.  This involves
	// putting the rfc7951 data (d) into a data tree, and then 'set'ting
	// each line into the union tree.
	validationStart := time.Now()
	ut, err = union.NewUnmarshaller(encoding.RFC7951).
		SetValidation(yang.ValidateState).
		Unmarshal(s.schemaFull, d)
	if err != nil {
		return nil, err, nil
	}
	logStateTime(errLogger, "Validate back into union tree", validationStart)

	// 5. Filter based on path
	filterStart := time.Now()
	if !s.existsInTree(ut, ctx, path, opts.Defaults) {
		if opts.CouldExistIsAllowed && pathCouldExist(s.schemaFull, path) {
			return nil, nil, errAndWarns.warns
		}
		cerr := mgmterror.NewUnknownElementApplicationError(path[len(path)-1])
		cerr.Path = pathutil.Pathstr(path[:len(path)-1])
		return nil, nil, errAndWarns.warns
	}
	out, err := ut.Descendant(s.newAuther(ctx), path)
	logStateTime(errLogger, "Filtering", filterStart)

	logStateTime(errLogger, fmt.Sprintf("Overall for path '%v'", path),
		stateStart)
	return out, err, errAndWarns.warns
}

func (s *session) changed(ctx *configd.Context) bool {
	mcan := s.getUnion().Merge()
	c := newctx(s.sid, ctx, nil, mcan, s.getRunning(), s.schema, "", false,
		0 /* no must debug */)
	return commit.Changed(c)
}

func (s *session) validate(ctx *configd.Context) *commitresp {
	var resp *commitresp
	if err := s.trylock(ctx.Pid); err != nil {
		return MakeCommitError(err)
	}

	//Lock the session from changes during validate
	pid, _ := s.locked()
	if pid != 0 {
		s.unlock(ctx.Pid)
		defer s.lock(ctx.Pid)
	}
	s.lock(int32(configd.COMMIT))
	defer s.unlock(int32(configd.COMMIT))

	mcan := s.getUnion().Merge()
	mustThreshold, _ := common.LoggingValueAndStatus(common.TypeMust)
	c := newctx(s.sid, ctx, nil, mcan, s.getRunning(), s.schema, "",
		common.LoggingIsEnabledAtLevel(common.LevelDebug, common.TypeCommit),
		mustThreshold)

	respch := make(chan *commitresp)
	go func() {
		outs, errs, ok := commit.Validate(c)
		respch <- &commitresp{out: outs, err: errs, ok: ok}
	}()

	//Process requests that don't modify the session during commit
Loop:
	for {
		select {
		case resp = <-respch:
			break Loop
		case req := <-s.reqch:
			s.processreq(req, nil)
		}
	}

	return resp
}

func (s *session) lock(pid int32) (int32, error) {
	if s.lpid == 0 {
		s.lpid = pid
		return pid, nil
	} else if s.lpid == pid {
		return pid, lockDenied(strconv.Itoa(int(s.lpid)))
	}
	if s.lpid < 0 {
		return s.lpid, lockDenied(configd.LockId(s.lpid).String())
	}
	return s.lpid, lockDenied(strconv.Itoa(int(s.lpid)))
}

func (s *session) trylock(pid int32) error {
	if s.lpid == 0 {
		//unlocked
		return nil
	} else if s.lpid == pid {
		//locked by self
		return nil
	}
	if s.lpid < 0 {
		return lockDenied(configd.LockId(s.lpid).String())
	}
	return lockDenied(strconv.Itoa(int(s.lpid)))

}

func (s *session) locked() (int32, error) {
	if s.lpid == 0 {
		return s.lpid, nil
	}
	return s.lpid, nil
}

func (s *session) unlock(pid int32) (int32, error) {
	if s.lpid == 0 {
		err := mgmterror.NewOperationFailedProtocolError()
		err.Message = "session is not locked"
		return s.lpid, err
	} else if s.lpid == pid {
		s.lpid = 0
		return pid, nil
	}
	if s.lpid < 0 {
		return s.lpid, lockDenied(configd.LockId(s.lpid).String())
	}
	return s.lpid, lockDenied(strconv.Itoa(int(s.lpid)))
}

func (s *session) comment(ctx *configd.Context, path []string) error {
	if err := s.trylock(ctx.Pid); err != nil {
		return err
	}
	return nil
}

func (s *session) marksaved(ctx *configd.Context, saved bool) error {
	if err := s.trylock(ctx.Pid); err != nil {
		return err
	}
	s.saved = saved
	return nil
}

func (s *session) show(ctx *configd.Context, path []string, hideSecrets, showDefaults, forceShowSecrets bool) (string, error) {
	options := []union.UnionOption{union.Authorizer(s.newAuther(ctx))}
	if hideSecrets {
		options = append(options, union.HideSecrets)
	}
	if showDefaults {
		options = append(options, union.IncludeDefaults)
	}
	if forceShowSecrets {
		options = append(options, union.ForceShowSecrets)
	}
	out, err := s.getUnion().Show(path, options...)
	if err != nil {
		return "", err
	}
	return out, nil
}

func (s *session) discard(ctx *configd.Context) error {
	if err := s.trylock(ctx.Pid); err != nil {
		return err
	}
	s.candidate = data.New("root")
	return nil
}

func (s *session) preCommitChecks(ctx *configd.Context) error {
	// Check that the disk has not entered read-only mode
	err := syscall.Access("/", syscall.O_RDWR)
	if err != nil && err == syscall.Errno(syscall.EROFS) {
		r := s.cmgr.Running()
		if r.NumChildren() != 0 {
			// Block commit, system will get into weird state
			err := mgmterror.NewOperationFailedProtocolError()
			err.Message = "Commit blocked, disk is read-only"
			ctx.Elog.Println(err.Message)
			return err
		} else {
			// Allow system to attempt coming up during reboot
			ctx.Elog.Println("Commit allowed, but disk is read-only.")
		}
	}
	return nil
}

func (s *session) commit(ctx *configd.Context, message string, debug bool) *commitresp {
	var resp *commitresp

	if err := s.trylock(ctx.Pid); err != nil {
		return MakeCommitError(err)
	}

	if !s.changed(ctx) {
		err := mgmterror.NewOperationFailedProtocolError()
		err.Message = "No configuration changes to commit"
		return MakeCommitError(err)
	}
	if err := s.preCommitChecks(ctx); err != nil {
		return MakeCommitError(err)
	}

	//Lock the session from changes during commit
	pid, _ := s.locked()
	if pid != 0 {
		s.unlock(ctx.Pid)
		defer s.lock(ctx.Pid)
	}
	s.lock(int32(configd.COMMIT))
	defer s.unlock(int32(configd.COMMIT))
	//cache the diff tree during commit
	//this is a speed hack to help out legacy
	//scripts.
	diffCache := diff.NewNode(s.getUnion().Merge(), s.getRunning(), s.schema, nil)
	respch := make(chan *commitresp)
	go func() {
		respch <- s.cmgr.Commit(s.sid, ctx, s.candidate, message, debug)
	}()

	//Process requests that don't modify the session during commit
Loop:
	for {
		select {
		case resp = <-respch:
			break Loop
		case req := <-s.reqch:
			s.processreq(req, diffCache)
		}
	}

	if !resp.ok {
		return resp
	}

	s.candidate = data.New("root")
	return resp
}

func (s *session) gethelp(ctx *configd.Context, fromSchema bool, path []string) map[string]string {
	out, _ := s.getUnion().GetHelp(s.newAuther(ctx), fromSchema, path)
	return out
}

func (s *session) processreq(req request, diffCache *diff.Node) {
	switch v := req.(type) {
	case *mergetreereq:
		v.resp <- s.mergetree(v.ctx, v.defaults)
	case *setreq:
		v.resp <- s.set(v.ctx, v.path)
	case *validatesetreq:
		v.resp <- s.validateSetPath(
			v.ctx, v.path, incompletePathIsInvalid, cfgSchemaOnly)
	case *delreq:
		v.resp <- s.del(v.ctx, v.path)
	case *existsreq:
		v.resp <- s.existsInTree(s.getUnion(), v.ctx, v.path, true)
	case *typereq:
		vs, err := s.gettype(v.ctx, v.path)
		v.resp <- typeresp{vs, err}
	case *statusreq:
		vs, err := s.getstatus(v.ctx, v.path, diffCache)
		v.resp <- statusresp{vs, err}
	case *defaultreq:
		vs, err := s.isdefault(v.ctx, v.path)
		v.resp <- defaultresp{vs, err}
	case *getreq:
		vs, err := s.get(v.ctx, v.path)
		v.resp <- getresp{vs, err}
	case *gettreereq:
		vs, err := s.gettree(v.ctx, v.path, v.opts)
		v.resp <- gettreeresp{vs, err}
	case *getfulltreereq:
		vs, err, warns := s.getfulltree(v.ctx, v.path, v.opts)
		v.resp <- getfulltreeresp{vs, err, warns}
	case *validatereq:
		v.resp <- s.validate(v.ctx)
	case *lockreq:
		pid, err := s.lock(v.ctx.Pid)
		v.resp <- lockresp{pid, err}
	case *unlockreq:
		pid, err := s.unlock(v.ctx.Pid)
		v.resp <- lockresp{pid, err}
	case *lockedreq:
		pid, err := s.locked()
		v.resp <- lockresp{pid, err}
	case *commentreq:
		v.resp <- s.set(v.ctx, v.path)
	case *savedreq:
		v.resp <- s.saved
	case *changedreq:
		v.resp <- s.changed(v.ctx)
	case *marksavedreq:
		v.resp <- s.marksaved(v.ctx, v.saved)
	case *showreq:
		d, err := s.show(v.ctx, v.path, v.hideSecrets, v.showDefaults, v.forceShowSecrets)
		v.resp <- showresp{d, err}
	case *discardreq:
		v.resp <- s.discard(v.ctx)
	case *loadreq:
		err, invalidPaths := s.load(v.ctx, v.file, v.reader)
		v.resp <- loadresp{err, invalidPaths}
	case *mergereq:
		err, invalidPaths := s.merge(v.ctx, v.file, nil)
		v.resp <- mergeresp{err, invalidPaths}
	case *commitreq:
		v.resp <- s.commit(v.ctx, v.message, v.debug)
	case *gethelpreq:
		v.resp <- s.gethelp(v.ctx, v.schema, v.path)
	case *editconfigreq:
		v.resp <- s.editConfigXML(v.ctx, v.target, v.defop, v.testopt, v.erropt, v.config)
	case *copyconfigreq:
		v.resp <- s.copyConfig(v.ctx, v.sourceDatastore,
			v.sourceEncoding, v.sourceConfig,
			v.sourceURL, v.targetDatastore, v.targetURL)
	}
}

func (s *session) run() {
	for {
		select {
		case req := <-s.reqch:
			s.processreq(req, nil)
		case <-s.kill:
			close(s.term)
			return
		}
	}
}
