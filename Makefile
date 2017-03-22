VERSION=0.1.1
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

.PHONY: build
build:
	go-bindata static/... templates/... 
	go build

.PHONY: release
release:
	rm -rf dist/
	mkdir dist/
	env GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_linux_amd64
	cd dist && upx --brute cowyo_linux_amd64
	env GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_windows_amd64.exe
	cd dist && upx --brute cowyo_windows_amd64.exe
	env GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_osx_amd64
	cd dist && upx --brute cowyo_osx_amd64


