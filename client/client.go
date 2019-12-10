// Copyright (c) 2017-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/danos/configd/rpc"
)

var defaultOpts = map[string]interface{}{"Defaults": true, "Secrets": true}

//GetFuncName() returns the unqualified name of the caller
func GetFuncName() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "invalid"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "invalid"
	}
	name := fn.Name()
	i := strings.LastIndex(name, ".")
	return name[i+1:]
}

type Client struct {
	conn net.Conn
	sid  string
	enc  *json.Encoder
	dec  *json.Decoder
	id   int
}

func Dial(network, address, sid string) (*Client, error) {
	c, e := net.Dial(network, address)
	if e != nil {
		return nil, e
	}

	client := &Client{
		conn: c,
		enc:  json.NewEncoder(c),
		dec:  json.NewDecoder(c),
		id:   0,
		sid:  sid,
	}

	return client, nil
}

func (c *Client) Close() {
	if c.conn == nil {
		return
	}
	c.conn.Close()
}

func (c *Client) call(method string, args ...interface{}) (interface{}, error) {
	var rep rpc.Response
	c.id++
	c.enc.Encode(&rpc.Request{Method: method, Args: args, Id: c.id})
	c.dec.Decode(&rep)
	//fmt.Printf("%#v\n", &rpc.Request{Method: method, Args: args, Id: c.id})
	//fmt.Printf("%#v\n", rep)
	if err, ok := rep.Error.(string); ok {
		return rep.Result, errors.New(err)
	}
	return rep.Result, nil
}

//Per JSON RPC spec we must return a value upon success. This is not idomatic for go,
//so if the method will only return an error just ignore the bool.
func (c *Client) callBoolIgnore(method string, args ...interface{}) error {
	i, err := c.call(method, args...)
	if err != nil {
		return err
	}
	if _, ok := i.(bool); ok {
		return nil
	} else {
		return fmt.Errorf("wrong return type for %s got %T expecting bool", method, i)
	}
}

func (c *Client) callBool(method string, args ...interface{}) (bool, error) {
	i, err := c.call(method, args...)
	if err != nil {
		return false, err
	}
	if v, ok := i.(bool); ok {
		return v, nil
	} else {
		return false, fmt.Errorf("wrong return type for %s got %T expecting bool", method, i)
	}
}

func (c *Client) callInt(method string, args ...interface{}) (int, error) {
	i, err := c.call(method, args...)
	if err != nil {
		return -1, err
	}
	if v, ok := i.(float64); ok {
		return int(v), nil
	} else {
		return -1, fmt.Errorf("wrong return type for %s got %T expecting float64", method, i)
	}
}

func (c *Client) callString(method string, args ...interface{}) (string, error) {
	i, err := c.call(method, args...)
	if err != nil {
		return "", err
	}
	if v, ok := i.(string); ok {
		return v, nil
	} else {
		return "", fmt.Errorf("wrong return type for %s got %T expecting string", method, i)
	}
}

func (c *Client) callMap(method string, args ...interface{}) (map[string]interface{}, error) {
	i, err := c.call(method, args...)
	if err != nil {
		return nil, err
	}
	if v, ok := i.(map[string]interface{}); ok {
		return v, nil
	} else {
		return nil, fmt.Errorf("wrong return type for %s got %T expecting map[string]interface{}", method, i)
	}
}

func (c *Client) callMapString(method string, args ...interface{}) (map[string]string, error) {
	i, err := c.call(method, args...)
	if err != nil {
		return nil, err
	}
	if v, ok := i.(map[string]interface{}); ok {
		out := make(map[string]string)
		for k, val := range v {
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("wrong return type for %s got %T expecting string", method, val)
			}
			out[k] = str
		}
		return out, nil
	} else {
		return nil, fmt.Errorf("wrong return type for %s got %T expecting map[string]interface{}", method, i)
	}
}

func (c *Client) callMapStruct(method string, args ...interface{}) (map[string]struct{}, error) {
	v, err := c.callMap(method, args...)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{})
	for k, _ := range v {
		out[k] = struct{}{}
	}
	return out, nil
}

func (c *Client) callSlice(method string, args ...interface{}) ([]interface{}, error) {
	i, err := c.call(method, args...)
	if err != nil {
		return nil, err
	}
	if v, ok := i.([]interface{}); ok {
		return v, nil
	} else {
		return nil, fmt.Errorf("wrong return type for %s got %T expecting []string", method, i)
	}
}

func (c *Client) callSliceString(method string, args ...interface{}) ([]string, error) {
	i, err := c.call(method, args...)
	if err != nil {
		return nil, err
	}
	if v, ok := i.([]interface{}); ok {
		out := make([]string, 0, len(v))
		for _, val := range v {
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("wrong return type for %s got %T expecting string", method, val)
			}
			out = append(out, str)
		}
		return out, nil
	} else {
		return nil, fmt.Errorf("wrong return type for %s got %T expecting []string", method, i)
	}
}

func (c *Client) SessionExists() (bool, error) {
	return c.callBool(GetFuncName(), c.sid)
}
func (c *Client) SessionSetup() error {
	return c.callBoolIgnore(GetFuncName(), c.sid)
}
func (c *Client) SessionTeardown() error {
	return c.callBoolIgnore(GetFuncName(), c.sid)
}
func (c *Client) SessionChanged() (bool, error) {
	return c.callBool(GetFuncName(), c.sid)
}
func (c *Client) SessionSaved() (bool, error) {
	return c.callBool(GetFuncName(), c.sid)
}
func (c *Client) SessionMarkSaved() error {
	return c.callBoolIgnore(GetFuncName(), c.sid)
}
func (c *Client) SessionMarkUnsaved() error {
	return c.callBoolIgnore(GetFuncName(), c.sid)
}
func (c *Client) SessionGetEnv() (map[string]interface{}, error) {
	return c.callMap(GetFuncName(), c.sid)
}

func (c *Client) TmplGet(path string) (map[string]string, error) {
	return c.callMapString(GetFuncName(), path)
}
func (c *Client) TmplGetChildren(path string) ([]string, error) {
	return c.callSliceString(GetFuncName(), path)
}
func (c *Client) TmplGetAllowed(path string) ([]string, error) {
	return c.callSliceString(GetFuncName(), c.sid, path)
}
func (c *Client) TmplValidatePath(path string) (bool, error) {
	return c.callBool(GetFuncName(), path)
}
func (c *Client) TmplValidateValues(path string) (bool, error) {
	return c.callBool(GetFuncName(), path)
}

func (c *Client) Get(db rpc.DB, path string) ([]string, error) {
	return c.callSliceString(GetFuncName(), db, c.sid, path)
}
func (c *Client) TreeGet(db rpc.DB, path, encoding string) (string, error) {
	return c.callString(GetFuncName(), db, c.sid, path, encoding, defaultOpts)
}
func (c *Client) TreeGetFull(db rpc.DB, path, encoding string) (string, error) {
	return c.callString(GetFuncName(), db, c.sid, path, encoding, defaultOpts)
}
func (c *Client) Exists(db rpc.DB, path string) (bool, error) {
	return c.callBool(GetFuncName(), db, c.sid, path)
}
func (c *Client) NodeGetStatus(db rpc.DB, path string) (int, error) {
	return c.callInt(GetFuncName(), db, c.sid, path)
}
func (c *Client) NodeGetType(path string) (rpc.NodeType, error) {
	nt, err := c.callInt(GetFuncName(), c.sid, path)
	return rpc.NodeType(nt), err
}

func (c *Client) Set(path string) (string, error) {
	return c.callString(GetFuncName(), c.sid, path)
}
func (c *Client) ValidatePath(path string) (string, error) {
	return c.callString(GetFuncName(), c.sid, path)
}
func (c *Client) Delete(path string) error {
	return c.callBoolIgnore(GetFuncName(), c.sid, path)
}
func (c *Client) Rename(fpath, tpath string) error {
	return c.callBoolIgnore(GetFuncName(), c.sid, fpath, tpath)
}
func (c *Client) Copy(fpath, tpath string) error {
	return c.callBoolIgnore(GetFuncName(), c.sid, fpath, tpath)
}
func (c *Client) Comment(path string) error {
	return c.callBoolIgnore(GetFuncName(), c.sid, path)
}
func (c *Client) CancelCommit(comment string, force, debug bool) (string, error) {
	return c.callString(GetFuncName(), c.sid, comment, "", force, debug)
}
func (c *Client) Rollback(revision, comment string, debug bool) (string, error) {
	return c.callString(GetFuncName(), c.sid, revision, comment, debug)
}
func (c *Client) Confirm() (string, error) {
	return c.callString(GetFuncName(), c.sid)
}
func (c *Client) ConfirmPersistId(persistid string) (string, error) {
	return c.callString(GetFuncName(), persistid)
}
func (c *Client) ConfirmSilent() (string, error) {
	return c.callString(GetFuncName(), c.sid)
}
func (c *Client) CommitConfirm(
	message string,
	debug bool,
	mins int,
) (string, error) {
	return c.callString(GetFuncName(), c.sid, message, debug, mins)
}
func (c *Client) Commit(message string, debug bool) (string, error) {
	return c.callString(GetFuncName(), c.sid, message, debug)
}
func (c *Client) Discard() error {
	return c.callBoolIgnore(GetFuncName(), c.sid)
}
func (c *Client) Save(file string) error {
	return c.callBoolIgnore(GetFuncName(), file)
}
func (c *Client) SaveTo(dest, routingInstance string) error {
	return c.callBoolIgnore(GetFuncName(), dest, routingInstance)
}
func (c *Client) ExtractArchive(file, destination string) (string, error) {
	s, e := c.callString(GetFuncName(), c.sid, file, destination)
	return s, e
}
func (c *Client) Load(file string) error {
	return c.callBoolIgnore(GetFuncName(), c.sid, file)
}
func (c *Client) LoadFrom(source string, routingInstance string) error {
	return c.callBoolIgnore(GetFuncName(), c.sid, source, routingInstance)
}
func (c *Client) LoadKeys(user, source, routingInstance string) (string, error) {
	return c.callString(GetFuncName(), c.sid, user, source, routingInstance)
}
func (c *Client) Merge(file string) error {
	return c.callBoolIgnore(GetFuncName(), c.sid, file)
}
func (c *Client) LoadReportWarnings(file string) (bool, error) {
	return c.callBool(GetFuncName(), c.sid, file)
}
func (c *Client) MergeReportWarnings(file string) (bool, error) {
	return c.callBool(GetFuncName(), c.sid, file)
}
func (c *Client) Validate() (string, error) {
	return c.callString(GetFuncName(), c.sid)
}
func (c *Client) Show(db rpc.DB, path string) (string, error) {
	return c.callString(GetFuncName(), db, c.sid, path)
}
func (c *Client) ShowConfigWithContextDiffs(path string, showDefaults bool) (string, error) {
	return c.callString(GetFuncName(), c.sid, path, showDefaults)
}
func (c *Client) SchemaGet(module string, format string) (string, error) {
	return c.callString(GetFuncName(), module, format)
}
func (c *Client) GetSchemas() (string, error) {
	return c.callString(GetFuncName())
}
func (c *Client) GetModuleSchemas() (string, error) {
	return c.callString(GetFuncName())
}
func (c *Client) GetFeatures() (map[string]string, error) {
	return c.callMapString(GetFuncName())
}
func (c *Client) GetDeviations() (map[string]string, error) {
	return c.callMapString(GetFuncName())
}
func (c *Client) GetCommitLog() (map[string]string, error) {
	return c.callMapString(GetFuncName())
}
func (c *Client) GetConfigSystemFeatures() (map[string]struct{}, error) {
	return c.callMapStruct(GetFuncName())
}
func (c *Client) AuthAuthorize(path string, perm int) (bool, error) {
	return c.callBool(GetFuncName(), path, perm)
}

func (c *Client) AuthGetPerms() (map[string]string, error) {
	return c.callMapString(GetFuncName())
}

func (c *Client) GetCompletions(schema bool, path string) (map[string]string, error) {
	return c.callMapString(GetFuncName(), c.sid, schema, path)
}

func (c *Client) GetHelp(schema bool, path string) (map[string]string, error) {
	return c.callMapString(GetFuncName(), c.sid, schema, path)
}

func (c *Client) ReadConfigFile(filename string) (string, error) {
	return c.callString(GetFuncName(), filename)
}

func (c *Client) ReadConfigFileRaw(filename string) (string, error) {
	return c.callString(GetFuncName(), filename)
}

func (c *Client) CallRpc(namespace, name, args, encoding string) (string, error) {
	return c.callString(GetFuncName(), namespace, name, args, encoding)
}

// TODO: Eventually remove this
func (c *Client) CallRpcXml(namespace, name, args string) (string, error) {
	return c.callString(GetFuncName(), namespace, name, args)
}

func (c *Client) MigrateConfigFile(filename string) (string, error) {
	return c.callString(GetFuncName(), filename)
}

func (c *Client) Expand(path string) (string, error) {
	return c.callString(GetFuncName(), path)
}

func (c *Client) ExpandWithPrefix(
	path, prefix string,
	pos int,
) (string, error) {
	return c.callString(GetFuncName(), path, prefix, pos)
}

func (c *Client) Compare(old, new, spath string, ctxdiff bool) (string, error) {
	return c.callString(GetFuncName(), old, new, spath, ctxdiff)
}

func (c *Client) CompareConfigRevisions(revOne string, revTwo string) (string, error) {
	return c.callString(GetFuncName(), c.sid, revOne, revTwo)
}

func (c *Client) CompareSessionChanges() (string, error) {
	return c.callString(GetFuncName(), c.sid)
}

func (c *Client) SetConfigDebug(dbgType, level string) (string, error) {
	return c.callString(GetFuncName(), c.sid, dbgType, level)
}
