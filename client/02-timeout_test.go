package client

import (
	"testing"
	"firepear.net/asock"
	"time"
)

func waitwhat(args [][]byte) ([]byte, error) {
	time.Sleep(40 * time.Millisecond)
	return args[0], nil
}

func TestClientTimeout(t *testing.T) {
	// instantiate unix asock
	asdisp := make(asock.Dispatch)
	asdisp["echo"] = &asock.DispatchFunc{hollaback, "nosplit"}
	asdisp["slow"] = &asock.DispatchFunc{waitwhat, "nosplit"}
	asconf := asock.Config{Sockname: "/tmp/clienttest2.sock", Msglvl: asock.Fatal}
	as, err := asock.NewUnix(asconf, asdisp)
	if err != nil {
		t.Errorf("Failed to create asock instance: %v", err)
	}
	// and now a client
	cconf := Config{Addr: "/tmp/clienttest2.sock", Timeout: 25}
	c, err := NewUnix(cconf)
	if err != nil {
		t.Errorf("Failed to create client: %v", err)
	}
	// and send a message
	resp, err := c.Dispatch([]byte("echo just the one test"))
	if err != nil {
		t.Errorf("Dispatch returned error: %v", err)
	}
	if string(resp) != "just the one test\n\n" {
		t.Errorf("Expected `just the one test\\n\\n` but got: `%v`", string(resp))
	}
	// now send a message which will take too long to come back
	resp, err = c.Dispatch([]byte("slow just the one test, slowly"))
	if err == nil {
		t.Errorf("Dispatch should have timed out, but no error. Got: %v", string(resp))
	}
	if err.Error() != "read unix /tmp/clienttest2.sock: i/o timeout" {
		t.Errorf("Expected read timeout, but got: %v", err)
	}
	resp, err = c.Read()
	if err != nil {
		t.Errorf("Read returned error: %v", err)
	}
	if string(resp) != "just the one test, slowly\n\n" {
		t.Errorf("Expected `just the one test, slowly\\n\\n` but got: `%v`", string(resp))
	}
	c.Close()
	as.Quit()
}
