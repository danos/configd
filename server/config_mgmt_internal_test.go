// Copyright (c) 2019-2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

var productionTmpDir = tmpDir

type SpawnCommandAsCallerFn func(*Disp, []string) (string, error)

func SetSpawnCommandAsCallerFn(fn SpawnCommandAsCallerFn) {
	spawnCommandAsCallerFn = fn
}

func ResetSpawnCommandAsCallerFn() {
	spawnCommandAsCallerFn = spawnCommandAsCaller
}

func SetTmpDir(dir string) {
	tmpDir = dir
}

func GetProductionTmpDir() string {
	return productionTmpDir
}
