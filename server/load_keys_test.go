// Copyright (c) 2019-2020, AT&T Intellectual Property Inc. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/common"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/server"
	"github.com/danos/utils/pathutil"
)

const loadKeysSchema = `
	container system {
		container login {
			list user {
				key name;
				leaf name {
					type string;
				}
				container authentication {
					list public-keys {
						key name;
						leaf name {
							type string;
						}
						leaf key {
							type string;
						}
						leaf type {
							type string;
						}
					}
				}
			}
		}
	}`

const (
	pubKeyType   = "ssh-rsa"
	pubBase64Key = "AAAAB3NzaC1yc2EAAAADAQABAAABAQCw9Sgl10ho7A4+c7QV6ofCpjfhPqWHxj2i7idX7dIJgZzPo1SBNJqS3N56r5HxbFcVIdosiVg2bymHLmHDG2t1KLejvDB1uyowr3UeQ4yjzRbLaxiUAeuQvKSHxYGCLDHG+GVmXIdESE5ZD3wptxd9Hw5E9YTokjC9uPyx3CkF24bXqisZYpvkeKveCLiYnienQASpI/UN0bR9TtTa3s/TsCvOhUCsd5ZhGIM6eISxG4tJ467UYT4fNBqYInDY92mgHWiN63dIyMBVo1OJj369qgsASZJBeb0xGOEjNiAJmEEepPhEhFB0TA2uHkPnkD783ZS86QY0l20ZpTOidWwz"
	pubKeyName   = "rsa-key"
	pubKey       = pubKeyType + " " + pubBase64Key + " " + pubKeyName
)

var vyattaUserCfgPath = []string{"system", "login", "user", "vyatta"}

func generateKeyFile(keys []string) (string, error) {
	file, err := ioutil.TempFile("", "loadkeys")
	if err != nil {
		return "", err
	}

	for _, key := range keys {
		if _, err = file.WriteString(key); err != nil {
			os.Remove(file.Name())
			return "", err
		}
	}

	if err = file.Close(); err != nil {
		os.Remove(file.Name())
		return "", err
	}

	return file.Name(), nil
}

func TestLoadKeysNotSupported(t *testing.T) {
	d := newTestDispatcher(
		t, auth.TestAutherAllowAll(), defaultSchema, emptyConfig)

	dispTestSetupSession(t, d, testSID)

	feats, err := d.GetConfigSystemFeatures()
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	if _, exists := feats[common.LoadKeysFeature]; exists {
		t.Fatalf("%v was unexpectedly reported as supported", common.LoadKeysFeature)
	}

	_, err = d.LoadKeys(testSID, "vyatta", "/config/keys", "")
	if err == nil {
		t.Fatalf("Unexpected LoadKeys success")
	}

	expErrs := assert.NewExpectedMessages("not supported")
	expErrs.ContainedIn(t, err.Error())
}

func TestLoadKeysSupported(t *testing.T) {
	d := newTestDispatcher(
		t, auth.TestAutherAllowAll(), loadKeysSchema, emptyConfig)

	dispTestSetupSession(t, d, testSID)

	feats, err := d.GetConfigSystemFeatures()
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	if _, exists := feats[common.LoadKeysFeature]; !exists {
		t.Fatalf("%v was unexpectedly not reported as supported", common.LoadKeysFeature)
	}
}

func loadKeysTest(t *testing.T) (auth.TestAuther, *server.Disp, string) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a, loadKeysSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	server.SetCallerCmdSetPrivs(false)
	defer server.SetCallerCmdSetPrivs(server.GetProductionCallerCmdSetPrivs())

	dispTestSetupSession(t, d, testSID)

	// Configure a user to load keys for
	dispTestSet(t, d, testSID, pathutil.Pathstr(vyattaUserCfgPath))
	clearAllCmdRequestsAndUserAuditLogs(a) /* Set will have generated requests */

	keyFile, err := generateKeyFile([]string{pubKey})
	if err != nil {
		t.Fatalf("Error generating key file: %s", err)
	}
	defer os.Remove(keyFile)

	out, err := d.LoadKeys(testSID, "vyatta", keyFile, "")
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}

	expErrs := assert.NewExpectedMessages("Loaded keys from '" + keyFile + "'")
	expErrs.ContainedIn(t, out)
	return a, d, keyFile
}

func TestLoadKeys(t *testing.T) {
	_, d, _ := loadKeysTest(t)

	// Check the key was configured
	baseCheckPath := append(vyattaUserCfgPath, "authentication", "public-keys", "rsa-key")
	dispTestExists(t, d, rpc.RUNNING, testSID, pathutil.Pathstr(baseCheckPath), true)

	dispTestExists(t, d, rpc.RUNNING, testSID,
		pathutil.Pathstr(append(baseCheckPath, "key", pubBase64Key)), true)

	dispTestExists(t, d, rpc.RUNNING, testSID,
		pathutil.Pathstr(append(baseCheckPath, "type", pubKeyType)), true)
}

func TestLoadKeysAaa(t *testing.T) {
	a, _, keyFile := loadKeysTest(t)
	assertCommandAaaNoSecrets(t, a, []string{"loadkey", "vyatta", keyFile})
}

func TestLoadKeysRoutingInstanceCommandAuthz(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a, loadKeysSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)

	// Right now we're just checking command authorization
	_, _ = d.LoadKeys(testSID, "vyatta", "scp://foo:bar@localhost/keys", "blue")

	assertCommandAaaNoSecrets(t, a,
		[]string{"loadkey", "vyatta", "routing-instance",
			"blue", "scp://foo:**@localhost/keys"})
}
