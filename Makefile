# Make a release with
# make -j4 release

VERSION=$(shell git describe)
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}" -a -installsuffix cgo

.PHONY: build
build:
	go-bindata static/... templates/...
	go build ${LDFLAGS}

.PHONY: quick
quick:
	go-bindata static/... templates/...
	go build

.PHONY: linuxarm
linuxarm:
	env GOOS=linux GOARCH=arm go build ${LDFLAGS} -o dist/cowyo_linux_arm
	#cd dist && upx --brute cowyo_linux_arm

.PHONY: linux32
linux32:
	env GOOS=linux GOARCH=386 go build ${LDFLAGS} -o dist/cowyo_linux_32bit
	#cd dist && upx --brute cowyo_linux_amd64

.PHONY: linux64
linux64:
	env GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_linux_amd64
	ssh camlistore "sudo initctl stop cowyo"
	scp dist/cowyo_linux_amd64 camlistore:cowyo/cowyo
	ssh camlistore "chmod +x cowyo/cowyo"
	ssh camlistore "sudo initctl start cowyo"

.PHONY: windows
windows:
	env GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_windows_amd64.exe
	#cd dist && upx --brute cowyo_windows_amd64.exe

.PHONY: osx
osx:
	env GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_osx_amd64
	#cd dist && upx --brute cowyo_osx_amd64

.PHONY: release
release: osx windows linux64 linux32 linuxarm
