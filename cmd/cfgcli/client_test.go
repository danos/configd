// Copyright (c) 2018-2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"testing"

	"fmt"
	"github.com/danos/configd/rpc"
)

type MockReturnParams struct {
	retStr  string
	retInt  int
	retBool bool
	retErr  error
}

type MockExpectation struct {
	fnName     string
	callParams []string
	retParams  *MockReturnParams
}

type Mocker struct {
	expectations []MockExpectation
}

func (m *Mocker) AddExpectedCall(expCall MockExpectation) {
	m.expectations = append(m.expectations, expCall)
}

func (m *Mocker) MakeActualCall(
	t *testing.T,
	name string,
	callParams []string,
) *MockReturnParams {

	if len(m.expectations) == 0 {
		t.Fatalf("Call to '%s' not expected.", name)
		return nil
	}

	expectation := m.expectations[0]
	m.expectations = m.expectations[1:]

	if name != expectation.fnName {
		t.Fatalf("Expected '%s' to be called, got '%s'",
			expectation.fnName, name)
		return nil
	}
	if len(expectation.callParams) != len(callParams) {
		t.Fatalf("%s(): expected %d params, got %d", name,
			len(expectation.callParams), len(callParams))
		return nil
	}
	for index, param := range expectation.callParams {
		if param != callParams[index] {
			t.Fatalf("%s(): expected param '%s', got '%s'", name,
				param, callParams[index])
		}
	}
	return expectation.retParams
}

func (m *Mocker) CheckAllCallsMade(t *testing.T) {
	if len(m.expectations) > 0 {
		t.Fatalf("Not all expected calls were made!")
	}
}

type testClient struct {
	t              *testing.T
	cfgSysFeatures map[string]struct{}
	commitLog      map[string]string
	Mocker
}

func newTestClient(t *testing.T) *testClient {
	featMap := make(map[string]struct{}, 0)
	cmtLogMap := make(map[string]string, 0)
	return &testClient{
		t:              t,
		cfgSysFeatures: featMap,
		commitLog:      cmtLogMap}
}

func (tc *testClient) enableFeature(feature string) *testClient {
	tc.cfgSysFeatures[feature] = struct{}{}
	return tc
}

func (tc *testClient) setCommitLog(numEntries int) *testClient {
	for i := 0; i < numEntries; i++ {
		tc.commitLog[fmt.Sprintf("%d", i)] = fmt.Sprintf(
			"2019-08-21 09:00:%d vyatta", i)
	}
	return tc
}

func (tc *testClient) CancelCommit(comment string, force, debug bool) (string, error) {
	panic("Rollback testClient method not yet implemented")
}

func (tc *testClient) Commit(message string, debug bool) (string, error) {
	panic("Commit testClient method not yet implemented")
}

func (tc *testClient) CommitConfirm(message string, debug bool, mins int,
) (string, error) {
	panic("CommitConfirm testClient method not yet implemented")
}

func (tc *testClient) CompareConfigRevisions(revOne, revTwo string) (string, error) {
	panic("CompareConfigRevisions testClient method not yet implemented")
}

func (tc *testClient) CompareSessionChanges() (string, error) {
	panic("CompareSessionChanges testClient method not yet implemented")
}

func (tc *testClient) Confirm() (string, error) {
	panic("Confirm testClient method not yet implemented")
}

func (tc *testClient) ConfirmPersistId(persistid string) (string, error) {
	panic("ConfirmPersistId testClient method not yet implemented")
}
func (tc *testClient) ConfirmSilent() (string, error) {
	panic("ConfirmSilent testClient method not yet implemented")
}

func (tc *testClient) Delete(path string) error {
	panic("Delete testClient method not yet implemented")
}

func (tc *testClient) Discard() error {
	panic("Discard testClient method not yet implemented")
}
func (tc *testClient) Exists(db rpc.DB, path string) (bool, error) {
	panic("Exists testClient method not yet implemented")
}

func (tc *testClient) Expand(path string) (string, error) {

	retParams := tc.MakeActualCall(tc.t, "Expand", []string{path})
	return retParams.retStr, retParams.retErr
}

func (tc *testClient) ExpandWithPrefix(
	path, prefix string,
	pos int,
) (string, error) {

	retParams := tc.MakeActualCall(tc.t, "ExpandWithPrefix",
		[]string{path, prefix, fmt.Sprintf("%d", pos)})
	return retParams.retStr, retParams.retErr
}

func (tc *testClient) ExtractArchive(file, destination string) (string, error) {
	panic("ExtractArchive testClient method not yet implemented")
}

func (tc *testClient) Get(db rpc.DB, path string) ([]string, error) {
	panic("Get testClient method not yet implemented")
}

func (tc *testClient) GetCommitLog() (map[string]string, error) {
	return tc.commitLog, nil
}

func (tc *testClient) GetConfigSystemFeatures() (map[string]struct{}, error) {
	return tc.cfgSysFeatures, nil
}

func (tc *testClient) GetCompletions(
	schema bool, path string,
) (map[string]string, error) {
	panic("GetCompletions testClient method not yet implemented")
}

func (tc *testClient) Load(file string) error {
	panic("Load testClient method not yet implemented")
}

func (tc *testClient) LoadFrom(source, routingInstance string) error {
	panic("LoadFrom testClient method not yet implemented")
}

func (tc *testClient) LoadKeys(user, source, routingInstance string) (string, error) {
	panic("LoadKeys testClient method not yet implemented")
}

func (tc *testClient) MergeReportWarnings(file string) (bool, error) {
	panic("MergeReportWarnings testClient method not yet implemented")
}

func (tc *testClient) NodeGetType(path string) (rpc.NodeType, error) {
	panic("NodeGetType testClient method not yet implemented")
}

func (tc *testClient) Rollback(revision, comment string, debug bool) (string, error) {
	panic("Rollback testClient method not yet implemented")
}

func (tc *testClient) Save(file string) error {
	panic("Save testClient method not yet implemented")
}

func (tc *testClient) SaveTo(dest, routingInstance string) error {
	panic("SaveTo testClient method not yet implemented")
}

func (tc *testClient) SessionChanged() (bool, error) {
	panic("SessionChanged testClient method not yet implemented")
}

func (tc *testClient) SessionMarkSaved() error {
	panic("SessionMarkSaved testClient method not yet implemented")
}

func (tc *testClient) Set(path string) (string, error) {
	panic("Set testClient method not yet implemented")
}

func (tc *testClient) ShowConfigWithContextDiffs(path string, showDefs bool,
) (string, error) {
	panic("ShowConfigWithContextDiffs testClient method not yet implemented")
}

func (tc *testClient) TmplGet(path string) (map[string]string, error) {
	panic("TmplGet testClient method not yet implemented")
}

func (tc *testClient) TmplValidatePath(path string) (bool, error) {
	retParams := tc.MakeActualCall(tc.t, "TmplValidatePath",
		[]string{path})
	return retParams.retBool, retParams.retErr
}

func (tc *testClient) Validate() (string, error) {
	panic("Validate testClient method not yet implemented")
}
