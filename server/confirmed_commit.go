// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/danos/mgmterror"
)

const (
	DefaultTimeout = 600
)

type ConfirmedCommitInfo struct {
	Session   string `json:"session"`
	PersistId string `json:"persist-id"`
}

func getConfirmedCommitInfo() *ConfirmedCommitInfo {
	info := &ConfirmedCommitInfo{}

	fl, err := os.Open("/config/confirmed_commit.job")
	if err != nil {
		// Ignore errors, likely no pending
		// confirmed commit
		return info
	}
	defer fl.Close()
	dec := json.NewDecoder(fl)
	dec.Decode(info)

	return info
}

type commitInfo struct {
	confirmed bool
	timeout   uint32
	persist   string
	persistId string
	session   int
}

func newCommitInfo(confirmed bool, timeout, persist, persistid string) (*commitInfo, error) {
	cmt := &commitInfo{}
	if timeout != "" {
		seconds, err := strconv.ParseUint(timeout, 10, 32)
		if err != nil {
			merr := mgmterror.NewInvalidValueProtocolError()
			merr.Message = err.Error()
			return cmt, merr
		}
		if seconds == 0 {
			merr := mgmterror.NewInvalidValueProtocolError()
			merr.Message = "timeout value out of range, 0 is not permitted"
			return cmt, merr
		}
		cmt.timeout = uint32(seconds)
	} else {
		cmt.timeout = DefaultTimeout
	}

	cmt.persist = persist
	cmt.persistId = persistid
	cmt.confirmed = confirmed
	return cmt, nil
}

func (c *commitInfo) arguments(session string) []string {
	args := make([]string, 0)
	args = append(args, "--action=confirmed-commit")
	args = append(args, fmt.Sprintf("--seconds=%d", c.timeout))
	if c.persist != "" {
		args = append(args, fmt.Sprintf("--persist=%s", c.persist))
	}
	if c.persistId != "" {
		args = append(args, fmt.Sprintf("--persistid=%s", c.persistId))
	}

	//if c.minutes != 0 {
	//	args = append(args, fmt.Sprintf("--minutes=%d", c.minutes))
	//}
	args = append(args, fmt.Sprintf("--session=%s", session))
	return args
}

func (d *Disp) isCommitAllowed(pid string, cmt *commitInfo, revert bool) error {
	info := getConfirmedCommitInfo()

	if info.Session != "" {
		// There is an outstanding confirmed-commit
		switch {
		case revert == true:
			d.ConfirmingCommit()
		case cmt == nil:
			// CLI commit, can't proceed if ongoing confirmed commit
			err := mgmterror.NewAccessDeniedApplicationError()
			err.Message = "Operation blocked by outstanding confirmed commit"
			return err
		case info.PersistId != cmt.persistId:
			// persist-id does not match outstanding confirmed-commit
			err := mgmterror.NewInvalidValueProtocolError()
			err.Message = "persist-id does not match outstanding confirmed commit"
			return err

		case cmt.persistId == "" && info.Session != pid:
			// Only consider the session identifier if there given persist-id
			err := mgmterror.NewAccessDeniedApplicationError()
			err.Message = "operation blocked by outstanding confirmed commit"
			return err
		case cmt.confirmed == false:
			// We have a valid confirming commit
			// confirm the pending confirmed-commit
			d.ConfirmingCommit()
		default:
			//Follow-up confirmed commit
		}
	}

	return nil
}
