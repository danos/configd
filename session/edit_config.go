// Copyright (c) 2018-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"bytes"
	"encoding/xml"
	"runtime"

	"github.com/danos/config/auth"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/configd/rpc"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/pathutil"
	yang "github.com/danos/yang/schema"
)

const (
	target_notset = iota
	target_candidate
	target_running
)

type config_target uint32

func (t *config_target) Set(target string) error {
	values := map[string]config_target{
		"candidate": target_candidate,
		"running":   target_running,
	}
	if v, ok := values[target]; ok {
		*t = v
		return nil
	}
	return mgmterror.NewUnknownElementProtocolError(target)
}

const (
	defop_notset = iota
	defop_merge
	defop_replace
	defop_none
)

type default_operation uint32

func (o *default_operation) Set(op string) error {
	values := map[string]default_operation{
		"merge":   defop_merge,
		"replace": defop_replace,
		"none":    defop_none,
	}
	if v, ok := values[op]; ok {
		*o = v
		return nil
	}
	err := mgmterror.NewInvalidValueProtocolError()
	err.Message = "Invalid default-operation"
	return err
}

// Effectively this maps default_operation to operation
func (o default_operation) Get() operation {
	values := map[default_operation]operation{
		defop_merge:   op_merge,
		defop_replace: op_replace,
		defop_none:    op_notset,
	}
	return values[o]
}

const (
	testopt_notset = iota
	testopt_testset
	testopt_set
	testopt_testonly
)

type test_option uint32

func (o *test_option) Set(opt string) error {
	values := map[string]test_option{
		"test-then-set": testopt_testset,
		"set":           testopt_set,
		"test-only":     testopt_testonly,
	}
	if v, ok := values[opt]; ok {
		*o = v
		return nil
	}
	err := mgmterror.NewInvalidValueProtocolError()
	err.Message = "Invalid test-option"
	return err
}

const (
	erropt_notset = iota
	erropt_stop
	erropt_cont
	erropt_rollback
)

type error_option uint32

func (o *error_option) Set(opt string) error {
	values := map[string]error_option{
		"stop-on-error":     erropt_stop,
		"continue-on-error": erropt_cont,
		"rollback-on-error": erropt_rollback,
	}
	if v, ok := values[opt]; ok {
		*o = v
		return nil
	}
	err := mgmterror.NewInvalidValueProtocolError()
	err.Message = "Invalid error-option"
	return err
}

const (
	op_notset = iota
	op_merge
	op_replace
	op_create
	op_delete
	op_remove
)

type operation uint32

func (o *operation) set(op string) error {
	values := map[string]operation{
		"merge":   op_merge,
		"replace": op_replace,
		"create":  op_create,
		"delete":  op_delete,
		"remove":  op_remove,
	}
	if v, ok := values[op]; ok {
		*o = v
		return nil
	}
	return mgmterror.NewUnknownAttrProtocolError(op, "operation")
}

func (o *operation) UnmarshalXMLAttr(attr xml.Attr) error {
	const netconfNS = "urn:ietf:params:xml:ns:netconf:base:1.0"
	if attr.Name.Space != netconfNS {
		return mgmterror.NewUnknownNamespaceProtocolError("operation", attr.Name.Space)
	}
	return o.set(attr.Value)
}

type edit_node struct {
	XMLName   xml.Name
	Operation operation   `xml:"operation,attr"`
	Value     string      `xml:",chardata"`
	Children  []edit_node `xml:",any"`
	Path      string
	Type      rpc.NodeType
}

func (en edit_node) getOperation(parentop operation) operation {
	if en.Operation == op_notset {
		return parentop
	}
	return en.Operation
}

func (en *edit_node) setOperation(parentop operation) {
	en.Operation = en.getOperation(parentop)
}

func (en *edit_node) setPath(curPath string) error {
	en.Path = curPath + "/" + en.XMLName.Local
	return nil
}

type edit_op struct {
	op   operation
	path []string
}

func (ec edit_config) authorize(
	perm auth.AuthPerm,
	path []string,
) bool {

	// Generate a corresponding command to perform command authorization
	var perm_cmd string
	switch perm {
	case auth.P_CREATE, auth.P_UPDATE:
		perm_cmd = "set"
	case auth.P_DELETE:
		perm_cmd = "delete"
	default:
		return false
	}

	attrs := schema.AttrsForPath(ec.sess.schemaFull, path)

	// Prepend command keyword and corresponding attributes
	cmd := append([]string{perm_cmd}, path...)
	perm_cmd_attrs := []pathutil.PathElementAttrs{
		pathutil.PathElementAttrs{Secret: false},
	}
	attrs.Attrs = append(perm_cmd_attrs, attrs.Attrs...)

	// Do command authorization and accounting
	if !ec.ctx.Auth.AuthorizeCommand(ec.ctx.Uid, ec.ctx.Groups, cmd, attrs) {
		return false
	}
	ec.ctx.Auth.AccountCommand(ec.ctx.Uid, ec.ctx.Groups, cmd, attrs)

	// Drop path attrs for "command" keyword and do path authorization
	attrs.Attrs = attrs.Attrs[1:]
	return ec.ctx.Auth.AuthorizePath(ec.ctx.Uid, ec.ctx.Groups, path, attrs, perm)
}

func (e edit_op) Auth(ec edit_config) bool {
	ctx := ec.ctx
	exist := ec.sess.existsInTree(ec.sess.getUnion(), ctx, e.path, excludeDefault)
	switch e.op {
	case op_merge:
		if exist {
			return ec.authorize(auth.P_UPDATE, e.path)
		}
		return ec.authorize(auth.P_CREATE, e.path)
	case op_replace:
		if exist {
			if !ec.authorize(auth.P_DELETE, e.path) {
				return false
			}
		}
		return ec.authorize(auth.P_CREATE, e.path)
	case op_create:
		return ec.authorize(auth.P_CREATE, e.path)
	case op_delete, op_remove:
		return ec.authorize(auth.P_DELETE, e.path)
	case op_notset:
		return true
	}
	return false
}

func (e edit_op) Test(ec edit_config) error {
	if !e.Auth(ec) {
		return mgmterror.NewAccessDeniedApplicationError()
	}
	switch e.op {
	case op_create:
		if ec.sess.existsInTree(ec.sess.getUnion(), ec.ctx, e.path, excludeDefault) {
			return yang.NewNodeExistsError(e.path)
		}
	case op_delete:
		if !ec.sess.existsInTree(ec.sess.getUnion(), ec.ctx, e.path, excludeDefault) {
			return yang.NewNodeNotExistsError(e.path)
		}
		fallthrough
	case op_remove:
		if tmpl := ec.sess.schema.PathDescendant(e.path); tmpl == nil {
			return yang.NewInvalidPathError(e.path)
		}
	}
	var sn schema.Node = ec.sess.schema
	var err error
	if e.path, err = schema.NormalizePath(sn, e.path); err != nil {
		return err
	}
	for i, p := range e.path {
		if sn = sn.SchemaChild(p); sn == nil {
			cerr := mgmterror.NewUnknownElementApplicationError(p)
			cerr.Path = pathutil.Pathstr(e.path[:i])
			return cerr
		}
	}
	return nil
}

func (e edit_op) Merge(ec edit_config) error {
	if ec.sess.existsInTree(ec.sess.getUnion(), ec.ctx, e.path, excludeDefault) {
		return nil
	}
	return ec.sess._set(ec.ctx, e.path)
}

func (e edit_op) Replace(ec edit_config) error {
	if err := e.Remove(ec); err != nil {
		return err
	}
	return e.Merge(ec)
}

func (e edit_op) Create(ec edit_config) error {
	if ec.sess.existsInTree(ec.sess.getUnion(), ec.ctx, e.path, excludeDefault) {
		return yang.NewNodeExistsError(e.path)
	}
	return ec.sess._set(ec.ctx, e.path)
}

func (e edit_op) Delete(ec edit_config) error {
	if !ec.sess.existsInTree(ec.sess.getUnion(), ec.ctx, e.path, excludeDefault) {
		return yang.NewNodeNotExistsError(e.path)
	}
	return e.Remove(ec)
}

func (e edit_op) Remove(ec edit_config) error {
	t := ec.sess.getUnion()
	// Remove succeeds even when delete fails
	t.Delete(ec.sess.newAuther(ec.ctx), e.path, union.DontCheckAuth)
	return nil
}

func (e edit_op) Set(ec edit_config) error {
	switch e.op {
	case op_merge:
		return e.Merge(ec)
	case op_replace:
		return e.Replace(ec)
	case op_create:
		return e.Create(ec)
	case op_delete:
		return e.Delete(ec)
	case op_remove:
		return e.Remove(ec)
	case op_notset:
		// e.op should only be op_notset if it has been inherited
		// from the default operation. Otherwise e.op will have been
		// set to one of the other op_* values either by its own operation
		// attribute or that of a parent, or by inheriting defop_merge
		// or defop_replace.
		if ec.DefaultOperation == defop_none {
			return nil
		}
		fallthrough
	default:
		return mgmterror.NewOperationFailedApplicationError()
	}
}

func (e edit_op) Perform(ec edit_config) error {
	switch ec.TestOption {
	case testopt_notset, testopt_testset:
		if err := e.Test(ec); err != nil {
			return err
		}
		return e.Set(ec)
	case testopt_set:
		if !e.Auth(ec) {
			return mgmterror.NewAccessDeniedApplicationError()
		}
		return e.Set(ec)
	case testopt_testonly:
		return e.Test(ec)
	}
	return nil
}

type edit_config struct {
	XMLName          xml.Name     `xml:"config,"`
	Children         []*edit_node `xml:",any"`
	Target           config_target
	DefaultOperation default_operation
	TestOption       test_option
	ErrorOption      error_option
	sess             *session
	ctx              *configd.Context
	ops              []edit_op
}

func newEditConfigXML(s *session, ctx *configd.Context, config_target, def_operation, test_option, error_option string, config []byte) (*edit_config, error) {
	ec := edit_config{sess: s, ctx: ctx}
	if err := ec.Target.Set(config_target); err != nil {
		return nil, err
	}
	if err := ec.DefaultOperation.Set(def_operation); err != nil {
		return nil, err
	}
	if err := ec.TestOption.Set(test_option); err != nil {
		return nil, err
	}
	if err := ec.ErrorOption.Set(error_option); err != nil {
		return nil, err
	}
	if err := xml.Unmarshal(config, &ec); err != nil {
		return nil, err
	}
	return &ec, nil
}

func (ec *edit_config) Add(op operation, path []string) {
	// Make our own copy of the path
	p := make([]string, len(path))
	copy(p, path)
	ec.ops = append(ec.ops, edit_op{op: op, path: p})
}

func (en edit_node) traversePostOrder(ec *edit_config, parentop operation, curpath []string) {
	op := en.getOperation(parentop)
	for _, c := range en.Children {
		c.traverse(ec, op, curpath)
	}
	ec.Add(op, curpath)
}

func (en edit_node) traversePreOrder(ec *edit_config, parentop operation, curpath []string) {
	op := en.getOperation(parentop)
	ec.Add(op, curpath)
	for _, c := range en.Children {
		c.traverse(ec, op, curpath)
	}
}

func (en edit_node) traverseSubtree(ec *edit_config, parentop operation, curpath []string) {
	if (parentop == op_delete) || (parentop == op_remove) {
		en.traversePostOrder(ec, parentop, curpath)
		return
	}
	en.traversePreOrder(ec, parentop, curpath)
}

func (en edit_node) traverseContainer(ec *edit_config, parentop operation, curpath []string) {
	op := en.getOperation(parentop)
	sch := schema.Descendant(ec.sess.schema, curpath)
	if sch == nil {
		cerr := mgmterror.NewUnknownElementApplicationError(curpath[len(curpath)-1])
		cerr.Path = pathutil.Pathstr(curpath[:len(curpath)-1])
		panic(cerr)
	}
	if sch.Namespace() != en.XMLName.Space {
		panic(mgmterror.NewUnknownNamespaceApplicationError(pathutil.Pathstr(curpath), en.XMLName.Space))
	}
	if !sch.HasPresence() && (len(en.Children) > 0) &&
		(op != op_delete) && (op != op_remove) {
		for _, c := range en.Children {
			c.traverse(ec, parentop, curpath)
		}
		return
	}
	en.traverseSubtree(ec, parentop, curpath)
}

func (en edit_node) traverseList(ec *edit_config, parentop operation, curpath []string) {
	sch := schema.Descendant(ec.sess.schema, curpath)
	if sch == nil {
		cerr := mgmterror.NewUnknownElementApplicationError(curpath[len(curpath)-1])
		cerr.Path = pathutil.Pathstr(curpath[:len(curpath)-1])
		panic(cerr)
	}
	if sch.Namespace() != en.XMLName.Space {
		panic(mgmterror.NewUnknownNamespaceApplicationError(pathutil.Pathstr(curpath), en.XMLName.Space))
	}
	n, ok := sch.(schema.List)
	if !ok {
		return // This should not happen; bail.
	}

	// Find list key
	var path []string
	for i, c := range en.Children {
		if c.XMLName.Local == n.Keys()[0] {
			if i != 0 {
				// Key must be first child, if not bail
				return
			}
			path = append(curpath, c.Value)
			sch := schema.Descendant(ec.sess.schema, path)
			if sch == nil {
				cerr := mgmterror.NewUnknownElementApplicationError(curpath[len(curpath)-1])
				cerr.Path = pathutil.Pathstr(curpath[:len(curpath)-1])
				panic(cerr)
			}
			if sch.Namespace() != en.XMLName.Space {
				panic(mgmterror.NewUnknownNamespaceApplicationError(pathutil.Pathstr(curpath), en.XMLName.Space))
			}
			// Remove key so it does not get processed as a leaf later
			en.Children = append(en.Children[:i], en.Children[i+1:]...)
			break
		}
	}

	if len(path) == 0 {
		// Check if we are deleting the tag
		op := en.getOperation(parentop)
		if (op == op_delete) || (op == op_remove) {
			en.traverseSubtree(ec, parentop, curpath)
		}
		return
	}
	en.traverseSubtree(ec, parentop, path)
}

func (en edit_node) traverseLeaf(ec *edit_config, parentop operation, curpath []string) {
	sch := schema.Descendant(ec.sess.schema, curpath)
	if sch == nil {
		cerr := mgmterror.NewUnknownElementApplicationError(curpath[len(curpath)-1])
		cerr.Path = pathutil.Pathstr(curpath[:len(curpath)-1])
		panic(cerr)
	}
	if sch.Namespace() != en.XMLName.Space {
		panic(mgmterror.NewUnknownNamespaceApplicationError(pathutil.Pathstr(curpath), en.XMLName.Space))
	}
	op := en.getOperation(parentop)
	_, isEmpty := sch.Type().(schema.Empty)
	if !isEmpty && en.Value != "" {
		path := append(curpath, en.Value)
		ec.Add(op, path)
		return
	}
	ec.Add(op, curpath)
}

func (en edit_node) traverse(ec *edit_config, parentop operation, curpath []string) error {
	path := append(curpath, en.XMLName.Local)
	op := en.getOperation(parentop)

	sch := schema.Descendant(ec.sess.schema, path)
	if sch == nil {
		return nil // invalid path; bail
	}
	switch sch.(type) {
	case schema.List:
		en.traverseList(ec, op, path)
	case schema.Leaf, schema.LeafList, schema.LeafValue:
		en.traverseLeaf(ec, op, path)
	default:
		en.traverseContainer(ec, op, path)
	}
	return nil
}

func (ec edit_config) test() error {

	for _, o := range ec.ops {
		if err := o.Test(ec); err != nil {
			return err
		}
	}
	return nil
}

type perform_error []error

func (errs perform_error) Error() string {
	var out bytes.Buffer
	for _, e := range errs {
		out.WriteString(e.Error() + "\n")
	}
	return out.String()
}

func (ec edit_config) perform() error {
	var perr perform_error
	for _, o := range ec.ops {
		if err := o.Perform(ec); err != nil {
			switch ec.ErrorOption {
			case erropt_stop, erropt_notset:
				perr = append(perr, err)
				return err
			case erropt_cont:
				perr = append(perr, err)
				continue
			}
		}
	}
	if len(perr) == 0 {
		return nil
	}
	return perr
}

func (ec edit_config) EditConfig() (reterr error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			reterr = r.(error)
		}
	}()

	for _, en := range ec.Children {
		err := en.traverse(&ec, ec.DefaultOperation.Get(), []string{})
		if err != nil {
			return err
		}
	}

	if ec.ErrorOption == erropt_rollback {
		// We can't rollback, but we can prevalidate which is
		// the same result
		if err := ec.test(); err != nil {
			return err
		}
	}

	return ec.perform()
}

func (s *session) editConfigXML(
	ctx *configd.Context,
	config_target, default_operation, test_option, error_option, config string) (reterr error) {
	if err := s.trylock(ctx.Pid); err != nil {
		return err
	}

	ec, err := newEditConfigXML(s, ctx, config_target, default_operation, test_option, error_option, []byte(config))
	if err != nil {
		return err
	}
	return ec.EditConfig()
}
