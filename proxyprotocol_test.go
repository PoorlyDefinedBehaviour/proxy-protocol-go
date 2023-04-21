package proxyprotocol

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzParseProtocolHeaderDoesntCrash(f *testing.F) {
	f.Add([]byte("PROXY TCP4 255.255.255.255 255.255.255.254 65535 65534\r\n"))
	f.Add([]byte("PROXY TCP6 255.255.255.255 255.255.255.254 65535 65534\r\n"))
	f.Add([]byte("PROXY UNKNOWN 255.255.255.255 255.255.255.254 65535 65534\r\n"))
	f.Add([]byte("PROXY UNKNOWN\r\n"))

	f.Fuzz(func(t *testing.T, input []byte) {
		_, _ = ParseProtocolHeader(bufio.NewReader(bytes.NewReader(input)))
	})
}

func TestWriteHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description            string
		header                 Header
		expectedErr            error
		expectedWriterContents string
	}{
		{
			description: "valid protocol version 1 header",
			header: Header{
				Version:    ProtocolVersion1,
				InetFamily: "TCP6",
				Src: net.TCPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 8080,
				},
				Dest: net.TCPAddr{
					IP:   net.ParseIP("127.0.0.2"),
					Port: 8081,
				},
			},
			expectedWriterContents: "PROXY TCP6 127.0.0.1 127.0.0.2 8080 8081\r\n",
			expectedErr:            nil,
		},
		{
			description: "invalid protocol version, returns error",
			header: Header{
				Version:    3,
				InetFamily: "TCP6",
				Src: net.TCPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 8080,
				},
				Dest: net.TCPAddr{
					IP:   net.ParseIP("127.0.0.2"),
					Port: 8081,
				},
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
		expected    Header
		err         error
	}{
		{
			description: "empty proxy protocol header, should return error",
			input:       "",
			expected: Header{
				Version:    0,
				InetFamily: "",
				Src:        net.TCPAddr{},
				Dest:       net.TCPAddr{},
			},
			err: ErrInvalidProtocolHeader,
		},
		{
			description: "valid tcp4 header",
			input:       "PROXY TCP4 255.255.255.255 255.255.255.254 65535 65534\r\n",
			expected: Header{
				Version:    ProtocolVersion1,
				InetFamily: "TCP4",
				Src: net.TCPAddr{
					IP:   net.ParseIP("255.255.255.255"),
					Port: 65535,
				},
				Dest: net.TCPAddr{
					IP:   net.ParseIP("255.255.255.254"),
					Port: 65534,
				},
			},
			err: nil,
		},
		{
			description: "valid unknown protocol header, should ignore everything after the protocol until \r\n is found",
			input:       "PROXY UNKNOWN 255.255.255.255 255.255.255.254 65535 65534\r\n",
			expected: Header{
				Version:    ProtocolVersion1,
				InetFamily: protocolVersion1Unknown,
				Src:        net.TCPAddr{},
				Dest:       net.TCPAddr{},
			},
			err: nil,
		},
		{
			description: "invalid signature, should return error",
			input:       "ANYTHING_THATS_NOT_PROXY TCP4 255.255.255.255 255.255.255.254 65535 65534\r\n",
			expected: Header{
				Version:    0,
				InetFamily: "",
				Src:        net.TCPAddr{},
				Dest:       net.TCPAddr{},
			},
			err: ErrInvalidProtocolHeader,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			header, err := ParseProtocolHeader(bufio.NewReader(strings.NewReader(tt.input)))

			if !assert.True(t, errors.Is(err, tt.err)) {
				t.Fatalf("expected err: %s, but got: %s", tt.err, err)
			}

			assert.Equal(t, tt.expected, header)
		})
	}
}
