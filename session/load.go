// Copyright (c) 2018-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2015,2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"bytes"
	"fmt"
	"io"

	"github.com/danos/config/data"
	"github.com/danos/config/load"
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/pathutil"
	"github.com/danos/yang/data/encoding"
)

type lderrs []error

func (l lderrs) Error() string {
	var b bytes.Buffer
	for _, err := range l {
		b.WriteString(err.Error())
		b.WriteByte('\n')
	}
	return b.String()
}

func (s *session) readFile(file string, r io.Reader) (union.Node, error, []error) {
	var err error
	var can *data.Node
	var invalidPaths []error

	if r == nil {
		can, err, invalidPaths = load.Load(file, s.schema)
	} else {
		can, err, invalidPaths = load.LoadFile(file, r, s.schema)
	}
	if err != nil {
		return nil, err, invalidPaths
	}
	return union.NewNode(nil, can, s.schema, nil, 0), nil, invalidPaths
}

func (s *session) merge(ctx *configd.Context, file string, r io.Reader) (error, []error) {
	ltree, err, invalidPaths := s.readFile(file, r)
	if err != nil {
		return err, invalidPaths
	}

	return s.merge_tree(ctx, ltree), invalidPaths
}

func (s *session) load(ctx *configd.Context, file string, r io.Reader) (error, []error) {
	ltree, err, invalidPaths := s.readFile(file, r)
	if err != nil {
		return err, invalidPaths
	}

	return s.delete_then_merge_tree(ctx, ltree), invalidPaths
}

func (s *session) loadFromStringUsingEncoding(
	input string,
	encType encoding.EncType,
) (union.Node, error) {
	um := union.NewUnmarshaller(encType)

	return um.Unmarshal(s.schema, []byte(input))
}

func (s *session) copyConfig(
	ctx *configd.Context,
	sourceDatastore,
	sourceConfig,
	sourceURL,
	targetDatastore,
	targetURL string,
) error {

	// Don't support URL capability.
	if sourceURL != "" || targetURL != "" {
		// TODO - details
		err := mgmterror.NewOperationNotSupportedApplicationError()
		err.Message = "URL capability is not supported"
		return err
	}

	// As we only support candidate datastore, targetDatastore must be set to
	// this, sourceDatastore must not be set, and sourceConfig must be set.
	if sourceDatastore != "" {
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = "Source must be specified in <config> tags."
		return err
	}
	if sourceConfig == "" {
		// error-info <bad-element>
		return mgmterror.NewMissingElementApplicationError("<source>")
	}
	if targetDatastore != "candidate" {
		// TODO details!
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = fmt.Sprintf(
			"Target datastore only supports candidate, not %s",
			targetDatastore)
		return err
	}

	ltree, err := s.loadFromStringUsingEncoding(sourceConfig, encoding.XML)
	if err != nil {
		return err
	}

	return s.delete_then_merge_tree(ctx, ltree)
}

func (s *session) delete_then_merge_tree(
	ctx *configd.Context,
	ltree union.Node,
) error {
	stree := s.getUnion()

	stree.Delete(s.newAuther(ctx), []string{} /* unused */, union.CheckAuth)

	return s.merge_tree(ctx, ltree)
}

func (s *session) merge_tree(ctx *configd.Context, ltree union.Node) error {
	var errors []error
	ut := s.getUnion()
	setFn := func(n union.Node, path []string) {
		if !n.GetSchema().HasPresence() {
			//if a node doesn't have presence
			//setting it will result in an error
			//skip it.
			return
		}
		if s.existsInTree(ut, ctx, path, false) {
			//already in tree, skip
			return
		}
		err := s.set(ctx, path)
		if err != nil {
			errors = append(errors, err)
		}
	}
	//create the nodes that are in the loaded configuration tree
	//use a preorder walk to find the 'leaves' i.e. any node that has
	//presence.
	var preord func(n union.Node, curPath []string)
	preord = func(n union.Node, curPath []string) {
		sch := n.GetSchema()
		if sch == nil {
			//invalid path, skip
			return
		}
		if n.Default() {
			//once a node is 'default' so are all of its children
			//and defaults don't get set in the session tree.
			return
		}
		curPath = pathutil.CopyAppend(curPath, n.Name())
		setFn(n, curPath)
		for _, ch := range n.SortedChildren() {
			preord(ch, curPath)
		}
		return
	}
	for _, ch := range ltree.SortedChildren() {
		preord(ch, nil)
	}
	if len(errors) == 0 {
		return nil
	}

	var merr mgmterror.MgmtErrorList
	merr.MgmtErrorListAppend(errors...)
	return merr
}
