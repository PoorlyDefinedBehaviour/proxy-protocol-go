package proxyprotocol

import (
	"bufio"
	"net"
	"time"
)

type ListenerAdapter struct {
	Listener                       net.Listener
	ProxyProtocolHeaderReadTimeout time.Duration
}

type connAdapter struct {
	remoteAddr net.Addr
	conn       net.Conn
	connReader *bufio.Reader
}

func (adapter *ListenerAdapter) Accept() (net.Conn, error) {
	conn, err := adapter.Listener.Accept()
	if err != nil {
		return conn, err
	}
	return newConnAdapter(conn, adapter.ProxyProtocolHeaderReadTimeout)
}

func (adapter *ListenerAdapter) Close() error {
	return adapter.Listener.Close()
}

func (adapter *ListenerAdapter) Addr() net.Addr {
	return adapter.Listener.Addr()
}

func newConnAdapter(conn net.Conn, proxyProtocolHeaderReadTimeout time.Duration) (net.Conn, error) {
	if proxyProtocolHeaderReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(proxyProtocolHeaderReadTimeout))
	}

	connReader := bufio.NewReader(conn)

	header, err := ParseProtocolHeader(connReader)
	if err != nil {
		return nil, err
	}

	addr := &net.TCPAddr{IP: header.src, Port: int(header.srcPort)}

	return &connAdapter{remoteAddr: addr, conn: conn, connReader: connReader}, nil
}

func (adapter *connAdapter) Read(b []byte) (n int, err error) {
	return adapter.connReader.Read(b)
}

func (adapter *connAdapter) Write(b []byte) (n int, err error) {
	return adapter.conn.Write(b)
}

func (adapter *connAdapter) Close() error {
	return adapter.conn.Close()
}

func (adapter *connAdapter) LocalAddr() net.Addr {
	return adapter.conn.LocalAddr()
}

func (adapter *connAdapter) RemoteAddr() net.Addr {
	return adapter.remoteAddr
}

func (adapter *connAdapter) SetDeadline(t time.Time) error {
	return adapter.conn.SetDeadline(t)
}

func (adapter *connAdapter) SetReadDeadline(t time.Time) error {
	return adapter.conn.SetReadDeadline(t)
}

func (adapter *connAdapter) SetWriteDeadline(t time.Time) error {
	return adapter.conn.SetWriteDeadline(t)
}
