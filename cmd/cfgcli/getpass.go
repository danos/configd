/*
The MIT License (MIT)

Copyright (c) 2013-2014 Andrew Dunham

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.

SPDX-License-Identifier: MIT
*/

package main

import (
	"io"
	"syscall"
	"unsafe"
)

const ioctlReadTermios = syscall.TCGETS
const ioctlWriteTermios = syscall.TCSETS

func GetPass(prompt string, prompt_fd, input_fd uintptr) ([]byte, error) {
	// Firstly, print the prompt.
	written := 0
	buf := []byte(prompt)
	for written < len(prompt) {
		n, err := syscall.Write(int(prompt_fd), buf[written:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, io.EOF
		}

		written += n
	}

	// Write a newline after we're done, since it won't be echoed when the
	// user presses 'Enter'.
	defer syscall.Write(int(prompt_fd), []byte("\n"))

	// Get the current state of the terminal
	var oldState syscall.Termios
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL,
		uintptr(input_fd),
		ioctlReadTermios,
		uintptr(unsafe.Pointer(&oldState)),
		0, 0, 0); err != 0 {
		return nil, err
	}

	// Turn off echo and write the new state.
	newState := oldState
	newState.Lflag &^= syscall.ECHO
	newState.Lflag |= syscall.ICANON | syscall.ISIG
	newState.Iflag |= syscall.ICRNL
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL,
		uintptr(input_fd),
		ioctlWriteTermios,
		uintptr(unsafe.Pointer(&newState)),
		0, 0, 0); err != 0 {
		return nil, err
	}

	// Regardless of how we exit, we need to restore the old state.
	defer func() {
		syscall.Syscall6(syscall.SYS_IOCTL,
			uintptr(input_fd),
			ioctlWriteTermios,
			uintptr(unsafe.Pointer(&oldState)),
			0, 0, 0)
	}()

	// Read in increments of 16 bytes.
	var readBuf [16]byte
	var ret []byte
	for {
		n, err := syscall.Read(int(input_fd), readBuf[:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			if len(ret) == 0 {
				return nil, io.EOF
			}
			break
		}

		// Trim the trailing newline.
		if readBuf[n-1] == '\n' {
			n--
		}

		ret = append(ret, readBuf[:n]...)
		if n < len(readBuf) {
			break
		}
	}

	return ret, nil
}
