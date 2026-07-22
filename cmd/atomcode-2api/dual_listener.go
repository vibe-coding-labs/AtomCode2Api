package main

import (
	"crypto/tls"
	"net"
	"net/http"
)

// dualListener listens on a single port and upgrades HTTP to HTTPS when possible.
type dualListener struct {
	net.Listener
	tlsConfig *tls.Config
	handler   http.Handler
}

func newDualListener(ln net.Listener, tlsCfg *tls.Config, handler http.Handler) *dualListener {
	return &dualListener{Listener: ln, tlsConfig: tlsCfg, handler: handler}
}

func (d *dualListener) Accept() (net.Conn, error) {
	conn, err := d.Listener.Accept()
	if err != nil {
		return conn, err
	}
	return &dualConn{Conn: conn, tlsConfig: d.tlsConfig}, nil
}

type dualConn struct {
	net.Conn
	tlsConfig *tls.Config
	isTLS     bool
	peeked    bool
	firstByte byte
}

func (c *dualConn) Read(b []byte) (int, error) {
	if !c.peeked {
		c.peeked = true
		tmp := make([]byte, 1)
		n, err := c.Conn.Read(tmp)
		if n > 0 {
			c.firstByte = tmp[0]
			if c.firstByte == 0x16 {
				c.isTLS = true
				tlsConn := tls.Server(c.Conn, c.tlsConfig)
				return tlsConn.Read(b)
			}
			b[0] = c.firstByte
			if len(b) > 1 {
				rest, err := c.Conn.Read(b[1:])
				return rest + 1, err
			}
			return 1, nil
		}
		return n, err
	}
	return c.Conn.Read(b)
}
