package proxyprotocol

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type ListenerAdapter struct {
	Listener                       net.Listener
	ProxyProtocolHeaderReadTimeout time.Duration
}

type connAdapter struct {
	header     Header
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

func newConnAdapter(conn net.Conn, proxyProtocolHeaderReadTimeout time.Duration) (newConn net.Conn, err error) {
	if proxyProtocolHeaderReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(proxyProtocolHeaderReadTimeout))
	}
	defer func() {
		if proxyProtocolHeaderReadTimeout > 0 {
			setReadDeadlineErr := conn.SetReadDeadline(time.Time{})
			if setReadDeadlineErr != nil {
				if err != nil {
					err = fmt.Errorf("error setting conn deadline to no deadline: %w: %s", setReadDeadlineErr, err.Error())
				} else {
					err = fmt.Errorf("error setting conn deadline to no deadline: %w", setReadDeadlineErr)
				}
			}
		}
	}()

	connReader := bufio.NewReader(conn)

	header, err := ParseProtocolHeader(connReader)
	if err != nil {

		return nil, fmt.Errorf("error parsing proxy protocol header during Read([]byte) call: %w", err)
	}

	newConn = &connAdapter{header: header, conn: conn, connReader: connReader}

	return newConn, err
}

func (adapter *connAdapter) Read(b []byte) (n int, err error) {
	n, err = adapter.connReader.Read(b)
	if err != nil {
		return n, fmt.Errorf("error reading connection: %w", err)
	}

	return n, err
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
	return &adapter.header.Src
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
