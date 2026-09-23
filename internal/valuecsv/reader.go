// Package valuecsv reads CSV without changing database field values.
package valuecsv

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"unicode/utf8"
)

// Reader accepts the strict CSV dialect used by the importer. Unlike
// encoding/csv.Reader, it preserves CRLF and arbitrary bytes inside fields.
// Blank physical lines are skipped and the first record fixes the field count.
type Reader struct {
	Comma        rune
	input        *bufio.Reader
	fields       int
	line, column int
}

func NewReader(input io.Reader) *Reader {
	return &Reader{Comma: ',', input: bufio.NewReader(input), line: 1}
}

func (r *Reader) readByte() (byte, error) {
	b, err := r.input.ReadByte()
	if err == nil {
		r.column++
		if b == '\n' {
			r.line++
			r.column = 0
		}
	}
	return b, err
}

func (r *Reader) Read() ([]string, error) {
	if r.Comma == 0 || r.Comma == '"' || r.Comma == '\r' || r.Comma == '\n' || r.Comma == utf8.RuneError || !utf8.ValidRune(r.Comma) {
		return nil, fmt.Errorf("csv: invalid field delimiter")
	}
	delimiter := []byte(string(r.Comma))
	startLine := r.line
	var record []string
	var value []byte
	quoted, closed, started := false, false, false
	parseError := func(err error) error {
		return &csv.ParseError{StartLine: startLine, Line: r.line, Column: r.column, Err: err}
	}
	finish := func() ([]string, error) {
		record = append(record, string(value))
		if r.fields == 0 {
			r.fields = len(record)
		}
		if len(record) != r.fields {
			return nil, parseError(csv.ErrFieldCount)
		}
		return record, nil
	}
	for {
		b, err := r.readByte()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			if quoted {
				return nil, parseError(csv.ErrQuote)
			}
			if !started && len(record) == 0 {
				return nil, io.EOF
			}
			return finish()
		}
		if quoted {
			if b == '"' {
				quoted = false
				closed = true
			} else {
				value = append(value, b)
			}
			continue
		}
		if closed && b == '"' {
			value = append(value, '"')
			quoted = true
			closed = false
			continue
		}
		if b == delimiter[0] {
			matches := len(delimiter) == 1
			if !matches {
				next, _ := r.input.Peek(len(delimiter) - 1)
				matches = bytes.Equal(next, delimiter[1:])
			}
			if matches {
				for i := 1; i < len(delimiter); i++ {
					if _, err = r.readByte(); err != nil {
						return nil, err
					}
				}
				record = append(record, string(value))
				value = nil
				started = false
				closed = false
				continue
			}
		}
		newline := b == '\n'
		if b == '\r' {
			next, peekErr := r.input.Peek(1)
			if len(next) == 1 && next[0] == '\n' {
				_, err = r.readByte()
				if err != nil {
					return nil, err
				}
				newline = true
			} else if len(next) == 0 && peekErr == io.EOF {
				// Match encoding/csv's trailing record CR convention. A CR
				// inside a quoted field is handled above and remains data.
				newline = true
			}
		}
		if newline {
			if !started && len(record) == 0 {
				startLine = r.line
				continue
			}
			return finish()
		}
		if closed {
			return nil, parseError(csv.ErrQuote)
		}
		if b == '"' {
			if started {
				return nil, parseError(csv.ErrBareQuote)
			}
			quoted = true
			started = true
			continue
		}
		value = append(value, b)
		started = true
	}
}
