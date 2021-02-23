// In this file, JSON refers to WTF-JSON, a variant of JSON used by Minetest
// where \u00XX escapes in string literals act like Go's \xXX escapes.

package mt

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Stack struct {
	Item
	Count uint16
}

type Item struct {
	Name string
	Wear uint16
	ItemMeta
}

type ItemMeta string

func (m ItemMeta) Field(name string) (s string, ok bool) {
	if len(m) > 0 && m[0] == 1 {
		m = m[1:]
		eat := func(stop byte) string {
			for i := 0; i < len(m); i++ {
				if m[i] == stop {
					defer func() {
						m = m[i+1:]
					}()
					return string(m[:i])
				}
			}
			defer func() {
				m = ""
			}()
			return string(m)
		}
		for len(m) > 0 {
			if eat(2) == name {
				s = eat(3)
				ok = true
			}
		}
		return
	}

	if name == "" {
		return string(m), true
	}

	return "", false
}

func (s Stack) String() string {
	if s.Count == 0 {
		return ""
	}

	n := 1
	if s.ItemMeta != "" {
		n = 4
	} else if s.Wear > 0 {
		n = 3
	} else if s.Count > 1 {
		n = 2
	}

	return strings.Join([]string{
		optJSONStr(s.Name),
		fmt.Sprint(s.Count),
		fmt.Sprint(s.Wear),
		optJSONStr(string(s.ItemMeta)),
	}[:n], " ")
}

func optJSONStr(s string) string {
	for _, r := range s {
		if r <= ' ' || r == '"' || r >= utf8.RuneSelf {
			return jsonStr(s)
		}
	}
	return s
}

func jsonStr(s string) string {
	esc := [256]byte{
		'\\': '\\',
		'"':  '"',
		'/':  '/',
		'\b': 'b',
		'\f': 'f',
		'\n': 'n',
		'\r': 'r',
		'\t': 't',
	}

	b := new(strings.Builder)

	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case esc[c] != 0:
			fmt.Fprintf(b, "\\%c", esc[c])
		case ' ' <= c && c <= '~':
			b.WriteByte(c)
		default:
			fmt.Fprintf(b, "\\u%04x", c)
		}
	}
	b.WriteByte('"')

	return b.String()
}

func (stk *Stack) Scan(state fmt.ScanState, verb rune) (err error) {
	*stk = Stack{}

	defer func() {
		if err == io.EOF {
			err = nil
		}
	}()

	nm, err := scanOptJSONStr(state)
	if err != nil {
		return err
	}
	stk.Name = nm
	stk.Count = 1

	if _, err := fmt.Fscan(state, &stk.Count, &stk.Wear); err != nil {
		return err
	}

	s, err := scanOptJSONStr(state)
	if err != nil {
		return err
	}
	stk.ItemMeta = ItemMeta(s)

	return nil
}

func scanOptJSONStr(state fmt.ScanState) (string, error) {
	state.SkipSpace()

	r, _, err := state.ReadRune()
	if err != nil {
		return "", err
	}
	state.UnreadRune()

	if r == '"' {
		return scanJSONStr(state)
	}

	token, err := state.Token(false, func(r rune) bool {
		return r != ' ' && r != '\n'
	})
	return string(token), err
}

func scanJSONStr(state fmt.ScanState) (s string, rerr error) {
	r, _, err := state.ReadRune()
	if err != nil {
		return "", err
	}
	if r != '"' {
		return "", fmt.Errorf("unexpected rune: %q", r)
	}

	defer func() {
		if rerr == io.EOF {
			rerr = io.ErrUnexpectedEOF
		}
	}()

	b := new(strings.Builder)
	for {
		r, _, err := state.ReadRune()
		if err != nil {
			return b.String(), err
		}

		switch r {
		case '"':
			return b.String(), nil
		case '\\':
			r, _, err := state.ReadRune()
			if err != nil {
				return b.String(), err
			}

			switch r {
			case '\\', '"', '/':
				b.WriteRune(r)
			case 'b':
				b.WriteRune('\b')
			case 'f':
				b.WriteRune('\f')
			case 'n':
				b.WriteRune('\n')
			case 'r':
				b.WriteRune('\r')
			case 't':
				b.WriteRune('\t')
			case 'u':
				var hex [4]rune
				for i := range hex {
					r, _, err := state.ReadRune()
					if err != nil {
						return b.String(), err
					}
					hex[i] = r
				}
				n, err := strconv.ParseUint(string(hex[:]), 16, 8)
				if err != nil {
					return b.String(), err
				}
				b.WriteByte(byte(n))
			default:
				return b.String(), fmt.Errorf("invalid escape: \\%c", r)
			}
		default:
			b.WriteRune(r)
		}
	}
}
