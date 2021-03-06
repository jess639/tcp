// Copyright 2014 Mikio Hara. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tcp

import (
	"errors"
	"net"
	"os"

	"github.com/mikioh/netreflect"
	"github.com/mikioh/tcpopt"
)

var _ net.Conn = &Conn{}

// A Conn represents an end point that uses TCP connection.
// It allows to set non-portable, platform-dependent TCP-level socket
// options.
type Conn struct {
	net.Conn
	s uintptr // socket descriptor for configuring options
}

// SetOption sets a socket option.
func (c *Conn) SetOption(o tcpopt.Option) error {
	b, err := o.Marshal()
	if err != nil {
		return &net.OpError{Op: "set", Net: c.LocalAddr().Network(), Source: nil, Addr: c.LocalAddr(), Err: err}
	}
	if err := setsockopt(c.s, o.Level(), o.Name(), b); err != nil {
		return &net.OpError{Op: "set", Net: c.LocalAddr().Network(), Source: nil, Addr: c.LocalAddr(), Err: os.NewSyscallError("setsockopt", err)}
	}
	return nil
}

// Option returns a socket option.
func (c *Conn) Option(level, name int, b []byte) (tcpopt.Option, error) {
	if len(b) == 0 {
		return nil, errors.New("short buffer")
	}
	if err := getsockopt(c.s, level, name, b); err != nil {
		return nil, &net.OpError{Op: "get", Net: c.LocalAddr().Network(), Source: nil, Addr: c.LocalAddr(), Err: os.NewSyscallError("getsockopt", err)}
	}
	o, err := tcpopt.Parse(level, name, b)
	if err != nil {
		return nil, &net.OpError{Op: "get", Net: c.LocalAddr().Network(), Source: nil, Addr: c.LocalAddr(), Err: err}
	}
	return o, nil
}

// Buffered returns the number of bytes that can be read from the
// underlying socket read buffer.
// It returns -1 when the platform doesn't support this feature.
func (c *Conn) Buffered() int { return buffered(c.s) }

// Available returns how many bytes are unused in the underlying
// socket write buffer.
// It returns -1 when the platform doesn't support this feature.
func (c *Conn) Available() int { return available(c.s) }

// OriginalDst returns an original destination address, which is an
// address not modified by intermediate entities such as network
// address and port translators inside the kernel, on the connection.
//
// Only Linux and BSD variants using PF support this feature.
func (c *Conn) OriginalDst() (net.Addr, error) {
	la := c.LocalAddr().(*net.TCPAddr)
	od, err := originalDst(c.s, la, c.RemoteAddr().(*net.TCPAddr))
	if err != nil {
		return nil, &net.OpError{Op: "get", Net: c.LocalAddr().Network(), Source: nil, Addr: la, Err: err}
	}
	return od, nil
}

// NewConn returns a new end point.
func NewConn(c net.Conn) (*Conn, error) {
	s, err := netreflect.SocketOf(c)
	if err != nil {
		return nil, err
	}
	return &Conn{Conn: c, s: s}, nil
}
