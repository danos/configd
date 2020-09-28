// Copyright (c) 2019-2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/danos/configd/rpc"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/pathutil"
	spawn "os/exec"
)

const (
	transferUrlBin = "/opt/vyatta/sbin/vyatta-transfer-url"
)

// Globals which can be manipulated by UTs (see config_mgmt_internal_test.go)
var configDir = "/config"
var tmpDir = "/var/tmp/configd"
var callerCmdSetPrivs = true

func userSandboxPath(user string) string {
	return "/run/cli-sandbox/" + user
}

func (d *Disp) callerIsSandboxed() bool {
	_, err := os.Stat(userSandboxPath(d.ctx.User))
	return !os.IsNotExist(err)
}

// Parse/validate source/destination URIs provided to management functions
// such as load, save etc.
// True is returned if uri is a local path, otherwise False
// The uri is also returned with any password redacted
// An error is returned in case of an invalid URI
func parseMgmtURI(uri string) (bool, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return false, "", err
	}
	if !u.IsAbs() {
		/* local path */
		return true, uri, nil
	}

	if u.Scheme != "tftp" &&
		u.Scheme != "ftp" &&
		u.Scheme != "http" &&
		u.Scheme != "scp" {
		operr := mgmterror.NewOperationFailedApplicationError()
		operr.Message = "Invalid protocol [" + u.Scheme + "]"
		return false, "", operr
	}

	// Redact password if one is present
	// Use string manipulation rather than updating u.User and returning
	// u.String() to avoid the password replacement being escaped.
	// We only care about the redacted version for doing command authorization
	// so it should not be escaped.
	if pass, passSet := u.User.Password(); passSet {
		match := u.Scheme + "://" + u.User.Username() + ":"
		uri = strings.Replace(uri, match+pass, match+"**", 1)
	}

	return false, uri, nil
}

func getCurrentConfigVersion() string {
	out, err := spawn.Command("/opt/vyatta/sbin/vyatta_current_conf_ver.pl").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func (d *Disp) isVyattaConfigFile(file string) bool {
	cfg, err := d.readCfgFile(file, true, false)
	if err != nil {
		return false
	}

	var line string
	cfgBuf := bytes.NewBufferString(cfg)
	for err = nil; err != io.EOF; line, err = cfgBuf.ReadString('\n') {
		if strings.Contains(line, "=== vyatta-config") {
			return true
		}
	}
	return false
}

func (d *Disp) validLocalConfigPath(path string) error {
	// For isolated users there are two valid local paths, /config
	// and the user's home
	if d.callerIsSandboxed() &&
		!strings.HasPrefix(path, d.ctx.UserHome+"/") &&
		!strings.HasPrefix(path, configDir+"/") {
		operr := mgmterror.NewOperationFailedApplicationError()
		operr.Message = path + " is not a valid path"
		operr.Message += "\nValid paths are beneath " + configDir + " or " + d.ctx.UserHome
		return operr
	}
	return nil
}

func (d *Disp) validLocalSaveToDest(dest string) error {
	if err := d.validLocalConfigPath(dest); err != nil {
		return err
	}

	info, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.IsDir() {
		operr := mgmterror.NewOperationFailedApplicationError()
		operr.Message = dest + " is a directory"
		return operr
	}

	if !d.isVyattaConfigFile(dest) {
		operr := mgmterror.NewOperationFailedApplicationError()
		operr.Message = dest + " exists and is not a Vyatta configuration"
		return operr
	}

	return nil
}

func (d *Disp) parseLocalPath(path string) string {
	// Clean the destination path to prevent using parent specifiers (..)
	// to manipulate the save location
	path = filepath.Clean(path)

	// A relative destination is placed beneath /config
	if !strings.HasPrefix(path, "/") {
		path = configDir + "/" + path
	}

	return path
}

func (d *Disp) writeRunningConfigToFile(file *os.File) error {
	cfg, err := d.show(rpc.RUNNING, "", pathutil.Makepath(""), false, false)
	if err != nil {
		return err
	}
	_, err = file.WriteString(cfg + getCurrentConfigVersion())
	if err != nil {
		return err
	}
	file.Sync()
	return nil
}

func (d *Disp) writeTempRunningConfigFile() (string, error) {
	tmpFile, err := ioutil.TempFile(tmpDir, ".save.")
	if err != nil {
		return "", err
	}

	err = d.writeRunningConfigToFile(tmpFile)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func (d *Disp) spawnCommandAsCaller(cmd []string) (string, error) {
	// Drop to calling user privileges to try prevent the user from accessing
	// or doing things they shouldn't be able to.
	// NB. If user isolation is enabled this is *not* run in the context of the
	//     calling user's container but rather the main container.
	if !d.ctx.Configd && callerCmdSetPrivs {
		cmd = append([]string{"/opt/vyatta/sbin/lu", "--setprivs", "--user=" + d.ctx.User},
			cmd...)
	}

	out, err := spawn.Command(cmd[0], cmd[1:]...).CombinedOutput()

	// If there was output replace error with something a bit more meaningful
	if err != nil && len(out) > 0 {
		operr := mgmterror.NewOperationFailedApplicationError()
		operr.Message = strings.Trim(string(out), "\n")
		return operr.Message, operr
	}

	return string(out), err
}

func (d *Disp) copyFile(from, to string) error {
	// Don't preserve existing permissions on destination file
	_, err := d.spawnCommandAsCaller([]string{"cp", "-T", from, to})
	return err
}

func (d *Disp) downloadFile(source, file, routingInstance string) error {
	cmd := []string{transferUrlBin, "--outfile=" + file}
	if routingInstance != "" {
		cmd = append(cmd, "--ri="+routingInstance)
	}

	_, err := d.spawnCommandAsCaller(append(cmd, source))
	return err
}

func (d *Disp) downloadTempFile(source, dir, prefix, routingInstance string) (string, error) {
	var file string

	// Create a temporary file which we can overwrite
	t, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		return "", err
	}
	file = t.Name()

	// Set owner of the temp file to the requesting user
	// This is necessary since the download operation happens as the
	// requesting user, and we need to overwrite the existing temporary file.
	if !d.ctx.Configd {
		err = os.Chown(file, int(d.ctx.Uid), -1)
		if err != nil {
			os.Remove(file)
			return "", err
		}
	}

	err = d.downloadFile(source, file, routingInstance)
	if err != nil {
		os.Remove(file)
		return "", err
	}
	return file, nil
}

func (d *Disp) uploadFile(file, dest, routingInstance string) error {
	cmd := []string{transferUrlBin, "--infile=" + file}
	if routingInstance != "" {
		cmd = append(cmd, "--ri="+routingInstance)
	}

	_, err := d.spawnCommandAsCaller(append(cmd, dest))
	return err
}

func (d *Disp) cfgMgmtCommandArgs(cmd, uri, routingInstance string) *commandArgs {
	var args []string
	if routingInstance != "" {
		args = []string{"routing-instance", routingInstance, uri}
	} else {
		args = []string{uri}
	}

	return d.newCommandArgsForAaa(cmd, args, nil)
}

func (d *Disp) loadFromInternal(
	sid, source, routingInstance string, local bool,
) (bool, error) {
	var cfgFile string
	if local {
		cfgFile = d.parseLocalPath(source)
		if err := d.validLocalConfigPath(cfgFile); err != nil {
			return false, err
		}
	} else {
		cfgFile, err := d.downloadTempFile(source, tmpDir, ".load.", routingInstance)
		if err != nil {
			return false, err
		}
		defer os.Remove(cfgFile)
	}

	return d.loadReportWarningsReader(sid, cfgFile, nil)
}

func (d *Disp) LoadFrom(sid, source, routingInstance string) (bool, error) {
	local, redactedSource, err := parseMgmtURI(source)
	if err != nil {
		return false, err
	}

	args := d.cfgMgmtCommandArgs("load", redactedSource, routingInstance)
	if !d.authCommand(args) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	if !d.ctx.Configd {
		d.ctx.Wlog.Println("Load config [" + redactedSource + "] by " + d.ctx.User)
	}

	return d.accountCmdWrapBoolErr(args, func() (interface{}, error) {
		return d.loadFromInternal(sid, source, routingInstance, local)
	})
}

func (d *Disp) saveToInternal(dest, routingInstance string, local bool) (bool, error) {
	if local {
		dest = d.parseLocalPath(dest)
		if err := d.validLocalSaveToDest(dest); err != nil {
			return false, err
		}
	}

	tmpFile, err := d.writeTempRunningConfigFile()
	if err != nil {
		return false, err
	}
	defer os.Remove(tmpFile)

	// Set owner of the saved config to the requesting user
	// This is necessary since future operations on the file will be performed
	// as the requesting user
	if !d.ctx.Configd {
		err = os.Chown(tmpFile, int(d.ctx.Uid), -1)
		if err != nil {
			return false, err
		}
	}

	if local {
		err = d.copyFile(tmpFile, dest)
	} else {
		err = d.uploadFile(tmpFile, dest, routingInstance)
	}

	return err == nil, err
}

func (d *Disp) SaveTo(dest, routingInstance string) (bool, error) {
	local, redactedDest, err := parseMgmtURI(dest)
	if err != nil {
		return false, err
	}

	args := d.cfgMgmtCommandArgs("save", redactedDest, routingInstance)
	if !d.authCommand(args) {
		return false, mgmterror.NewAccessDeniedApplicationError()
	}

	return d.accountCmdWrapBoolErr(args, func() (interface{}, error) {
		return d.saveToInternal(dest, routingInstance, local)
	})
}
