TAGS = sqlite fts5
DIRS = ./core ./sqlite

all:
	go test -i -v -tags "$(TAGS)" $(DIRS)

deps:
	go get -v -d -tags "$(TAGS)" $(DIRS)

test:
	go test -v -tags "$(TAGS)" $(DIRS)

_testdata/testdata.txt:
	go get -d ./_testdata
	go run ./_testdata/main.go -f _testdata/testdata.txt

testlarge:	_testdata/testdata.txt
	MARKOV_TESTFILE=$(PWD)/_testdata/testdata.txt go test -v -tags "$(TAGS)" ./sqlite -run TestLarge

bench:
	go test -v -tags "$(TAGS)" -bench . -run - $(DIRS)

clean:
	rm -f markov.db* _testdata/testdata.txt

.PHONY: all deps test testlarge bench clean

