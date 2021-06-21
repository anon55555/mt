package mt

import "io"

type byteReader struct {
	io.Reader
}

func (br byteReader) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(br, buf)
	return buf[0], err
}
