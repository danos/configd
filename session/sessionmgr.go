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

//Internal unprotected function, reduces lock pressure
func (mgr *SessionMgr) get(sid string) (*Session, error) {
	sess, ok := mgr.sessions[sid]
	if !ok {
		err := mgmterror.NewOperationFailedApplicationError()
		err.Message = "session " + sid + " does not exist"
		return nil, err
	}
	return sess, nil
}

func (mgr *SessionMgr) Get(_ *configd.Context, sid string) (*Session, error) {
	if mgr == nil {
		return nil, nilSessionMgrError()
	}
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.get(sid)
}

func (mgr *SessionMgr) create(ctx *configd.Context, sid string, cmgr *CommitMgr, st, stFull schema.ModelSet) (*Session, error) {
	sess, ok := mgr.sessions[sid]
	if ok {
		lpid, _ := sess.Locked(ctx)
		if lpid != 0 && lpid != ctx.Pid {
			return nil, lockDenied(strconv.Itoa(int(lpid)))
		}
		return sess, nil
	}

	sess = NewSession(sid, cmgr, st, stFull)
	mgr.sessions[sid] = sess
	return sess, nil
}

func (mgr *SessionMgr) Create(ctx *configd.Context, sid string, cmgr *CommitMgr, st, stFull schema.ModelSet) (*Session, error) {
	if mgr == nil {
		return nil, nilSessionMgrError()
	}
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	return mgr.create(ctx, sid, cmgr, st, stFull)
}

func (mgr *SessionMgr) destroy(ctx *configd.Context, sid string) error {
	sess, ok := mgr.sessions[sid]
	if ok {
		lpid, _ := sess.Locked(ctx)
		if lpid != 0 && lpid != ctx.Pid {
			return lockDenied(strconv.Itoa(int(lpid)))
		}
		delete(mgr.sessions, sid)
		go sess.Kill()
	}
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
	sess, err := mgr.get(sid)
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
	sess, err := mgr.get(sid)
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
