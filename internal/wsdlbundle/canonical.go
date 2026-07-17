package wsdlbundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"unicode/utf16"
	"unicode/utf8"
)

var canonicalInteger = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// canonicalJSON implements the RFC 8785 rules needed by the evidence model.
// Evidence contains only objects, arrays, strings, booleans, null, and integer
// counts; non-integer JSON numbers are rejected instead of being approximated.
func canonicalJSON(value interface{}) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var decoded interface{}
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	var result bytes.Buffer
	if err := writeCanonical(&result, decoded); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func writeCanonical(target *bytes.Buffer, value interface{}) error {
	switch typed := value.(type) {
	case nil:
		target.WriteString("null")
	case bool:
		if typed {
			target.WriteString("true")
		} else {
			target.WriteString("false")
		}
	case string:
		return writeCanonicalString(target, typed)
	case json.Number:
		if !canonicalInteger.MatchString(typed.String()) {
			return fmt.Errorf("non-integer number is outside the evidence canonicalization profile")
		}
		target.WriteString(typed.String())
	case []interface{}:
		target.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				target.WriteByte(',')
			}
			if err := writeCanonical(target, item); err != nil {
				return err
			}
		}
		target.WriteByte(']')
	case map[string]interface{}:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		target.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				target.WriteByte(',')
			}
			if err := writeCanonicalString(target, key); err != nil {
				return err
			}
			target.WriteByte(':')
			if err := writeCanonical(target, typed[key]); err != nil {
				return err
			}
		}
		target.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}

func writeCanonicalString(target *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("invalid UTF-8 string")
	}
	target.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"', '\\':
			target.WriteByte('\\')
			target.WriteRune(r)
		case '\b':
			target.WriteString(`\b`)
		case '\t':
			target.WriteString(`\t`)
		case '\n':
			target.WriteString(`\n`)
		case '\f':
			target.WriteString(`\f`)
		case '\r':
			target.WriteString(`\r`)
		default:
			if r < 0x20 {
				fmt.Fprintf(target, `\u%04x`, r)
			} else {
				target.WriteRune(r)
			}
		}
	}
	target.WriteByte('"')
	return nil
}

func utf16Less(left, right string) bool {
	l := utf16.Encode([]rune(left))
	r := utf16.Encode([]rune(right))
	limit := len(l)
	if len(r) < limit {
		limit = len(r)
	}
	for index := 0; index < limit; index++ {
		if l[index] != r[index] {
			return l[index] < r[index]
		}
	}
	return len(l) < len(r)
}
