package core

import (
	"bytes"
	"strings"
)

// Prefix is a Markov chain prefix of one or more words.
type TextTuple struct {
	e     []string
	order int
}

func NewTupleFromSlice(s []string) *TextTuple {
	return &TextTuple{s, len(s)}
}

func NewTextTuple(order int) *TextTuple {
	tp := make([]string, order)
	return &TextTuple{
		e:     tp,
		order: order,
	}
}

func (t *TextTuple) Copy() *TextTuple {
	tp := NewTextTuple(t.order)
	copy(tp.e, t.e)
	return tp
}

func (t *TextTuple) Shift(word string, d Direction) {
	switch d {
	case Forward:
		copy(t.e, t.e[1:])
		t.e[len(t.e)-1] = word
	case Backward:
		copy(t.e[1:], t.e)
		t.e[0] = word
	}
}

func (t *TextTuple) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *TextTuple) UnmarshalBinary(p []byte) error {
	f := bytes.Fields(p)
	tt := NewTextTuple(len(f))
	for i, w := range f {
		tt.e[i] = string(w)
	}

	t.e = tt.e
	t.order = tt.order

	return nil
}

func (t *TextTuple) MarshalText() ([]byte, error) {
	return t.MarshalBinary()
}

func (t *TextTuple) UnmarshalText(p []byte) error {
	return t.UnmarshalBinary(p)
}

func (t TextTuple) MarshalDB() (interface{}, error) {
	return t.MarshalText()
}

// sql.Scanner
func (t *TextTuple) Scan(i interface{}) error {
	return t.UnmarshalText(i.([]byte))
}

// String returns the Prefix as a string (for use as a map key).
func (t TextTuple) String() string {
	return strings.Join(t.e, " ")
}

func (t *TextTuple) Order() int {
	return t.order
}

func (t *TextTuple) Elements() []string {
	return t.e
}
