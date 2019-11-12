# configd daemon

## Overview

'configd' is the daemon that runs the BVNOS configuration system.  This involves:

  * Starting 'yangd' - validates YANG directory (including configd extensions)
    and VCI component files
  * Determining 'listeners' ...
  * Creating a server which handles incoming session requests for the life
    of the daemon
