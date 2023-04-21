package proxyprotocol

import (
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

		clientConn, err := net.Dial("tcp", serverAddr)

		assert.NoError(t, err)

		defer clientConn.Close()

		serverSideConn, err := listener.Accept()

		assert.True(t, strings.Contains(err.Error(), "i/o timeout"))
		assert.Nil(t, serverSideConn)
	})

	t.Run("listener adapter wraps a net.Listener and reads the proxy protocol header", func(t *testing.T) {
		t.Parallel()

		serverAddr := "localhost:9881"

		ln, err := net.Listen("tcp", serverAddr)
		assert.NoError(t, err)

		listener := ListenerAdapter{
			Listener:                       ln,
			ProxyProtocolHeaderReadTimeout: 5 * time.Second,
		}
		defer listener.Close()

		clientSideConn, err := net.Dial("tcp", serverAddr)
		assert.NoError(t, err)
		defer clientSideConn.Close()

		header := Header{
			Version:    ProtocolVersion1,
			InetFamily: "TCP4",
			Src: net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 9000,
			},
			Dest: net.TCPAddr{
				IP:   net.ParseIP("127.0.0.2"),
				Port: 9001,
			},
		}

		assert.NoError(t, WriteHeader(header, clientSideConn))

		serverSideConn, err := listener.Accept()
		assert.NoError(t, err)
		defer serverSideConn.Close()

		assert.Equal(t, &header.Src, serverSideConn.RemoteAddr())
	})

	t.Run("using the listener in a http server", func(t *testing.T) {
		t.Parallel()

		serverAddr := "localhost:9883"

		ln, err := net.Listen("tcp", serverAddr)
		assert.NoError(t, err)

		listener := ListenerAdapter{
			Listener:                       ln,
			ProxyProtocolHeaderReadTimeout: 5 * time.Second,
		}

		clientSideConn, err := net.Dial("tcp", serverAddr)
		assert.NoError(t, err)

		header := Header{
			Version:    ProtocolVersion1,
			InetFamily: "TCP4",
			Src: net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 9000,
			},
			Dest: net.TCPAddr{
				IP:   net.ParseIP("127.0.0.2"),
				Port: 9001,
			},
		}

		assert.NoError(t, WriteHeader(header, clientSideConn))

		_, err = clientSideConn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
		assert.NoError(t, err)

		var remoteAddrReceivedByHttpServer string

		server := &http.Server{Addr: "9500" /*ReadHeaderTimeout: 50 * time.Millisecond, ReadTimeout: 50 * time.Millisecond*/}
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			if _, err := w.Write([]byte(r.RemoteAddr)); err != nil {
				panic(err)
			}

			remoteAddrReceivedByHttpServer = r.RemoteAddr

			// Close the server for the test to exit.
			if err := server.Close(); err != nil {
				panic(err)
			}
		})

		assert.Equal(t, http.ErrServerClosed, server.Serve(&listener))
		assert.Equal(t, header.Src.String(), remoteAddrReceivedByHttpServer)
	})
}
