package asock // import "firepear.net/asock"

// Copyright (c) 2014,2015 Shawn Boyette <shawn@firepear.net>. All
// rights reserved.  Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.

import (
	"crypto/tls"
	"fmt"
	"os"
	"net"
	"sync"
)


// Message levels control which messages will be sent to as.Msgr
const (
	All = iota
	Conn
	Error
	Fatal
	Version = "0.18.0"
)

// Asock is a handle on an asock instance. It contains the
// Msgr channel, which is the conduit for notifications from the
// instance.
type Asock struct {
	Msgr chan *Msg
	q    chan bool
	w    *sync.WaitGroup
	s    string       // socket name
	l    net.Listener // listener socket
	d    dispatch     // dispatch table
	t    int64        // timeout
	ml   int          // message level
	eom  []byte       // end-of-message
	help string       // bad command help
}

// AddHandler adds a handler function to the Asock instance.
//
// argmode has two legal values: "split" and "nosplit"
func (a *Asock) AddHandler(name string, argmode string, df DispatchFunc) error {
	if _, ok := a.d[name]; ok {
		return fmt.Errorf("handler '%v' already exists", name)
	}
	if argmode != "split" && argmode != "nosplit" {
		return fmt.Errorf("invalid argmode '%v'", argmode)
	}
	a.d[name] = &dispatchFunc{df, argmode}
	a.help = ""
	for cmd := range a.d {
		a.help = a.help + cmd + " "
	}
	return nil
}

// genMsg creates messages and sends them to the Msgr channel.
func (a *Asock) genMsg(conn, req uint, code, ml int, txt string, err error) {
	// if this message's level is below the instance's level, don't
	// generate the message
	if ml < a.ml {
		return
	}
	select {
	case a.Msgr <- &Msg{conn, req, code, txt, err}:
	default:
	}
}

// Quit handles shutdown and cleanup for an asock instance,
// including waiting for any connections to terminate. When it
// returns, the Asock is fully shut down. See the package Overview
// for more info.
func (a *Asock) Quit() {
	a.q <- true
	a.l.Close()
	a.w.Wait()
	close(a.q)
	close(a.Msgr)
}

// Msg is the format which asock uses to communicate informational
// messages and errors to its host program. See the package Overview
// for more info.
type Msg struct {
	Conn uint   // connection id
	Req  uint   // connection request number
	Code int    // numeric status code
	Txt  string // textual description of Msg
	Err  error  // error (if any) passed along from underlying condition
}

// Error implements the error interface, returning a nicely (if
// blandly) formatted string containing all information present in a
// given Msg.
func (m *Msg) Error() string {
	s := fmt.Sprintf("conn %d req %d status %d", m.Conn, m.Req, m.Code)
	if m.Txt != "" {
		s = s + fmt.Sprintf(" (%s)", m.Txt)
	}
	if m.Err != nil {
		s = s + fmt.Sprintf("; err: %s", m.Err)
	}
	return s
}

// Config holds values to be passed to the constuctor.
type Config struct {
	// For Unix sockets, Sockname takes the form
	// "/path/to/socket". For TCP socks, it is either an IPv4 or IPv6
	// address followed by the desired port number ("127.0.0.1:9090",
	// "[::1]:9090").
	Sockname string

	// Timeout is the number of milliseconds the socket will wait
	// before timing out due to inactivity. Default (zero) is no
	// timeout. Negative values cause the connection to close after
	// handling one request (e.g. -25 closes after one request or a
	// read wait of 25 milliseconds, whichever happens first).
	Timeout int64

	// EOM is the end-of-message marker. Asock reads from its socket
	// until it encounters EOM. Defaults to "\n\n".
	EOM string

	// Buffer is the buffer size, in instances of asock.Msg, for
	// Asock.Msgr. Defaults to 32.
	Buffer int

	// Msglvl determines which messages will be sent to the socket's
	// message channel. Valid values are asock.All, asock.Conn,
	// asock.Error, and asock.Fatal.
	Msglvl int
}

// dispatch is the dispatch table which drives asock's behavior. See
// the package Overview for more info on this and DispatchFunc.
type dispatch map[string]*dispatchFunc

// DispatchFunc instances are the functions called via Dispatch.
type DispatchFunc func ([][]byte) ([]byte, error)

// DispatchFunc instances are the functions called via Dispatch.
type dispatchFunc struct {
	// df is the function to be called.
	df DispatchFunc

	// argmode can be "split" or "nosplit". It determines how the
	// bytestream read from the socket will be turned into arguments
	// to Func.
	//
	// Given the input `"echo echo" foo "bar baz" quux`, a function
	// with an Argmode of "nosplit" will receive an arguments list of
	//
	//    []byte{[]byte{`foo "bar baz" quux`}}
	//
	// A fuction with Argmode "split" would get:
	//
	//    []byte{[]byte{`foo`}, []byte{`bar baz`}, []byte{`quux`}}
	argmode string
}

// NewTCP returns an instance of Asock which uses TCP networking.
func NewTCP(c Config) (*Asock, error) {
	tcpaddr, err := net.ResolveTCPAddr("tcp", c.Sockname)
	l, err := net.ListenTCP("tcp", tcpaddr)
	if err != nil {
		return nil, err
	}
	return commonNew(c, l), nil
}

// NewTLS returns an instance of Asock which uses TCP networking,
// secured with TLS.
func NewTLS(c Config, t *tls.Config) (*Asock, error) {
	l, err := tls.Listen("tcp", c.Sockname, t)
	if err != nil {
		return nil, err
	}
	return commonNew(c, l), nil
}

// NewUnix returns an instance of Asock which uses Unix domain
// networking. Argument `p` is the Unix permissions to set on the
// socket (e.g. 770)
func NewUnix(c Config, p uint32) (*Asock, error) {
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: c.Sockname, Net: "unix"})
	if err != nil {
		return nil, err
	}
	err = os.Chmod(c.Sockname, os.FileMode(p))
	if err != nil {
		return nil, err
	}
	return commonNew(c, l), nil
}

// commonNew does shared setup work for the constructors (mostly so
// that changes to Asock don't have to be mirrored)
func commonNew(c Config, l net.Listener) *Asock {
	// spawn a WaitGroup and add one to it for a.sockAccept()
	var w sync.WaitGroup
	w.Add(1)
	// set c.Buffer to the default if it's zero
	if c.Buffer < 1 {
		c.Buffer = 32
	}
	// set c.EOM to the default if it's the empty string
	if c.EOM == "" {
		c.EOM = "\n\n"
	}
	// create the Asock instance, start listening, and return
	a := &Asock{make(chan *Msg, c.Buffer), make(chan bool, 1), &w, c.Sockname, l, make(dispatch), c.Timeout, c.Msglvl, []byte(c.EOM), ""}
	go a.sockAccept()
	return a
}
