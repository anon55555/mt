package mt

import (
	"fmt"
	"io"
	"reflect"
)

type sentinal struct {
	err error
}

func (s *sentinal) ret(err error) {
	s.err = err
	panic(s)
}

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
	for _, l := range i {
		if _, err := fmt.Fprintln(w, "List", l.Name, len(l.Stacks)); err != nil {
			return err
		}
		if err := l.Serialize(w); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "EndInventory")
	return err
}

func (i *Inv) Deserialize(r io.Reader) (err error) {
	s := new(sentinal)
	defer func() {
		r := recover()
		if r, ok := r.(sentinal); ok {
			err = r.err
		}
	}()

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
				s.ret(io.ErrUnexpectedEOF)
			}
			s.ret(err)
		}
	}
}

type InvList struct {
	Width  int
	Stacks []Stack
}

func (l InvList) Serialize(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "Width", l.Width); err != nil {
		return err
	}
	for _, i := range l.Stacks {
		if i.Count > 0 {
			if _, err := fmt.Fprintln(w, "Item", i); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w, "Empty"); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(w, "EndInventoryList")
	return err
}

func (i *InvList) Deserialize(r io.Reader) (err error) {
	s := new(sentinal)
	defer func() {
		r := recover()
		if r, ok := r.(sentinal); ok {
			err = r.err
		}
	}()

	if _, err := fmt.Fscanf(r, "Width %d\n", &i.Width); err != nil {
		s.ret(err)
	}

	i.Stacks = i.Stacks[:0]

	for {
		if err := readCmdLn(r, map[string]interface{}{
			"Empty": func() {
				i.Stacks = append(i.Stacks, Stack{})
			},
			"Item": func(stk Stack) {
				i.Stacks = append(i.Stacks, stk)
			},
			"Keep": func() {
				if len(i.Stacks) < cap(i.Stacks) {
					i.Stacks = i.Stacks[:len(i.Stacks)+1]
				} else {
					i.Stacks = append(i.Stacks, Stack{})
				}
			},
			"EndInventoryList": func() {
				s.ret(nil)
			},
		}); err != nil {
			if err == io.EOF {
				s.ret(io.ErrUnexpectedEOF)
			}
			s.ret(err)
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
