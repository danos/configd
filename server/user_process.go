// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"bytes"
	"io"
	"os/exec"
)

// Provides a ReadCloser implementation allowing a local file to be read
// as the user who opened the server connection. This is done by execing
// cat, avoiding the need to use LockOSThread() / UnlockOSThread() and
// adjust the effective UID.
func (d *Disp) newUserFileReader(file string) *userProcessReader {
	return d.newUserProcessReader([]string{"cat", file})
}

// Provides a ReadCloser implementation allowing a remote file to be
// read using vyatta-transfer-url.
func (d *Disp) newUserRemoteFileReader(uri, routingInstance string) *userProcessReader {
	cmd := []string{transferUrlBin}
	if routingInstance != "" {
		cmd = append(cmd, "--ri="+routingInstance)
	}
	return d.newUserProcessReader(append(cmd, uri))
}

// Implements the ReadCloser interface around the stdout stream of a
// process executed as the user who opened the server connection.
type userProcessReader struct {
	d        *Disp
	stdout_r *io.PipeReader
	stdout_w *io.PipeWriter
	stderr   bytes.Buffer
	cmd      *exec.Cmd
}

func (d *Disp) newUserProcessReader(cmd []string) *userProcessReader {
	r := &userProcessReader{d: d}
	r.stdout_r, r.stdout_w = io.Pipe()
	r.cmd = r.d.newCommandAsCaller(cmd)
	r.cmd.Stdout = r.stdout_w
	r.cmd.Stderr = &r.stderr
	return r
}

func (r *userProcessReader) run() error {
	if r.cmd.Process != nil {
		// Already running
		return nil
	}

	err := r.cmd.Start()
	if err != nil {
		return err
	}

	// Wait for process to exit in a separate goroutine, then close the pipe
	// to ensure the reader routine doesn't forever block on a pipe read.
	// If the process terminates with an error, we close the write side of the
	// pipe with a representation of that error (instead of io.EOF). This makes
	// the reader aware that something went wrong.
	go func() {
		err := r.cmd.Wait()
		r.stdout_w.CloseWithError(
			handleCallerCommandError(r.stderr.Bytes(), err))
	}()

	return nil
}

func (r *userProcessReader) Read(buf []byte) (int, error) {
	if err := r.run(); err != nil {
		return 0, err
	}

	return r.stdout_r.Read(buf)
}

func (r *userProcessReader) Close() error {
	if r.cmd.Process != nil {
		r.cmd.Process.Kill()
	}
	r.stdout_r.Close()
	r.stdout_w.Close()
	return nil
}
