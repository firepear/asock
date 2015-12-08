package asock

// Copyright (c) 2014,2015 Shawn Boyette <shawn@firepear.net>. All
// rights reserved.  Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.

// Socket code for asock

import (
	"bytes"
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
				a.Msgr <- &Msg{0, 0, 199, "Quit called: closing listener socket", nil}
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
	b1 := make([]byte, 128) // buffer 1:  network reads go here, 128B at a time
	var b2 []byte           // buffer 2:  data accumulates here; requests pulled from here
	var rs [][]byte         // a request, split by word
	var reqnum uint         // request counter for this connection

	a.genMsg(n, reqnum, 100, Conn, "client connected", nil)
	for {
		// check if we're a one-shot connection, and if we're done
		if a.t < 0 && reqnum > 0 {
			a.genMsg(n, reqnum, 197, Conn, "ending session", nil)
			return
		}
		// set conn timeout deadline if needed
		if a.t != 0 {
			a.setConnTimeout(c)
		}
		// get some data from the client
		b, err := c.Read(b1)
		if err != nil {
			if err.Error() == "EOF" {
				a.genMsg(n, reqnum, 198, Conn, "client disconnected", err)
			} else {
				a.genMsg(n, reqnum, 197, Conn, "ending session", err)
			}
			return
		}
		// append what we read into the b2 slice
		b2 = append(b2, b1[:b]...)
		// and enter the dispatch loop
		for {
			// scan b2 for eom; break from loop if we don't find it.
			eom := bytes.Index(b2, a.eom)
			if eom == -1 {
				break
			}
			// we did find it, so we have a request. increment reqnum
			// and slice the req into b3, then reslice b2 to remove
			// this request.
			reqnum++
			b3 := b2[:eom]
			b2 = b2[eom + len(a.eom):]
			// extract the command form b3
			if len(b3) == 0 {
				c.Write([]byte(fmt.Sprintf("Received empty request. Available commands: %v%v",
					a.help, string(a.eom))))
				a.genMsg(n, reqnum, 401, All, "nil request", nil)
				continue
			}
			cl := qsplit.Locations(b3)
			dcmd := string(b3[cl[0][0]:cl[0][1]])
			// now get the args
			var dargs []byte
			if len(cl) == 1 {
				dargs = nil
			} else {
				dargs = b3[cl[1][0]:]
			}
			// send error and list of known commands if we don't
			// recognize the command
			dfunc, ok := a.d[dcmd]
			if !ok {
				c.Write([]byte(fmt.Sprintf("Unknown command '%v'. Available commands: %v%v",
					dcmd, a.help, string(a.eom))))
				a.genMsg(n, reqnum, 400, All, fmt.Sprintf("bad command '%v'", dcmd), nil)
				continue
			}
			// ok, we know the command and we have its dispatch
			// func. call it and send response
			a.genMsg(n, reqnum, 101, All, fmt.Sprintf("dispatching [%v]", dcmd), nil)
			switch dfunc.argmode {
			case "split":
				rs = qsplit.ToBytes(dargs)
			case "nosplit":
				rs = rs[:0]
				rs = append(rs, dargs)
			}
			reply, err := dfunc.df(rs)
			if err != nil {
				c.Write([]byte("Sorry, an error occurred and your request could not be completed." + string(a.eom)))
				a.genMsg(n, reqnum, 500, Error, "request failed", err)
				continue
			}
			reply = append(reply, a.eom...)
			c.Write(reply)
			a.genMsg(n, reqnum, 200, All, "reply sent", nil)
		}
	}
}

func (a *Asock) setConnTimeout(c net.Conn) {
	var t time.Duration
	if a.t > 0 {
		t = time.Duration(a.t)
	} else {
		t = time.Duration(a.t - (a.t * 2))
	}
	_ = c.SetReadDeadline(time.Now().Add(t * time.Millisecond))
}
