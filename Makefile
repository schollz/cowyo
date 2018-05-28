# Make a release with
# make -j4 release

VERSION=$(shell git describe)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: build
build: server/bindata.go
	go build ${LDFLAGS}

STATICFILES := $(wildcard static/*)
TEMPLATES := $(wildcard templates/*)
server/bindata.go: $(STATICFILES) $(TEMPLATES)
	go-bindata -pkg server -tags '!debug' -o server/bindata.go static/... templates/...
	go fmt

server/bindata-debug.go: $(STATICFILES) $(TEMPLATES)
	go-bindata -pkg server -tags 'debug' -o server/bindata-debug.go -debug static/... templates/...
	go fmt

.PHONY: devel
devel: server/bindata-debug.go
	go build -tags debug

.PHONY: quick
quick: server/bindata.go
	go build

.PHONY: linuxarm
linuxarm: server/bindata.go
	env GOOS=linux GOARCH=arm go build ${LDFLAGS} -o dist/cowyo_linux_arm
	#cd dist && upx --brute cowyo_linux_arm

.PHONY: linux32
linux32: server/bindata.go
	env GOOS=linux GOARCH=386 go build ${LDFLAGS} -o dist/cowyo_linux_32bit
	#cd dist && upx --brute cowyo_linux_32bit

.PHONY: linux64
linux64: server/bindata.go
	env GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_linux_amd64

.PHONY: windows
windows: server/bindata.go
	env GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_windows_amd64.exe
	#cd dist && upx --brute cowyo_windows_amd64.exe

.PHONY: osx
osx: server/bindata.go
	env GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_osx_amd64
	#cd dist && upx --brute cowyo_osx_amd64

.PHONY: release
release: osx windows linux64 linux32 linuxarm
