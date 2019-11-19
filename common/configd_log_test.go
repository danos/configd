// Copyright (c) 2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package common_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/danos/configd/common"
)

func checkLoggingState(
	t *testing.T,
	logName, levelName string,
	expStatus bool,
) {
	level, _ := common.MapLevelNameToLevel(levelName)
	logType, _ := common.MapLogNameToType(logName)
	actStatus := common.LoggingIsEnabledAtLevel(level, logType)

	if actStatus != expStatus {
		t.Fatalf("Log State (%s / %s):\nExp: %t\nGot: %t\n",
			logName, levelName, expStatus, actStatus)
	}
}

func checkDebugDefaults(t *testing.T, status string) {
	checkDebugStatus(t, status,
		common.LevelError, common.LevelNone)
}

func checkDebugStatus(
	t *testing.T,
	msg string,
	commitLevel common.LogLevel,
	stateLevel common.LogLevel,
) {
	checkDebugStatusForType(t, msg, commitLevel, common.TypeCommit)
	checkDebugStatusForType(t, msg, stateLevel, common.TypeState)
}

func checkDebugStatusForType(
	t *testing.T,
	msg string,
	level common.LogLevel,
	logType common.LogType,
) {
	if !common.LoggingIsEnabledAtLevel(level, logType) {
		t.Logf("Log settings:\n%s\n", msg)
		t.Fatalf("Validate logging should be at least '%s'",
			common.MapLogLevelToName(level))
	}
	expStatus := fmt.Sprintf("%-8s\t%s",
		common.MapLogTypeToName(logType),
		common.MapLogLevelToName(level))
	if !strings.Contains(msg, expStatus) {
		t.Fatalf("Unexpected status reported:\nExp:\n%s\n\nGot:\n%s\n",
			expStatus, msg)
	}
}

func TestConfigDebugInvalidName(t *testing.T) {

	msg, err := common.SetConfigDebug("invalidName", "debug")
	if err == nil {
		t.Fatalf("Expected error for invalid debug name")
	}

	checkDebugDefaults(t, msg)
	expErr := "LogType 'invalidName' not recognised"
	if !strings.Contains(err.Error(), expErr) {
		t.Fatalf("Unexpected error content:\nExp: %s\nGot: %s\n",
			expErr, err)
	}
}

func TestConfigDebugInvalidType(t *testing.T) {

	msg, err := common.SetConfigDebug("commit", "invalidLevel")
	if err == nil {
		t.Fatalf("Expected error for invalid debug name")
	}

	checkDebugDefaults(t, msg)
	expErr := "LogLevel 'invalidLevel' not recognised"
	if !strings.Contains(err.Error(), expErr) {
		t.Fatalf("Unexpected error content:\nExp: %s\nGot: %s\n",
			expErr, err)
	}
}

func TestConfigDebugEnable(t *testing.T) {

	msg, err := common.SetConfigDebug("commit", "debug")
	if err != nil {
		t.Fatalf("Unexpected error for valid settings.")
	}

	msg, err = common.SetConfigDebug("state", "error")
	if err != nil {
		t.Fatalf("Unexpected error for valid settings.")
	}

	checkDebugStatus(t, msg,
		common.LevelDebug,
		common.LevelError)

	msg, _ = common.SetConfigDebug("commit", "error")
	msg, _ = common.SetConfigDebug("state", "none")

	checkDebugDefaults(t, msg)

}

func TestConfigDebugErrorEnabledIfDebugSet(t *testing.T) {

	common.SetConfigDebug("commit", "error")
	checkLoggingState(t, "commit", "none", true)
	checkLoggingState(t, "commit", "error", true)
	checkLoggingState(t, "commit", "debug", false)

	common.SetConfigDebug("state", "debug")
	checkLoggingState(t, "state", "none", true)
	checkLoggingState(t, "state", "error", true)
	checkLoggingState(t, "state", "debug", true)
}
