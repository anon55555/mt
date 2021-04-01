package mt

import (
	"fmt"
	"io"
	"reflect"
)

type Inv []NamedInvList

type NamedInvList struct {
	Name string
	InvList
}

func (inv Inv) List(name string) *NamedInvList {
	for i, l := range inv {
		if l.Name == name {
			return &inv[i]
		}
	}
	return nil
}

func (i Inv) Serialize(w io.Writer) error {
	return i.SerializeKeep(w, nil)
}

func (i Inv) SerializeKeep(w io.Writer, old Inv) error {
	ew := &errWriter{w: w}

	for _, l := range i {
		if reflect.DeepEqual(&i, old.List(l.Name)) {
			fmt.Fprintln(ew, "KeepList", l.Name)
			continue
		}

		fmt.Fprintln(ew, "List", l.Name, len(l.Stacks))
		l.Serialize(ew)
	}
	fmt.Fprintln(ew, "EndInventory")

	return ew.err
}

func (i *Inv) Deserialize(r io.Reader) (err error) {
	s := new(sentinal)
	defer s.recover(&err)

	old := *i
	*i = nil

	for {
		if err := readCmdLn(r, map[string]interface{}{
			"List": func(name string, size int) {
				l := old.List(name)
				if l == nil {
					l = &NamedInvList{Name: name}
				}

				if err := l.Deserialize(r); err != nil {
					s.ret(fmt.Errorf("List %s %d: %w", name, size, err))
				}
				if len(l.Stacks) != size {
					s.ret(fmt.Errorf("List %s %d: contains %d stacks", name, size, len(l.Stacks)))
				}

				*i = append(*i, *l)
			},
			"KeepList": func(name string) {
				l := old.List(name)
				if l == nil {
					s.ret(fmt.Errorf("KeepList %s: list does not exist", name))
				}

				*i = append(*i, *l)
			},
			"EndInventory": func() {
				s.ret(nil)
			},
		}); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
	}
}

type InvList struct {
	Width  int
	Stacks []Stack
}

func (l InvList) Serialize(w io.Writer) error {
	return l.SerializeKeep(w, InvList{})
}

func (l InvList) SerializeKeep(w io.Writer, old InvList) error {
	ew := &errWriter{w: w}

	fmt.Fprintln(ew, "Width", l.Width)
	for i, s := range l.Stacks {
		if i < len(old.Stacks) && s == old.Stacks[i] {
			fmt.Fprintln(ew, "Keep")
		}

		if s.Count > 0 {
			fmt.Fprintln(ew, "Item", s)
		} else {
			fmt.Fprintln(ew, "Empty")
		}
	}
	fmt.Fprintln(ew, "EndInventoryList")

	return ew.err
}

func (l *InvList) Deserialize(r io.Reader) (err error) {
	s := new(sentinal)
	defer s.recover(&err)

	if _, err := fmt.Fscanf(r, "Width %d\n", &l.Width); err != nil {
		return err
	}

	old := l.Stacks
	l.Stacks = nil

	for {
		if err := readCmdLn(r, map[string]interface{}{
			"Empty": func() {
				l.Stacks = append(l.Stacks, Stack{})
			},
			"Item": func(stk Stack) {
				l.Stacks = append(l.Stacks, stk)
			},
			"Keep": func() {
				if i := len(l.Stacks); i < len(old) {
					l.Stacks = append(l.Stacks, old[i])
				} else {
					l.Stacks = append(l.Stacks, Stack{})
				}
			},
			"EndInventoryList": func() {
				s.ret(nil)
			},
		}); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
	}
}

func readCmdLn(r io.Reader, cmds map[string]interface{}) error {
	if _, ok := r.(io.RuneScanner); !ok {
		r = &readRune{Reader: r, peekRune: -1}
	}

	var cmd string
	if _, err := fmt.Fscan(r, &cmd); err != nil {
		return err
	}

	f, ok := cmds[cmd]
	if !ok {
		return fmt.Errorf("unsupported line type: %+q", cmd)
	}

	t := reflect.TypeOf(f)

	a := make([]interface{}, t.NumIn())
	for i := range a {
		a[i] = reflect.New(t.In(i)).Interface()
	}
	fmt.Fscanln(r, a...)

	args := make([]reflect.Value, t.NumIn())
	for i := range args {
		args[i] = reflect.ValueOf(a[i]).Elem()
	}
	reflect.ValueOf(f).Call(args)

	return nil
}

type sentinal struct {
	err error
}

func (s *sentinal) ret(err error) {
	s.err = err
	panic(s)
}

func (s *sentinal) recover(p *error) {
	if r := recover(); r != nil {
		if r == s {
			*p = s.err
		} else {
			panic(r)
		}
	}
}

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}

	n, err := ew.w.Write(p)
	if err != nil {
		ew.err = err
	}
	return n, err
}
