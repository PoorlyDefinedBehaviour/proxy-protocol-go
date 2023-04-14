package proxyprotocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

const PROXY_WORD_LENGTH = 5

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
	protocolVersion1 = iota
	protocolVersion2 = iota
)

type header struct {
	version  protocolVersion
	srcPort  uint16
	destPort uint16
	src      net.IP
	dest     net.IP
}

func handleConnection(conn net.Conn) error {
	buffer := make([]byte, 108)

	bytesRead, err := conn.Read(buffer)

	buffer = buffer[:bytesRead]

	fmt.Printf("\n\naaaaaaa bytesRead %+v\n\n", bytesRead)
	fmt.Printf("\n\naaaaaaa err %+v\n\n", err)

	fmt.Printf("\n\naaaaaaa string(buffer) %+v\n\n", string(buffer))
	return nil
}

func parseProtocolHeader(buffer []byte) (header, error) {
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
			fmt.Printf("\n\naaaaaaa signature does not match %+v\n\n", string(signature))
			return header{}, ErrInvalidProtocolHeader
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			fmt.Printf("\n\naaaaaaa line 78 expected whitespace\n\n")
			return header{}, err
		}

		// Followed by a the INET protocol and family.
		inetProtocolAndFamily, err := lexer.readUntilDelimiter(' ')
		if err != nil {
			fmt.Printf("\n\naaaaaaa line 85 invalid inetProtocolAndFamily\n\n")
			return header{}, err
		}

		if !isValidProtocolVersion1InetProtocolAndFamily(string(inetProtocolAndFamily)) {
			fmt.Printf("\n\naaaaaaa inetProtocolAndFamily '%+v'\n\n", inetProtocolAndFamily)
			fmt.Printf("\n\naaaaaaa len(inetProtocolAndFamily) %+v\n\n", len(inetProtocolAndFamily))
			fmt.Printf("\n\naaaaaaa line 89 invalid inetProtocolAndFamily\n\n")
			return header{}, ErrInvalidProtocolHeader
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			fmt.Printf("\n\naaaaaaa line 78 expected whitespace\n\n")
			return header{}, err
		}

		// Followed by the layer 3 source address.
		srcAddress, err := lexer.readUntilDelimiter(' ')
		if err != nil {
			fmt.Printf("\n\naaaaaaa error reading srcAddress\n\n")
			return header{}, err
		}
		srcIP := net.ParseIP(string(srcAddress))
		if srcIP == nil {
			fmt.Printf("\n\naaaaaaa error parsing src ip\n\n")
			return header{}, ErrInvalidProtocolHeader
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			fmt.Printf("\n\naaaaaaa line 113 expected whitespace\n\n")
			return header{}, err
		}

		// Followed by the layer 3 destination address.
		destAddress, err := lexer.readUntilDelimiter(' ')
		if err != nil {
			fmt.Printf("\n\naaaaaaa error reading destAddress\n\n")
			return header{}, err
		}
		destIP := net.ParseIP(string(destAddress))
		if destIP == nil {
			fmt.Printf("\n\naaaaaaa error parsing srcAddress\n\n")
			return header{}, ErrInvalidProtocolHeader
		}

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			fmt.Printf("\n\naaaaaaa line 131 expected whitespace\n\n")
			return header{}, err
		}

		fmt.Printf("\n\naaaaaaa before reading srcPort\n\n")
		lexer.debugBuffer()

		srcPort, err := lexer.readUint16()
		if err != nil {
			fmt.Printf("\n\naaaaaaa error src port\n\n")
			return header{}, err
		}

		fmt.Printf("\n\naaaaaaa after reading srcPort\n\n")
		lexer.debugBuffer()

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			fmt.Printf("\n\naaaaaaa line 146 expected whitespace\n\n")
			return header{}, err
		}

		fmt.Printf("\n\naaaaaaa before reading destPort\n\n")
		lexer.debugBuffer()

		destPort, err := lexer.readUint16()
		if err != nil {
			fmt.Printf("\n\naaaaaaa error dest port: '%+v'\n\n", err)
			return header{}, err
		}

		fmt.Printf("\n\naaaaaaa after reading destPort\n\n")
		lexer.debugBuffer()

		// Followed by a single whitespace.
		if err := lexer.expectByte(' '); err != nil {
			fmt.Printf("\n\naaaaaaa line 160 expected whitespace\n\n")
			return header{}, err
		}

		fmt.Printf("\n\naaaaaaa after line 162 \n\n")
		lexer.debugBuffer()

		fmt.Printf("\n\naaaaaaa lexer.buffer %+v\n\n", string(lexer.buffer[lexer.nextByteIndex:]))
		if err := lexer.expectCRLF(); err != nil {
			fmt.Printf("\n\naaaaaaa expected CRLF\n\n")
			return header{}, err
		}

		return header{version: protocolVersion1, src: srcIP, dest: destIP, srcPort: srcPort, destPort: destPort}, nil
	}

	fmt.Printf("\n\naaaaaaa  protocol is not version 1 nor 2")
	return header{}, ErrInvalidProtocolHeader
}

type lexer struct {
	nextByteIndex int
	buffer        []byte
}

func newLexer(buffer []byte) lexer {
	return lexer{nextByteIndex: 0, buffer: buffer}
}

func (l *lexer) debugBuffer() {
	fmt.Printf("\n\naaaaaaa l.buffer '%+v'\n\n", string(l.buffer[l.nextByteIndex:]))
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
	fmt.Printf("\n\naaaaaaa *l %+v\n\n", *l)
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
		fmt.Printf("\n\naaaaaaa expectedByte read byte: %+v but expected: %+v\n\n", string(b), string(expectedByte))
		return ErrInvalidProtocolHeader
	}
	return nil
}

func isValidProtocolVersion1InetProtocolAndFamily(v string) bool {
	return v == protocolVersion1Tcp4 || v == protocolVersion1Tcp6 || v == protocolVersion1Unknown
}

func isProtocolVersion1(buffer []byte) bool {
	fmt.Printf("\n\naaaaaaa len(buffer) %+v\n\n", len(buffer))
	return len(buffer) >= minBytesForProtocolVersion1 && len(buffer) <= maxBytesForProtocolVersion1 &&
		bytes.Equal(buffer[:len(protocolVersion1HeaderSignature)], protocolVersion1HeaderSignature)
}

func isProtocolVersion2(buffer []byte) bool {
	return len(buffer) >= minBytesForProtocolVersion2 &&
		bytes.Equal(buffer[:len(protocolVersion2HeaderSignature)], protocolVersion2HeaderSignature)
}
