// Package mt implements the high-level Minetest protocol.
// This version is compatible with
// https://github.com/ClamityAnarchy/minetest/commit/66adeade9d5c45a5499b5ad1ad4bd91dae82482a.
package mt

type Node struct {
	Param0         Content
	Param1, Param2 uint8
}

type Content uint16

const (
	Unknown Content = 125
	Air     Content = 126
	Ignore  Content = 127
)

type Group struct {
	Name   string
	Rating int16
}

type Field struct {
	Name string

	//mt:len32
	Value string
}
