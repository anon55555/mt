package mt

import (
	"encoding/json"
	"strings"
)

type ItemMeta string

var sanitizer = strings.NewReplacer(
	string(1), "",
	string(2), "",
	string(3), "",
)

func NewItemMeta(fields []Field) ItemMeta {
	if len(fields) == 0 {
		return ""
	}

	b := new(strings.Builder)
	b.WriteByte(1)
	for _, f := range fields {
		sanitizer.WriteString(b, f.Name)
		b.WriteByte(2)
		sanitizer.WriteString(b, f.Value)
		b.WriteByte(3)
	}
	return ItemMeta(b.String())
}

func (m ItemMeta) Fields() []Field {
	var f []Field
	if len(m) > 0 && m[0] == 1 {
		m = m[1:]
		eat := func(stop byte) string {
			if i := strings.IndexByte(string(m), stop); i != -1 {
				defer func() {
					m = m[i+1:]
				}()
				return string(m[:i])
			}

			defer func() {
				m = ""
			}()
			return string(m)
		}
		for len(m) > 0 {
			f = append(f, Field{eat(2), eat(3)})
		}
		return f
	}

	return []Field{{"", string(m)}}
}

func (m ItemMeta) Field(name string) (s string, ok bool) {
	for _, f := range m.Fields() {
		if f.Name == name {
			s, ok = f.Value, true
		}
	}
	return
}

func (m *ItemMeta) SetField(name, value string) {
	var fields []Field
	for _, f := range m.Fields() {
		if f.Name != name {
			fields = append(fields, f)
		}
	}
	fields = append(fields, Field{name, value})
	*m = NewItemMeta(fields)
}

func (m ItemMeta) ToolCaps() (ToolCaps, bool) {
	f, ok := m.Field("tool_capabilities")
	if !ok {
		return ToolCaps{}, false
	}

	var tc ToolCaps
	if err := json.Unmarshal([]byte(f), &tc); err != nil {
		return tc, false
	}
	return tc, true
}

func (m *ItemMeta) SetToolCaps(tc ToolCaps) {
	b, err := tc.MarshalJSON()
	if err != nil {
		panic(err)
	}

	m.SetField("tool_capabilities", string(b))
}
