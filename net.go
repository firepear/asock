package asock

// Copyright (c) 2014,2015 Shawn Boyette <shawn@firepear.net>. All
// rights reserved.  Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.

// Socket code for asock

import (
	"fmt"
	"net"
	"time"

	"firepear.net/qsplit"
)

// sockAccept monitors the listener socket and spawns connections for
// clients.
func (a *Asock) sockAccept() {
	defer a.w.Done()
	var n uint
	for n = 1; true; n++ {
		c, err := a.l.Accept()
		if err != nil {
			select {
			case <-a.q:
				// a.Quit() was invoked; close up shop
				a.genMsg(0, 0, 199, 1, "closing listener socket", nil)
				return
			default:
				// we've had a networking error
				a.Msgr <- &Msg{0, 0, 599, "read from listener socket failed", err}
				return
			}
		}
		// we have a new client
		a.w.Add(1)
		go a.connHandler(c, n)
	}
}

// connHandler dispatches commands from, and sends reponses to, a client. It
// is launched, per-connection, from sockAccept().
func (a *Asock) connHandler(c net.Conn, n uint) {
	defer a.w.Done()
	defer c.Close()
	b1 := make([]byte, 64) // buffer 1:  network reads go here, 64B at a time
	var b2 []byte          // buffer 2:  then are accumulated here
	var bs [][]byte        // b2, sliced by word
	var reqnum uint        // request counter for this connection
	var cmdhelp string     // list of commands for the auto-help msg
	var cmd string         // builds cmdhelp, then holds command for dispatch
	for cmd := range a.d {
		cmdhelp = cmdhelp + "    " + cmd + "\n"
	}
	a.genMsg(n, reqnum, 100, 1, "client connected", nil)
	for {
		// check if we're a one-shot connection, and if we're done
		if a.t < 0 && reqnum > 0 {
			a.genMsg(n, reqnum, 197, 1, "ending session", nil)
			return
		}
		// set conn timeout deadline if needed
		if a.t != 0 {
			err := a.setConnTimeout(c)
			if err != nil {
				a.genMsg(n, reqnum, 501, 2, "deadline set failed; disconnecting client", err)
				c.Write([]byte("Sorry, an error occurred. Terminating connection."))
				return
			}
		}
		// get request from the client
		for {
			b, err := c.Read(b1)
			if err != nil {
				if err.Error() == "EOF" {
					a.genMsg(n, reqnum, 198, 1, "client disconnected", err)
				} else {
					a.genMsg(n, reqnum, 197, 1, "ending session", err)
				}
				return
			}
			if b > 0 {
				// then copy those bytes into the b2 slice
				b2 = append(b2, b1[:b]...)
				// if we read 64 bytes, loop back to get anything that
				// might be left on the wire
				if b == 64 {
					continue
				}
				// break inner loop; drop to dispatch
				break 
			}
		}
		reqnum++
		// extract the command
		cl := qsplit.Locations(b2)
		cmd = string(b2[cl[0][0]:cl[0][1]])
		// send error and list of known commands if cmd isn't known
		dfunc, ok := a.d[cmd]
		if !ok {
			c.Write([]byte(fmt.Sprintf("Unknown command '%v'\nAvailable commands:\n%v",
				cmd, cmdhelp)))
			a.genMsg(n, reqnum, 400, 0, fmt.Sprintf("bad command '%v'", cmd), nil)
			continue
		}
		// dispatch command and send response
		a.genMsg(n, reqnum, 101, 0, fmt.Sprintf("dispatching [%v]", cmd), nil)
		switch dfunc.Argmode {
		case "split":
			bs = qsplit.ToBytes(b2[cl[1][0]:])
		case "nosplit":
			bs = bs[:0]
			bs = append(bs, b2[cl[1][0]:])
		}
		reply, err := dfunc.Func(bs)
		if err != nil {
			c.Write([]byte("Sorry, an error occurred and your request could not be completed."))
			a.genMsg(n, reqnum, 500, 2, "request failed", err)
			continue
		}
		c.Write(reply)
		a.genMsg(n, reqnum, 200, 0, "reply sent", nil)
		// reslice b2 so that it will be "empty" on the next read
		b2 = b2[:0]
	}
}

func (a *Asock) setConnTimeout(c net.Conn) error {
	var t time.Duration
	if a.t > 0 {
		t = time.Duration(a.t)
	} else {
		t = time.Duration(a.t - (a.t * 2))
	}
	err := c.SetReadDeadline(time.Now().Add(t * time.Second))
	if err != nil {
		return err
	}
	return nil
}