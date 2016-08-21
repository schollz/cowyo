SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')

BINARY=cowyo

VERSION=1.1
BUILD_TIME=`date +%FT%T%z`
BUILD=`git rev-parse HEAD`

LDFLAGS=-ldflags "-X main.VersionNum=${VERSION} -X main.Build=${BUILD} -X main.BuildTime=${BUILD_TIME}"

.DEFAULT_GOAL: $(BINARY)

$(BINARY): $(SOURCES)
	go get github.com/boltdb/bolt
	go get github.com/gin-gonic/contrib/sessions
	go get github.com/gin-gonic/gin
	go get github.com/gorilla/websocket
	go get github.com/microcosm-cc/bluemonday
	go get github.com/russross/blackfriday
	go get github.com/sergi/go-diff/diffmatchpatch
	go build ${LDFLAGS} -o ${BINARY} ${SOURCES}

.PHONY: clean
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	rm -rf binaries

.PHONY: binaries
binaries:
	rm -rf binaries
	rm -f cowyo
	mkdir binaries
	env GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY} ${SOURCES}
	zip -9 -r cowyo-linux-64bit.zip cowyo static/* templates/*
	rm -f cowyo
	env GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY} ${SOURCES}
	zip -9 -r cowyo-windows-64bit.zip cowyo.exe static/* templates/*
	rm -f cowyo.exe
	env GOOS=linux GOARCH=arm go build ${LDFLAGS} -o ${BINARY} ${SOURCES}
	zip -9 -r cowyo-raspberrypi.zip cowyo static/* templates/*
	rm -f cowyo
	env GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY} ${SOURCES}
	zip -9 -r cowyo-macosx-64bit.zip cowyo static/* templates/*
	rm -f cowyo
	mv *.zip binaries/
