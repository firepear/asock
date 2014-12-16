package adminsock

// Copyright (c) 2014 Shawn Boyette. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Socket code for adminsock

import (
	"net"
	"sync"
)

// sockAccept monitors the listener socket and spawns connections for
// clients.
func sockAccept(l net.Listener, q chan bool, e chan error, w *sync.WaitGroup) {
	defer w.Done()
	w.Add(1)
	go sockWatchdog(l, q, w)
	// TODO we need to know which connections are open so we can wait
	// on them before closing the listener
	for {
		// TODO see conn.SetDeadline for idle timeouts
		conn, err := l.Accept()
		if err != nil {
			// is the error because sockWatchdog closed the sock?
			select {
			case <-q:    // yes; close up shop
				close(e)
				close(q)
				return
			default:      // no
				e <- err  // send error to client
				close(e)  // close error chan
				q <- true // send for sockWatchdog
				return
			}
		}
		w.Add(1)
		go connHandler(conn, q, e, w)
	}
}

// sockWatchdog waits to get a signal on the quitter chan and closes
// the listener.
func sockWatchdog(l net.Listener, q chan bool, w *sync.WaitGroup) {
	defer w.Done()
	<-q        // block until signalled
	l.Close()  // close the socket
	q <- true  // send a signal for sockAccept to find
}

// connHandler dispatches commands from, and talks back to, a client. It
// is launched, per-connection, from sockAccept().
func connHandler(c net.Conn, q chan bool, e chan error, w *sync.WaitGroup) {
	defer w.Done()
	defer c.Close()
	//log.Println("Accepted connection on adm sock!")
	b1 := make([]byte, 64)  // buffer 1:  network reads go straight here
	b2 := make([]byte, 0)   // buffer 2:  then are accumulated here to handle overruns
	var blen int            // bufferlen: cumulative length of bytes read
	var bstr string         // bufferstr: bytes finally go here when we have them all
	//c.Write([]byte(hellomsg))
//ReadLoop:
	for {
		for {
			// try to read. n is bytes read.
			n, err := c.Read(b1)
			if err != nil {
				//log.Println("Adm socket connection dropped:", err)
				return
			}
			if n > 0 {
				// we read some bytes. first, track how many
				blen += n
				// then copy those bytes into the b2 slice
				b2 = append(b2, b1[:n]...)
				// if we read 64 bytes, loop back to get anything that
				// might be left on the wire
				if n == 64 {
					continue
				}
				// else, we've got a complete command read in. turn it
				// into a string
				bstr = string(b2)
				// reslice b2 so that it will be "empty" on the next read
				b2 = b2[:0]
				// reset total bytes read
				blen = 0
				// break inner loop; drop to dispatch
				break 
			}
		}
		// TODO bstr dispatch table action goes here. fake it for now
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
