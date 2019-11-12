# configd code design

## Overview

The configd package is written in Go.  It provides functionality to parse,
compile and validate Yang schemas, and to validate base configurations and
configuration updates against those schemas.  Further to this it manages the
configuration change process (set/delete, commit etc).

Much of the subsidiary functionality, eg the YANG compiler and handling for
configd YANG extensions, is implemented in separate repositories.

## Packages in this repository

The following packages are included in this repository.

### Yang cmds

  * yang2path
  * yang2rev
  * yangc       - standalone version of YANG compiler (inc cfgd xtns)
  * yanggraph

### Configd cmds

  * args2cpath
  * callrpc
  * cfgcli      - implements 'config' mode commands
  * cfgdiff
  * cfgparse
  * cfgread
  * configd     - configd process (parses YANG, config, runs sessions)
  * configdcaps
  * featcaps
  * gettree
  * normalize

### Configd packages

  * configd (configd)         - authentication, logging, error formatting
  * commit  (configd/commit)  - builds config trees for commit
  * session (configd/session) - session management
  * server  (configd/server)  - server for incoming RPCs
