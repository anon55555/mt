package mt

import "strings"

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
