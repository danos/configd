// Copyright (c) 2018-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"io/ioutil"
	"log"
	"log/syslog"
	"strconv"
	"sync"

	"github.com/danos/config/schema"
	"github.com/danos/configd"
	"github.com/danos/mgmterror"
)

//Session manager is a monitor that provides access to the shared session state.
//All methods must be protected by Mutex
type SessionMgr struct {
	mu       *sync.RWMutex
	sessions map[string]*Session
	Elog     *log.Logger
}

func NewSessionMgr() *SessionMgr {
	elog, err := syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_DAEMON, 0)
	if err != nil {
		elog = log.New(ioutil.Discard, "", 0)
	}

	return NewSessionMgrCustomLog(elog)
}

func NewSessionMgrCustomLog(elog *log.Logger) *SessionMgr {
	return &SessionMgr{
		mu:       &sync.RWMutex{},
		sessions: make(map[string]*Session),
		Elog:     elog,
	}
}

func (mgr *SessionMgr) lookup(ctx *configd.Context, sid string) (*Session, error) {
	sess, ok := mgr.sessions[sid]
	if !ok {
		return nil, nil
	}

	/*
	 * Access to a session is permitted iff:
	 *   - the requesting user owns the session, or
	 *   - the session is shared (eg. NETCONF, RUNNING), or
	 *   - the requester is configd, or
	 *   - the requester is a superuser (for debugging)
	 */
	if sess.OwnedBy(ctx.Uid) || sess.IsShared() || ctx.Configd || ctx.Superuser {
		return sess, nil
	}

	return nil, mgmterror.NewAccessDeniedApplicationError()
}

//Internal unprotected function, reduces lock pressure
func (mgr *SessionMgr) get(ctx *configd.Context, sid string) (*Session, error) {
	sess, err := mgr.lookup(ctx, sid)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = "session " + sid + " does not exist"
		return nil, err
	}
	return sess, nil
}

func (mgr *SessionMgr) Get(ctx *configd.Context, sid string) (*Session, error) {
	if mgr == nil {
		return nil, nilSessionMgrError()
	}
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.get(ctx, sid)
}

func (mgr *SessionMgr) create(
	ctx *configd.Context, sid string, cmgr *CommitMgr, st, stFull schema.ModelSet, shared bool,
) (*Session, error) {

	sess, err := mgr.lookup(ctx, sid)
	if err != nil {
		return nil, err
	}
	if sess != nil {
		if shared != sess.IsShared() {
			err := mgmterror.NewOperationFailedApplicationError()
			err.Message = sid + " already exists as "
			if !sess.IsShared() {
				err.Message += "an un-shared session"
			} else {
				err.Message += "a shared session"
			}
			return nil, err
		}

		lpid, _ := sess.Locked(ctx)
		if lpid != 0 && lpid != ctx.Pid {
			return nil, lockDenied(strconv.Itoa(int(lpid)))
		}
		return sess, nil
	}

	opts := []SessionOption{}
	if !shared {
		opts = append(opts, WithOwner(ctx.Uid))
	}

	sess = NewSession(sid, cmgr, st, stFull, opts...)
	mgr.sessions[sid] = sess
	return sess, nil
}

func (mgr *SessionMgr) Create(
	ctx *configd.Context, sid string, cmgr *CommitMgr, st, stFull schema.ModelSet, shared bool,
) (*Session, error) {

	if mgr == nil {
		return nil, nilSessionMgrError()
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	return mgr.create(ctx, sid, cmgr, st, stFull, shared)
}

func (mgr *SessionMgr) destroy(ctx *configd.Context, sid string) error {
	sess, err := mgr.lookup(ctx, sid)
	if sess == nil || err != nil {
		return err
	}

	lpid, _ := sess.Locked(ctx)
	if lpid != 0 && lpid != ctx.Pid {
		return lockDenied(strconv.Itoa(int(lpid)))
	}
	delete(mgr.sessions, sid)
	go sess.Kill()

	return nil
}

func (mgr *SessionMgr) Destroy(ctx *configd.Context, sid string) error {
	if mgr == nil {
		return nilSessionMgrError()
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	return mgr.destroy(ctx, sid)
}

func (mgr *SessionMgr) Lock(ctx *configd.Context, sid string) (int32, error) {
	if mgr == nil {
		return -1, nilSessionMgrError()
	}
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	sess, err := mgr.get(ctx, sid)
	if err != nil {
		return -1, err
	}
	return sess.Lock(ctx)
}

func (mgr *SessionMgr) Unlock(ctx *configd.Context, sid string) (int32, error) {
	if mgr == nil {
		return -1, nilSessionMgrError()
	}
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	sess, err := mgr.get(ctx, sid)
	if err != nil {
		return -1, err
	}
	return sess.Unlock(ctx)
}

func (mgr *SessionMgr) UnlockAllPid(ctx *configd.Context) error {
	var err error
	if mgr == nil {
		return nilSessionMgrError()
	}
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	for _, sess := range mgr.sessions {
		if lkr, _ := sess.Locked(ctx); lkr != 0 && lkr == ctx.Pid {
			_, err = sess.Unlock(ctx)
		}
	}
	return err
}
