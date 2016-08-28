// +build sqlite

package sqlite

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/mattn/go-sqlite3"
	"upper.io/db.v2"
	"upper.io/db.v2/lib/sqlbuilder"
	"upper.io/db.v2/sqlite"

	"github.com/mischief/markov/core"
)

var (
	ErrNoSuffixes = errors.New("markov: no suffixes")
	ErrNoPrefixes = errors.New("markov: no prefixes match")
)

var _ core.Database = (*sqlitedb)(nil)

type sqlitedb struct {
	db sqlbuilder.Database
}

func NewSQLite(path string) (*sqlitedb, error) {
	var settings = sqlite.ConnectionURL{
		Database: path,
	}

	sess, err := sqlite.Open(settings)
	if err != nil {
		return nil, err
	}

	for _, s := range schema {
		_, err = sess.Driver().(*sql.DB).Exec(s)
		if err != nil {
			return nil, err
		}
	}

	return &sqlitedb{sess}, nil
}

func (s *sqlitedb) Close() error {
	return s.db.Close()
}

func (s *sqlitedb) PrefixContaining(word string) ([]*SQLPrefix, error) {
	word = fmt.Sprintf(`"tuple": %s`, word)
	rows, err := s.db.Query(`SELECT rowid, tuple, author FROM tuples_idx WHERE tuples_idx MATCH ?`, word)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	it := sqlbuilder.NewIterator(rows)
	var prefixes []*SQLPrefix

	if err := it.All(&prefixes); err != nil {
		return nil, err
	}

	if len(prefixes) == 0 {
		return nil, ErrNoPrefixes
	}

	for _, p := range prefixes {
		p.O = p.T.Order()
	}

	return prefixes, nil
}

func (s *sqlitedb) randKey(table string) (int64, error) {
	var maxid int64
	row, err := s.db.QueryRow(fmt.Sprintf(`SELECT MAX(rowid) FROM %s`, table))
	if err != nil {
		return 0, err
	}

	if err := row.Scan(&maxid); err != nil {
		return 0, err
	}

	return 1 + rand.Int63n(maxid), nil
}

func (s *sqlitedb) RandomPrefix() (*SQLPrefix, error) {
	r, err := s.randKey("tuples")
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`SELECT id, tuple, ord, author FROM tuples WHERE id = ?`, r)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	it := sqlbuilder.NewIterator(rows)
	var p SQLPrefix
	var tt core.TextTuple
	err = it.ScanOne(&p.ID, &tt, &p.O, &p.A)
	p.T = &tt
	return &p, err
}

func (s *sqlitedb) RandomSuffix(p *SQLPrefix) (*SQLSuffix, error) {
	word := fmt.Sprintf(`"tuple": "%s"`, p.T)
	rows, err := s.db.Query(`SELECT tuple, word, count
	FROM suffixes
	WHERE tuple = (
		SELECT rowid FROM tuples_idx WHERE tuples_idx MATCH ?
	)
	ORDER BY RANDOM()
	LIMIT 1;
	`, word)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	it := sqlbuilder.NewIterator(rows)
	var suf SQLSuffix
	err = it.ScanOne(&suf.Prefix, &suf.Word, &suf.Count)
	if err == db.ErrNoMoreRows {
		return nil, ErrNoSuffixes
	}

	return &suf, err

}

func (s *sqlitedb) Suffixes(p *SQLPrefix) ([]*SQLSuffix, error) {
	word := fmt.Sprintf(`"tuple": %s`, p.T)
	rows, err := s.db.Query(`SELECT tuple, word, count
	FROM suffixes
	WHERE tuple = (
		SELECT rowid FROM tuples_idx WHERE tuples_idx MATCH ?
	)`, word)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	it := sqlbuilder.NewIterator(rows)
	var suf []*SQLSuffix
	return suf, it.All(&suf)
}

func (s *sqlitedb) AddPrefix(p *SQLPrefix) error {
	c := s.db.Collection("tuples")
	idx, err := c.Insert(p)
	if err != nil {
		if e, ok := err.(sqlite3.Error); ok {
			if e.Code != sqlite3.ErrConstraint {
				return err
			}
			rows, err := s.db.Query("SELECT rowid from tuples WHERE tuple = CAST(? as BLOB) AND author = ?", p.T.String(), p.A)
			if err != nil {
				return err
			}
			defer rows.Close()
			it := sqlbuilder.NewIterator(rows)
			return it.ScanOne(&p.ID)
		} else {
			return err
		}
	}
	p.ID = idx.(int64)
	return nil
}

func (s *sqlitedb) IncrementSuffix(suf *SQLSuffix) error {
	return s.db.Tx(func(sess sqlbuilder.Tx) error {
		c := sess.Collection("suffixes")
		idx, err := c.Insert(suf)
		if err != nil {
			if e, ok := err.(sqlite3.Error); ok {
				if e.Code != sqlite3.ErrConstraint {
					return err
				}
				_, err = sess.Exec("UPDATE suffixes SET count = count + 1 WHERE tuple = ? AND word = ?", suf.Prefix, suf.Word)
				return err
			}

			return err
		}

		suf.ID = idx.(int64)
		return err
	})
}

func (s *sqlitedb) processTuple(t *SQLPrefix, suf string) error {
	if err := s.AddPrefix(t); err != nil {
		return err
	}

	ssuf := &SQLSuffix{Prefix: t.ID, Word: suf, Count: 1}
	return s.IncrementSuffix(ssuf)
}

// Ingest reads text from r, splitting it into lines and then words.
// If author is empty, input lines are expected to be prefixed with the author.
// If author is non-empty, input lines should have no author prefix and all
// input will be attributed to that author.
//
// TODO: generalize around DB interface
func (s *sqlitedb) Ingest(r io.Reader, order int, author string) error {
	ord := order
	pre := core.NewTextTuple(ord)

	sl := bufio.NewScanner(r)

	ts := time.Now()
	for sl.Scan() {
		t := sl.Text()
		words := 0
		auth := author
		if author == "" {
			spl := strings.SplitN(t, " ", 2)
			auth = spl[0]
			t = spl[1]
		}

		sw := bufio.NewScanner(strings.NewReader(t))
		sw.Split(bufio.ScanWords)
		for sw.Scan() {
			words++
			if words < ord+1 {
				pre.Shift(sw.Text(), core.Forward)
				continue
			}

			p := &SQLPrefix{O: ord, A: auth}
			p.T = pre.Copy()

			s.processTuple(p, sw.Text())

			pre.Shift(sw.Text(), core.Forward)
		}

		now := time.Now()
		fmt.Printf("%-12s: %q\n", now.Sub(ts), t)
		ts = now
	}

	return sl.Err()
}

func (s *sqlitedb) Generate(wr io.Writer, w ...string) error {
	var p *SQLPrefix
	var err error
	if len(w) > 0 {
		ps, e := s.PrefixContaining(w[0])
		err = e
		if e == nil {
			p = ps[0]
		}
	} else {
		p, err = s.RandomPrefix()
	}

	if err != nil {
		return err
	}

	words := p.Words()

	for {
		sf, err := s.RandomSuffix(p)
		if err != nil {
			if err == ErrNoSuffixes {
				break
			}

			return err
		}

		words = append(words, sf.Word)
		p.T.Shift(sf.Word, core.Forward)
	}

	_, err = io.WriteString(wr, strings.Join(words, " "))
	return err
}

type SQLPrefix struct {
	ID int64 `db:"id,omitempty"`
	// prefix
	T *core.TextTuple `db:"tuple"`
	// order
	O int `db:"ord"`
	// author
	A string `db:"author"`
}

func (sp SQLPrefix) String() string {
	return fmt.Sprintf("%s(%d): %s", sp.A, sp.O, sp.T)
}

func (sp *SQLPrefix) Words() []string {
	w := make([]string, len(sp.T.Elements()))
	copy(w, sp.T.Elements())
	return w
}

type SQLSuffix struct {
	ID     int64  `db:"id,omitempty"`
	Prefix int64  `db:"tuple"`
	Word   string `db:"word"`
	Count  int64  `db:"count"`
}

func (ss SQLSuffix) String() string {
	return fmt.Sprintf("%s(%d)", ss.Word, ss.Count)
}
