package petrel

import (
	"strings"
	"testing"
)

// the echo function for our dispatch table, and readConn for the
// client, are defined in test02

// implement an echo server
func TestEchoTCPServer(t *testing.T) {
	// instantiate petrel (failure)
	c := &ServerConfig{Sockname: "127.0.0.1:1", Msglvl: All}
	as, err := TCPServ(c)
	if err == nil {
		as.Quit()
		t.Errorf("Tried to listen on an impossible IP, but it worked")
	}

	// instantiate petrel
	c = &ServerConfig{Sockname: "127.0.0.1:50709", Msglvl: All}
	as, err = TCPServ(c)
	if err != nil {
		t.Errorf("Couldn't create socket: %v", err)
	}
	if as.s != "127.0.0.1:50709" {
		t.Errorf("Socket name should be '127.0.0.1:50709' but got '%v'", as.s)
	}
	// load the echo func into the dispatch table
	err = as.AddFunc("echo", "blob", echo)
	if err != nil {
		t.Errorf("Couldn't add handler func: %v", err)
	}
	if len(as.d) != 1 {
		t.Errorf("as.d should be len 1, but got %v", len(as.d))
	}
	if _, ok := as.d["echo"]; !ok {
		t.Errorf("Can't find dispatch function 'echo'")
	}

	// launch echoclient. we should get a message about the
	// connection.
	go echoTCPclient(as.s, t)
	msg := <-as.Msgr
	if msg.Err != nil {
		t.Errorf("connection creation returned error: %v", msg.Err)
	}
	if !strings.HasPrefix(msg.Txt, "client connected") {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// and a message about dispatching the command
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("successful cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "dispatching: [echo]" {
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
	// and a message about dispatching the command
	// and a message that we have replied (again)
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("successful cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "dispatching: [echo]" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 101 {
		t.Errorf("msg.Code should have been 101 but got: %v", msg.Code)
	}
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
	if msg.Txt != "bad command: [foo]" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 400 {
		t.Errorf("msg.Code should have been 400 but got: %v", msg.Code)
	}
	// wait for msg from nil command
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("nil cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "nil request" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 401 {
		t.Errorf("msg.Code should have been 401 but got: %v", msg.Code)
	}
	// wait for disconnect Msg
	msg = <-as.Msgr
	if msg.Err == nil {
		t.Errorf("connection drop should be an err, but got nil")
	}
	if msg.Txt != "client disconnected" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// shut down petrel
	as.Quit()
}

// now do it in ipv6
func TestEchoTCP6Server(t *testing.T) {
	// instantiate petrel
	c := &ServerConfig{Sockname: "[::1]:50709", Msglvl: All}
	as, err := TCPServ(c)
	if err != nil {
		t.Errorf("Couldn't create socket: %v", err)
	}
	if as.s != "[::1]:50709" {
		t.Errorf("Socket name should be '[::1]:50709' but got '%v'", as.s)
	}
	// load the echo func into the dispatch table, with mode of
	// split this time
	err = as.AddFunc("echo", "args", echo)
	if err != nil {
		t.Errorf("Couldn't add handler func: %v", err)
	}

	// launch echoclient. we should get a message about the
	// connection.
	go echoTCPclient(as.s, t)
	msg := <-as.Msgr
	if msg.Err != nil {
		t.Errorf("connection creation returned error: %v", msg.Err)
	}
	if !strings.HasPrefix(msg.Txt, "client connected") {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// and a message about dispatching the command
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("successful cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "dispatching: [echo]" {
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
	// and a message about dispatching the command
	// and a message that we have replied (again)
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("successful cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "dispatching: [echo]" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 101 {
		t.Errorf("msg.Code should have been 101 but got: %v", msg.Code)
	}
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
	if msg.Txt != "bad command: [foo]" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 400 {
		t.Errorf("msg.Code should have been 400 but got: %v", msg.Code)
	}
	// wait for msg from nil command
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("nil cmd shouldn't be err, but got %v", err)
	}
	if msg.Txt != "nil request" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	if msg.Code != 401 {
		t.Errorf("msg.Code should have been 401 but got: %v", msg.Code)
	}
	// wait for disconnect Msg
	msg = <-as.Msgr
	if msg.Err == nil {
		t.Errorf("connection drop should be an err, but got nil")
	}
	if msg.Txt != "client disconnected" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// shut down petrel
	as.Quit()
}

// this time our (less) fake client will send a string over the
// connection and (hopefully) get it echoed back.
func echoTCPclient(sn string, t *testing.T) {
	ac, err := client.NewTCP(&ClientConfig{Addr: sn})
	if err != nil {
		t.Fatalf("client instantiation failed! %s", err)
	}
	defer ac.Close()

	resp, err := ac.Dispatch([]byte("echo it works!"))
	if err != nil {
		t.Errorf("Error on read: %v", err)
	}
	if string(resp) != "it works!" {
		t.Errorf("Expected 'it works!' but got '%v'", string(resp))
	}
	// let's try echoing nothing
	resp, err = ac.Dispatch([]byte("echo"))
	if err != nil {
		t.Errorf("Error on read: %v", err)
	}
	if string(resp) != "" {
		t.Errorf("Expected '' but got '%v'", string(resp))
	}
	// for bonus points, let's send a bad command
	resp, err = ac.Dispatch([]byte("foo bar"))
	if err != nil {
		t.Errorf("Error on read: %v", err)
	}
	if string(resp) != "PERRPERR400" {
		t.Errorf("Expected bad command error but got '%v'", string(resp))
	}
	// and a null command!
	resp, err = ac.Dispatch([]byte())
	if err != nil {
		t.Errorf("Error on read: %v", err)
	}
	if string(resp) != "PERRPERR401" {
		t.Errorf("Expected bad command error but got '%v'", string(resp))
	}
}

