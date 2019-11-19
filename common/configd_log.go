// Copyright (c) 2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package common

import (
	"fmt"
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
	TypeLast // Keep at end so we can size slices
)

var cfgDebugSettings = []LogLevel{
	LevelNone,  // TypeNone
	LevelError, // TypeCommit
	LevelNone,  // TypeState
}

func MapLogNameToType(name string) (LogType, error) {
	switch strings.ToLower(name) {
	case "commit":
		return TypeCommit, nil
	case "state":
		return TypeState, nil
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
	default:
		return "none"
	}
}

func LoggingIsEnabledAtLevel(level LogLevel, logType LogType) bool {
	if logType >= TypeLast || level >= LevelLast {
		return false
	}
	return cfgDebugSettings[logType] >= level
}

func CurrentLogStatus() string {
	var retStr = "\nCurrent Debug Status:\n\n"
	for logType, level := range cfgDebugSettings {
		if LogType(logType) == TypeNone {
			continue
		}
		retStr += fmt.Sprintf("%-8s\t%s\n",
			MapLogTypeToName(LogType(logType)),
			MapLogLevelToName(level))
	}
	retStr += "\nValid levels: none, error, debug\n"

	return retStr
}

func SetConfigDebug(logName, level string) (string, error) {
	// Allows us to let users know what valid options are w/o encoding them
	// explicitly in API, and also to get current status.
	if logName == "" && level == "" {
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
	logLevel, levelErr := MapLevelNameToLevel(level)
	if levelErr != nil {
		return CurrentLogStatus(),
			fmt.Errorf("%s\n%s", levelErr, CurrentLogStatus())
	}

	cfgDebugSettings[logType] = logLevel
	return CurrentLogStatus(), nil
}
