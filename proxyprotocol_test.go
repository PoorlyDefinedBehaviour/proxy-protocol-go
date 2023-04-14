package proxyprotocol

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockConn struct {
	readValue []byte
}

func (conn *MockConn) Read(b []byte) (int, error) {
	fmt.Printf("\n\naaaaaaa len(b) %+v\n\n", len(b))
	fmt.Printf("\n\naaaaaaa conn.readValue %+v\n\n", conn.readValue)
	return copy(b, conn.readValue), io.EOF
}

func (conn *MockConn) Write(b []byte) (n int, err error) {
	return n, err
}

func (conn *MockConn) Close() error {
	return nil
}

func (conn *MockConn) LocalAddr() net.Addr {
	panic("todo")
}

func (conn *MockConn) RemoteAddr() net.Addr {
	panic("todo")
}

func (conn *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (conn *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (conn *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestTodo(t *testing.T) {
	t.Parallel()

	conn := MockConn{readValue: []byte("PROXY")}
	handleConnection(&conn)

}

func Test_parseProtocolHeader(t *testing.T) {
	t.Parallel()

	header, err := parseProtocolHeader([]byte("PROXY TCP4 255.255.255.255 255.255.255.255 65535 65534\r\n"))
	assert.NoError(t, err)

	fmt.Printf("\n\naaaaaaa header %+v\n\n", header)
}
