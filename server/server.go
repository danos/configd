// Copyright (c) 2018-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"log/syslog"
	"net"
	"os/user"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/danos/config/auth"
	"github.com/danos/config/data"
	"github.com/danos/config/load"
	"github.com/danos/config/schema"
	"github.com/danos/configd"
	"github.com/danos/configd/session"
)

type Srv struct {
	*net.UnixListener
	ms         schema.ModelSet
	msFull     schema.ModelSet
	m          map[string]reflect.Method
	smgr       *session.SessionMgr
	cmgr       *session.CommitMgr
	authGlobal *auth.AuthGlobal
	uid        uint32
	Dlog       *log.Logger
	Elog       *log.Logger
	Wlog       *log.Logger
	Config     *configd.Config
	CompMgr    schema.ComponentManager
}

func loadRunning(config *configd.Config, ms schema.ModelSet) *data.Node {
	t, _, _ := load.Load(config.Runfile, ms)
	return t
}

func NewSrv(
	l *net.UnixListener,
	ms, msFull schema.ModelSet,
	username string,
	config *configd.Config,
	elog *log.Logger,
	compMgr schema.ComponentManager,
) *Srv {
	rt := loadRunning(config, ms)

	dlog, err := configd.NewLogger(syslog.LOG_DEBUG|syslog.LOG_DAEMON, 0)
	if err != nil {
		elog.Println(err)
		dlog = log.New(ioutil.Discard, "", 0)
	}

	wlog, err := configd.NewLogger(syslog.LOG_WARNING|syslog.LOG_DAEMON, 0)
	if err != nil {
		elog.Println(err)
		wlog = log.New(ioutil.Discard, "", 0)
	}

	u, _ := user.Lookup(username)
	uid, _ := strconv.ParseUint(u.Uid, 10, 32)

	s := &Srv{
		UnixListener: l,
		ms:           ms,
		msFull:       msFull,
		m:            make(map[string]reflect.Method),
		smgr:         session.NewSessionMgr(),
		cmgr:         session.NewCommitMgr(data.NewAtomicNode(rt), ms),
		uid:          uint32(uid),
		Dlog:         dlog,
		Elog:         elog,
		Wlog:         wlog,
		Config:       config,
		CompMgr:      compMgr,
	}

	s.authGlobal = auth.NewAuthGlobal(username, s.Dlog, s.Elog)

	//Create sessions so access to RUNNING and EFFECTIVE
	//state is not special.
	ctx := &configd.Context{
		Pid:    int32(configd.SYSTEM),
		Auth:   auth.NewAuth(s.authGlobal),
		Config: config,
		Dlog:   s.Dlog,
		Elog:   s.Elog,
		Wlog:   s.Wlog,
	}
	s.smgr.Create(ctx, "RUNNING", s.cmgr, s.ms, s.msFull, session.Shared)
	s.smgr.Lock(ctx, "RUNNING")

	effective, _ := s.smgr.Create(
		ctx, "EFFECTIVE", s.cmgr, s.ms, s.msFull, session.Shared)
	s.smgr.Lock(ctx, "EFFECTIVE")
	s.cmgr.SetEffective(effective)

	t := reflect.TypeOf(new(Disp))
	for m := 0; m < t.NumMethod(); m++ {
		meth := t.Method(m)
		ftype := meth.Func.Type()
		if unicode.IsLower(rune(meth.Name[0])) {
			//only exported methods
			continue
		}
		if ftype.NumOut() != 2 {
			//with 2 return values
			continue
		}
		if ftype.Out(1).Name() != "error" {
			//whose second return value is an error
			continue
		}

		s.m[meth.Name] = meth
	}
	return s
}

//Serve is the server main loop. It accepts connections and spawns a goroutine to handle that connection.
func (s *Srv) Serve() error {
	var err error
	for {
		conn, err := s.AcceptUnix()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			s.LogError(err)
			break
		}
		sconn := s.NewConn(conn)

		go sconn.Handle(s.CompMgr)
	}
	return err
}

//NewConn creates a new SrvConn and returns a reference to it.
func (s *Srv) NewConn(conn *net.UnixConn) *SrvConn {
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	c := &SrvConn{
		UnixConn: conn,
		srv:      s,
		uid:      0,
		enc:      enc,
		dec:      dec,
		sending:  new(sync.Mutex),
	}
	return c
}

//Log is a common place to do logging so that the implementation may change in the future.
func (d *Srv) Log(fmt string, v ...interface{}) {
	d.Dlog.Printf(fmt, v...)
}

//LogError logs an error if the passed in value is non nil
func (d *Srv) LogError(err error) {
	if err != nil {
		d.Elog.Printf("%s", err)
	}
}

func (d *Srv) LogFatal(err error) {
	if err != nil {
		d.Elog.Fatal(err)
	}
}
