// Copyright (c) 2019-2020, AT&T Intellectual Property.
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

const (
	COMMIT = "commit"
	STATE  = "state"
	MUST   = "must"
	NONE   = "none"
	ERROR  = "error"
	DEBUG  = "debug"
)

func checkLoggingValue(
	t *testing.T,
	logName string,
	expVal int,
	expStatus bool,
) {
	logType, _ := common.MapLogNameToType(logName)

	actVal, actStatus := common.LoggingValueAndStatus(logType)

	if actStatus != expStatus {
		t.Fatalf("Status: exp %v, got %v", expStatus, actStatus)
	}
	if actVal != expVal {
		t.Fatalf("Value: exp %v, got %v", expVal, actVal)
	}
}

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
		common.LevelError, common.LevelNone, 0)
}

func checkDebugStatus(
	t *testing.T,
	msg string,
	commitLevel common.LogLevel,
	stateLevel common.LogLevel,
	mustThreshold int,
) {
	checkDebugStatusForType(t, msg, commitLevel, common.TypeCommit)
	checkDebugStatusForType(t, msg, stateLevel, common.TypeState)
	checkDebugStatusForIntType(t, msg, mustThreshold, common.TypeMust)
}

func checkDebugStatusForType(
	t *testing.T,
	msg string,
	level common.LogLevel,
	logType common.LogType,
) {
	if !common.LoggingIsEnabledAtLevel(level, logType) {
		t.Logf("Log settings:\n%s\n", msg)
		t.Fatalf("'%s' logging should be at least '%s'",
			common.MapLogTypeToName(logType),
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

func checkDebugStatusForIntType(
	t *testing.T,
	msg string,
	threshold int,
	logType common.LogType,
) {
	expStatus := fmt.Sprintf("%-8s\t%d",
		common.MapLogTypeToName(logType),
		threshold)
	if !strings.Contains(msg, expStatus) {
		t.Fatalf("Unexpected status reported:\nExp:\n%s\n\nGot:\n%s\n",
			expStatus, msg)
	}
}

func restoreDefaults() {
	common.SetConfigDebug(COMMIT, ERROR)
	common.SetConfigDebug(STATE, NONE)
	common.SetConfigDebug(MUST, "0")
}

func TestConfigDebugInvalidName(t *testing.T) {

	msg, err := common.SetConfigDebug("invalidName", DEBUG)
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

	msg, err := common.SetConfigDebug(COMMIT, "invalidLevel")
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

	msg, err := common.SetConfigDebug(COMMIT, DEBUG)
	if err != nil {
		t.Fatalf("Unexpected error for valid settings.")
	}

	msg, err = common.SetConfigDebug(STATE, ERROR)
	if err != nil {
		t.Fatalf("Unexpected error for valid settings.")
	}

	checkDebugStatus(t, msg,
		common.LevelDebug,
		common.LevelError,
		0)

	msg, _ = common.SetConfigDebug(COMMIT, ERROR)
	msg, _ = common.SetConfigDebug(STATE, NONE)

	checkDebugDefaults(t, msg)

}

func TestConfigDebugErrorEnabledIfDebugSet(t *testing.T) {

	common.SetConfigDebug(COMMIT, ERROR)
	checkLoggingState(t, COMMIT, NONE, true)
	checkLoggingState(t, COMMIT, ERROR, true)
	checkLoggingState(t, COMMIT, DEBUG, false)

	common.SetConfigDebug(STATE, DEBUG)
	checkLoggingState(t, STATE, NONE, true)
	checkLoggingState(t, STATE, ERROR, true)
	checkLoggingState(t, STATE, DEBUG, true)

	restoreDefaults()
}

func TestConfigDebugIntValueRejectedForStringType(t *testing.T) {
	_, err := common.SetConfigDebug(COMMIT, "66")
	if err == nil {
		t.Fatalf("Should have failed")
	}
	if !strings.Contains(err.Error(), "LogLevel '66' not recognised") {
		t.Fatalf("Unexpected error: %s\n", err)
	}

	restoreDefaults()
}

func TestConfigDebugIntValueAcceptedForIntType(t *testing.T) {
	_, err := common.SetConfigDebug(MUST, "66")
	if err != nil {
		t.Fatalf("Should have passed: %s", err)
	}

	restoreDefaults()
}

func TestConfigDebugStringValueRejectedForIntType(t *testing.T) {
	_, err := common.SetConfigDebug(MUST, "not-a-number")
	if err == nil {
		t.Fatalf("Should have failed.")
	}

	restoreDefaults()
}

func TestConfigDebugIntValueRetrievedCorrectly(t *testing.T) {
	common.SetConfigDebug(MUST, "666")
	checkLoggingValue(t, MUST, 666, true)

	restoreDefaults()
	checkLoggingValue(t, MUST, 0, false)
}

func TestConfigDebugIntValueDisplayedCorrectly(t *testing.T) {
	out, _ := common.SetConfigDebug(MUST, "123")

	checkDebugStatus(t, out,
		common.LevelError,
		common.LevelNone,
		123)

	restoreDefaults()
}
