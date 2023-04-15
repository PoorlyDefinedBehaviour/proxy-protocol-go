package proxyprotocol

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
)

const minBytesForProtocolVersion1 = 8
const maxBytesForProtocolVersion1 = 108

const minBytesForProtocolVersion2 = 16

const protocolVersion1Tcp4 = "TCP4"
const protocolVersion1Tcp6 = "TCP6"
const protocolVersion1Unknown = "UNKNOWN"

var (
	protocolVersion1HeaderSignature = []byte("PROXY")
	protocolVersion2HeaderSignature = []byte{'\x0D', '\x0A', '\x0D', '\x0A', '\x00', '\x0D', '\x0A', '\x51', '\x55', '\x49', '\x54', '\x0A'}

	ErrInvalidProtocolHeader = errors.New("the protocol header is not in the expected format")
)

type protocolVersion byte

const (
	protocolVersion1 = 1
	protocolVersion2 = 2
)

type header struct {
	version      protocolVersion
	srcPort      uint16
	destPort     uint16
	inetProtocol string
	src          net.IP
	dest         net.IP
}

func WriteHeader(header header, writer io.Writer) error {
	switch header.version {
	case protocolVersion1:
		_, err := writer.Write([]byte(
			fmt.Sprintf("PROXY %s %s %s %d %d\r\n",
				header.inetProtocol,
				header.src,
				header.dest,
				header.srcPort, header.destPort,
			)))
		if err != nil {
			return err
		}
	case protocolVersion2:
		panic("TODO")
	default:
		return fmt.Errorf("unexpected protocol version: %d: %w", header.version, ErrInvalidProtocolHeader)
	}
	return nil
}

func ParseProtocolHeader(reader *bufio.Reader) (header, error) {
	parser := newParser(reader)

	// Example header: PROXY TCP4 255.255.255.255 255.255.255.255 65535 65535\r\n
	isVersion1, err := isProtocolVersion1(reader)
	if err != nil {
		return header{}, fmt.Errorf("error checking if protocol is version 1: %w: %w", err, ErrInvalidProtocolHeader)
	}
	if isVersion1 {
		// The header must start with `PROXY`.
		signature, err := parser.readBytes(len(protocolVersion1HeaderSignature))
		if err != nil {
			return header{}, fmt.Errorf("error reading version 1 signature bytes: %w", err)
		}
		if !bytes.Equal(signature, protocolVersion1HeaderSignature) {
			return header{}, fmt.Errorf("invalid version 1 signature: %w", ErrInvalidProtocolHeader)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return header{}, fmt.Errorf("expected whitespace after version 1 signature:%w", err)
		}

		// Followed by a the INET protocol and family.
		inetProtocolAndFamily, err := parser.readUntilDelimiter(' ')
		if err != nil {
			return header{}, fmt.Errorf("error reading inet protocol and family bytes: %w", err)
		}

		if !isValidProtocolVersion1InetProtocolAndFamily(string(inetProtocolAndFamily)) {
			return header{}, fmt.Errorf("invalid inet protocol and family: %w", ErrInvalidProtocolHeader)
		}

		if bytes.Equal(inetProtocolAndFamily, []byte(protocolVersion1Unknown)) {
			if _, err := parser.readUntilCRLF(); err != nil {
				return header{}, fmt.Errorf("error reading bytes until \r\n is found after finding unknown protocol version: %w", ErrInvalidProtocolHeader)
			}
			return header{
					version:      protocolVersion1,
					inetProtocol: string(inetProtocolAndFamily),
					src:          nil,
					dest:         nil,
					srcPort:      0,
					destPort:     0,
				},
				nil
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return header{}, fmt.Errorf("expected whitespace after inet protocol and family: %w", err)
		}

		// Followed by the layer 3 source address.
		srcAddress, err := parser.readUntilDelimiter(' ')
		if err != nil {
			return header{}, fmt.Errorf("error reading source address bytes:%w", err)
		}
		srcIP := net.ParseIP(string(srcAddress))
		if srcIP == nil {
			return header{}, fmt.Errorf("error parsing source address: %w", ErrInvalidProtocolHeader)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return header{}, fmt.Errorf("expected whitespace after source address: %w", err)
		}

		// Followed by the layer 3 destination address.
		destAddress, err := parser.readUntilDelimiter(' ')
		if err != nil {
			return header{}, fmt.Errorf("error reading destination address bytes: %w", err)
		}
		destIP := net.ParseIP(string(destAddress))
		if destIP == nil {
			return header{}, fmt.Errorf("error parsing destination address: %w", ErrInvalidProtocolHeader)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return header{}, fmt.Errorf("expected whitespace after destination address: %w", err)
		}

		srcPort, err := parser.readUint16()
		if err != nil {
			return header{}, fmt.Errorf("error reading source port: %w", err)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return header{}, fmt.Errorf("expected whitespace after source port: %w", err)
		}

		destPort, err := parser.readUint16()
		if err != nil {
			return header{}, fmt.Errorf("error reading destination port: %w", err)
		}

		// Followed by \r\n
		if err := parser.expectCRLF(); err != nil {
			return header{}, fmt.Errorf("expected whitespace after destination port: %w", err)
		}

		return header{
				version:      protocolVersion1,
				inetProtocol: string(inetProtocolAndFamily),
				src:          srcIP,
				dest:         destIP,
				srcPort:      srcPort,
				destPort:     destPort,
			},
			nil
	}

	return header{}, fmt.Errorf("unexpected protocol version: %w", ErrInvalidProtocolHeader)
}

func isValidProtocolVersion1InetProtocolAndFamily(v string) bool {
	return v == protocolVersion1Tcp4 || v == protocolVersion1Tcp6 || v == protocolVersion1Unknown
}

func isProtocolVersion1(reader *bufio.Reader) (bool, error) {
	buffer, err := reader.Peek(len(protocolVersion1HeaderSignature))
	if err != nil {
		return false, err
	}
	return bytes.Equal(buffer, protocolVersion1HeaderSignature), nil
}
