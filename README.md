binswap
-------

## Intro
`binswap` is a tool created to automatically relaunch a binary when a replacement is created at a specific location on the filesystem.

## Usage
```
$ BINSWAP_REPLACEMENT=/tmp/new-server binswap /usr/bin/my-server --port 8080 --other-flag extra args
```
The above command will run /usr/bin/my-server with all arguments given. If a file is created at /tmp/new-server while it is running, binswap will kill the original process, move the new binary for /tmp/new-server to /usr/bin/my-server and restart it.
The cycle continues until my-server exits on its own.
