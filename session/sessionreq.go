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
	"github.com/danos/config/union"
	"github.com/danos/configd"
	"github.com/danos/configd/rpc"
)

// a request type defines the alphabet of the request channel.
// this allows a polymorphic channel request for each different communcation.
type request interface {
	reqty()
}
type mergetreereq struct {
	ctx      *configd.Context
	defaults bool
	resp     chan *data.Node
}

func (*mergetreereq) reqty() {}

type setreq struct {
	ctx  *configd.Context
	path []string
	resp chan error
}

func (*setreq) reqty() {}

type validatesetreq struct {
	setreq
}

type getresp struct {
	vals []string
	err  error
}

type getreq struct {
	ctx  *configd.Context
	path []string
	resp chan getresp
}

func (*getreq) reqty() {}

type typeresp struct {
	val rpc.NodeType
	err error
}

type typereq struct {
	ctx  *configd.Context
	path []string
	resp chan typeresp
}

func (*typereq) reqty() {}

type statusresp struct {
	val rpc.NodeStatus
	err error
}

type statusreq struct {
	ctx  *configd.Context
	path []string
	resp chan statusresp
}

func (*statusreq) reqty() {}

type defaultresp struct {
	val bool
	err error
}

type defaultreq struct {
	ctx  *configd.Context
	path []string
	resp chan defaultresp
}

func (*defaultreq) reqty() {}

type gettreeresp struct {
	val union.Node
	err error
}

type gettreereq struct {
	ctx  *configd.Context
	path []string
	opts *TreeOpts
	resp chan gettreeresp
}

func (*gettreereq) reqty() {}

type getfulltreeresp struct {
	val   union.Node
	err   error
	warns []error
}

type getfulltreereq struct {
	ctx  *configd.Context
	path []string
	opts *TreeOpts
	resp chan getfulltreeresp
}

func (*getfulltreereq) reqty() {}

type existsreq struct {
	ctx  *configd.Context
	path []string
	resp chan bool
}

func (*existsreq) reqty() {}

type delreq struct {
	ctx  *configd.Context
	path []string
	resp chan error
}

func (*delreq) reqty() {}

type validatereq struct {
	ctx  *configd.Context
	resp chan *commitresp
}

func (*validatereq) reqty() {}

type lockresp struct {
	pid int32
	err error
}

type lockreq struct {
	ctx  *configd.Context
	pid  int
	resp chan lockresp
}

func (*lockreq) reqty() {}

type unlockreq struct {
	ctx  *configd.Context
	pid  int
	resp chan lockresp
}

func (*unlockreq) reqty() {}

type lockedreq struct {
	ctx  *configd.Context
	resp chan lockresp
}

func (*lockedreq) reqty() {}

type commentreq struct {
	ctx  *configd.Context
	path []string
	resp chan error
}

func (*commentreq) reqty() {}

type savedreq struct {
	ctx  *configd.Context
	resp chan bool
}

func (*savedreq) reqty() {}

type changedreq struct {
	ctx  *configd.Context
	resp chan bool
}

func (*changedreq) reqty() {}

type marksavedreq struct {
	ctx   *configd.Context
	saved bool
	resp  chan error
}

func (*marksavedreq) reqty() {}

type showresp struct {
	data string
	err  error
}

type showreq struct {
	ctx          *configd.Context
	path         []string
	hideSecrets  bool
	showDefaults bool
	resp         chan showresp
}

func (*showreq) reqty() {}

type discardreq struct {
	ctx  *configd.Context
	resp chan error
}

func (*discardreq) reqty() {}

type loadresp struct {
	err          error
	invalidPaths []error
}

type loadreq struct {
	ctx    *configd.Context
	file   string
	reader io.Reader
	resp   chan loadresp
}

func (*loadreq) reqty() {}

type mergeresp struct {
	err          error
	invalidPaths []error
}

type mergereq struct {
	ctx  *configd.Context
	file string
	resp chan mergeresp
}

func (*mergereq) reqty() {}

type commitreq struct {
	ctx     *configd.Context
	message string
	resp    chan *commitresp
	debug   bool
}

func (*commitreq) reqty() {}

type gethelpreq struct {
	ctx    *configd.Context
	schema bool
	path   []string
	resp   chan map[string]string
}

func (*gethelpreq) reqty() {}

type editconfigreq struct {
	ctx     *configd.Context
	target  string
	defop   string
	testopt string
	erropt  string
	config  string
	resp    chan error
}

func (*editconfigreq) reqty() {}
