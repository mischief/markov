package core

import (
	"testing"
)

var _ Tuple = &TextTuple{nil, 0}

func checkPrefix(t *testing.T, p Tuple, str string) {
	s, err := p.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if string(s) != str {
		t.Fatalf("expected %q, got %q", str, s)
	}
}

func TestPrefixForward(t *testing.T) {
	d := Forward
	p := NewTextTuple(3)
	words := []string{"foo", "bar", "baz"}

	for _, w := range words {
		p.Shift(w, d)
	}

	checkPrefix(t, p, "foo bar baz")
	p.Shift("quux", d)
	checkPrefix(t, p, "bar baz quux")
	for _, w := range words {
		p.Shift(w, d)
	}
	checkPrefix(t, p, "foo bar baz")
}

func TestPrefixBackward(t *testing.T) {
	d := Backward
	p := NewTextTuple(3)
	words := []string{"foo", "bar", "baz"}

	for _, w := range words {
		p.Shift(w, d)
	}

	checkPrefix(t, p, "baz bar foo")
	p.Shift("quux", d)
	checkPrefix(t, p, "quux baz bar")
	for _, w := range words {
		p.Shift(w, d)
	}
	checkPrefix(t, p, "baz bar foo")
}
