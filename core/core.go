package core

import (
	"database/sql"
	"encoding"
	"io"
)

type Encodable interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	encoding.TextMarshaler
	encoding.TextUnmarshaler
	sql.Scanner
}

type Direction int

const (
	Forward Direction = iota
	Backward
)

type Tuple interface {
	Encodable
	String() string
	Shift(word string, d Direction)
	Order() int
	Elements() []string
}

type Suffix interface {
	Encodable
	String() string
}

type Database interface {
	io.Closer
	//Update(core.Tuple, core.Suffix) error
	//Get(core.Tuple) core.Suffix
	//VisitPrefixes(func(core.Tuple) bool)
	Ingest(r io.Reader, order int, author string) error
	Generate(wr io.Writer, w ...string) error
}
