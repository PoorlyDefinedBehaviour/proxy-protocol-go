package proxyprotocol

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListener(t *testing.T) {
	t.Parallel()

	t.Run("returns error if proxy header cannot be read before timeout", func(t *testing.T) {
		t.Parallel()

		serverAddr := "localhost:9880"

		ln, err := net.Listen("tcp", serverAddr)
		assert.NoError(t, err)

		listener := ListenerAdapter{
			Listener:                       ln,
			ProxyProtocolHeaderReadTimeout: 10 * time.Millisecond,
		}

		go func() {
			clientConn, err := net.Dial("tcp", serverAddr)
			assert.NoError(t, err)
			defer clientConn.Close()
			// Client does not send the proxy protocol header.
			time.Sleep(10 * time.Second)
		}()

		require.Eventuallyf(t, func() bool {
			_, err := listener.Accept()
			return strings.Contains(err.Error(), "i/o timeout")
		},
			100*time.Millisecond,
			10*time.Millisecond,
			"the server should timeout reading the proxy protocol header because the client took to long to send it",
		)
	})

	t.Run("listener adapter wraps a net.Listener and reads the proxy protocol header", func(t *testing.T) {
		t.Parallel()

		serverAddr := "localhost:9879"

		ln, err := net.Listen("tcp", serverAddr)
		assert.NoError(t, err)

		listener := ListenerAdapter{
			Listener:                       ln,
			ProxyProtocolHeaderReadTimeout: 5 * time.Second,
		}

		type data struct {
			remoteAddr net.Addr
			payload    string
		}

		dataChan := make(chan data)

		go func() {
			serverConn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			defer serverConn.Close()
			buffer := make([]byte, 5)
			_, err = serverConn.Read(buffer)
			if err != nil {
				panic(err)
			}
			dataChan <- data{remoteAddr: serverConn.RemoteAddr(), payload: string(buffer)}
		}()

		clientConn, err := net.Dial("tcp", serverAddr)
		assert.NoError(t, err)
		defer clientConn.Close()

		header := header{
			version:      protocolVersion1,
			src:          net.ParseIP("127.0.0.3"),
			dest:         net.ParseIP("127.0.0.4"),
			inetProtocol: "TCP4",
			srcPort:      8080,
			destPort:     8081,
		}

		assert.NoError(t, WriteHeader(header, clientConn))

		_, err = clientConn.Write([]byte("hello"))
		assert.NoError(t, err)

		require.Eventuallyf(t, func() bool {
			data := <-dataChan
			tcpAddr := data.remoteAddr.(*net.TCPAddr)

			return tcpAddr.IP.Equal(header.src) && tcpAddr.Port == int(header.srcPort) && data.payload == "hello"
		},
			500*time.Millisecond,
			10*time.Millisecond,
			"server should receive a connection with the remote addr that came in the proxy protocol header",
		)
	})

}
