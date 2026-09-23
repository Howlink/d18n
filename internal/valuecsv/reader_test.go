package valuecsv

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReaderPreservesDatabaseValues(t *testing.T) {
	rows := [][]string{
		{"NAME", "NOTE", "AMOUNT"},
		{"plain", "line1\r\nline2", "12345678901234.123456"},
		{"quotes", "a,\"b\"\r\nc", "-0.000001"},
		{"carriage", "a\rb\r", "0"},
		{"linefeed", "a\nb\n", "1"},
		{"empty", "", ""},
		{"null marker", "NULL/*CDM_D18N_test*/", "\\N"},
		{"中文", "\x00\r\n\xff\\r\\n", "000123"},
		{"long", strings.Repeat("x", 8191) + "\r\nend", "1.230000"},
	}
	for _, comma := range []rune{',', '\t', '|', '界'} {
		t.Run(string(comma), func(t *testing.T) {
			var data bytes.Buffer
			w := csv.NewWriter(&data)
			w.Comma = comma
			if err := w.WriteAll(rows); err != nil {
				t.Fatal(err)
			}
			for _, input := range []io.Reader{bytes.NewReader(data.Bytes()), iotest.OneByteReader(bytes.NewReader(data.Bytes()))} {
				r := NewReader(input)
				r.Comma = comma
				for i, want := range rows {
					got, err := r.Read()
					if err != nil {
						t.Fatalf("row %d: %v", i, err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("row %d changed: got %q, want %q", i, got, want)
					}
				}
				if _, err := r.Read(); err != io.EOF {
					t.Fatalf("want EOF, got %v", err)
				}
			}
		})
	}
}

func TestReaderRecordBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        [][]string
	}{
		{"CRLF records", "a,b\r\n\r\n\"x\r\ny\",z\r\n", [][]string{{"a", "b"}, {"x\r\ny", "z"}}},
		{"blank lines", "\n\r\na,b\n\n\"\",\n", [][]string{{"a", "b"}, {"", ""}}},
		{"quoted EOF", "\"a\r\nb\"", [][]string{{"a\r\nb"}}},
		{"unquoted EOF", "a,b", [][]string{{"a", "b"}}},
		{"empty input", "", nil},
		{"trailing separator", "a,", [][]string{{"a", ""}}},
		{"quoted trailing CR", "\"a\"\r", [][]string{{"a"}}},
		{"unquoted trailing CR", "a,b\r", [][]string{{"a", "b"}}},
		{"empty quoted trailing CR", "\"\"\r", [][]string{{""}}},
		{"blank trailing CR", "\r", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReader(iotest.OneByteReader(strings.NewReader(tc.input)))
			var got [][]string
			for {
				row, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, row)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReaderRejectsMalformedCSV(t *testing.T) {
	for _, input := range []string{"a,\"unterminated", "a,b\"c\n", "a,\"b\"c\n", "a,b\nc\n", "a,b\nc,d,e\n"} {
		t.Run(input, func(t *testing.T) {
			r := NewReader(strings.NewReader(input))
			for {
				_, err := r.Read()
				if err == io.EOF {
					t.Fatal("accepted malformed CSV")
				}
				if err != nil {
					return
				}
			}
		})
	}
}

func TestReaderPropagatesReadFailure(t *testing.T) {
	want := errors.New("input failed")
	r := NewReader(io.MultiReader(strings.NewReader("a,\"line\r\n"), iotest.ErrReader(want)))
	if _, err := r.Read(); !errors.Is(err, want) {
		t.Fatalf("want read failure, got %v", err)
	}
}

func TestReaderRejectsInvalidDelimiter(t *testing.T) {
	for _, comma := range []rune{0, '"', '\r', '\n', -1, '\uFFFD'} {
		r := NewReader(strings.NewReader("a,b"))
		r.Comma = comma
		if _, err := r.Read(); err == nil {
			t.Fatalf("accepted delimiter %U", comma)
		}
	}
}

func TestReaderMatchesStrictCSVGrammar(t *testing.T) {
	// Exhaustively compare grammar with encoding/csv, allowing only its CRLF
	// normalization difference. Importers discard records returned with errors.
	alphabet := []byte{'a', ' ', ',', '"', '\r', '\n'}
	var check func([]byte, int)
	check = func(input []byte, remaining int) {
		a, b := NewReader(bytes.NewReader(input)), csv.NewReader(bytes.NewReader(input))
		for {
			got, gotErr := a.Read()
			want, wantErr := b.Read()
			if (gotErr == nil) != (wantErr == nil) || (gotErr == io.EOF) != (wantErr == io.EOF) {
				t.Fatalf("input %q: error %v, want %v", input, gotErr, wantErr)
			}
			if gotErr != nil {
				break
			}
			for i := range got {
				got[i] = strings.ReplaceAll(got[i], "\r\n", "\n")
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("input %q: got %q, want %q", input, got, want)
			}
		}
		if remaining > 0 {
			for _, c := range alphabet {
				check(append(input, c), remaining-1)
			}
		}
	}
	check(nil, 6)
}

func TestReaderGeneratedByteRoundTrips(t *testing.T) {
	rng := rand.New(rand.NewSource(20260923))
	for _, comma := range []rune{',', '\t', '|', '界'} {
		var rows [][]string
		for i := 0; i < 300; i++ {
			row := make([]string, 3)
			for j := range row {
				value := make([]byte, rng.Intn(128))
				if _, err := rng.Read(value); err != nil {
					t.Fatal(err)
				}
				row[j] = string(value) + "\r\n" + string(comma) + "\""
			}
			rows = append(rows, row)
		}
		var input bytes.Buffer
		w := csv.NewWriter(&input)
		w.Comma = comma
		if err := w.WriteAll(rows); err != nil {
			t.Fatal(err)
		}
		r := NewReader(iotest.OneByteReader(bytes.NewReader(input.Bytes())))
		r.Comma = comma
		for i, want := range rows {
			got, err := r.Read()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("delimiter %U row %d changed bytes", comma, i)
			}
		}
		if _, err := r.Read(); err != io.EOF {
			t.Fatalf("want EOF, got %v", err)
		}
	}
}
