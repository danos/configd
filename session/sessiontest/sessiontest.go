// Copyright (c) 2017-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package sessiontest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"os"
	"os/user"
	"strconv"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/compmgrtest"
	"github.com/danos/config/data"
	"github.com/danos/config/load"
	"github.com/danos/config/schema"
	"github.com/danos/config/testutils"
	"github.com/danos/configd"
	. "github.com/danos/configd/session"
	"github.com/danos/vci/conf"
)

// For some operations, 'false' indicates pass, whereas in others, 'true'
// does.  Hide this mess with some enumerated values.
const (
	SetPass    = false
	SetFail    = true
	CommitPass = true
	CommitFail = false
)

var emptypath = []string{}

const (
	NotConfigdUser    = false
	ConfigdUser       = true
	NotInSecretsGroup = false
	InSecretsGroup    = true
)

var stackLogEnabled = true

func DisableStackLog() {
	stackLogEnabled = false
}

func EnableStackLog() {
	stackLogEnabled = true
}

func logStack(t *testing.T) {
	if stackLogEnabled {
		testutils.LogStackFatal(t)
	}
}

const schemaTemplate = `
module test-configd-session {
	namespace "urn:vyatta.com:test:configd-session";
	prefix test;
	organization "Brocade Communications Systems, Inc.";
	contact
		"Brocade Communications Systems, Inc.
		 Postal: 130 Holger Way
		         San Jose, CA 95134
		 E-mail: support@Brocade.com
		 Web: www.brocade.com";
	revision 2014-12-29 {
		description "Test schema for configd";
	}
	%s
}
`

const schemaImportTemplate = `
	import %s {
	    prefix %s;
    }
`

const schemaIncludeTemplate = `
	include %s;
`

const schemaModuleTemplate = `
module %s {
	namespace "urn:vyatta.com:test:%s";
	prefix %s;
    %s
    %s
	organization "Brocade Communications Systems, Inc.";
	contact
		"Brocade Communications Systems, Inc.
		 Postal: 130 Holger Way
		         San Jose, CA 95134
		 E-mail: support@Brocade.com
		 Web: www.brocade.com";
	revision 2014-12-29 {
		description "Test schema for configd";
	}
	%s
}
`

const schemaSubmoduleTemplate = `
submodule %s {
	belongs-to %s {
		prefix %s;
	}
	%s
	%s
}
`

// Used for creating tests with multiple modules without resorting to reading
// them in from file as this means you can't read the schema and the test
// together easily.
type TestSchema struct {
	Name          NameDef
	Imports       []NameDef
	Includes      []string
	BelongsTo     NameDef
	Prefix        string
	SchemaSnippet string
}

type NameDef struct {
	Namespace string
	Prefix    string
}

func NewTestSchema(namespace, prefix string) *TestSchema {
	return &TestSchema{Name: NameDef{Namespace: namespace, Prefix: prefix}}
}

func (ts *TestSchema) AddInclude(module string) *TestSchema {
	ts.Includes = append(ts.Includes, module)
	return ts
}

func (ts *TestSchema) AddBelongsTo(namespace, prefix string) *TestSchema {
	ts.BelongsTo.Namespace = namespace
	ts.BelongsTo.Prefix = prefix
	return ts
}

func (ts *TestSchema) AddImport(namespace, prefix string) *TestSchema {
	nd := NameDef{
		Namespace: namespace,
		Prefix:    prefix,
	}
	ts.Imports = append(ts.Imports, nd)
	return ts
}

func (ts *TestSchema) AddSchemaSnippet(snippet string) *TestSchema {
	ts.SchemaSnippet = snippet
	return ts
}

func constructSchema(schemaDef TestSchema) (schema string) {
	var importStr, includeStr string

	for _, inc := range schemaDef.Includes {
		includeStr = includeStr + fmt.Sprintf(schemaIncludeTemplate, inc)
	}

	if schemaDef.BelongsTo.Namespace != "" {
		schema = fmt.Sprintf(schemaSubmoduleTemplate,
			schemaDef.Name.Namespace,
			schemaDef.BelongsTo.Namespace, schemaDef.BelongsTo.Prefix,
			includeStr, schemaDef.SchemaSnippet)
	} else {
		for _, imp := range schemaDef.Imports {
			importStr = importStr + fmt.Sprintf(schemaImportTemplate,
				imp.Namespace, imp.Prefix)
		}

		schema = fmt.Sprintf(schemaModuleTemplate,
			schemaDef.Name.Namespace, schemaDef.Name.Namespace,
			schemaDef.Name.Prefix, importStr, includeStr,
			schemaDef.SchemaSnippet)
	}

	return schema
}

func ValidateTestSchemaSnippet(t *testing.T, schema string,
) (schema.ModelSet, schema.ModelSet, error) {

	sch := bytes.NewBufferString(fmt.Sprintf(schemaTemplate, schema))

	return testutils.NewModelSetSpec(t).
		SetSchemas(sch.Bytes()).
		GenerateModelSets()
}

// loosely modelled on Srv from configd/server/server.go
type TstSrv struct {
	Ms       schema.ModelSet
	MsFull   schema.ModelSet
	Smgr     *SessionMgr
	Cmgr     *CommitMgr
	Auth     auth.Auther
	Dlog     *log.Logger
	Elog     *log.Logger
	Wlog     *log.Logger
	Ctx      *configd.Context
	username string
	capsdir  string
}

// Setup enough infrastructure to enable test cases to operate
func tstInit(
	t *testing.T,
	ms, msFull schema.ModelSet,
	config, capDir string,
	a auth.Auther,
	isConfigdUser, inSecretsGroup bool,
	smgrLog *bytes.Buffer,
	compMgr schema.ComponentManager,
) (*TstSrv, error) {

	var rt *data.Node
	cfg := bytes.NewBufferString(config)
	rt, err, invalidPaths := load.LoadFile("config", cfg, ms)
	if err != nil {
		return nil, err
	}
	if len(invalidPaths) > 0 {
		return nil, fmt.Errorf("Unable to set the following path(s):\n%v\n",
			invalidPaths)
	}

	var u *user.User
	u, err = user.Current()
	if err != nil {
		return nil, err
	}

	var elog *log.Logger
	elog, err = syslog.NewLogger(syslog.LOG_ERR|syslog.LOG_DAEMON, 0)
	if err != nil {
		elog = log.New(ioutil.Discard, "", 0)
	}
	var slog *log.Logger
	if smgrLog != nil {
		slog = log.New(smgrLog, "SLOG:", 0)
	}

	s := &TstSrv{
		Ms:       ms,
		MsFull:   msFull,
		Smgr:     NewSessionMgrCustomLog(slog),
		Cmgr:     NewCommitMgr(data.NewAtomicNode(rt), ms),
		Dlog:     log.New(ioutil.Discard, "", 0),
		Elog:     elog,
		Wlog:     log.New(ioutil.Discard, "", 0),
		username: u.Username,
		capsdir:  capDir,
	}

	authGlobal := auth.NewAuthGlobal(u.Username, s.Dlog, s.Elog)
	s.Auth = auth.NewAuth(authGlobal)

	// Create sessions so access to RUNNING and EFFECTIVE
	// state is not special.
	s.Ctx = &configd.Context{
		User:     u.Username,
		UserHome: u.HomeDir,
		Pid:      int32(configd.SYSTEM),
		Auth:     s.Auth,
		Dlog:     s.Dlog,
		Elog:     s.Elog,
		Wlog:     s.Wlog,
		CompMgr:  compMgr,
		Configd:  isConfigdUser,
		Config: &configd.Config{
			Runfile:      "session_test.runfile",
			Yangdir:      "../../yang",
			Capabilities: capDir,
		},
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, err
	}
	s.Ctx.Uid = uint32(uid)

	if inSecretsGroup {
		s.Ctx.Groups = append(s.Ctx.Groups, s.Ctx.Config.SecretsGroup)
	}
	if a != nil {
		s.Ctx.Auth = a
	}
	s.Smgr.Create(s.Ctx, "RUNNING", s.Cmgr, s.Ms, s.MsFull, Shared)
	s.Smgr.Lock(s.Ctx, "RUNNING")

	effective, _ := s.Smgr.Create(
		s.Ctx, "EFFECTIVE", s.Cmgr, s.Ms, s.MsFull, Shared)
	s.Smgr.Lock(s.Ctx, "EFFECTIVE")
	s.Cmgr.SetEffective(effective)
	return s, nil
}

// Test startup with default authentication settings
func TstStartup(t *testing.T, schema, config string) (*TstSrv, *Session) {
	return NewTestSpec(t).
		SetSingleSchema(schema).
		SetConfig(config).
		Init()
}

// Test startup with default authentication settings
func TstStartupWithCapabilities(t *testing.T, schema, config, caps string) (*TstSrv, *Session) {
	return NewTestSpec(t).
		SetSingleSchema(schema).
		SetConfig(config).
		SetCapabilities(caps).
		Init()
}

func TstStartupWithCustomAuth(
	t *testing.T,
	schema, config string,
	a auth.Auther,
	isConfigdUser, inSecretsGroup bool,
) (*TstSrv, *Session) {

	return NewTestSpec(t).
		SetSingleSchema(schema).
		SetConfig(config).
		SetAuther(a, isConfigdUser, inSecretsGroup).
		Init()
}

func TstStartupMultipleSchemas(
	t *testing.T,
	schemaDefs []TestSchema,
	config string,
) (*TstSrv, *Session) {

	return NewTestSpec(t).
		SetSchemaDefs(schemaDefs).
		SetConfig(config).
		Init()
}

func TstStartupMultipleSchemasWithCustomAuth(
	t *testing.T,
	schemaDefs []TestSchema,
	config string,
	a auth.Auther,
	isConfigdUser, inSecretsGroup bool,
) (*TstSrv, *Session) {

	return NewTestSpec(t).
		SetSchemaDefs(schemaDefs).
		SetConfig(config).
		SetAuther(a, isConfigdUser, inSecretsGroup).
		Init()
}

func TstStartupSchemaDir(t *testing.T, schemaDir, config, capabilities string) (*TstSrv, *Session) {

	return NewTestSpec(t).
		SetSchemaDir(schemaDir).
		SetCapabilities(capabilities).
		SetConfig(config).
		Init()
}

// Clean up any resources that we created, but no longer need
func (ts *TstSrv) Cleanup() {
	if ts.capsdir != "" {
		os.RemoveAll(ts.capsdir)
		ts.capsdir = ""
	}
}

func (ts *TstSrv) LoadConfig(t *testing.T, config string, sess *Session) {

	cfgFile, err := ioutil.TempFile("", "config")
	if err != nil {
		t.Fatalf("Unable to create config file\n")
		return
	}
	defer os.Remove(cfgFile.Name())
	if _, err := cfgFile.Write([]byte(config)); err != nil {
		t.Fatalf("Unable to write config file\n")
		return
	}

	err, _ = sess.Load(ts.Ctx, cfgFile.Name(), nil)
	if err != nil {
		t.Fatalf("Load Error: %s\n", err.Error())
		return
	}
}

type TestSpec struct {
	// Provided at setup
	t              *testing.T
	singleSchema   string
	schemaDir      string
	schemaDefs     []TestSchema
	config         string
	capabilities   string
	components     []string
	compMgr        schema.ComponentManager
	auther         auth.Auther
	isConfigdUser  bool
	inSecretsGroup bool
	smgrLog        *bytes.Buffer

	// Derived internally
	schemas    [][]byte
	extensions *schema.CompilationExtensions
	capsDir    string
}

func NewTestSpec(t *testing.T) *TestSpec {

	ts := &TestSpec{
		t:              t,
		config:         "\n", // Tests hang with no trailing newline
		isConfigdUser:  ConfigdUser,
		inSecretsGroup: NotInSecretsGroup}

	ts.compMgr = compmgrtest.NewTestCompMgr(t)

	return ts
}

func (ts *TestSpec) GetCompMgr() schema.ComponentManager { return ts.compMgr }

func (ts *TestSpec) SetConfig(config string) *TestSpec {
	// Ensure we have trailing '\n' or test will hang.  Extra blank lines
	// are harmless.
	ts.config = config + "\n"
	return ts
}

func (ts *TestSpec) SetCapabilities(caps string) *TestSpec {
	ts.capabilities = caps
	return ts
}

func (ts *TestSpec) SetComponents(comps []string) *TestSpec {
	ts.components = comps
	return ts
}

func (ts *TestSpec) SetSessionMgrLog(smgrLog *bytes.Buffer) *TestSpec {
	ts.smgrLog = smgrLog
	return ts
}

func (ts *TestSpec) SetAuther(
	auther auth.Auther,
	isConfigdUser bool,
	inSecretsGroup bool,
) *TestSpec {
	ts.auther = auther
	ts.isConfigdUser = isConfigdUser
	ts.inSecretsGroup = inSecretsGroup
	return ts
}

func (ts *TestSpec) SetSingleSchema(singleSchema string) *TestSpec {
	if ts.schemaDir != "" || ts.schemaDefs != nil {
		ts.t.Fatalf("Can't set single schema - schema already set.\n")
		return ts
	}

	ts.singleSchema = singleSchema
	return ts
}

func (ts *TestSpec) SetSchemaDir(schemaDir string) *TestSpec {
	if ts.singleSchema != "" || ts.schemaDefs != nil {
		ts.t.Fatalf("Can't set schema dir - schema already set.\n")
		return ts
	}
	ts.schemaDir = schemaDir
	return ts
}

func (ts *TestSpec) SetSchemaDefs(schemaDefs []TestSchema) *TestSpec {
	if ts.schemaDir != "" || ts.singleSchema != "" {
		ts.t.Fatalf("Can't set schemaDefs - schema already set.\n")
		return ts
	}
	ts.schemaDefs = schemaDefs
	return ts
}

// When we have created schemaDefs using NewTestSchema, it is easier to
// pass by reference.
func (ts *TestSpec) SetSchemaDefsByRef(schemaDefs []*TestSchema) *TestSpec {
	if ts.schemaDir != "" || ts.singleSchema != "" {
		ts.t.Fatalf("Can't set schemaDefs - schema already set.\n")
		return ts
	}
	for _, schema := range schemaDefs {
		ts.schemaDefs = append(ts.schemaDefs, *schema)
	}
	return ts
}

func (ts *TestSpec) checkAndProcessSchemas() {
	if ts.singleSchema != "" {
		ts.schemas = append(ts.schemas,
			[]byte(fmt.Sprintf(schemaTemplate, ts.singleSchema)))
		return
	}

	if ts.schemaDefs != nil {
		ts.schemas = make([][]byte, len(ts.schemaDefs))
		for index, schemaDef := range ts.schemaDefs {
			ts.schemas[index] = []byte(constructSchema(schemaDef))
		}
		return
	}

	// schemaDir processed directly into ModelSets later.
}

func (ts *TestSpec) processComponents() {
	if ts.components == nil {
		return
	}

	var parsedComps []*conf.ServiceConfig

	for _, comp := range ts.components {
		parsedComp, err := conf.ParseConfiguration([]byte(comp))
		if err != nil {
			ts.t.Fatalf("Unable to parse component.")
			return
		}
		parsedComps = append(parsedComps, parsedComp)
	}

	ts.extensions = &schema.CompilationExtensions{
		ComponentConfig: parsedComps,
	}
}

func (ts *TestSpec) generateModelSets() (
	schema.ModelSet, schema.ModelSet, error) {
	ts.processComponents()
	ts.checkAndProcessSchemas()

	return testutils.NewModelSetSpec(ts.t).
		SetSchemas(ts.schemas...). // Either schemas or schemaDir non-nil.
		SetSchemaDir(ts.schemaDir).
		SetCapabilities(ts.capabilities).
		SetExtensions(ts.extensions).
		GenerateModelSets()
}

func (ts *TestSpec) Init() (*TstSrv, *Session) {
	ms, msFull, err := ts.generateModelSets()
	if err != nil {
		ts.t.Fatalf("Unable to generate model sets: %s", err)
		return nil, nil
	}

	srv, err := tstInit(ts.t, ms, msFull, ts.config, ts.capsDir,
		ts.auther, ts.isConfigdUser, ts.inSecretsGroup, ts.smgrLog,
		ts.compMgr)

	if err != nil {
		ts.t.Fatalf("Unable to initialize testspec; %s", err)
		return nil, nil
	}
	sess := NewSession("TEST", srv.Cmgr, srv.Ms, srv.MsFull)
	if _, errs, ok := sess.Validate(srv.Ctx); !ok {
		ts.t.Fatalf("Unable to validate initial configuration: %v", errs)
	}

	return srv, sess
}

func (ts *TestSpec) ClearCompLogEntries() {
	ts.compMgr.(*compmgrtest.TestCompMgr).ClearLogEntries()
}

// Checks exact match for number and order of entries, after filtering for
// given specific type of log entry (eg SetRunning)
func (ts *TestSpec) CheckCompLogEntries(
	name, filter string,
	entries ...compmgrtest.TestLogEntry,
) {
	ts.compMgr.(*compmgrtest.TestCompMgr).CheckLogEntries(
		ts.t, name, entries, filter)
}

func (ts *TestSpec) SetCurrentState(model, stateJson string) {
	ts.compMgr.(*compmgrtest.TestCompMgr).SetCurrentState(
		model, stateJson)
}
