package asock

import (
	"testing"
)

// sleeperclient is defined in test 07. oneshotclient is defined in
// test 05.

func TestConnNegTimeout(t *testing.T) {
	//
	// rerun timeout test.
	//
	d := make(Dispatch) // create Dispatch
	d["echo"] = &DispatchFunc{echo, "split"} // and put a function in it
	// instantiate an asocket
	c := Config{"/tmp/test09.sock", -1, 32, All, nil}
	as, err := NewUnix(c, d)
	if err != nil {
		t.Errorf("Couldn't create socket: %v", err)
	}
	// rerun oneshot test. we should close this one ourselves.
	go oneshotclient(as.s, t)
	msg := <-as.Msgr
	if msg.Err != nil {
		t.Errorf("connection creation returned error: %v", msg.Err)
	}
	if msg.Txt != "client connected" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}
	// wait for disconnect Msg
	msg = <-as.Msgr // dispatch
	msg = <-as.Msgr // response
	msg = <-as.Msgr
	if msg.Err != nil {
		t.Errorf("connection drop should be nil, but got %v", err)
	}
	if msg.Txt != "ending session" {
		t.Errorf("unexpected msg.Txt: %v", msg.Txt)
	}	
	if msg.Code != 197 {
		t.Errorf("msg.Code should be 197 but got %v", msg.Code)
	}	
	// shut down asocket
	as.Quit()
}
