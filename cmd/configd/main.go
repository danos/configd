// Copyright (c) 2019-2020, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

/*
configd is a daemon that manages run-time configuration based on YANG definition files.

Usage:
	-cpuprofile=<filename>
		Defines a file which to write a cpu profile that can be parsed with go pprof.
		When defined, the daemon will begin recording cpu profile information when it
		receives a SIGUSR1 signal. Then on a subsequent SIGUSR1 it will write the profile
		information to the defined file.

	-logfile=<filename>
		When defined configd will redirect its stdout and stderr to the defined file.

	-pidfile=<filename>
		Sepecify file for the daemon to write pid in (default: /run/configd/configd.pid).

	-runfile=<filename>
		Sepecify file for the daemon to write running configuration into (default:
		/run/configd/running.config).

	-socketfile=<filename>
		When defined configd will write its pid to the defined file (defualt:
		/run/configd/main.sock).

	-yangdir=<dir>
		Directory configd will load YANG files and watch for updates (default:
		/usr/share/configd/yang).

	SIGUSR1
		Issuing SIGUSR1 to the daemon will toggle run-time profiling. Profile data will
		be written to the file specified by the cpuprofile option.

*/
package main

import (
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/activation"
	"github.com/danos/configd"
	"github.com/danos/configd/server"
	"github.com/danos/utils/os/group"
	"github.com/danos/yang/compile"
)

var basepath string = "/run/configd"
var runningprof bool
var cpuproffile os.File
var elog *log.Logger

/* Command line options */
var cpuprofile *string = flag.String("cpuprofile",
	basepath+"/configd.pprof",
	"Write cpu profile to supplied file on SIGUSR1.")

var memprofile = flag.String("memprofile", basepath+"/configd_mem.pprof",
	"Write memory profile to specified file on SIGUSR2")

var logfile *string = flag.String("logfile",
	"",
	"Redirect std{out,err} to supplied file.")

var pidfile *string = flag.String("pidfile",
	basepath+"/configd.pid",
	"Write pid to supplied file.")

var socket *string = flag.String("socketfile",
	basepath+"/main.sock",
	"Path to socket used to comminicate with daemon.")

var yangdir *string = flag.String("yangdir",
	"/usr/share/configd/yang",
	"Load YANG from specified directory.")

var compdir *string = flag.String("compdir",
	"/lib/vci/components",
	"Load Component Config from specified directory.")

var username *string = flag.String("user",
	"configd",
	"Username to explicitly allow without authorization")

var groupname *string = flag.String("group",
	"configd",
	"Group that owns the socket")

var runfile *string = flag.String("runfile",
	basepath+"/running.config",
	"File to store current running config into incase of restart")

var secretsgroup *string = flag.String("secretsgroup",
	"secrets",
	"Group that is allowed to view nodes marked as secret")

var supergroup *string = flag.String("supergroup",
	"",
	"Group that is permitted access to all sessions")

var capabilities *string = flag.String("capabilities",
	compile.DefaultCapsLocation,
	"File specifying system capabilities")

func sigstartprof() {
	sigch := make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGUSR1)
	signal.Notify(sigch, syscall.SIGUSR2)
	for {
		sig := <-sigch
		switch sig {
		case syscall.SIGUSR1:
			if !runningprof {
				cpuproffile, err := os.Create(*cpuprofile)
				if err != nil {
					log.Fatal(err)
				}
				pprof.StartCPUProfile(cpuproffile)
				runningprof = true
			} else {
				pprof.StopCPUProfile()
				cpuproffile.Close()
				runningprof = false
			}
		case syscall.SIGUSR2:
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal(err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	}
}

func fatal(err error) {
	if err != nil {
		log.Println(err)
		elog.Fatal(err)
	}
}

func openLogfile() {
	if logfile == nil {
		return
	}
	f, e := os.OpenFile(*logfile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0640)
	if e != nil {
		fmt.Fprintf(os.Stderr, "%s\n", e)
		return
	}
	defer f.Close()
	syscall.Dup2(int(f.Fd()), 1)
	syscall.Dup2(int(f.Fd()), 2)
}

func writePid() {
	if pidfile == nil {
		return
	}
	f, e := os.OpenFile(*pidfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		fmt.Fprintf(os.Stderr, "%s\n", e)
		return
	}
	defer f.Close()
	pid := os.Getpid()
	fmt.Fprintf(f, "%d\n", pid)
}

func getIds(username, groupname string) (uid, gid int) {
	u, err := user.Lookup(username)
	if err != nil {
		uid = 0
	} else {
		uid, _ = strconv.Atoi(u.Uid)
	}
	g, err := group.Lookup(groupname)
	if err != nil {
		gid = 0
	} else {
		gid = int(g.Gid)
	}
	return uid, gid
}

func initialiseLogging() {
	var err error

	openLogfile()

	if logfile == nil || *logfile == "" {
		// log to stderr
		elog = log.New(os.Stderr, "", 0)
	} else {
		//rsyslog may not be up even though it returns to the init system so we
		//have to do this mess to ensure that logging works.
		for i := 0; i < 5; i++ {
			elog, err = configd.NewLogger(syslog.LOG_ERR|syslog.LOG_DAEMON, 0)

			if err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err != nil {
			//give up and log to stderr (mapped to configd.log)
			elog = log.New(os.Stderr, "", 0)
		}
	}
}

func getListeners() net.Listener {
	listeners, err := activation.Listeners(true)
	fatal(err)
	if len(listeners) == 0 {
		fmt.Println("No systemd listeners")
		if !os.IsNotExist(os.Remove(*socket)) {
			fatal(err)
		}

		ua, err := net.ResolveUnixAddr("unix", *socket)
		fatal(err)

		l, err := net.ListenUnix("unix", ua)
		fatal(err)

		err = os.Chmod(*socket, 0777)
		fatal(err)

		uid, gid := getIds(*username, *groupname)
		err = os.Chown(*socket, uid, gid)
		fatal(err)

		listeners = append(listeners, l)
	}
	return listeners[0]
}

func main() {
	debug.SetGCPercent(25)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	initialiseLogging()

	fatal(os.MkdirAll(basepath, 0755))

	go sigstartprof()

	st, stFull := startYangd()

	l := getListeners()

	config := &configd.Config{
		User:         *username,
		Runfile:      *runfile,
		Logfile:      *logfile,
		Pidfile:      *pidfile,
		Yangdir:      *yangdir,
		Socket:       *socket,
		SecretsGroup: *secretsgroup,
		SuperGroup:   *supergroup,
		Capabilities: *capabilities,
	}

	srv := server.NewSrv(l.(*net.UnixListener), st, stFull, *username,
		config, elog)

	writePid()

	// Initialization may generate significant garbage ensure that
	// it is cleaned up immediately.
	runtime.GC()
	debug.FreeOSMemory()

	fatal(srv.Serve())
}
