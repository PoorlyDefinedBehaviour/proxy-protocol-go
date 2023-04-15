package proxyprotocol

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

type parser struct {
	nextByteIndex int
	reader        *bufio.Reader
}

func newParser(reader *bufio.Reader) parser {
	return parser{nextByteIndex: 0, reader: reader}
}

func (l *parser) readUint16() (uint16, error) {
	buffer := make([]byte, 0)

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

		b, err := l.nextByte()
		if err != nil {
			return 0, err
		}
		buffer = append(buffer, b)
	}

	n, err := strconv.ParseUint(string(buffer), 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(n), nil
}

func (l *parser) peekNextByte() (byte, error) {
	buffer, err := l.reader.Peek(1)
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}

func (l *parser) readBytes(n int) ([]byte, error) {
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

func (l *parser) nextByte() (byte, error) {
	b, err := l.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	l.nextByteIndex += 1
	return b, nil
}

func (l *parser) readUntilCRLF() ([]byte, error) {
	buffer := make([]byte, 0)

	for {
		b, err := l.nextByte()
		if err != nil {
			return buffer, err
		}

		if b == '\r' {
			nextByte, err := l.peekNextByte()
			if err != nil {
				return buffer, err
			}
			if nextByte == '\n' {
				// Consume '\n'
				_, err := l.nextByte()
				if err != nil {
					return buffer, err
				}
				return buffer, nil
			}
		}

	}
}

func (l *parser) readUntilDelimiter(delimiter byte) ([]byte, error) {
	buffer := make([]byte, 0)

	for {
		nextByte, err := l.peekNextByte()
		if err != nil {
			return []byte{}, err
		}
		if nextByte == delimiter {
			return buffer, nil
		}

		b, err := l.nextByte()
		if err != nil {
			return []byte{}, err
		}
		buffer = append(buffer, b)
	}
}

func (l *parser) expectCRLF() error {
	if err := l.expectByte('\r'); err != nil {
		return err
	}
	if err := l.expectByte('\n'); err != nil {
		return err
	}
	return nil
}

func (l *parser) expectByte(expectedByte byte) error {
	b, err := l.nextByte()
	if err != nil {
		return err
	}
	if b != expectedByte {
		return ErrInvalidProtocolHeader
	}
	return nil
}
