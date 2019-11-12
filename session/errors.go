// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"github.com/danos/mgmterror"
)

func lockDenied(sess string) error {
	err := mgmterror.NewLockDeniedError(sess)
	err.Message = "session is locked by " + sess
	return err
}

func nilSessionMgrError() error {
	err := mgmterror.NewOperationFailedApplicationError()
	err.Message = "cannot get a session on a nil manager"
	return err
}

func sessTermError() error {
	err := mgmterror.NewOperationFailedApplicationError()
	err.Message = "session terminated"
	return err
}
