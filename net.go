package petrel

// Copyright (c) 2014,2015 Shawn Boyette <shawn@firepear.net>. All
// rights reserved.  Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.

// Socket code for petrel

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"firepear.net/qsplit"
)

var (
	// these errors are for internal signalling; they do not propagate
	errshortread = fmt.Errorf("too few bytes")
	errbadcmd = fmt.Errorf("bad command")
	errcmderr = fmt.Errorf("dispatch cmd errored")
)

// sockAccept monitors the listener socket and spawns connections for
// clients.
func (h *Handler) sockAccept() {
	defer h.w.Done()
	var cn uint
	for cn = 1; true; cn++ {
		c, err := h.l.Accept()
		if err != nil {
			select {
			case <-h.q:
				// h.Quit() was invoked; close up shop
				h.Msgr <- &Msg{0, 0, 199, "Quit called: closing listener socket", nil}
				return
			default:
				// we've had a networking error
				h.Msgr <- &Msg{0, 0, 599, "read from listener socket failed", err}
				return
			}
		}
		// we have a new client
		h.w.Add(1)
		go h.connHandler(c, cn)
	}
}

// connHandler dispatches commands from, and sends reponses to, a client. It
// is launched, per-connection, from sockAccept().
func (h *Handler) connHandler(c net.Conn, cn uint) {
	defer h.w.Done()
	defer c.Close()
	// request counter for this connection
	var reqnum uint

	h.genMsg(cn, reqnum, 100, Conn, "client connected", nil)
	for {
		reqnum++

		// read the request
		req, err := h.connRead(c, cn, reqnum)
		if err != nil {
			// TODO write "you're being dropped" msg
			return
		}
		if len(req) == 0 {
			h.sendMsg(c, cn, reqnum, []byte("Received empty request."))
			h.genMsg(cn, reqnum, 401, All, "nil request", nil)
			continue
		}

		// dispatch the request and get the reply
		reply, err := h.reqDispatch(c, cn, reqnum, req)
		if err != nil {
			continue
		}

		// send reply
		err = h.sendMsg(c, cn, reqnum, reply)
		if err != nil {
			return
		}
		h.genMsg(cn, reqnum, 200, All, "reply sent", nil)
	}
}

// connRead does all network reads and assembles the request. If it
// returns an error, then the connection terminates because the state
// of the connection cannot be known.
func (h *Handler) connRead(c net.Conn, cn, reqnum uint) ([]byte, error) {
	// buffer 0 holds the message length
	b0 := make([]byte, 4)
	// buffer 1: network reads go here, 128B at a time
	b1 := make([]byte, 128)
	// buffer 2: data accumulates here; requests pulled from here
	var b2 []byte
	// message length
	var mlen int32
	// bytes read so far
	var bread int32

	// get the response message length
	if h.t > 0 {
		c.SetReadDeadline(time.Now().Add(h.t))
	}
	n, err := c.Read(b0)
	if err != nil {
		if err == io.EOF {
			h.genMsg(cn, reqnum, 198, Conn, "client disconnected", err)
		} else {
			h.genMsg(cn, reqnum, 196, Conn, "failed to read mlen from socket", err)
		}
		return nil, err
	}
	if  n != 4 {
		h.genMsg(cn, reqnum, 196, Conn, "short read on message length", err)
		return nil, errshortread
	}
	buf := bytes.NewReader(b0)
	err = binary.Read(buf, binary.BigEndian, &mlen)
	if err != nil {
		h.genMsg(cn, reqnum, 501, Conn, "could not decode message length", err)
		return nil, err
	}

	for bread < mlen {
		// if there are less than 128 bytes remaining to read in this
		// message, resize b1 to fit. this avoids reading across a
		// message boundary.
		if x := mlen - bread; x < 128 {
			b1 = make([]byte, x)
		}
		if h.t > 0 {
			c.SetReadDeadline(time.Now().Add(h.t))
		}
		n, err = c.Read(b1)
		if err != nil {
			if err == io.EOF {
				h.genMsg(cn, reqnum, 198, Conn, "client disconnected", err)
			} else {
				h.genMsg(cn, reqnum, 196, Conn, "failed to read req from socket", err)
			}
			return nil, err
		}
		if n == 0 {
			// short-circuit just in case this ever manages to happen
			return b2[:mlen], err
		}
		bread += int32(n)
		b2 = append(b2, b1[:n]...)
	}
	return b2[:mlen], err
}

// reqDispatch turns the request into a command and arguments, and
// dispatches these components to a handler.
func (h *Handler) reqDispatch(c net.Conn, cn, reqnum uint, req []byte) ([]byte, error) {
	cl := qsplit.Locations(req)
	dcmd := string(req[cl[0][0]:cl[0][1]])
	// now get the args
	var dargs []byte
	if len(cl) == 1 {
		dargs = nil
	} else {
		dargs = req[cl[1][0]:]
	}
	// send error and list of known commands if we don't
	// recognize the command
	dfunc, ok := h.d[dcmd]
	if !ok {
		h.sendMsg(c, cn, reqnum, []byte(fmt.Sprintf("Unknown command '%s'.", dcmd)))
		h.genMsg(cn, reqnum, 400, All, fmt.Sprintf("bad command '%s'", dcmd), nil)
		return nil, errbadcmd
	}
	// ok, we know the command and we have its dispatch
	// func. call it and send response
	h.genMsg(cn, reqnum, 101, All, fmt.Sprintf("dispatching [%s]", dcmd), nil)
	var rs [][]byte // req, split by word
	switch dfunc.mode {
	case "args":
		rs = qsplit.ToBytes(dargs)
	case "blob":
		rs = rs[:0]
		rs = append(rs, dargs)
	}
	resp, err := dfunc.df(rs)
	if err != nil {
		h.genMsg(cn, reqnum, 500, Error, "request failed", err)
		h.sendMsg(c, cn, reqnum, []byte("Sorry, an error occurred and your request could not be completed."))
		return nil, errcmderr
	}
	return resp, nil
}

// sendMsg handles all network writes.
func (h *Handler) sendMsg(c net.Conn, cn, reqnum uint, resp []byte) error {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, int32(len(resp)))
	if err != nil {
		h.genMsg(cn, reqnum, 501, Conn, "could not encode message length", err)
		return err
	}
	resp = append(buf.Bytes(), resp...)
	if h.t > 0 {
		c.SetReadDeadline(time.Now().Add(h.t))
	}
	_, err = c.Write(resp)
	if err != nil {
		h.genMsg(cn, reqnum, 196, Error, "failed to write resp to socket", err)
	}
	return err
}
