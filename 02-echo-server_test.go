package adminsock

import (
	"net"
	"strings"
	"testing"
)

// the echo function for our dispatch table
func echo(s []string) ([]byte, error) {
	return []byte(strings.Join(s, " ")), nil
}

// implement an echo server
func TestEchoServer(t *testing.T) {
	d := make(Dispatch)   // create Dispatch
	d["echo"] = echo // and put a function in it
	// instantiate an adminsocket
	as, err := New(d, 0)
	if err != nil {
		t.Errorf("Couldn't create socket: %v", err)
	}
	// launch fakeclient. we should get a message about the
	// connection.
	go echoclient(buildSockName(), t)
	msg := <-as.Msgr
	if msg.Err != nil {
		t.Errorf("connection creation returned error: %v", msg.Err)
	}
	if msg.Txt != "adminsock conn 1 opened" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// wait for disconnect Msg
	msg = <-as.Msgr
	if msg.Err == nil {
		t.Errorf("connection drop should be an err, but got nil")
	}
	if msg.Txt != "adminsock conn 1 closed" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// shut down adminsocket
	as.Quit()
}

// this time our (less) fake client will send a string over the
// connection and (hopefully) get it echoed back.
func echoclient(sn string, t *testing.T) {
	conn, err := net.Dial("unix", sn)
	defer conn.Close()
	if err != nil {
		t.Errorf("Couldn't connect to %v: %v", sn, err)
	}
	conn.Write([]byte("echo it works!"))
	res := readConn(conn, t)
	if string(res) != "it works!" {
		t.Errorf("Expected 'it works!' but got '%v'", string(res))
	}
	// for bonus points, let's send a bad command
	conn.Write([]byte("foo bar"))
	res = readConn(conn, t)	
	if string(res) != "Unknown command 'foo'\nAvailable commands:\n    echo\n" {
		t.Errorf("Expected 'it works!' but got '%v'", string(res))
	}
}

func readConn(conn net.Conn, t *testing.T) []byte {
	b1 := make([]byte, 64)
	var b2 []byte
	for {
		n, err := conn.Read(b1)
		if err != nil {
			t.Errorf("Couldn't read response from adminsock: %v", err)
		}
		b2 = append(b2, b1[:n]...)
		if n == 64 {
			continue
		}
		break
	}
	return b2
}
