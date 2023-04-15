package proxyprotocol

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzParseProtocolHeaderDoesntCrash(f *testing.F) {
	f.Add([]byte("PROXY TCP4 255.255.255.255 255.255.255.254 65535 65534\r\n"))
	f.Add([]byte("PROXY TCP6 255.255.255.255 255.255.255.254 65535 65534\r\n"))
	f.Add([]byte("PROXY UNKNOWN 255.255.255.255 255.255.255.254 65535 65534\r\n"))
	f.Add([]byte("PROXY UNKNOWN\r\n"))

	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = ParseProtocolHeader(input)
	})
}

func TestWriteHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description            string
		header                 header
		expectedErr            error
		expectedWriterContents string
	}{
		{
			description: "valid protocol version 1 header",
			header: header{
				version:      protocolVersion1,
				inetProtocol: "TCP6",
				src:          net.ParseIP("127.0.0.1"),
				dest:         net.ParseIP("127.0.0.2"),
				srcPort:      8080,
				destPort:     8081,
			},
			expectedWriterContents: "PROXY TCP6 127.0.0.1 127.0.0.2 8080 8081\r\n",
			expectedErr:            nil,
		},
		{
			description: "invalid protocol version, returns error",
			header: header{
				version:      3,
				inetProtocol: "TCP6",
				src:          net.ParseIP("127.0.0.1"),
				dest:         net.ParseIP("127.0.0.2"),
				srcPort:      8080,
				destPort:     8081,
			},
			expectedWriterContents: "",
			expectedErr:            ErrInvalidProtocolHeader,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			writer := bytes.Buffer{}

			err := WriteHeader(tt.header, &writer)

			if !assert.True(t, errors.Is(err, tt.expectedErr)) {
				t.Fatalf("expected error %s but got %s", tt.expectedErr, err)
			}
			assert.Equal(t, tt.expectedWriterContents, writer.String())
		})
	}
}

func TestParseProtocolHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		input       string
		expected    header
		err         error
	}{
		{
			description: "empty proxy protocol header, should return error",
			input:       "",
			expected: header{
				version:      0,
				inetProtocol: "",
				srcPort:      0,
				destPort:     0,
				src:          nil,
				dest:         nil,
			},
			err: ErrInvalidProtocolHeader,
		},
		{
			description: "valid tcp4 header",
			input:       "PROXY TCP4 255.255.255.255 255.255.255.254 65535 65534\r\n",
			expected: header{
				version:      protocolVersion1,
				inetProtocol: "TCP4",
				srcPort:      65535,
				destPort:     65534,
				src:          net.ParseIP("255.255.255.255"),
				dest:         net.ParseIP("255.255.255.254"),
			},
			err: nil,
		},
		{
			description: "valid unknown protocol header, should ignore everything after the protocol until \r\n is found",
			input:       "PROXY UNKNOWN 255.255.255.255 255.255.255.254 65535 65534\r\n",
			expected: header{
				version:      protocolVersion1,
				inetProtocol: protocolVersion1Unknown,
				srcPort:      0,
				destPort:     0,
				src:          nil,
				dest:         nil,
			},
			err: nil,
		},
		{
			description: "invalid signature, should return error",
			input:       "ANYTHING_THATS_NOT_PROXY TCP4 255.255.255.255 255.255.255.254 65535 65534\r\n",
			expected: header{
				version:      0,
				inetProtocol: "",
				srcPort:      0,
				destPort:     0,
				src:          nil,
				dest:         nil,
			},
			err: ErrInvalidProtocolHeader,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			header, err := ParseProtocolHeader([]byte(tt.input))
			assert.Equal(t, tt.err, err)

			assert.Equal(t, tt.expected, header)
		})
	}
}

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

		remoteAddrChan := make(chan net.Addr)

		go func() {
			serverConn, err := listener.Accept()
			assert.NoError(t, err)
			defer serverConn.Close()
			remoteAddrChan <- serverConn.RemoteAddr()
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

		WriteHeader(header, clientConn)

		require.Eventuallyf(t, func() bool {
			remoteAddr := <-remoteAddrChan
			tcpAddr := remoteAddr.(*net.TCPAddr)
			return tcpAddr.IP.Equal(header.src) && tcpAddr.Port == int(header.srcPort)
		},
			100*time.Millisecond,
			10*time.Millisecond,
			"server should receive a connection with the remote addr that came in the proxy protocol header",
		)
	})
}
