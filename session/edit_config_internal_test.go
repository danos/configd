// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"github.com/danos/configd"
)

func NewTestEditConfig(
	s *Session,
	ctx *configd.Context,
	config_target,
	def_operation,
	test_option,
	error_option,
	config string,
) (*edit_config, error) {
	return newEditConfigXML(&s.s, ctx, config_target, def_operation,
		test_option, error_option, []byte(config))
}

func NewTestEditOp(path []string, op_str string) (*edit_op, error) {
	op := &edit_op{path: path}

	if op_str == "" {
		op.op = op_notset
	} else if err := op.op.set(op_str); err != nil {
		return nil, err
	}
	return op, nil
}
