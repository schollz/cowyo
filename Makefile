# Make a release with
# make -j4 release

VERSION=$(shell git describe)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: build
build: bindata.go
	go build ${LDFLAGS}

STATICFILES := $(wildcard static/*)
TEMPLATES := $(wildcard templates/*)
bindata.go: $(STATICFILES) $(TEMPLATES)
	go-bindata -tags '!debug' static/... templates/...

bindata-debug.go: $(STATICFILES) $(TEMPLATES)
	go-bindata -tags 'debug' -o bindata-debug.go -debug static/... templates/...

.PHONY: devel
devel: bindata-debug.go
	go build -tags debug

.PHONY: quick
quick: bindata.go
	go build

.PHONY: linuxarm
linuxarm: bindata.go
	env GOOS=linux GOARCH=arm go build ${LDFLAGS} -o dist/cowyo_linux_arm
	#cd dist && upx --brute cowyo_linux_arm

.PHONY: linux32
linux32: bindata.go
	env GOOS=linux GOARCH=386 go build ${LDFLAGS} -o dist/cowyo_linux_32bit
	#cd dist && upx --brute cowyo_linux_32bit

.PHONY: linux64
linux64: bindata.go
	env GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_linux_amd64

.PHONY: windows
windows: bindata.go
	env GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_windows_amd64.exe
	#cd dist && upx --brute cowyo_windows_amd64.exe

.PHONY: osx
osx: bindata.go
	env GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_osx_amd64
	#cd dist && upx --brute cowyo_osx_amd64

.PHONY: release
release: osx windows linux64 linux32 linuxarm
