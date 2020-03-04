// Copyright (c) 2019-2020, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package common

import (
	"fmt"
	"strconv"
	"strings"
)

type LogLevel int

const (
	// Current log levels used in configd are Error (Elog) and Debug (Dlog).
	// Commit 'error' level logs (which might be better described as info)
	// are always on.
	//
	// Order must be least verbose (none) to most verbose (debug) so we can
	// check what is enabled by simple numeric comparison.
	LevelNone LogLevel = iota
	LevelError
	LevelDebug
	LevelLast // Keep at end for sizing slices etc.
)

func MapLevelNameToLevel(level string) (LogLevel, error) {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug, nil
	case "error":
		return LevelError, nil
	case "none":
		return LevelNone, nil
	}
	return LevelNone, fmt.Errorf(
		"LogLevel '%s' not recognised. Use <none|error|debug>.", level)
}

func MapLogLevelToName(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelError:
		return "error"
	case LevelNone:
		return "none"
	default:
		return "none"
	}
}

type LogType int

const (
	// Any changes need to be reflected in cfgDebugSettings
	TypeNone LogType = iota
	TypeCommit
	TypeState
	TypeMust
	TypeLast // Keep at end so we can size slices
)

type ValueType int

const (
	StringVal ValueType = iota
	IntVal
)

// cfgDebugSetting
//
// StringVal: use LogLevel and ignore value
// Intval:    use value, disabled if LogLevel is LevelNone
type cfgDebugSetting struct {
	valType ValueType
	level   LogLevel
	value   int
}

var cfgDebugSettings = map[LogType]cfgDebugSetting{
	TypeNone:   {valType: StringVal, level: LevelNone, value: 0},
	TypeCommit: {valType: StringVal, level: LevelError, value: 0},
	TypeState:  {valType: StringVal, level: LevelNone, value: 0},
	TypeMust:   {valType: IntVal, level: LevelNone, value: 0},
}

func MapLogNameToType(name string) (LogType, error) {
	switch strings.ToLower(name) {
	case "commit":
		return TypeCommit, nil
	case "state":
		return TypeState, nil
	case "must":
		return TypeMust, nil
	}
	return TypeNone, fmt.Errorf(
		"LogType '%s' not recognised. Use <validate|commit|state>.", name)
}

func MapLogTypeToName(logType LogType) string {
	switch logType {
	case TypeCommit:
		return "commit"
	case TypeState:
		return "state"
	case TypeMust:
		return "must"
	default:
		return "none"
	}
}

func LoggingIsEnabledAtLevel(level LogLevel, logType LogType) bool {
	if logType >= TypeLast || level >= LevelLast {
		return false
	}
	return cfgDebugSettings[logType].level >= level
}

func LoggingValueAndStatus(logType LogType) (int, bool) {
	if logType >= TypeLast {
		return 0, false
	}
	if cfgDebugSettings[logType].valType != IntVal {
		return 0, false
	}
	enabled := (cfgDebugSettings[logType].level != LevelNone)
	return cfgDebugSettings[logType].value, enabled
}

func CurrentLogStatus() string {
	var retStr = "\nCurrent Debug Status:\n\n"
	for logType, dbgSetting := range cfgDebugSettings {
		if LogType(logType) == TypeNone {
			continue
		}
		switch dbgSetting.valType {
		case StringVal:
			retStr += fmt.Sprintf("%-8s\t%s\n",
				MapLogTypeToName(LogType(logType)),
				MapLogLevelToName(dbgSetting.level))
		case IntVal:
			retStr += fmt.Sprintf("%-8s\t%d\n",
				MapLogTypeToName(LogType(logType)),
				dbgSetting.value)
		default:
			// Ignore.
		}
	}
	retStr += "\nValid levels: none, error, debug\n"

	return retStr
}

func SetConfigDebug(logName, levelOrValue string) (string, error) {
	// Allows us to let users know what valid options are w/o encoding them
	// explicitly in API, and also to get current status.
	if logName == "" && levelOrValue == "" {
		return CurrentLogStatus(), nil
	}

	// If we return an error over the client / dispatcher API, the returned
	// string appears to get ignored, so we add currentLogStatus() output to
	// the error string.
	logType, typeErr := MapLogNameToType(logName)
	if typeErr != nil {
		return CurrentLogStatus(),
			fmt.Errorf("%s\n%s", typeErr, CurrentLogStatus())
	}
	switch cfgDebugSettings[logType].valType {
	case StringVal:
		logLevel, levelErr := MapLevelNameToLevel(levelOrValue)
		if levelErr != nil {
			return CurrentLogStatus(),
				fmt.Errorf("%s\n%s", levelErr, CurrentLogStatus())
		}

		newCfgSetting := cfgDebugSetting{valType: StringVal, level: logLevel}
		cfgDebugSettings[logType] = newCfgSetting

	case IntVal:
		// If we can parse value as number, all good
		val, err := strconv.Atoi(levelOrValue)
		if val < 0 || err != nil {
			return CurrentLogStatus(),
				fmt.Errorf("Use +ve integer value for %s (0 = disable)\n%s",
					logName, CurrentLogStatus())
		}
		newLevel := LevelDebug
		if val == 0 {
			newLevel = LevelNone
		}
		newCfgSetting := cfgDebugSetting{
			valType: IntVal, level: newLevel, value: val}
		cfgDebugSettings[logType] = newCfgSetting
	}

	return CurrentLogStatus(), nil
}
