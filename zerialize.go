// This file is called zerialize.go so the following go:generate runs last.

//go:generate ./mkserialize.sh

package mt

import (
	"encoding/binary"
	"errors"
	"io"
)

// ErrTooLong reports a length that is too long to serialize.
var ErrTooLong = errors.New("len too long")

var be = binary.BigEndian

type serializer interface {
	serialize(w io.Writer)
}

func serialize(w io.Writer, s interface{}) error {
	return pcall(func() { s.(serializer).serialize(w) })
}

type deserializer interface {
	deserialize(r io.Reader)
}

func deserialize(r io.Reader, d interface{}) error {
	return pcall(func() { d.(deserializer).deserialize(r) })
}

type serializationError struct {
	error
}

func pcall(f func()) (rerr error) {
	defer func() {
		switch r := recover().(type) {
		case serializationError:
			rerr = r.error
		case nil:
		default:
			panic(r)
		}
	}()
	f()
	return
}

func chk(err error) {
	if err != nil {
		panic(serializationError{err})
	}
}
