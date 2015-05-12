package asock

import (
	"net"
	"testing"
)

// the echo function for our dispatch table
func echo(args [][]byte) ([]byte, error) {
	var bs []byte
	for i, arg := range args {
		bs = append(bs, arg...)
		if i != len(args) - 1 {
			bs = append(bs, byte(32))
		}
	}
	return bs, nil
}

// implement an echo server
func TestEchoServer(t *testing.T) {
	d := make(Dispatch) // create Dispatch
	d["echo"] = &DispatchFunc{echo, "split"} // and put a function in it
	// instantiate an asocket
	c := Config{"/tmp/test02.sock", 0, All}
	as, err := NewUnix(c, d)
	if err != nil {
		t.Errorf("Couldn't create socket: %v", err)
	}
	// launch echoclient. we should get a message about the
	// connection.
	go echoclient(as.s, t)
	msg := <-as.Msgr
	if msg.Err != nil {
		t.Errorf("connection creation returned error: %v", msg.Err)
	}
	if msg.Txt != "client connected" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// and a message about dispatching the command
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("successful cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "dispatching [echo]" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 101 {
		t.Errorf("msg.Code should have been 101 but got: %v", msg.Code)
	}
	// and a message that we have replied
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("successful cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "reply sent" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 200 {
		t.Errorf("msg.Code should have been 200 but got: %v", msg.Code)
	}
	// wait for msg from unsuccessful command
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("unsuccessful cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "bad command 'foo'" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 400 {
		t.Errorf("msg.Code should have been 400 but got: %v", msg.Code)
	}
	// wait for disconnect Msg
	msg = <-as.Msgr
	if msg.Err == nil {
		t.Errorf("connection drop should be an err, but got nil")
	}
	if msg.Txt != "client disconnected" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// shut down asocket
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
	res, err := readConn(conn)
	if err != nil {
		t.Errorf("Error on read: %v", err)
	}
	if string(res) != "it works!" {
		t.Errorf("Expected 'it works!' but got '%v'", string(res))
	}
	// for bonus points, let's send a bad command
	conn.Write([]byte("foo bar"))
	res, err = readConn(conn)
	if err != nil {
		t.Errorf("Error on read: %v", err)
	}
	if string(res) != "Unknown command 'foo'\nAvailable commands:\n    echo\n" {
		t.Errorf("Expected 'it works!' but got '%v'", string(res))
	}
}

func readConn(conn net.Conn) ([]byte, error) {
	b1 := make([]byte, 64)
	var b2 []byte
	for {
		n, err := conn.Read(b1)
		if err != nil {
			return nil, err
		}
		b2 = append(b2, b1[:n]...)
		if n == 64 {
			continue
		}
		break
	}
	return b2, nil
}
