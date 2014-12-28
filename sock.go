package adminsock

// Copyright (c) 2014 Shawn Boyette <shawn@firepear.net>. All rights
// reserved.  Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Socket code for adminsock

import (
	"net"
	"sync"

	"firepear.net/goutils/qsplit"
)

// sockAccept monitors the listener socket and spawns connections for
// clients.
func sockAccept(l net.Listener, t int, m chan *Msg, q chan bool, w *sync.WaitGroup) {
	defer w.Done()
	w.Add(1)
	go sockWatchdog(l, q, w)
	// TODO make list of known commands and hand them to connHandlers
	// for better "unknown command" handling
	for {
		// TODO see conn.SetDeadline for idle timeouts
		c, err := l.Accept()
		if err != nil {
			// is the error because sockWatchdog closed the sock?
			select {
			case <-q:
				// yes; close up shop
				m <- &Msg{"adminsock shutting down", nil}
				close(m)
				return
			default:
				// no, we've had a networking error
				m <- &Msg{"ENOSOCK" ,err}
				close(m)
				q <- true // kill off the watchdog
				return
			}
		}
		w.Add(1)
		go connHandler(c, m, w)
	}
}

// sockWatchdog waits to get a signal on the quitter chan, then closes
// it and the listener.
func sockWatchdog(l net.Listener, q chan bool, w *sync.WaitGroup) {
	defer w.Done()
	<-q        // block until signalled
	l.Close()
	q <- true  // signal to sockAccept
	close(q)
}

// connHandler dispatches commands from, and talks back to, a client. It
// is launched, per-connection, from sockAccept().
func connHandler(c net.Conn, m chan *Msg, w *sync.WaitGroup) {
	// TODO blen may be dead code. check after completion
	defer w.Done()
	defer c.Close()
	m <- &Msg{"adminsock accepted new connection", nil}
	b1 := make([]byte, 64) // buffer 1:  network reads go here, 64B at a time
	var b2 []byte          // buffer 2:  then are accumulated here
	var bs [][]byte        // byteslices, from qsplit.Split()
//ReadLoop:
	for {
		for {
			// try to read. n is bytes read.
			n, err := c.Read(b1)
			if err != nil {
				m <- &Msg{"adminsock connection lost", err}
				return
			}
			if n > 0 {
				// then copy those bytes into the b2 slice
				b2 = append(b2, b1[:n]...)
				// if we read 64 bytes, loop back to get anything that
				// might be left on the wire
				if n == 64 {
					continue
				}
				// TODO maybe this should end when '\n' is encountered
				// instead of when less than 64 bytes is read?
				bs = qsplit.Split(b2)
				// reslice b2 so that it will be "empty" on the next read
				b2 = b2[:0]
				// break inner loop; drop to dispatch
				break 
			}
		}
		// TODO dispatch table action goes here. fake it for now
		// to get around compile errors
		c.Write([]byte(bstr))

		//switch {
		//default:
		//	log.Println("Unknown command")
		//	msg := fmt.Sprintf("Unknown command '%s'. Type 'help' for command list.", bstr)
		//	if _, err := c.Write([]byte(msg)); err != nil {
		//		log.Println("Error writing to adm socket; ending connection")
		//		break ReadLoop
		//	}
		//}
	}
}
