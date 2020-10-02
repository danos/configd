// Copyright (c) 2018-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2015,2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"reflect"
	"strconv"
	"sync"
	"syscall"

	"github.com/danos/config/auth"
	"github.com/danos/configd"
	"github.com/danos/configd/rpc"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/audit"
	"github.com/danos/utils/os/group"
	"github.com/danos/utils/tty"
)

type any interface{}

func newResponse(result any, err error, id int) *rpc.Response {
	var resp rpc.Response
	if err != nil {
		switch val := err.(type) {
		case mgmterror.MgmtErrorList:
			resp = rpc.Response{MgmtErrList: val, Id: id}
		case mgmterror.MgmtErrorRef:
			var mel mgmterror.MgmtErrorList
			mel.MgmtErrorListAppend(err)
			resp = rpc.Response{MgmtErrList: mel, Id: id}
		default:
			resp = rpc.Response{Error: err.Error(), Id: id}
		}
	} else {
		resp = rpc.Response{Result: result, Id: id}
	}
	return &resp
}

type SrvConn struct {
	*net.UnixConn
	srv     *Srv
	uid     uint32
	pid     int
	cred    *syscall.Ucred
	enc     *json.Encoder
	dec     *json.Decoder
	sending *sync.Mutex
}

type LoginPidError struct {
	pid int32
}

func (e *LoginPidError) Error() string {
	return fmt.Sprintf("Login User Id is not set for PID %d", e.pid)
}

func newLoginPidError(pid int32) error {
	return &LoginPidError{pid: pid}
}

func IsLoginPidError(err error) bool {
	_, ok := err.(*LoginPidError)

	return ok
}

//Send an rpc response with appropriate data or an error
func (conn *SrvConn) sendResponse(resp *rpc.Response) error {
	conn.sending.Lock()
	err := conn.enc.Encode(&resp)
	conn.sending.Unlock()
	return err

}

//Receive an rpc request and do some preprocessing.
func (conn *SrvConn) readRequest() (*rpc.Request, error) {
	var req = new(rpc.Request)
	err := conn.dec.Decode(req)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// Get User ID for connecting process
func getLoginUid(pid int32) (uint32, error) {

	u, e := audit.GetPidLoginuid(pid)
	if e != nil {
		return 0, e
	}

	// The special value of -1 is used when login ID is not set. This
	// is the case for daemons and boot processes. Since we're using
	// unsigned numbers we take the bitwise complement of 0.
	if u == ^uint32(0) {
		return 0, newLoginPidError(pid)
	}

	return u, nil
}

//Grab the credentials off of the unix socet using SO_PEERCRED and store them int the SrvConn
func (conn *SrvConn) getCreds() (*syscall.Ucred, error) {
	uf, err := conn.File()
	if err != nil {
		return nil, err
	}
	cred, err := syscall.GetsockoptUcred(
		int(uf.Fd()),
		syscall.SOL_SOCKET,
		syscall.SO_PEERCRED)
	if err != nil {
		conn.srv.LogError(err)
		return nil, err
	}
	uf.Close()

	cred.Uid, err = getLoginUid(cred.Pid)

	return cred, err
}

// Handle is the main loop for a connection. It receives the requests,  authorizes
// the request, calls the request method and returns the response to the client.
func (conn *SrvConn) Handle() {

	var err error

	conn.cred, err = conn.getCreds()
	if err != nil {
		if !os.IsNotExist(err) {
			conn.srv.LogError(err)
		}
		if !IsLoginPidError(err) {
			conn.Close()
			return
		}
	}

	disp := &Disp{
		smgr:   conn.srv.smgr,
		cmgr:   conn.srv.cmgr,
		ms:     conn.srv.ms,
		msFull: conn.srv.msFull,
		ctx: &configd.Context{
			Configd:   conn.cred.Uid == conn.srv.uid,
			Uid:       conn.cred.Uid,
			Pid:       conn.cred.Pid,
			Groups:    make([]string, 0),
			Superuser: conn.cred.Uid == 0,
			Config:    conn.srv.Config,
			Elog:      conn.srv.Elog,
			Dlog:      conn.srv.Dlog,
			Wlog:      conn.srv.Wlog,
		},
	}

	//Group lookup is expensive, do it once per connection.
	//groups are not needed for commit spawned processes
	//if the uid is the same as configd auth allows it implicitly
	//don't include groups for these users
	if conn.cred.Uid != conn.srv.uid {
		groups, err := group.LookupUid(strconv.Itoa(int(disp.ctx.Uid)))
		conn.srv.LogError(err)
		haveSuperGroup := conn.srv.Config.SuperGroup != ""
		for _, gr := range groups {
			disp.ctx.Groups = append(disp.ctx.Groups, gr.Name)
			if haveSuperGroup && gr.Name == conn.srv.Config.SuperGroup {
				disp.ctx.Superuser = true
			}
		}
	}

	ttyName, err := tty.TtyNameForPid(int(conn.cred.Pid))
	if err != nil && !os.IsNotExist(err) {
		conn.srv.LogError(err)
	}

	authEnv := &auth.AuthEnv{Tty: ttyName}
	disp.ctx.Auth = auth.NewAuthForUser(conn.srv.authGlobal, disp.ctx.Uid, disp.ctx.Groups, authEnv)

	u, err := user.LookupId(strconv.Itoa(int(disp.ctx.Uid)))
	if err != nil {
		conn.srv.LogError(err)
		conn.Close()
		return
	}
	disp.ctx.User = u.Username
	disp.ctx.UserHome = u.HomeDir

	//Unlock all sessions this connection may have locked on return
	defer conn.srv.smgr.UnlockAllPid(disp.ctx)
	for {
		req, err := conn.readRequest()
		if err != nil {
			if err != io.EOF {
				conn.srv.LogError(err)
			}
			break
		}

		result, err := conn.Call(disp, req.Method, req.Args)
		err = conn.sendResponse(newResponse(result, err, req.Id))
		if err != nil {
			break
		}
	}
	if err = disp.sessionTermination(); err != nil {
		conn.srv.LogError(err)
	}
	conn.Close()
	return
}

func (conn *SrvConn) Call(
	disp *Disp,
	method string,
	args []interface{},
) (any, error) {

	m, ok := conn.srv.m[method]
	if !ok {
		return nil, &rpc.MethErr{Name: method}
	}

	if !disp.ctx.Auth.AuthorizeFn(disp.ctx.Uid, disp.ctx.Groups, method) {
		return nil, mgmterror.NewAccessDeniedApplicationError()
	}

	typ := m.Func.Type()

	//Number of args are equal?
	if len(args) != typ.NumIn()-1 {
		return nil, &rpc.ArgNErr{Method: method, Len: len(args), Elen: typ.NumIn() - 1}
	}

	//validate arguments
	//prepending the first argument *Disp
	vals := make([]reflect.Value, len(args)+1)
	vals[0] = reflect.ValueOf(disp)
	for i, v := range args {
		t1 := reflect.TypeOf(v)
		t2 := typ.In(i + 1)
		if t1 != t2 {
			if !t1.ConvertibleTo(t2) {
				return nil, &rpc.ArgErr{Method: method, Farg: v, Typ: t1.Name(), Etyp: t2.Name()}
			}
			vals[i+1] = reflect.ValueOf(v).Convert(t2)
		} else {
			vals[i+1] = reflect.ValueOf(v)
		}
	}

	//call the function
	rets := m.Func.Call(vals)
	err, ok := rets[1].Interface().(error)
	if ok {
		return rets[0].Interface(), err
	}

	return rets[0].Interface(), nil
}
