// +build sqlite

package sqlite

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/mischief/markov/core"
)

func getTemp(t testing.TB) (path string, remover func()) {
	dbname := os.Getenv("MARKOV_DB")
	if dbname != "" {
		return dbname, func() {}
	}

	tmpfile, err := ioutil.TempFile("", "markov")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("tmpfile %q created", tmpfile.Name())

	c := func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Logf("tmpfile %q removed", tmpfile.Name())
		}
	}

	return tmpfile.Name(), c
}

func getDB(t testing.TB) (*sqlitedb, func()) {
	tmp, c := getTemp(t)
	sql, err := NewSQLite(tmp)
	if err != nil {
		c()
		t.Fatal(err)
	}

	s := func() {
		c()
		sql.Close()
	}

	return sql, s
}

func TestSQliteOpen(t *testing.T) {
	sql, c := getDB(t)
	defer c()
	t.Logf("%T %+v", sql.db.Driver(), sql)
}

func processPrefixes(r io.Reader, cb func(*SQLPrefix, string), order int) error {
	ord := order
	pre := core.NewTextTuple(ord)

	sl := bufio.NewScanner(r)
	for sl.Scan() {
		t := sl.Text()
		words := 0
		spl := strings.SplitN(t, " ", 2)
		author := spl[0]
		line := spl[1]
		sw := bufio.NewScanner(strings.NewReader(line))
		sw.Split(bufio.ScanWords)
		for sw.Scan() {
			words++
			if words < ord+1 {
				pre.Shift(sw.Text(), core.Forward)
				continue
			}

			p := &SQLPrefix{O: ord, A: author}
			p.T = pre.Copy()
			cb(p, sw.Text())

			pre.Shift(sw.Text(), core.Forward)
		}
	}

	return nil
}

func TestSQliteAddPrefix(t *testing.T) {
	sql, c := getDB(t)
	defer c()

	st := strings.NewReader("foo bar baz foo baz bar foo bar bar baz bar baz foo")
	sql.Ingest(st, 2, "test")

	st = strings.NewReader("baz foo bar baz baz foo foo foo bar baz")
	sql.Ingest(st, 2, "test")

	res, err := sql.PrefixContaining("baz")
	if err != nil {
		t.Fatalf("%T %v", err, err)
	}

	for _, r := range res {
		t.Logf("%+v", r)
	}

	if len(res) != 5 {
		t.Fatalf("expected 5 results, got %d", len(res))
	}

	rp, err := sql.RandomPrefix()
	if err != nil {
		t.Fatalf("%T %v", err, err)
	}

	for i := 0; i < 5; i++ {
		t.Logf("random prefix: %+v %v", rp, err)
		sf, err := sql.RandomSuffix(rp)
		if err != nil {
			t.Fatalf("%T %v", err, err)
		}

		t.Logf("sf %+v", sf)

		rp.T.Shift(sf.Word, core.Forward)
	}
}

var testdata = []string{
	"foo",
	"bar",
	"baz",
	"quux",

	/*
		"foo",
		"bar",
		"baz",
		"quux",
	*/
}

func getRandData() (*SQLPrefix, *SQLSuffix) {
	dst := make([]string, len(testdata))
	perm := rand.Perm(len(testdata))
	for i, v := range perm {
		dst[v] = testdata[i]
	}

	t := core.NewTupleFromSlice(dst[:3])

	sp := &SQLPrefix{0, t, 3, "user"}
	ss := &SQLSuffix{Word: dst[3], Count: 1}

	return sp, ss
}

func TestRandData(t *testing.T) {
	p, s := getRandData()
	t.Logf("%s %+v", p, s)
}

func TestIngestGenerate(t *testing.T) {
	sql, c := getDB(t)
	defer c()

	txt := "user The quick brown fox jumps over the lazy dog"
	sr := strings.NewReader(txt)
	if err := sql.Ingest(sr, 3, "test"); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := sql.Generate(buf, "The quick"); err != nil {
		t.Fatal(err)
	}

	if txt != buf.String() {
		t.Fatalf("expected %q got %q", txt, buf.String())
	}
}

func BenchmarkProcessPrefixes(b *testing.B) {
	sql, c := getDB(b)
	defer c()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pre, suf := getRandData()
		err := sql.AddPrefix(pre)
		if err != nil {
			b.Fatalf("%T %+v", err, err)
		}

		suf.Prefix = pre.ID
		err = sql.IncrementSuffix(suf)
		if err != nil {
			b.Fatalf("%T %+v", err, err)
		}
	}
}

func getLines(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	s := make([]string, 0)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		s = append(s, sc.Text())
	}

	return s, sc.Err()
}

func TestLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	testfile := "testdata.txt"
	if s := os.Getenv("MARKOV_TESTFILE"); s != "" {
		testfile = s
	}

	f, err := os.Open(testfile)
	if err != nil {
		t.Logf("cannot open test file %q: %v", testfile, err)
		t.Skip()
	}

	defer f.Close()

	sql, c := getDB(t)
	defer c()

	if err := sql.Ingest(f, 3, ""); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := sql.Generate(buf); err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", buf)
}
