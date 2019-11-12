// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// Functions required for server_test but not part of public API.

package server

import (
	"github.com/danos/config/schema"
	"github.com/danos/configd"
	"github.com/danos/configd/session"
)

func (d *Disp) CallRpcWithCaller(
	moduleIdOrNamespace, rpcName, args, encoding string,
	vrc VciRpcCaller,
) (string, error) {
	return d.callRpcInternal(moduleIdOrNamespace, rpcName, args, encoding, vrc)
}

func (d *Disp) SchemaGetUnescaped(modOrSubmod string) (string, error) {
	schema, err := d.getModuleOrSubmoduleSchema(modOrSubmod)
	if err != nil {
		return "", err
	}
	return schema, nil
}

func NewDispatcher(
	smgr *session.SessionMgr,
	cmgr *session.CommitMgr,
	ms schema.ModelSet,
	msFull schema.ModelSet,
	ctx *configd.Context,
) *Disp {
	return &Disp{
		smgr:   smgr,
		cmgr:   cmgr,
		ms:     ms,
		msFull: msFull,
		ctx:    ctx,
	}
}
