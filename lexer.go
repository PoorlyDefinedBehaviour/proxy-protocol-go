package proxyprotocol

import (
	"errors"
	"io"
	"strconv"
)

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
