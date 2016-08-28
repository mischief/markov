// download test data for markov chains
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/mischief/botbot"
	"golang.org/x/net/context"
)

var (
	month   = flag.String("m", "2016-08", "month to download data for")
	timeout = flag.Duration("t", 10*time.Minute, "timeout for download")
	file    = flag.String("f", "test.txt", "output file")
)

func m2d(yr int, m time.Month) int {
	return time.Date(yr, m, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, -1).Day()
}

func die(str string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, str+"\n", v...)
	os.Exit(1)
}

func main() {
	var wc io.WriteCloser

	flag.Parse()

	mt, err := time.Parse("2006-01", *month)
	if err != nil {
		die("invalid month %q: %v", *month, err)
	}

	if *file != "" {
		wc, err = os.OpenFile(*file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			die("failed to open output file %q: %v", *file, err)
		}

		defer wc.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	mc, ec := botbot.GetMessageStreamMonth(ctx, "freenode", "go-nuts", mt.Year(), mt.Month())

	bar := pb.New(m2d(mt.Year(), mt.Month()))
	bar.ShowCounters = true
	bar.ShowTimeLeft = true
	bar.Start()

	lt := mt

	for m := range mc {
		cd := m.Timestamp.Day()
		if cd > lt.Day() {
			bar.Set(cd)
		}

		lt = m.Timestamp

		if wc != nil {
			fmt.Fprintf(wc, "%s %s\n", m.User, m.Text)
		}
	}

	bar.Finish()

	if err := <-ec; err != nil {
		die("fetch error: %v", err)
	}

}
