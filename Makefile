VERSION=2.0.1
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

.PHONY: build
build:
	go-bindata static/... templates/... 
	go build

.PHONY: release
release:
	rm -rf dist/
	mkdir dist/
	go-bindata static/... templates/... 
	env GOOS=linux GOARCH=arm go build ${LDFLAGS} -o dist/cowyo_linux_arm
	cd dist && upx --brute cowyo_linux_arm
	env GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_linux_amd64
	cd dist && upx --brute cowyo_linux_amd64
	env GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_windows_amd64.exe
	cd dist && upx --brute cowyo_windows_amd64.exe
	env GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/cowyo_osx_amd64
	cd dist && upx --brute cowyo_osx_amd64


