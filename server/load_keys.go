// Copyright (c) 2019-2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/danos/configd/rpc"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/pathutil"

	"golang.org/x/crypto/ssh"
)

func loginSchemaPathForUser(user string) []string {
	return []string{"system", "login", "user", user}
}

func publicKeysSchemaPathForUser(user string) []string {
	return append(loginSchemaPathForUser(user), "authentication", "public-keys")
}

type sshPublicKey struct {
	key     ssh.PublicKey
	Comment string
	Options []string
}

func (k *sshPublicKey) Type() string {
	return k.key.Type()
}

func (k *sshPublicKey) Base64Key() string {
	key := ssh.MarshalAuthorizedKey(k.key)
	key = bytes.TrimPrefix(key, []byte(k.Type()+" "))
	return strings.TrimRight(string(key), "\n")
}

func (k *sshPublicKey) OptionsStr() string {
	return strings.Join(k.Options, ",")
}

func (k *sshPublicKey) ConfigurationCommands(user string) []string {
	out := make([]string, 0)
	base := append(publicKeysSchemaPathForUser(user), k.Comment)

	out = append(out, pathutil.Pathstr(append(base, "type", k.Type())),
		pathutil.Pathstr(append(base, "key", k.Base64Key())))

	opts := k.OptionsStr()
	if opts != "" {
		out = append(out, pathutil.Pathstr(append(base, "options", opts)))
	}
	return out
}

type keysParserCallback func(key *sshPublicKey) error

// Wrapper around ssh.ParseAuthorizedKey() which parses authorized_keys data
// See sshd(8) AUTHORIZED_KEYS FILE FORMAT
func loadKeysParse(reader io.Reader, callback keysParserCallback) error {
	lineNum := 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		lineNum += 1

		// Skip blank or commented lines since ssh.ParseAuthorizedKeys()
		// returns an error for those
		if len(line) == 0 || bytes.HasPrefix(line, []byte("#")) {
			continue
		}

		var err error
		pubKey := &sshPublicKey{}
		pubKey.key, pubKey.Comment, pubKey.Options, _, err = ssh.ParseAuthorizedKey(line)
		if err != nil {
			return fmt.Errorf("On line %v: %v", lineNum, err)
		}

		if err = callback(pubKey); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (d *Disp) setPublicKeyForUser(sid, user string, key *sshPublicKey) error {
	for _, cmd := range key.ConfigurationCommands(user) {
		normalizedCmd, err := d.normalizePath(pathutil.Makepath(cmd))
		if err != nil {
			return err
		}
		if _, err := d.setInternal(sid, normalizedCmd); err != nil {
			return err
		}
	}
	return nil
}

func (d *Disp) userIsConfigured(sid, user string) error {
	userPath := loginSchemaPathForUser(user)
	userExists, err := d.Exists(rpc.AUTO, sid, pathutil.Pathstr(userPath))
	if err != nil {
		return err
	}
	if !userExists {
		operr := mgmterror.NewOperationFailedApplicationError()
		operr.Message = "User " + user + " does not exist in the configuration"
		return operr
	}
	return nil
}

func (d *Disp) loadKeyCommandArgs(user, source, routingInstance string) *commandArgs {
	args := []string{user}
	if routingInstance != "" {
		args = append(args, "routing-instance", routingInstance)
	}

	return d.newCommandArgsForAaa("loadkey", append(args, source), nil)
}

func (d *Disp) loadKeysIsSupported() bool {
	// The LoadKeys RPC functionality is tightly tied to a particular schema
	// so check for the base path provided by that schema.
	// This doesn't check every path LoadKeys requires but it's sufficient for
	// disabling the RPC on systems without the Vyatta data models.
	supported, _ := d.TmplValidatePath(pathutil.Pathstr(publicKeysSchemaPathForUser("@@")))
	return supported
}

func (d *Disp) loadKeysInternal(
	sid, user, source, routingInstance string, local bool, args *commandArgs,
) (string, error) {
	if err := d.userIsConfigured(sid, user); err != nil {
		return "", err
	}

	var reader io.ReadCloser
	if local {
		source = d.parseLocalPath(source)
		if err := d.validLocalConfigPath(source); err != nil {
			return "", err
		}
		reader = d.newUserFileReader(source)
	} else {
		reader = d.newUserRemoteFileReader(source, routingInstance)
	}
	defer reader.Close()

	keySetFn := func(key *sshPublicKey) error {
		return d.setPublicKeyForUser(sid, user, key)
	}

	err := loadKeysParse(reader, keySetFn)
	if err != nil {
		operr := mgmterror.NewOperationFailedApplicationError()
		operr.Message = "Loading key file failed\n" + err.Error()
		return "", operr
	}

	if changed, _ := d.SessionChanged(sid); !changed {
		return "No keys were loaded from '" + source + "'", err
	}

	d.ConfirmSilent(sid)
	out, err := d.commitInternal(sid, strings.Join(args.cmd, " "), false, 0 /* no timeout */, false)
	if err == nil {
		if out != "" {
			out = strings.TrimRight(out, "\n") + "\n\n"
		}
		out += "Loaded keys from '" + source + "'"
	}
	return out, err
}

// LoadKeys RPC
// This provides the implementation for the "loadkey" cfgcli command
func (d *Disp) LoadKeys(sid, user, source, routingInstance string) (string, error) {
	if !d.loadKeysIsSupported() {
		return "", mgmterror.NewOperationNotSupportedApplicationError()
	}

	local, redactedSource, err := parseMgmtURI(source)
	if err != nil {
		return "", err
	}

	args := d.loadKeyCommandArgs(user, redactedSource, routingInstance)
	if !d.authCommand(args) {
		return "", mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapStrErr(args, func() (interface{}, error) {
		return d.loadKeysInternal(sid, user, source, routingInstance, local, args)
	})
}
