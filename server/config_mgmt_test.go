// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"os"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils"
	"github.com/danos/configd/server"
	spawn "os/exec"
)

func TestLoadFromSuccessCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	testConfig := testutils.Root(
		testutils.Leaf("testint", "8"))

	file, err := dispTestLoadOrMergeWriteConfigToFile(testConfig)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(file)

	ok, err := d.LoadFrom(testSID, file, "")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !ok {
		t.Fatalf("LoadFrom failed with no error returned")
	}

	assertCommandAaaNoSecrets(t, a, []string{"load", file})
}

func TestLoadFromRoutingInstanceCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.LoadFrom(testSID, "scp://bar:baz@localhost/conf", "red")

	assertCommandAaaNoSecrets(t, a,
		[]string{"load", "routing-instance", "red", "scp://bar:**@localhost/conf"})
}

func testSpawnCommandAsCaller(_ *server.Disp, cmd []string) (string, error) {
	out, err := spawn.Command(cmd[0], cmd[1:]...).CombinedOutput()
	return string(out), err
}

func TestSaveToCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	server.SetSpawnCommandAsCallerFn(testSpawnCommandAsCaller)
	defer server.ResetSpawnCommandAsCallerFn()

	server.SetConfigDir(os.TempDir())
	defer server.ResetConfigDir()

	dispTestSetupSession(t, d, testSID)

	file := os.TempDir() + "/saveto"

	// Right now we're mostly just checking command authorization
	success, err := d.SaveTo(file, "")
	if err != nil {
		t.Fatalf("Unexpected err: %s", err)
	}
	if !success {
		t.Fatalf("SaveTo failed unexpectedly")
	}

	// Cleanup the saved config
	defer os.Remove(file)

	assertCommandAaaNoSecrets(t, a, []string{"save", file})
}

func TestSaveToRoutingInstanceCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		loadOrMergeSchema, initConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.SaveTo("scp://bar:baz@localhost/conf", "red")

	assertCommandAaaNoSecrets(t, a,
		[]string{"save", "routing-instance", "red", "scp://bar:**@localhost/conf"})
}
