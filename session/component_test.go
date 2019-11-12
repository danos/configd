// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains helper functions for testing components with an
// active session.

package session_test

import (
	"testing"

	"github.com/danos/config/data"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/yangd"
)

const (
	submoduleHasNoPrefix = ""
)

type cfgTestDispatcher struct{}

type cfgTestService struct {
	name string
}

func (d *cfgTestDispatcher) NewService(name string) (yangd.Service, error) {
	return &cfgTestService{name: name}, nil
}

func (s *cfgTestService) GetRunning(path string) ([]byte, error) {
	return []byte(testCfg[s.name]), nil
}

func (s *cfgTestService) GetState(path string) ([]byte, error) {
	return nil, nil
}

func (s *cfgTestService) ValidateCandidate(candidate []byte) error {
	return nil
}

func (s *cfgTestService) SetRunning(candidate []byte) error {
	addLogEntry("SetRunning", s.name, string(candidate))
	return nil
}

var testCfg map[string]string

func addTestCfg(comp, config string) {
	testCfg[comp] = config
}

func clearTestCfg() {
	testCfg = make(map[string]string, 0)
}

type logEntry struct {
	fn     string
	params []string
}

var testLog = make([]logEntry, 0)

func clearTestLog() {
	testLog = nil
}

func newLogEntry(fn string, params ...string) logEntry {
	return logEntry{fn: fn, params: params}
}

func addLogEntry(fn string, params ...string) {
	testLog = append(testLog, newLogEntry(fn, params...))
}

func dumpLog(t *testing.T) {
	t.Logf("--- START TEST LOG ---\n")
	for _, entry := range testLog {
		t.Logf("%s:\n", entry.fn)
		for _, param := range entry.params {
			t.Logf("\t%s\n", param)
		}
	}
	t.Logf("---  END TEST LOG  ---\n")
}

// Checks entire log, so anything unexpected in log will cause error.
func checkLogEntries(t *testing.T, entries ...logEntry) {
	if len(entries) != len(testLog) {
		t.Fatalf("\nExp: %d entries\nGot %d\n", len(entries), len(testLog))
		return
	}

	for ix, entry := range entries {
		if entry.fn != testLog[ix].fn {
			dumpLog(t)
			t.Fatalf("\nExp fn: %s\nGot fn: %s\n", entry.fn, testLog[ix].fn)
			return
		}
		for iy, param := range entry.params {
			if param != testLog[ix].params[iy] {
				dumpLog(t)
				t.Fatalf("\nExp param: %s\nGot param: %s\n",
					param, testLog[ix].params[iy])
				return
			}
		}
	}
}

type compConfigTest struct {
	name       string
	config     []string
	logEntries []logEntry
}

// Checks entire log, so anything unexpected in log will cause error.
func checkTestLogEntries(t *testing.T, test compConfigTest) {
	if len(test.logEntries) != len(testLog) {
		t.Logf("\nTEST: %s\n", test.name)
		t.Logf("\nExp: %d entries\nGot: %d\n",
			len(test.logEntries), len(testLog))
		dumpLog(t)
		t.Fatalf("---\n")
		return
	}

	for ix, entry := range test.logEntries {
		if entry.fn != testLog[ix].fn {
			t.Logf("\nTEST: %s\n", test.name)
			dumpLog(t)
			t.Fatalf("\nExp fn: %s\nGot fn: %s\n", entry.fn, testLog[ix].fn)
			return
		}
		for iy, param := range entry.params {
			if param != testLog[ix].params[iy] {
				t.Logf("\nTEST: %s\n", test.name)
				dumpLog(t)
				t.Fatalf("\nExp param: %s\nGot param: %s\n",
					param, testLog[ix].params[iy])
				return
			}
		}
	}
}

func serialiseCfg(cfgTree *data.Node, ms schema.ModelSet) string {

	root := union.NewNode(cfgTree, nil, ms, nil, 0)
	var b union.StringWriter
	root.Serialize(&b, nil, union.IncludeDefaults)
	return b.String()
}
