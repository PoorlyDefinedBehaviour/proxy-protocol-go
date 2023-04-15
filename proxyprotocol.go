package proxyprotocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

const minBytesForProtocolVersion1 = 8
const maxBytesForProtocolVersion1 = 107

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

func ParseProtocolHeader(buffer []byte) (header, error) {
	lexer := newLexer(buffer)

	if isProtocolVersion2(buffer) {
		panic("TODO")
	}

	// Example header: PROXY TCP4 255.255.255.255 255.255.255.255 65535 65535\r\n
	if isProtocolVersion1(buffer) {
		// The header must start with `PROXY`.
		signature, err := lexer.readBytes(len(protocolVersion1HeaderSignature))
		if err != nil {
			return header{}, err
		}
		if !bytes.Equal(signature, protocolVersion1HeaderSignature) {
			return header{}, ErrInvalidProtocolHeader
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			return header{}, err
		}

		// Followed by a the INET protocol and family.
		inetProtocolAndFamily, err := lexer.readUntilDelimiter(' ')
		if err != nil {
			return header{}, err
		}

		if !isValidProtocolVersion1InetProtocolAndFamily(string(inetProtocolAndFamily)) {
			return header{}, ErrInvalidProtocolHeader
		}

		if bytes.Equal(inetProtocolAndFamily, []byte(protocolVersion1Unknown)) {
			if _, err := lexer.readUntilByteSequence([]byte{'\r', '\n'}); err != nil {
				return header{}, ErrInvalidProtocolHeader
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
		if err := lexer.expectByte(' '); err != nil {
			return header{}, err
		}

		// Followed by the layer 3 source address.
		srcAddress, err := lexer.readUntilDelimiter(' ')
		if err != nil {
			return header{}, err
		}
		srcIP := net.ParseIP(string(srcAddress))
		if srcIP == nil {
			return header{}, ErrInvalidProtocolHeader
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			return header{}, err
		}

		// Followed by the layer 3 destination address.
		destAddress, err := lexer.readUntilDelimiter(' ')
		if err != nil {
			return header{}, err
		}
		destIP := net.ParseIP(string(destAddress))
		if destIP == nil {
			return header{}, ErrInvalidProtocolHeader
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			return header{}, err
		}

		srcPort, err := lexer.readUint16()
		if err != nil {
			return header{}, err
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			return header{}, err
		}

		destPort, err := lexer.readUint16()
		if err != nil {
			return header{}, err
		}

		// Followed by \r\n
		if err := lexer.expectCRLF(); err != nil {
			return header{}, err
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

	return header{}, ErrInvalidProtocolHeader
}

type lexer struct {
	nextByteIndex int
	buffer        []byte
}

func newLexer(buffer []byte) lexer {
	return lexer{nextByteIndex: 0, buffer: buffer}
}

func (l *lexer) readUint16() (uint16, error) {
	startingIndex := l.nextByteIndex

	for {
		nextByte, err := l.peekNextByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}

		if nextByte < '0' || nextByte > '9' {
			break
		}

		_, err = l.nextByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}
	}

	n, err := strconv.ParseUint(string(l.buffer[startingIndex:l.nextByteIndex]), 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(n), nil
}

func (l *lexer) peekNextByte() (byte, error) {
	if l.nextByteIndex >= len(l.buffer) {
		return 0, io.EOF
	}
	return l.buffer[l.nextByteIndex], nil
}

func (l *lexer) readBytes(n int) ([]byte, error) {
	bytes := make([]byte, 0, n)

	for i := 0; i < n; i++ {
		b, err := l.nextByte()
		if err != nil {
			return bytes, err
		}
		bytes = append(bytes, b)
	}

	return bytes, nil
}

func (l *lexer) nextByte() (byte, error) {
	if l.nextByteIndex >= len(l.buffer) {
		return 0, io.EOF
	}
	index := l.nextByteIndex
	l.nextByteIndex++
	return l.buffer[index], nil
}

func (l *lexer) readUntilByteSequence(sequence []byte) ([]byte, error) {
loop:
	startingIndex := l.nextByteIndex

	for _, expectedByte := range sequence {
		b, err := l.nextByte()
		if err != nil {
			return []byte{}, err
		}
		if b != expectedByte {
			goto loop
		}
	}

	return l.buffer[startingIndex:l.nextByteIndex], nil
}

func (l *lexer) readUntilDelimiter(delimiter byte) ([]byte, error) {
	startingIndex := l.nextByteIndex

	for {
		nextByte, err := l.peekNextByte()
		if err != nil {
			return []byte{}, err
		}
		if nextByte == delimiter {
			return l.buffer[startingIndex:l.nextByteIndex], nil
		}

		_, err = l.nextByte()
		if err != nil {
			return []byte{}, err
		}
	}
}

func (l *lexer) expectCRLF() error {
	if err := l.expectByte('\r'); err != nil {
		return err
	}
	if err := l.expectByte('\n'); err != nil {
		return err
	}
	return nil
}

func (l *lexer) expectByte(expectedByte byte) error {
	b, err := l.nextByte()
	if err != nil {
		return err
	}
	if b != expectedByte {
		return ErrInvalidProtocolHeader
	}
	return nil
}

func isValidProtocolVersion1InetProtocolAndFamily(v string) bool {
	return v == protocolVersion1Tcp4 || v == protocolVersion1Tcp6 || v == protocolVersion1Unknown
}

func isProtocolVersion1(buffer []byte) bool {
	return len(buffer) >= minBytesForProtocolVersion1 && len(buffer) <= maxBytesForProtocolVersion1 &&
		bytes.Equal(buffer[:len(protocolVersion1HeaderSignature)], protocolVersion1HeaderSignature)
}

func isProtocolVersion2(buffer []byte) bool {
	return len(buffer) >= minBytesForProtocolVersion2 &&
		bytes.Equal(buffer[:len(protocolVersion2HeaderSignature)], protocolVersion2HeaderSignature)
}
