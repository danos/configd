// Copyright (c) 2017-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	spawn "os/exec"

	"github.com/danos/config/auth"
	"github.com/danos/config/data"
	"github.com/danos/config/diff"
	"github.com/danos/config/load"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/configd/common"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/session"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/exec"
	"github.com/danos/utils/pathutil"
	"github.com/danos/vci"
	"github.com/danos/yang/data/datanode"
	yangenc "github.com/danos/yang/data/encoding"
	yang "github.com/danos/yang/schema"
	"github.com/danos/yang/xpath/xutils"
)

func init() {
	exec.NewExecError = func(path []string, err string) error {
		return mgmterror.NewExecError(path, err)
	}
}

func isElemOf(list []string, elem string) bool {
	for _, v := range list {
		if v == elem {
			return true
		}
	}
	return false
}

func (d *Disp) getROSession(db rpc.DB, sid string) *session.Session {
	var sess *session.Session
	var err error
	switch db {
	case rpc.RUNNING:
		sess, err = d.smgr.Get(d.ctx, "RUNNING")
	case rpc.EFFECTIVE:
		sess, err = d.smgr.Get(d.ctx, "EFFECTIVE")
	case rpc.AUTO, rpc.CANDIDATE:
		sess, err = d.smgr.Get(d.ctx, sid)
	}
	if err != nil {
		sess, err = d.smgr.Get(d.ctx, "RUNNING")
	}
	return sess
}

func (d *Disp) normalizePath(ps []string) ([]string, error) {
	return schema.NormalizePath(d.ms, ps)
}

type Disp struct {
	smgr   *session.SessionMgr
	cmgr   *session.CommitMgr
	ms     schema.ModelSet
	msFull schema.ModelSet
	ctx    *configd.Context
}

func (d *Disp) GetConfigSystemFeatures() (map[string]struct{}, error) {
	feats := make(map[string]struct{})

	if _, err := os.Stat("/usr/sbin/chvrf"); err == nil {
		feats[common.RoutingInstanceFeature] = struct{}{}
	}

	if _, err := os.Stat("/opt/vyatta/sbin/vyatta-config-mgmt.pl"); err == nil {
		feats[common.ConfigManagementFeature] = struct{}{}
	}

	if d.loadKeysIsSupported() {
		feats[common.LoadKeysFeature] = struct{}{}
	}
	return feats, nil
}

func (d *Disp) SessionExists(sid string) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, nil
	}
	if sess == nil {
		return false, nil
	}
	return true, nil
}
func (d *Disp) SessionSetup(sid string) (bool, error) {
	_, err := d.smgr.Create(d.ctx, sid, d.cmgr, d.ms, d.msFull, session.Unshared)
	return err == nil, err
}
func (d *Disp) SessionSetupShared(sid string) (bool, error) {
	_, err := d.smgr.Create(d.ctx, sid, d.cmgr, d.ms, d.msFull, session.Shared)
	return err == nil, err
}
func (d *Disp) SessionTeardown(sid string) (bool, error) {
	err := d.smgr.Destroy(d.ctx, sid)
	if err != nil {
		return false, err
	}
	return true, nil
}
func (d *Disp) SessionChanged(sid string) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}
	changed := sess.Changed(d.ctx)
	return changed, nil
}
func (d *Disp) SessionSaved(sid string) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}
	saved := sess.Saved(d.ctx)
	return saved, nil
}
func (d *Disp) SessionMarkSaved(sid string) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}
	sess.MarkSaved(d.ctx, true)
	return true, nil
}
func (d *Disp) SessionMarkUnsaved(sid string) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}
	sess.MarkSaved(d.ctx, false)
	return true, nil
}
func (d *Disp) SessionGetEnv(sid string) (map[string]string, error) {
	return nil, mgmterror.NewOperationNotSupportedApplicationError()
}

func (d *Disp) SessionLock(sid string) (int32, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return -1, err
	}
	return sess.Lock(d.ctx)
}

func (d *Disp) SessionUnlock(sid string) (int32, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return -1, err
	}
	return sess.Unlock(d.ctx)
}

func (d *Disp) SessionLocked(sid string) (int32, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return -1, err
	}
	return sess.Locked(d.ctx)
}

func (d *Disp) authRead(path []string) bool {
	attrs := schema.AttrsForPath(d.msFull, path)
	return d.ctx.Auth.AuthorizeRead(d.ctx.Uid, d.ctx.Groups, path, attrs)
}

func (d *Disp) authDelete(path []string) bool {
	attrs := schema.AttrsForPath(d.msFull, path)
	return d.ctx.Auth.AuthorizeDelete(d.ctx.Uid, d.ctx.Groups, path, attrs)
}

func (d *Disp) authPath(path []string, perm int) bool {
	attrs := schema.AttrsForPath(d.msFull, path)
	return d.ctx.Auth.AuthorizePath(d.ctx.Uid, d.ctx.Groups, path,
		attrs, auth.AuthPerm(perm))
}

func (d *Disp) schemaPathDescendant(ps []string) (*schema.TmplCompat, error) {

	tmpl := d.ms.PathDescendant(ps)
	if tmpl == nil {
		return nil, d.getPathError(ps, "Schema not found")
	}
	return tmpl, nil
}

func (d *Disp) TmplGet(path string) (map[string]string, error) {
	//TODO(jhs): Remove this mess
	m := make(map[string]string)

	ps := pathutil.Makepath(path)

	tmpl, err := d.schemaPathDescendant(ps)
	if err != nil {
		return nil, err
	}

	sn := tmpl.Node
	ext := sn.ConfigdExt()
	ty := sn.Type()

	if tmpl.Val {
		m["is_value"] = "1"
	}

	desc := sn.Description()
	if desc != "" {
		m["comp_help"] = desc
	}

	if ext.Secret {
		m["secret"] = "1"
	}

	if ext.Help != "" {
		m["help"] = ext.Help
	}

	switch v := sn.(type) {
	case schema.List:
		m["tag"] = "1"
		m["key"] = v.Keys()[0]
	case schema.LeafList:
		m["multi"] = "1"
	case schema.Leaf:
		m["default"], _ = v.Default()
	case schema.Container:
		if v.Presence() {
			m["presence"] = "1"
		} else {
			m["presence"] = "0"
		}
		return m, nil
	}

	switch ty.(type) {
	case schema.Empty:
	case schema.Integer, schema.Uinteger:
		m["type"] = "u32"
	case schema.Boolean:
		m["type"] = "bool"
	default:
		m["type"] = "txt"
	}

	return m, nil
}

func (d *Disp) TmplGetChildren(path string) ([]string, error) {
	ps := pathutil.Makepath(path)

	if !d.authRead(ps) {
		return nil, mgmterror.NewAccessDeniedApplicationError()
	}

	tmpl, err := d.schemaPathDescendant(ps)
	if err != nil {
		return nil, err
	}

	if !tmpl.Val {
		switch tmpl.Node.(type) {
		case schema.List:
			return []string{"node.tag"}, nil
		}
	}
	chs := tmpl.Node.Children()
	strs := make([]string, 0, len(chs))
	for _, n := range chs {
		cpath := append(ps, n.Name())
		if !d.authRead(cpath) {
			continue
		}
		if sch, ok := tmpl.Node.(schema.List); ok {
			if n.Name() == sch.Keys()[0] {
				continue
			}
		}
		strs = append(strs, n.Name())
	}

	return strs, nil
}

// MakeNodeRef
//
// Used to convert a config path string ([]string) to the NodeRef format that
// looks like a XPath leafref-type reference to a node.
//
// NB:
//    (1) startNode represents root node
//
//    (2) ps represents path to leaf / leaf-list schema node, but NOT
//        to the value node underneath.  Think of NodeRefs as references
//        to a node generically, rather than to a specific value of that
//        node.
//
//    (3) All NodeRefs are absolute, not relative.
//
//    (4) We generate a single element in the NodeRef for List+ListEntry.
//        We get the key name from the List, and save it for use with the
//        extra data we get from the ListEntry
//
//    (5) This ought to be in configd/pathutil but with the schema reference
//        we end up with a circular reference to packages via the configd/exec
//        package and it all gets very messy trying to unentangle it.  Exercise
//        for the reader on another day ...
//
func MakeNodeRef(ps []string, startNode schema.Node) xutils.NodeRef {
	// Deal with root node (empty ps)
	if len(ps) == 0 {
		return xutils.NodeRef{}
	}

	retPath := xutils.NewNodeRef(0)
	curNode := startNode
	for _, elem := range ps {
		curNode = curNode.SchemaChild(elem)
		switch v := curNode.(type) {
		case schema.ListEntry:
			yangKey := xutils.NewNodeRefKey(v.Keys()[0], elem)
			retPath.AddElem(curNode.Name(), []xutils.NodeRefKey{yangKey})
		case schema.List:
			// Do nothing - if last element in path, handled below.
		case schema.LeafValue:
			// NodeRef stops at the Leaf / LeafList node.  Ignore value.
		default:
			retPath.AddElem(curNode.Name(), nil)
		}
	}

	// If we finish on a ListEntry we need to actually add the key node.
	switch v := curNode.(type) {
	case schema.ListEntry:
		retPath.AddElem(v.Keys()[0], nil)
	}

	return retPath
}

// Get possible options (if any) for completion of this leafref.
// On any error we just return no values - after all, this is just for
// tab completion and the user can still type the value to be validated
// later.
func (d *Disp) getLeafrefVals(
	sid string,
	ps []string,
	lrNode schema.Leafref,
) []string {

	if len(ps) == 0 {
		return []string{} // 'root' can't be a leafref
	}

	// As this operation is a user-requested tab-completion type event,
	// there shouldn't be a performance issue with creating an Xpath node
	// tree and navigating through it once per completion request.
	sess := d.getROSession(rpc.CANDIDATE, sid)
	sessRootNode, err := sess.GetTree(d.ctx, pathutil.Makepath(""),
		&session.TreeOpts{Defaults: false, Secrets: true})
	if err != nil {
		// Silently ignore error - we just don't have any tab-completions.
		return []string{}
	}

	// To evaluate the leafref statement we need a context node representing
	// the leafref.  If one isn't configured (quite likely!) we need to
	// create a temporary one to navigate from ... and ensure we remove it
	// once done.  The gotcha is that if the created dummy node is inside
	// a non-existent list entry, we need to delete the list entry, not the
	// child node, as doing the latter will NOT delete the list entry itself!
	createPS := ps
	if !sess.Exists(d.ctx, ps) {
		testPS := ps[:len(ps)-1]
		index := 0
		for index = 0; index < (len(ps) - 1); index++ {
			if sess.Exists(d.ctx, testPS) {
				break
			}
			testPS = testPS[:(len(testPS) - 1)]
		}
		deletePS := ps[:(len(ps) - index)]
		createPS = append(ps, "dummyValue")
		err = sess.Set(d.ctx, createPS)
		if err != nil {
			return []string{}
		}
		defer sess.Delete(d.ctx, deletePS)
	}

	// Once we have our (possibly dummy) node in the session union tree,
	// we need to create our Xpath root node and then locate the leafref
	// within the XpathNode tree to use as the context node for XPATH
	// evaluation.
	xRootNode := yang.ConvertToXpathNode(
		sessRootNode, sessRootNode.GetSchema())

	// Where there are multiple leaves at the same level with the same path
	// we only need to find one of them as our start point.
	xLeafRefNode := xutils.FindNode(
		xRootNode, MakeNodeRef(createPS, sessRootNode.GetSchema()))

	// Finally, run the Xpath expression and extract any values found.
	// If we get an error, just don't return any values.
	leafrefVals, err := lrNode.AllowedValues(xLeafRefNode, false)
	if err != nil {
		return []string{}
	}

	if len(leafrefVals) == 0 {
		return []string{}
	}
	return leafrefVals
}

func (d *Disp) TmplGetAllowed(sid, path string) ([]string, error) {
	ps := pathutil.Makepath(path)

	if !d.authRead(ps) {
		return nil, mgmterror.NewAccessDeniedApplicationError()
	}

	tmpl, err := d.schemaPathDescendant(ps)
	if err != nil {
		return nil, err
	}

	// If this is a leafref (but NOT the leaf value - Type() returns same
	// for both) then we need to get possible completions and return.
	if lrNode, ok := tmpl.Node.Type().(schema.Leafref); ok {
		if _, ok := tmpl.Node.(schema.LeafValue); !ok {
			leafrefVals := d.getLeafrefVals(sid, ps, lrNode)
			return leafrefVals, nil
		}
	}

	allowed := tmpl.Node.ConfigdExt().Allowed
	if allowed == "" || tmpl.Val {
		return []string{}, nil
	}
	/*
	 * Ignore stderr, we are mimicing the old implementation because of
	 * bugs in the exec'd scripts
	 */
	out, execErr := exec.ExecNoErr(exec.Env(sid, ps, "allowed", ""), ps, allowed)
	if execErr != nil {
		return nil, execErr
	}
	if out == nil {
		//no output
		return []string{}, nil
	}
	allowedvals := strings.Split(strings.TrimSpace(
		strings.Replace(out.Output, "\n", " ", -1)), " ")
	for i, v := range allowedvals {
		allowedvals[i] = strings.Replace(strings.Replace(v, "<", "\\<", -1), ">", "\\>", -1)
	}
	return allowedvals, nil
}

func (d *Disp) TmplValidatePath(path string) (bool, error) {
	ps := pathutil.Makepath(path)
	if _, err := d.schemaPathDescendant(ps); err != nil {
		return false, nil
	}
	return true, nil
}

func (d *Disp) TmplValidateValues(path string) (bool, error) {
	ps := pathutil.Makepath(path)

	if !d.authRead(ps) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}
	vctx := schema.ValidateCtx{
		Path:    path,
		CurPath: ps,
		Sid:     "RUNNING",
	}
	err := d.ms.Validate(vctx, []string{}, ps)
	return err == nil, nil
}

func (d *Disp) EditGetEnv(sid string) (map[string]string, error) {
	return nil, mgmterror.NewOperationNotSupportedApplicationError()
}

// NodeGet
func (d *Disp) Get(db rpc.DB, sid string, path string) ([]string, error) {
	ps := pathutil.Makepath(path)

	if !d.authRead(ps) {
		return nil, mgmterror.NewAccessDeniedApplicationError()
	}

	sess := d.getROSession(db, sid)
	chs, err := sess.Get(d.ctx, ps)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, ch := range chs {
		cpath := append(ps, ch)
		if !d.authRead(cpath) {
			continue
		}
		out = append(out, ch)
	}
	return out, nil
}

func (d *Disp) GetCommitLog() (map[string]string, error) {
	comps := make(map[string]string)
	buf, err := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
		"--action=show-commit-log-brief").Output()
	if err != nil {
		return comps, err
	}
	out := string(buf)
	vals := strings.Split(out, " ")
	for i, v := range vals {
		if v == "" {
			// Skip empty entries
			continue
		}
		val := strings.Replace(v, "_", " ", -1)
		comps[strconv.Itoa(i)] = val
	}
	return comps, nil
}

func (d *Disp) validatePath(ps []string) error {

	var sn schema.Node = d.ms

	for i, v := range ps {
		sn = sn.SchemaChild(v)
		if sn == nil {
			err := mgmterror.NewUnknownElementApplicationError(v)
			err.Path = pathutil.Pathstr(ps[:i])
			return err
		}
	}

	return nil
}

func (d *Disp) getPathError(ps []string, unexpected string) error {
	if err := d.validatePath(ps); err != nil {
		return err
	}
	err := mgmterror.NewOperationFailedApplicationError()
	err.Message = unexpected
	return err
}

// NodeExists
func (d *Disp) Exists(db rpc.DB, sid string, path string) (bool, error) {

	ps := pathutil.Makepath(path)
	if err := d.validatePath(ps); err != nil {
		return false, common.FormatConfigPathError(err)
	}

	if !d.authRead(ps) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	sess := d.getROSession(db, sid)
	return sess.Exists(d.ctx, ps), nil
}
func (d *Disp) NodeGetStatus(db rpc.DB, sid string, path string) (rpc.NodeStatus, error) {
	ps := pathutil.Makepath(path)

	if !d.authRead(ps) {
		return rpc.UNCHANGED, mgmterror.NewAccessDeniedApplicationError()
	}

	sess := d.getROSession(db, sid)
	return sess.GetStatus(d.ctx, ps)
}

func (d *Disp) NodeIsDefault(db rpc.DB, sid string, path string) (bool, error) {
	ps := pathutil.Makepath(path)

	if !d.authRead(ps) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	sess := d.getROSession(db, sid)
	return sess.IsDefault(d.ctx, ps)
}

func (d *Disp) NodeGetType(sid string, path string) (rpc.NodeType, error) {
	ps := pathutil.Makepath(path)

	if !d.authRead(ps) {
		return rpc.CONTAINER, mgmterror.NewAccessDeniedApplicationError()
	}

	sess := d.getROSession(rpc.AUTO, sid)
	return sess.GetType(d.ctx, ps)
}

func (d *Disp) NodeGetCompleteEnv(sid string, path string) (map[string]int, error) {
	return nil, mgmterror.NewOperationNotSupportedApplicationError()
}

func (d *Disp) NodeGetComment(sid string, path string) (map[string]int, error) {
	return nil, mgmterror.NewOperationNotSupportedApplicationError()
}

// NOTE: ps must already have been normalized
func (d *Disp) setInternal(sid string, ps []string) (string, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return "", err
	}

	err = sess.Set(d.ctx, ps)
	if err != nil {
		return "", common.FormatConfigPathErrorMultiline(err)
	}
	return "", nil
}

func (d *Disp) Set(sid string, path string) (string, error) {
	//Set data authorization is done in session_internal

	ps, err := d.normalizePath(pathutil.Makepath(path))
	if err != nil {
		return "", common.FormatConfigPathErrorMultiline(err)
	}

	// Do command authorization now
	args := d.newCommandArgsForAaa("set", nil, ps)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.setInternal(sid, ps)
	})
}

func (d *Disp) deleteInternal(sid string, ps []string) (bool, error) {
	if !d.authDelete(ps) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}

	err = sess.Delete(d.ctx, ps)
	if err != nil {
		return false, common.FormatConfigPathErrorMultiline(err)
	}
	return true, nil
}

func (d *Disp) Delete(sid string, path string) (bool, error) {
	ps := pathutil.Makepath(path)

	args := d.newCommandArgsForAaa("delete", nil, ps)
	if !d.authCommand(args) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapBoolErr(args, func() (interface{}, error) {
		return d.deleteInternal(sid, ps)
	})
}

func (d *Disp) Rename(sid string, fpath string, tpath string) (bool, error) {
	return false, mgmterror.NewOperationNotSupportedApplicationError()
}
func (d *Disp) Copy(sid string, fpath string, tpath string) (bool, error) {
	/*srv := d.conn.srv
	ps := normalizePath(pathutil.Makepath(path))

	//Copy authorization is done in session_internal

	sess, err := srv.smgr.Get(d.ctx, sid)
	if err != nil {
		return "", err
	}

	err = sess.Copy(d.ctx, ps)
	if err != nil {
		return "", err
	}
	return "", nil*/

	return false, mgmterror.NewOperationNotSupportedApplicationError()
}
func (d *Disp) Comment(sid string, path string) (bool, error) {
	return false, mgmterror.NewOperationNotSupportedApplicationError()
}

func (d *Disp) logRollbackError(err error) {
	d.logRollbackEvent(fmt.Sprintf("Failed with error: %s", err))
}

func (d *Disp) logRollbackEvent(msg string) {
	d.logEvent("Rollback", msg)
}

func (d *Disp) logConfirmedCommitEvent(msg string) {
	d.logEvent("Confirmed Commit", msg)
}

func (d *Disp) logEvent(pfx, msg string) {
	// Log only the first non-blank line
	for _, s := range strings.Split(msg, "\n") {
		if s != "" {
			d.ctx.Wlog.Println(pfx + ": " + s)
			break
		}
	}
}

func (d *Disp) loadArchivedConfig(sid, revision string) error {
	// Open the archived config file
	cfgFile, err := os.Open(configRevisionFileName(revision))
	if err != nil {
		d.logRollbackError(err)
		return err
	}
	defer cfgFile.Close()

	cfgFileReader, err := d.cfgFileReader(cfgFile)
	if err != nil {
		d.logRollbackError(err)
		return err
	}

	// Load the archived config file
	ok, err := d.loadReportWarningsReader(sid, cfgFile.Name(), cfgFileReader)
	if !ok {
		d.logRollbackError(err)
		return err
	}

	return nil
}

func (d *Disp) rollbackCommandAuthArgs(rev, comment string) *commandArgs {
	cmd := "rollback"

	args := make([]string, 0)
	if rev == "revert" {
		cmd = "cancel-commit"
	} else {
		args = append(args, rev)
	}
	if comment != "" {
		args = append(args, "comment", comment)
	}
	return d.newCommandArgsForAaa(cmd, args, nil)
}

func (d *Disp) sessionTermination() error {

	info := getConfirmedCommitInfo()
	if info.Session != "" && info.PersistId == "" &&
		info.Session == strconv.Itoa(int(d.ctx.Pid)) {
		cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
			"--action=revert-configuration")
		out, err := cmd.CombinedOutput()
		// out contains the output of both stdout and stderr. err is not really
		// user relevant so shouldn't be printed.
		if err != nil {
			err := mgmterror.NewOperationFailedApplicationError()
			err.Message = string(out)
			return err
		}
	}
	return nil
}

func (d *Disp) CancelCommit(sid, comment, persistid string, force, debug bool) (string, error) {
	info := getConfirmedCommitInfo()
	if !force {
		switch {
		case info.Session == "":
			err := mgmterror.NewOperationFailedApplicationError()
			err.Message = "No confirmed commit pending"
			return "", err
		case info.PersistId != persistid:
			err := mgmterror.NewInvalidValueProtocolError()
			err.Message = "persist-id does not match pending confirmed commit"
			return "", err
		case info.PersistId == "" && info.Session != strconv.Itoa(int(d.ctx.Pid)):
			err := mgmterror.NewAccessDeniedApplicationError()
			err.Message = "Pending confirmed commit initiated by another session"
			return "", err
		}
	}
	d.logConfirmedCommitEvent("Cancelling pending confirmed-commit with persist-id [" + info.PersistId + "]")

	res, err := d.Rollback(sid, "revert", comment, debug)
	return res, err
}

func (d *Disp) rollbackInternal(sid, revision, comment string, debug bool) (string, error) {
	var retStr string

	d.ConfirmSilent(sid)
	d.logRollbackEvent("Commit/Rollback operation - any pending rollback cancelled.")

	sessChngd, err := d.SessionChanged(sid)
	if err != nil {
		d.logRollbackError(err)
		return retStr, err
	}
	if sessChngd {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = fmt.Sprintf("%s\n%s",
			"Cannot rollback: configuration modified.",
			"Commit or discard the changes before rollback.")
		d.logRollbackError(err)
		return retStr, err
	}

	if revision != "revert" {
		log, _ := d.GetCommitLog()
		if _, exists := log[revision]; !exists {
			err := newInvalidConfigRevisionError(revision)
			d.logRollbackError(err)
			return retStr, err
		}

		d.logConfirmedCommitEvent(fmt.Sprintf("Reverting confirmed-commit revision %s [%s] from archive",
			revision, log[revision]))
	} else {
		if _, err := os.Stat(configRevisionFileName(revision)); err != nil {
			if os.IsNotExist(err) {
				err := mgmterror.NewOperationFailedApplicationError()
				err.Message = "No pending confirmed commit to cancel\n"
				return retStr, err
			}
		}
	}

	err = d.loadArchivedConfig(sid, revision)
	if err != nil {
		return retStr, err
	}

	// commit the changes
	sessChngd, err = d.SessionChanged(sid)
	if err != nil {
		d.logRollbackError(err)
		return retStr, err
	}
	if sessChngd {
		out, err := d.commitInternal(sid, comment, debug, 0, revision == "revert")
		if out != "" {
			retStr += out + "\n"
		}
		if err != nil {
			d.logRollbackError(err)
			return retStr, err
		}
	}
	d.logRollbackEvent("Completed successfully")

	return retStr, nil
}

func (d *Disp) Rollback(sid, revision, comment string, debug bool) (string, error) {
	args := d.rollbackCommandAuthArgs(revision, comment)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.rollbackInternal(sid, revision, comment, debug)
	})
}

func (d *Disp) confirmInternal(sid string) (string, error) {
	cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
		"--action=confirm")
	out, err := cmd.CombinedOutput()
	// out contains the output of both stdout and stderr. err is not really
	// user relevant so shouldn't be printed.
	if err != nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = string(out)
		return "", err
	}
	return string(out), err
}

func (d *Disp) Confirm(sid string) (string, error) {
	args := d.newCommandArgsForAaa("confirm", nil, nil)
	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.confirmInternal(sid)
	})
}

func (d *Disp) confirmPersistIdInternal(persistid string) (string, error) {
	cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
		"--action=confirm",
		fmt.Sprintf("--persistid=%s", persistid))
	out, err := cmd.CombinedOutput()
	// out contains the output of both stdout and stderr. err is not really
	// user relevant so shouldn't be printed.
	if err != nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = string(out)
		return "", err
	}
	return string(out), err
}

func (d *Disp) ConfirmPersistId(persistid string) (string, error) {
	args := d.newCommandArgsForAaa(
		"confirm", []string{"persist-id", persistid}, nil)

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.confirmPersistIdInternal(persistid)
	})
}

func (d *Disp) ConfirmingCommit() (string, error) {
	cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
		"--action=confirming-commit")
	out, err := cmd.CombinedOutput()
	// out contains the output of both stdout and stderr. err is not really
	// user relevant so shouldn't be printed.
	if err != nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = string(out)
		return "", err
	}
	return string(out), err

}

func (d *Disp) ConfirmSilent(sid string) (string, error) {
	cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
		"--action=confirm-silent")
	out, err := cmd.CombinedOutput()
	// out contains the output of both stdout and stderr. err is not really
	// user relevant so shouldn't be printed.
	if err != nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = string(out)
		return "", err
	}
	return string(out), err
}

func (d *Disp) setConfirmedCommitTimeout(cmt *commitInfo) (string, error) {
	cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
		cmt.arguments(strconv.Itoa(int(d.ctx.Pid)))...)
	out, err := cmd.CombinedOutput()
	// out contains the output of both stdout and stderr. err is not really
	// user relevant so shouldn't be printed.
	if err != nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = string(out)
		return "", err
	} else {
		d.logConfirmedCommitEvent("Scheduled revert for persist-id [" + cmt.persist + "]")
	}
	return string(out), err
}
func (d *Disp) setConfirmTimeout(mins int) (string, error) {
	cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl",
		"--action=commit-confirm",
		fmt.Sprintf("--minutes=%d", mins))
	out, err := cmd.CombinedOutput()
	// out contains the output of both stdout and stderr. err is not really
	// user relevant so shouldn't be printed.
	if err != nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = string(out)
		return "", err
	}
	return string(out), err
}

func (d *Disp) CommitConfirm(
	sid string,
	message string,
	debug bool,
	mins int,
) (string, error) {
	args := []string{fmt.Sprintf("%d", mins)}
	if message != "" {
		args = append(args, "comment", message)
	}
	cmdArgs := d.newCommandArgsForAaa("commit-confirm", args, nil)

	return d.accountCmdWrapStrErr(cmdArgs, func() (interface{}, error) {
		return d.commitInternal(sid, message, debug, mins, false)
	})
}

func (d *Disp) Commit(
	sid string,
	message string,
	debug bool,
) (string, error) {
	var args []string
	if message != "" {
		args = append(args, "comment", message)
	}
	cmdArgs := d.newCommandArgsForAaa("commit", args, nil)

	return d.accountCmdWrapStrErr(cmdArgs, func() (interface{}, error) {
		return d.commitInternal(sid, message, debug, 0, false)
	})
}

func (d *Disp) ConfirmedCommit(
	sid string,
	message string,
	confirmed bool,
	timeout string,
	persist string,
	persistid string,
	debug bool,
) (string, error) {
	var args []string
	if message != "" {
		args = append(args, "comment", message)
	}

	cmt, err := newCommitInfo(confirmed, timeout, persist, persistid)
	if err != nil {
		return "", err
	}

	cmdArgs := d.newCommandArgsForAaa("commit", args, nil)
	return d.accountCmdWrapStrErr(cmdArgs, func() (interface{}, error) {
		return d.confirmedCommitInternal(sid, message, debug, 0, cmt, false)
	})
}

func (d *Disp) commitInternal(
	sid string,
	message string,
	debug bool,
	confirmTimeout int,
	revert bool,
) (string, error) {
	return d.confirmedCommitInternal(sid, message, debug, confirmTimeout, nil, revert)
}

func (d *Disp) confirmedCommitInternal(
	sid string,
	message string,
	debug bool,
	confirmTimeout int,
	cmt *commitInfo,
	revert bool,
) (string, error) {

	var rpcout bytes.Buffer

	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return "", err
	}

	confirming, err := d.performConfirmingCommitIfRequired(strconv.Itoa(int(d.ctx.Pid)), cmt, revert)
	if err != nil {
		return "", err
	}

	if confirming && !sess.Changed(d.ctx) {
		// Confirming commit, but no changes to commit
		return "", err
	}

	outs, errs, ok := sess.Commit(d.ctx, message, debug)

	if outs != nil {
		for _, out := range outs {
			if out == nil {
				continue
			}
			if out.Path != nil {
				rpcout.WriteString(fmt.Sprint(out.Path))
				rpcout.WriteByte('\n')
			}
			if out.Output != "" {
				rpcout.WriteString(out.Output)
				rpcout.WriteByte('\n')
			}
		}
	}

	if ok && len(errs) == 0 {
		if ok, err := d.Save(""); !ok {
			return "", err
		}
		if cmt != nil && cmt.confirmed {

			out, err := d.setConfirmedCommitTimeout(cmt)
			if out != "" {
				rpcout.WriteByte('\n')
				rpcout.WriteString(out)
				rpcout.WriteByte('\n')
			}
			if err != nil {
				errs = append(errs, err)
			}
		} else if confirmTimeout != 0 {
			out, err := d.setConfirmTimeout(confirmTimeout)
			rpcout.WriteByte('\n')
			rpcout.WriteString(out)
			rpcout.WriteByte('\n')
			return rpcout.String(), err
		}
		return rpcout.String(), nil
	}

	var merr mgmterror.MgmtErrorList
	merr.MgmtErrorListAppend(errs...)
	if ok {
		if len(errs) != 0 {
			rpcout.WriteString(merr.CustomError(common.FormatCommitOrValErrors))
			rpcout.WriteByte('\n')
		}
		rpcout.WriteString(
			"\nCommit succeeded (non-fatal failures detected).\n")
		return rpcout.String(), nil
	}

	// NB: a validation error found during commit will be reported as a commit
	//     failure, with validation errors printed out.
	return "", merr
}

func (d *Disp) Compare(old, new, spath string, ctxdiff bool) (string, error) {
	t1, err := load.LoadStringNoValidate("old", old)
	if err != nil {
		return "", err
	}

	t2, err := load.LoadStringNoValidate("new", new)
	if err != nil {
		return "", err
	}

	dtree := diff.NewNode(t1, t2, d.ms, nil)
	dtree = dtree.Descendant(pathutil.Makepath(spath))
	hide := !configd.InSecretsGroup(d.ctx)
	return dtree.Serialize(ctxdiff, diff.HideSecrets(hide)), nil
}

func (d *Disp) validCompareConfigRevision(revision string) bool {
	if revision == "saved" || revision == "session" {
		return true
	}

	log, _ := d.GetCommitLog()
	_, exists := log[revision]
	return exists
}

func newInvalidConfigRevisionError(revision string) error {
	err := mgmterror.NewOperationFailedApplicationError()
	err.Message = "Invalid revision [" + revision + "]"
	return err
}

func (d *Disp) compareConfigRevisionsInternal(sid, revOne, revTwo string) (string, error) {
	if !d.validCompareConfigRevision(revOne) {
		return "", newInvalidConfigRevisionError(revOne)
	}
	if !d.validCompareConfigRevision(revTwo) {
		return "", newInvalidConfigRevisionError(revTwo)
	}

	var one string
	var err error
	if revOne == "session" {
		candSess := d.getROSession(rpc.CANDIDATE, sid)
		one, err = candSess.ShowForceSecrets(d.ctx, nil, false, false)
	} else {
		one, err = d.readConfigFileForceShowSecrets(configRevisionFileName(revOne))
	}
	if err != nil {
		return "", err
	}

	two, err := d.readConfigFileForceShowSecrets(configRevisionFileName(revTwo))
	if err != nil {
		return "", err
	}

	return d.Compare(one, two, "", true)
}

func (d *Disp) CompareConfigRevisions(sid, revOne, revTwo string) (string, error) {
	authArgs := []string{revTwo}
	if revOne != "session" {
		authArgs = append([]string{revOne}, authArgs...)
	}
	args := d.newCommandArgsForAaa("compare", authArgs, nil)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.compareConfigRevisionsInternal(sid, revOne, revTwo)
	})
}

func (d *Disp) compareSessionChangesInternal(sid string) (string, error) {
	runningSess := d.getROSession(rpc.RUNNING, sid)
	candSess := d.getROSession(rpc.CANDIDATE, sid)

	runningShow, err := runningSess.ShowForceSecrets(d.ctx, nil, false, false)
	if err != nil {
		return "", err
	}

	candShow, err := candSess.ShowForceSecrets(d.ctx, nil, false, false)
	if err != nil {
		return "", err
	}

	return d.Compare(candShow, runningShow, "", true)
}

func (d *Disp) CompareSessionChanges(sid string) (string, error) {
	args := d.newCommandArgsForAaa("compare", nil, nil)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.compareSessionChangesInternal(sid)
	})
}

// If conforms to interface

func (d *Disp) discardInternal(sid string) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}

	err = sess.Discard(d.ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *Disp) Discard(sid string) (bool, error) {
	args := d.newCommandArgsForAaa("discard", nil, nil)

	return d.accountCmdWrapBoolErr(args, func() (interface{}, error) {
		return d.discardInternal(sid)
	})
}

func (d *Disp) ExtractArchive(sid, revision, destination string) (string, error) {
	cmd := spawn.Command("/opt/vyatta/sbin/vyatta-config-mgmt.pl", "--action=extract-archive", "--revnum="+revision, "--dest="+destination)
	out, err := cmd.CombinedOutput()
	// out contains the output of both stdout and stderr. err is not really
	// user relevant so shouldn't be printed.
	if err != nil {
		cerr := mgmterror.NewOperationFailedApplicationError()
		cerr.Message = string(out)
		return "", cerr
	}
	return string(out), err
}

func (d *Disp) Save(_ string) (bool, error) {
	// In order to save the boot configuration we must raise privileges
	// which allows us to do several things:
	//   1) Access the entire config without being subjected to ACM
	//   2) Obtain un-redacted secrets
	//   3) Write to /config/config.boot which is owned by root
	if !d.ctx.Configd {
		d.ctx.RaisePrivileges()
		defer d.ctx.DropPrivileges()
	}
	return d.SaveTo("/config/config.boot", "")
}

func (d *Disp) Load(sid string, file string) (bool, error) {
	ok, errOrWarns := d.LoadReportWarnings(sid, file)
	if ok {
		// Suppress warnings to maintain legacy behaviour.
		return ok, nil
	}
	return ok, errOrWarns
}

func (d *Disp) LoadReportWarnings(sid string, file string) (bool, error) {
	args := d.newCommandArgsForAaa("load", []string{file}, nil)
	if !d.authCommand(args) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapBoolErr(args, func() (interface{}, error) {
		return d.loadReportWarningsReader(sid, file, nil)
	})
}

func (d *Disp) loadReportWarningsReader(sid string, file string, r io.Reader) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}

	err, warns := sess.Load(d.ctx, file, r)
	if err != nil {
		return false, err
	}

	return true, common.FormatWarnings(warns)
}

func (d *Disp) Merge(sid string, file string) (bool, error) {
	ok, errOrWarns := d.MergeReportWarnings(sid, file)
	if ok {
		// Suppress warnings to suppress legacy behaviour.
		return ok, nil
	}
	return ok, errOrWarns
}

func (d *Disp) mergeReportWarningsInternal(sid string, file string) (bool, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return false, err
	}

	err, warns := sess.Merge(d.ctx, file)
	if err != nil {
		return false, err
	}

	return true, common.FormatWarnings(warns)
}

func (d *Disp) MergeReportWarnings(sid string, file string) (bool, error) {
	args := d.cfgMgmtCommandArgs("merge", file, "", "")
	if !d.authCommand(args) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapBoolErr(args, func() (interface{}, error) {
		return d.mergeReportWarningsInternal(sid, file)
	})
}

func (d *Disp) validateInternal(sid string) (string, error) {
	var rpcout bytes.Buffer
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return "", err
	}

	outs, errs, ok := sess.Validate(d.ctx)
	if outs != nil {
		for _, out := range outs {
			if out == nil {
				continue
			}
			rpcout.WriteString(fmt.Sprint(out.Path))
			rpcout.WriteByte('\n')
			rpcout.WriteString(out.Output)
			rpcout.WriteByte('\n')
		}
	}
	if ok {
		return rpcout.String(), nil
	}

	var merr mgmterror.MgmtErrorList
	merr.MgmtErrorListAppend(errs...)
	return "", merr
}

func (d *Disp) Validate(sid string) (string, error) {
	args := d.newCommandArgsForAaa("validate", nil, nil)

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.validateInternal(sid)
	})
}

func (d *Disp) validateConfigInternal(sid, encoding, config string) (string, error) {
	sn := "VALIDATE" + strconv.Itoa(int(d.ctx.Pid))
	_, err := d.SessionSetup(sn)
	if err != nil {
		return "", err
	}
	defer d.SessionTeardown(sn)
	sess := d.getROSession(rpc.CANDIDATE, sn)
	if err != nil {
		return "", err
	}

	err = sess.CopyConfig(d.ctx, "", encoding, config, "", "candidate", "")
	if err != nil {
		return "", err
	}
	return d.Validate(sn)
}

func (d *Disp) ValidateConfig(sid, encoding, config string) (string, error) {
	args := d.newCommandArgsForAaa("validate", nil, nil)

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.validateConfigInternal(sid, encoding, config)
	})
}

func (d *Disp) ValidatePath(sid string, path string) (string, error) {
	ps, err := d.normalizePath(pathutil.Makepath(path))
	if err != nil {
		return "", err
	}

	if !d.authRead(ps) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	sess := d.getROSession(rpc.AUTO, sid)
	err = sess.ValidateSet(d.ctx, ps)
	if err != nil {
		return "", err
	}
	return "", nil
}

func (d *Disp) showCommandArgs(path []string, showDefaults bool) *commandArgs {
	var args []string
	if showDefaults {
		args = append(args, "-all")
	}

	return d.newCommandArgsForAaa("show", args, path)
}

func (d *Disp) show(db rpc.DB, sid string, path []string, hideSecrets, showDefaults bool) (string, error) {
	sess := d.getROSession(db, sid)
	return sess.Show(d.ctx, path, hideSecrets, showDefaults)
}

func (d *Disp) Show(db rpc.DB, sid string, path string, hideSecrets bool) (string, error) {
	ps := pathutil.Makepath(path)

	args := d.showCommandArgs(ps, false)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.show(db, sid, ps, hideSecrets, false)
	})
}

func (d *Disp) ShowDefaults(db rpc.DB, sid string, path string, hideSecrets bool) (string, error) {
	ps := pathutil.Makepath(path)

	args := d.showCommandArgs(ps, true)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.show(db, sid, ps, hideSecrets, true)
	})
}

func (d *Disp) showConfigWithContextDiffsInternal(
	sid string, path string, showDefaults bool,
) (string, error) {
	runningSess := d.getROSession(rpc.RUNNING, sid)
	candSess := d.getROSession(rpc.CANDIDATE, sid)

	runningShow, err := runningSess.ShowForceSecrets(d.ctx, nil, false, showDefaults)
	if err != nil {
		return "", err
	}

	candShow, err := candSess.ShowForceSecrets(d.ctx, nil, false, showDefaults)
	if err != nil {
		return "", err
	}

	return d.Compare(candShow, runningShow, path, false)
}

func (d *Disp) ShowConfigWithContextDiffs(sid string, path string, showDefaults bool) (string, error) {
	ps := pathutil.Makepath(path)

	args := d.showCommandArgs(ps, showDefaults)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.showConfigWithContextDiffsInternal(sid, path, showDefaults)
	})
}

func (d *Disp) AuthAuthorize(path string, perm int) (bool, error) {
	ps, err := d.normalizePath(pathutil.Makepath(path))
	if err != nil {
		return false, err
	}

	return d.authPath(ps, perm), nil
}

func (d *Disp) AuthGetPerms() (map[string]string, error) {
	return d.ctx.Auth.GetPerms(d.ctx.Groups), nil
}

func (d *Disp) TreeGet(db rpc.DB, sid, path, encoding string, flags map[string]interface{}) (string, error) {
	ps := pathutil.Makepath(path)
	sess := d.getROSession(db, sid)

	opts := session.NewTreeOpts(flags)
	// For NETCONF, it's not an error if a node could exist, but currently
	// is not configured.
	if encoding == "netconf" {
		opts.AllowCouldExist()
	}

	ut, err := sess.GetTree(d.ctx, ps, opts)
	if err != nil {
		return fixupEmptyStringForEncoding("", encoding), err
	}
	if ut == nil {
		return fixupEmptyStringForEncoding("", encoding), nil
	}

	options := opts.ToUnionOptions()
	options = append(options, union.Authorizer(sess.NewAuther(d.ctx)))
	return ut.Marshal("data", encoding, options...)
}

func (d *Disp) TreeGetFull(
	db rpc.DB, sid, path, encoding string,
	flags map[string]interface{},
) (string, error) {

	out, err, _ := d.TreeGetFullWithWarnings(db, sid, path, encoding, flags)
	return out, err
}

func (d *Disp) printWarnings(warns []error) {
	// Print warnings to journal.
	if len(warns) > 0 {
		d.smgr.Elog.Println(
			"Warnings generated creating full config and state tree:")
		for _, warn := range warns {
			d.smgr.Elog.Printf("\t%s", warn)
		}
	}
}

func fixupEmptyStringForEncoding(out, encoding string) string {
	if out == "" {
		switch encoding {
		case "json", "internal", "rfc7951":
			out = "{}"
		case "xml", "netconf":
			// See https://tools.ietf.org/html/rfc6241#section-6.4.2
			// This suggests the paired data tag format below is correct
			// for an 'empty' response.
			out = "<data></data>"
		}
	}
	return out
}

// TreeGetFullWithWarnings - full tree including state, with any warnings.
// Warnings are non-fatal, eg a specific node's state function(s) may have
// failed, but we can still return overall state and config.
// Error indicates a non-existent path typically - something we shouldn't
// ignore.
func (d *Disp) TreeGetFullWithWarnings(
	db rpc.DB, sid, path, encoding string,
	flags map[string]interface{},
) (string, error, []error) {

	ps := pathutil.Makepath(path)
	sess := d.getROSession(db, sid)

	opts := session.NewTreeOpts(flags)
	// Unconditionally allow for nodes that could exist, but don't have
	// any current config, or are state nodes.  This allows us to return
	// empty data rather than an error, saving that for when the path could
	// never exist, or something else went wrong.
	opts.AllowCouldExist()

	ut, err, warns := sess.GetFullTree(d.ctx, ps, opts)
	d.printWarnings(warns)
	if err != nil {
		return fixupEmptyStringForEncoding("", encoding), err, warns
	}
	// All ok - simply a case of no data to return for the requested path.
	// Most likely when requesting a specific piece of state or config deep
	// down in the YANG tree.
	if ut == nil {
		return fixupEmptyStringForEncoding("", encoding), nil, warns
	}

	// Return sub-tree with target node as root
	options := opts.ToUnionOptions()
	options = append(options, union.Authorizer(sess.NewAuther(d.ctx)))
	out, err := ut.Marshal("data", encoding, options...)

	return fixupEmptyStringForEncoding(out, encoding), err, warns
}

func (d *Disp) getModuleOrSubmoduleSchema(modOrSubmod string) (string, error) {
	mod, ok := d.ms.Modules()[modOrSubmod]
	if !ok {
		submod, ok := d.ms.Submodules()[modOrSubmod]
		if !ok {
			err := mgmterror.NewOperationFailedApplicationError()
			err.Message = fmt.Sprintf("unknown (sub)module %s", modOrSubmod)
			return "", err
		}
		return submod.Data(), nil
	}
	return mod.Data(), nil

}

func (d *Disp) SchemaGet(modOrSubmod string, format string) (string, error) {
	schema, err := d.getModuleOrSubmoduleSchema(modOrSubmod)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(schema))
	return buf.String(), nil
}

const (
	excludeSubmodules = false
	includeSubmodules = true
)

func (d *Disp) GetSchemas() (string, error) {
	return d.getSchemasInternal(includeSubmodules)
}

func (d *Disp) GetModuleSchemas() (string, error) {
	return d.getSchemasInternal(excludeSubmodules)
}

// getSchemasInternal - return XML-encoded list of (sub)modules supported
//
// This function returns an XML-encoded list of the modules and (optionally)
// submodules supported on the device.
//
// RFC 6020 section 4.2.1 opines that the external view should only consist
// of a single module even when submodule(s) are present.  This is therefore
// what should be presented in NETCONF module capabilities.
//
// Separately, RFC 6022 section 4.1 describes the retrieval of a list of all
// supported schemas using the ietf-netconf-monitoring netconf-state <schemas>
// element.  This explicitly includes modules and submodules.
//
func (d *Disp) getSchemasInternal(incSubmods bool) (string, error) {
	var b bytes.Buffer
	enc := xml.NewEncoder(&b)
	enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "schemas"}})

	mods := d.ms.Modules()
	for _, m := range mods {
		m.EncodeXML(enc)
	}

	if incSubmods {
		submods := d.ms.Submodules()
		for _, sm := range submods {
			sm.EncodeXML(enc)
		}
	}

	enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "schemas"}})
	enc.Flush()
	return b.String(), nil
}

func (d *Disp) GetDeviations() (map[string]string, error) {
	mods := d.ms.Modules()
	v := make(map[string]string, len(mods))
	for _, m := range mods {
		// C client cannot handle map[string][]string
		v[m.Identifier()] = strings.Join(m.Deviations(), ",")
	}
	return v, nil
}

func (d *Disp) GetFeatures() (map[string]string, error) {
	mods := d.ms.Modules()
	f := make(map[string]string, len(mods))
	for _, m := range mods {
		// C client cannot handle map[string][]string
		f[m.Identifier()] = strings.Join(m.Features(), ",")
	}
	return f, nil
}

func (d *Disp) GetHelp(sid string, schema bool, path string) (map[string]string, error) {
	ps := pathutil.Makepath(path)
	sess := d.getROSession(rpc.CANDIDATE, sid)
	return sess.GetHelp(d.ctx, schema, ps)
}

func (d *Disp) GetCompletions(sid string, schema bool, path string) (map[string]string, error) {
	ps := pathutil.Makepath(path)

	typ, err := d.NodeGetType(sid, path)
	if err != nil {
		return map[string]string{}, err
	}

	// To reduce CLI delay and AAA server load we only do remote authorization
	// when schema is false (ie. delete, show operation) or if completing on a
	// value (ie. not a container).
	if !schema || typ != rpc.CONTAINER {
		if !d.authRead(ps) {
			return map[string]string{}, mgmterror.NewAccessDeniedApplicationError()
		}
	}

	comps, err := d.GetHelp(sid, schema, path)
	if err != nil {
		return map[string]string{}, err
	}

	// Allowed values are only needed when schema is true (ie. set operation) and
	// we are completing on a value node (ie. not a container).
	needsAllowed := typ != rpc.CONTAINER
	if !schema || !needsAllowed {
		return comps, nil
	}

	allowed, err := d.TmplGetAllowed(sid, path)
	for _, v := range allowed {
		if strings.ContainsAny(v, "<>") {
			continue
		}
		comps[v] = ""
	}
	return comps, err
}

func configRevisionFileName(revision string) string {
	if revision == "saved" {
		return "/config/config.boot"
	}
	return "/config/archive/config.boot." + revision + ".gz"
}

func (d *Disp) cfgFileReader(file *os.File) (io.Reader, error) {
	if strings.HasSuffix(file.Name(), ".gz") {
		r, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	return file, nil
}

func (d *Disp) readCfgFile(file string, raw, forceShowSecrets bool) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	r, err := d.cfgFileReader(f)
	if err != nil {
		return "", err
	}
	if raw {
		text, err := ioutil.ReadAll(r)
		if err != nil && err != io.EOF {
			return "", err
		}
		return string(text), nil
	}
	dtree, err, _ := load.LoadFile(file, r, d.ms)
	if err != nil {
		return "", err
	}
	can := data.New("root")
	ut := union.NewNode(can, dtree, d.ms, nil, 0)

	sess := d.getROSession(rpc.RUNNING, "RUNNING")

	// ut.Show() will return 'access denied' error if we don't have read
	// access.  However, we used to use ut.String() here which just returned
	// an empty string and no error in this case, so to preserve identical
	// behaviour we do this here and ignore returned error.
	var options []union.UnionOption
	options = append(options, union.Authorizer(sess.NewAuther(d.ctx)))
	if forceShowSecrets {
		options = append(options, union.ForceShowSecrets)
	}
	out, _ := ut.Show(nil, options...)
	return out, nil
}

func (d *Disp) ReadConfigFileRaw(file string) (string, error) {
	return d.readCfgFile(file, true, false)
}

func (d *Disp) ReadConfigFile(file string) (string, error) {
	return d.readCfgFile(file, false, false)
}

func (d *Disp) readConfigFileForceShowSecrets(file string) (string, error) {
	return d.readCfgFile(file, false, true)
}

func (d *Disp) MigrateConfigFile(file string) (string, error) {
	// This is now obsolete and is due to be fully removed. For now, just do
	// nothing.
	return "", nil
}

func decodeTree(encoding string, sch schema.Node, input string) (datanode.DataNode, error) {
	switch encoding {
	case "json":
		return yangenc.UnmarshalJSON(sch, []byte(input))
	case "rfc7951":
		return yangenc.UnmarshalRFC7951(sch, []byte(input))
	case "xml", "netconf":
		return yangenc.UnmarshalXML(sch, []byte(input))
	default:
		cerr := mgmterror.NewOperationFailedApplicationError()
		cerr.Message = fmt.Sprintf("Unknown RPC encoding '%s'", encoding)
		return nil, cerr
	}
}

func encodeTree(encoding string, sch schema.Node, input datanode.DataNode) (string, error) {
	switch encoding {
	case "json":
		return string(yangenc.ToJSON(sch, input)), nil
	case "rfc7951":
		return string(yangenc.ToRFC7951(sch, input)), nil
	case "xml", "netconf":
		return string(yangenc.ToXML(sch, input)), nil
	default:
		cerr := mgmterror.NewOperationFailedApplicationError()
		cerr.Message = fmt.Sprintf("Unknown RPC encoding '%s'", encoding)
		return "", cerr
	}
}

func getModuleId(st schema.ModelSet, moduleIdOrNamespace, encoding string,
) (string, bool) {

	switch encoding {
	case "xml", "netconf":
		// With XML the module is identified by the namespace
		for name, module := range st.Modules() {
			if moduleIdOrNamespace == module.Namespace() {
				return name, true
			}
		}
		return "", false
	case "rfc7951":
		fallthrough
	case "json":
		// With JSON the module is identified by its module name
		if _, ok := st.Modules()[moduleIdOrNamespace]; !ok {
			return "", false
		}
		return moduleIdOrNamespace, true
	default:
		return "", false
	}
}

func (d *Disp) findRpc(
	moduleIdOrNamespace, name, encoding string,
) (schema.Rpc, string, bool) {

	allrpcs := d.ms.Rpcs()

	mod_id, ok := getModuleId(d.ms, moduleIdOrNamespace, encoding)
	if !ok {
		return nil, "", false
	}

	mod_ns := d.ms.Modules()[mod_id].Namespace()

	mod_rpcs, ok := allrpcs[mod_ns]
	if !ok {
		return nil, "", false
	}

	rpc, ok := mod_rpcs[name]
	if !ok || rpc.Input() == nil {
		return nil, "", false
	}

	return rpc.(schema.Rpc), mod_ns, true
}

func convertEncoding(rpc schema.Rpc, inputTree, fromEncoding, toEncoding string) (string, error) {
	if fromEncoding == toEncoding {
		return inputTree, nil
	}

	decodedTree, err := decodeTree(fromEncoding, rpc.Input().(schema.Node), inputTree)
	if err != nil {
		return "", err
	}
	encodedTree, err := encodeTree(toEncoding, rpc.Input().(schema.Node),
		decodedTree)
	if err != nil {
		return "", err
	}
	return encodedTree, nil
}

// Allows us to test without needing VCI DBUS infrastructure.
type VciRpcCaller interface {
	CallRpc(ctx *configd.Context, moduleName, rpcName, inputTreeJson string) (string, error)
}

type vciRpcCaller struct{}

func (vrc *vciRpcCaller) CallRpc(
	ctx *configd.Context,
	moduleName, rpcName, inputTreeJson string,
) (string, error) {
	metadata := vci.RPCMetadata{
		Pid:    ctx.Pid,
		Uid:    ctx.Uid,
		User:   ctx.User,
		Groups: ctx.Groups,
	}
	client, err := vci.Dial()
	if err != nil {
		return "", err
	}
	var out string
	err = client.CallWithMetadata(moduleName, rpcName, metadata, inputTreeJson).
		StoreOutputInto(&out)
	client.Close()
	return out, err
}

func (d *Disp) handleVciRpc(
	ctx *configd.Context,
	moduleName string,
	encoding string,
	rpc schema.Rpc,
	rpcName string,
	args string,
	vrc VciRpcCaller,
) (string, error) {

	var inputTreeJson = args
	var err error
	inputTreeJson, err = convertEncoding(rpc, args, encoding, "rfc7951")
	if err != nil {
		return "", err
	}

	output, err := vrc.CallRpc(ctx, moduleName, rpcName, inputTreeJson)
	if err != nil {
		return "", err
	}

	return convertJsonOutputToRpcReply(rpc, output, encoding)
}

func convertJsonOutputToRpcReply(rpc schema.Rpc, output, encoding string,
) (string, error) {

	if output == "" {
		output = "{}"
	}
	outputTree, err := yangenc.UnmarshalRFC7951(rpc.Output(), []byte(output))
	if err != nil {
		jerr := mgmterror.NewOperationFailedApplicationError()
		jerr.Message = fmt.Sprintf("Failed to process returned data: %s",
			err.Error())
		return "", jerr
	}
	repCh := outputTree.YangDataChildren()
	repVa := outputTree.YangDataValues()
	reply := datanode.CreateDataNode("rpc-reply", repCh, repVa)

	return encodeTree(encoding, rpc.Output().(schema.Node), reply)
}

func (d *Disp) CallRpc(moduleIdOrNamespace, rpcName, args, encoding string,
) (string, error) {
	return d.callRpcInternal(moduleIdOrNamespace, rpcName, args, encoding,
		&vciRpcCaller{})
}

func (d *Disp) callRpcInternal(
	moduleIdOrNamespace, rpcName, args, encoding string,
	vrc VciRpcCaller,
) (string, error) {

	rpc, moduleNs, ok := d.findRpc(moduleIdOrNamespace, rpcName, encoding)
	if !ok {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = fmt.Sprintf(
			"Unknown RPC (%s) %s:%s", encoding, moduleIdOrNamespace, rpcName)
		return "", err
	}

	_, found := d.ms.GetModelNameForNamespace(moduleNs)
	if found {
		moduleId, _ := getModuleId(d.ms, moduleIdOrNamespace, encoding)
		if !d.ctx.Auth.AuthorizeRPC(d.ctx.Uid, d.ctx.Groups, moduleId, rpcName) {
			return "", mgmterror.NewAccessDeniedApplicationError()
		}
		output, err := d.handleVciRpc(d.ctx,
			moduleId, encoding, rpc, rpcName, args, vrc)
		return output, common.FormatRpcPathError(err)
	}

	err := mgmterror.NewOperationFailedApplicationError()
	err.Message = fmt.Sprintf("Unknown model for RPC %s:%s",
		moduleIdOrNamespace, rpcName)
	return "", err
}

// TODO: eventually remove this.
func (d *Disp) CallRpcXml(moduleNamespace, name, args string) (string, error) {
	return d.CallRpc(moduleNamespace, name, args, "xml")
}

// Allow for the scenario where a user types 'tab' when the cursor is mid-word,
// eg 'set interfaces datadp0' with the cursor on the 'd' of 'dp0'.  We want
// to return 'set interfaces dataplanedp0' in this case rather than an error.
//
// <prefix> shows text up to cursor in the word in <path> indicated by the
// index <pos>.  <path> takes format '[/]0/1/2/...' ie indexed from zero, and
// with optional leading '/'.
//
// If <pos> < 0 then <prefix> will never be used, maintaining the original
// behaviour of the Expand() API.
//
func (d *Disp) ExpandWithPrefix(path, prefix string, pos int) (string, error) {
	// Need prefix, and 'argpos'
	ps, err := d.expandPath(pathutil.Makepath(path), prefix, pos+1)
	if err != nil {
		return "", common.FormatConfigPathError(err)
	}
	return pathutil.Pathstr(ps), nil
}

const (
	NoPrefix   = "TEST_NOT_USING_PREFIX"
	InvalidPos = -1
)

func (d *Disp) Expand(path string) (string, error) {
	return d.ExpandWithPrefix(path, NoPrefix, InvalidPos)
}

type processNodeFn func(
	sch schema.Node,
	path, cpath []string,
	prefix string,
	pos int,
) ([]string, error)

func (d *Disp) expandPath(path []string, prefix string, pos int,
) ([]string, error) {
	cpath := make([]string, 0, len(path))
	origPath := path

	var ( //predeclare recursive functions
		processnode         processNodeFn
		processleaf         processNodeFn
		processchildren     processNodeFn
		processchildrenskip func(
			sch schema.Node, path, cpath, skiplist []string,
			prefix string, pos int,
		) ([]string, error)
		processlist func(
			sch schema.List, path, cpath []string, prefix string, pos int,
		) ([]string, error)
	)

	processleaf = func(
		sch schema.Node, path, cpath []string, prefix string, pos int,
	) ([]string, error) {
		if len(path) < 1 {
			return cpath, nil
		}
		if _, ok := sch.Type().(schema.Empty); ok && len(path) > 0 {
			err := mgmterror.NewUnknownElementApplicationError(path[0])
			err.Path = pathutil.Pathstr(cpath)
			return nil, err
		}
		val, path := path[0], path[1:]
		if len(path) > 0 {
			//The path is longer than is present in the schema tree
			err := mgmterror.NewUnknownElementApplicationError(path[0])
			err.Path = pathutil.Pathstr(append(cpath, val))
			return nil, err
		}
		return append(cpath, val), nil
	}

	processlist = func(
		sch schema.List, path, cpath []string, prefix string, pos int,
	) ([]string, error) {
		if len(path) < 1 {
			return cpath, nil
		}
		key, path := path[0], path[1:]
		return processchildrenskip(sch, path, append(cpath, key),
			sch.Keys(), prefix, pos)
	}

	processnode = func(
		sch schema.Node, path, cpath []string, prefix string, pos int,
	) ([]string, error) {
		switch v := sch.(type) {
		case schema.Tree:
			return processchildren(sch, path, cpath, prefix, pos)
		case schema.Container:
			return processchildren(sch, path, cpath, prefix, pos)
		case schema.List:
			return processlist(v, path, cpath, prefix, pos)
		case schema.Leaf:
			return processleaf(sch, path, cpath, prefix, pos)
		case schema.LeafList:
			return processleaf(sch, path, cpath, prefix, pos)
		}
		err := mgmterror.NewUnknownElementApplicationError(cpath[len(cpath)-1])
		err.Path = pathutil.Pathstr(cpath[:len(cpath)-1])
		return nil, err
	}

	processchildren = func(
		sch schema.Node, path, cpath []string, prefix string, pos int,
	) ([]string, error) {
		return processchildrenskip(sch, path, cpath, nil, prefix, pos-1)
	}

	processchildrenskip = func(
		sch schema.Node, path, cpath, skiplist []string,
		prefix string, pos int,
	) ([]string, error) {
		var matches []schema.Node
		if len(path) == 0 {
			return cpath, nil
		}
		val, path := path[0], path[1:]

		prefixMatch := false
		for _, c := range sch.Children() {
			name := c.Name()
			if isElemOf(skiplist, name) {
				continue
			}
			if name == val {
				//exact matches are never ambiguous make a single match slice
				matches = []schema.Node{c.(schema.Node)}
				break
			} else if strings.HasPrefix(name, val) {
				matches = append(matches, c.(schema.Node))
			} else if pos == 0 {
				if strings.HasPrefix(name, prefix) {
					prefixMatch = true
					matches = append(matches, c.(schema.Node))
				}
			}
		}

		switch len(matches) {
		case 0:
			err := mgmterror.NewUnknownElementApplicationError(val)
			err.Path = pathutil.Pathstr(cpath)
			return nil, err
		case 1:
			nameToAppend := matches[0].Name()
			if prefixMatch {
				if len(prefix) >= len(val) {
					err := mgmterror.NewInvalidValueApplicationError()
					err.Message = fmt.Sprintf(
						"%v has invalid prefix '%s'",
						mgmterror.ErrPath(origPath), prefix)
					return nil, err
				}
				nameToAppend += val[len(prefix):]
			}
			return processnode(
				matches[0], path, append(cpath, nameToAppend), prefix, pos)
		default:
			matchnames := make(map[string]string)
			for _, v := range matches {
				matchnames[v.Name()] = v.ConfigdExt().Help
			}
			return nil, mgmterror.NewPathAmbiguousError(
				append(cpath, val), matchnames)
		}
	}

	return processnode(d.ms, path, cpath, prefix, pos)
}

func (d *Disp) EditConfigXML(sid, config_target, default_operation, test_option, error_option, config string) (string, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return "", err
	}

	return "", sess.EditConfigXML(d.ctx, config_target, default_operation, test_option, error_option, config)
}

func (d *Disp) copyConfigInternal(
	sid,
	sourceDatastore,
	sourceEncoding,
	sourceConfig,
	sourceURL,
	targetDatastore,
	targetURL string,
) (string, error) {
	sess, err := d.smgr.Get(d.ctx, sid)
	if err != nil {
		return "", err
	}

	return "", sess.CopyConfig(d.ctx, sourceDatastore, sourceEncoding,
		sourceConfig, sourceURL, targetDatastore, targetURL)
}

func (d *Disp) CopyConfig(
	sid,
	sourceDatastore,
	sourceEncoding,
	sourceConfig,
	sourceURL,
	targetDatastore,
	targetURL string,
) (string, error) {
	redactedSource := "copy-config"
	noRoutingInstance := ""
	args := d.cfgMgmtCommandArgs(
		"load", redactedSource, noRoutingInstance, sourceEncoding)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	if !d.ctx.Configd {
		d.ctx.Wlog.Println("copy-config by " + d.ctx.User)
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.copyConfigInternal(
			sid, sourceDatastore, sourceEncoding, sourceConfig,
			sourceURL, targetDatastore, targetURL)
	})

}
func (d *Disp) SetConfigDebug(sid, logName, level string) (string, error) {
	return common.SetConfigDebug(logName, level)
}
