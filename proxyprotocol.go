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
	ProtocolVersion1 = 1
	ProtocolVersion2 = 2
)

type Header struct {
	Version    protocolVersion
	InetFamily string
	Src        net.TCPAddr
	Dest       net.TCPAddr
}

func WriteHeader(header Header, writer io.Writer) error {
	switch header.Version {
	case ProtocolVersion1:
		_, err := writer.Write([]byte(
			fmt.Sprintf("PROXY %s %s %s %d %d\r\n",
				header.InetFamily,
				header.Src.IP,
				header.Dest.IP,
				header.Src.Port,
				header.Dest.Port,
			)))
		if err != nil {
			return err
		}
	case ProtocolVersion2:
		panic("TODO")
	default:
		return fmt.Errorf("unexpected protocol version: %d: %w", header.Version, ErrInvalidProtocolHeader)
	}
	return nil
}

func ParseProtocolHeader(reader *bufio.Reader) (Header, error) {
	parser := newParser(reader)

	// Example header: PROXY TCP4 255.255.255.255 255.255.255.255 65535 65535\r\n
	isVersion1, err := isProtocolVersion1(reader)
	if err != nil {
		return Header{}, fmt.Errorf("error checking if protocol is version 1: %w: %w", err, ErrInvalidProtocolHeader)
	}

	if isVersion1 {
		// The header must start with `PROXY`.
		signature, err := parser.readBytes(len(protocolVersion1HeaderSignature))
		if err != nil {
			return Header{}, fmt.Errorf("error reading version 1 signature bytes: %w", err)
		}
		if !bytes.Equal(signature, protocolVersion1HeaderSignature) {
			return Header{}, fmt.Errorf("invalid version 1 signature: %w", ErrInvalidProtocolHeader)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return Header{}, fmt.Errorf("expected whitespace after version 1 signature:%w", err)
		}

		// Followed by a the INET protocol and family.
		inetProtocolAndFamily, err := parser.readUntilDelimiter(' ')
		if err != nil {
			return Header{}, fmt.Errorf("error reading inet protocol and family bytes: %w", err)
		}

		if !isValidProtocolVersion1InetProtocolAndFamily(string(inetProtocolAndFamily)) {
			return Header{}, fmt.Errorf("invalid inet protocol and family: %w", ErrInvalidProtocolHeader)
		}

		if bytes.Equal(inetProtocolAndFamily, []byte(protocolVersion1Unknown)) {
			if _, err := parser.readUntilCRLF(); err != nil {
				return Header{}, fmt.Errorf("error reading bytes until \r\n is found after finding unknown protocol version: %w", ErrInvalidProtocolHeader)
			}
			return Header{
					Version:    ProtocolVersion1,
					InetFamily: string(inetProtocolAndFamily),
					Src:        net.TCPAddr{},
					Dest:       net.TCPAddr{},
				},
				nil
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return Header{}, fmt.Errorf("expected whitespace after inet protocol and family: %w", err)
		}

		// Followed by the layer 3 source address.
		srcAddress, err := parser.readUntilDelimiter(' ')
		if err != nil {
			return Header{}, fmt.Errorf("error reading source address bytes:%w", err)
		}
		srcIP := net.ParseIP(string(srcAddress))
		if srcIP == nil {
			return Header{}, fmt.Errorf("error parsing source address: %w", ErrInvalidProtocolHeader)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return Header{}, fmt.Errorf("expected whitespace after source address: %w", err)
		}

		// Followed by the layer 3 destination address.
		destAddress, err := parser.readUntilDelimiter(' ')
		if err != nil {
			return Header{}, fmt.Errorf("error reading destination address bytes: %w", err)
		}
		destIP := net.ParseIP(string(destAddress))
		if destIP == nil {
			return Header{}, fmt.Errorf("error parsing destination address: %w", ErrInvalidProtocolHeader)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return Header{}, fmt.Errorf("expected whitespace after destination address: %w", err)
		}

		srcPort, err := parser.readUint16()
		if err != nil {
			return Header{}, fmt.Errorf("error reading source port: %w", err)
		}

		// Followed by a single whitespace.
		if err := parser.expectByte(' '); err != nil {
			return Header{}, fmt.Errorf("expected whitespace after source port: %w", err)
		}

		destPort, err := parser.readUint16()
		if err != nil {
			return Header{}, fmt.Errorf("error reading destination port: %w", err)
		}

		// Followed by \r\n
		if err := parser.expectCRLF(); err != nil {
			return Header{}, fmt.Errorf("expected whitespace after destination port: %w", err)
		}

		return Header{
				Version:    ProtocolVersion1,
				InetFamily: string(inetProtocolAndFamily),
				Src:        net.TCPAddr{IP: srcIP, Port: int(srcPort)},
				Dest:       net.TCPAddr{IP: destIP, Port: int(destPort)},
			},
			nil
	}

	return Header{}, fmt.Errorf("unexpected protocol version: %w", ErrInvalidProtocolHeader)
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
