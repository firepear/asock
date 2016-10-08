package petrel

// Copyright (c) 2015-2016 Shawn Boyette <shawn@firepear.net>. All
// rights reserved.  Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.

// This file implements the Petrel client.

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
)

// Client is a Petrel client instance.
type Client struct {
	conn net.Conn
	// timeout length
	to time.Duration
}

// ClientConfig holds values to be passed to the client constructor.
type ClientConfig struct {
	// For Unix clients, Addr takes the form "/path/to/socket". For
	// TCP clients, it is either an IPv4 or IPv6 address followed by
	// the desired port number ("127.0.0.1:9090", "[::1]:9090").
	Addr string

	// Timeout is the number of milliseconds the client will wait
	// before timing out due to on a Dispatch() or Read()
	// call. Default (zero) is no timeout.
	Timeout int64
}

// TCPClient returns a Client which uses TCP.
func TCPClient(c *ClientConfig) (*Client, error) {
	conn, err := net.Dial("tcp", c.Addr)
	if err != nil {
		return nil, err
	}
	return newCommon(c, conn)
}

// TLSClient returns a Client which uses TLS + TCP.
func TLSClient(c *ClientConfig, t *tls.Config) (*Client, error) {
	conn, err := tls.Dial("tcp", c.Addr, t)
	if err != nil {
		return nil, err
	}
	return newCommon(c, conn)
}

// UnixClient returns a Client which uses Unix domain sockets.
func UnixClient(c *ClientConfig) (*Client, error) {
	conn, err := net.Dial("unix", c.Addr)
	if err != nil {
		return nil, err
	}
	return newCommon(c, conn)
}

func newCommon(c *ClientConfig, conn net.Conn) (*Client, error) {
	return &Client{conn, time.Duration(c.Timeout) * time.Millisecond}, nil
}

// Dispatch sends a request and returns the response.
func (c *Client) Dispatch(req []byte) ([]byte, error) {
	// generate packed message length header & prepend to request
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int32(len(req)))
	req = append(buf.Bytes(), req...)
	// send request
	if c.to > 0 {
		c.conn.SetDeadline(time.Now().Add(c.to))
	}
	_, err := c.conn.Write(req)
	if err != nil {
		return nil, err
	}
	if c.to > 0 {
		c.conn.SetDeadline(time.Now().Add(c.to))
	}
	resp, err := c.read()
	return resp, err
}

// read reads from the network.

func (c *Client) read() ([]byte, error) {
	resp, _, _, err := connRead(c.conn, c.to, 0)
	if err != nil {
		return nil, err
	}
	// check for/handle error responses
	if len(resp) == 11 && resp[0] == 80 { // 11 bytes, starting with 'P'
		pp := string(resp[0:8])
		if pp == "PERRPERR" {
			code, err := strconv.Atoi(string(resp[8:11]))
			if err != nil {
				return []byte{255}, fmt.Errorf("request error: unknown code %d", code)
			}
			return []byte{255}, perrs[perrmap[code]]
		}
	}
	return resp, err
}


// Close closes the client's connection.
func (c *Client) Close() {
	c.conn.Close()
}
