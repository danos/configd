// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

type SpawnCommandAsCallerFn func(*Disp, []string) (string, error)

func SetSpawnCommandAsCallerFn(fn SpawnCommandAsCallerFn) {
	spawnCommandAsCallerFn = fn
}

func ResetSpawnCommandAsCallerFn() {
	spawnCommandAsCallerFn = spawnCommandAsCaller
}

func SetConfigDir(dir string) {
	configDir = dir
}

func ResetConfigDir() {
	configDir = "/config"
}
