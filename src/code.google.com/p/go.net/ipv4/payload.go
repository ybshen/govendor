// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ipv4

import (
	"net"
	"syscall"
)

// A payloadHandler represents the IPv4 datagram payload handler.
type payloadHandler struct {
	c net.PacketConn
	rawOpt
}

func (c *payloadHandler) ok() bool { return c != nil && c.c != nil }

// Read reads a payload of the received IPv4 datagram, from the
// endpoint c, copying the payload into b.  It returns the number of
// bytes copied into b, the control message cm and the source address
// src of the received datagram.
func (c *payloadHandler) Read(b []byte) (n int, cm *ControlMessage, src net.Addr, err error) {
	if !c.ok() {
		return 0, nil, nil, syscall.EINVAL
	}
	oob := newControlMessage(&c.rawOpt)
	var oobn int
	switch rd := c.c.(type) {
	case *net.UDPConn:
		if n, oobn, _, src, err = rd.ReadMsgUDP(b, oob); err != nil {
			return 0, nil, nil, err
		}
	case *net.IPConn:
		nb := make([]byte, len(b)+maxHeaderLen)
		if n, oobn, _, src, err = rd.ReadMsgIP(nb, oob); err != nil {
			return 0, nil, nil, err
		}
		hdrlen := (int(b[0]) & 0x0f) << 2
		copy(b, nb[hdrlen:])
		n -= hdrlen
	default:
		return 0, nil, nil, errInvalidConnType
	}
	if cm, err = parseControlMessage(oob[:oobn]); err != nil {
		return 0, nil, nil, err
	}
	if cm != nil {
		cm.Src = netAddrToIP4(src)
	}
	return
}

// Write writes a payload of the IPv4 datagram, to the destination
// address dst through the endpoint c, copying the payload from b.
// It returns the number of bytes written.  The control message cm
// allows the datagram path and the outgoing interface to be
// specified.  Currently only Linux supports this.  The cm may be nil
// if control of the outgoing datagram is not required.
func (c *payloadHandler) Write(b []byte, cm *ControlMessage, dst net.Addr) (n int, err error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	oob := marshalControlMessage(cm)
	if dst == nil {
		return 0, errMissingAddress
	}
	switch wr := c.c.(type) {
	case *net.UDPConn:
		n, _, err = wr.WriteMsgUDP(b, oob, dst.(*net.UDPAddr))
	case *net.IPConn:
		n, _, err = wr.WriteMsgIP(b, oob, dst.(*net.IPAddr))
	default:
		return 0, errInvalidConnType
	}
	if err != nil {
		return 0, err
	}
	return
}
