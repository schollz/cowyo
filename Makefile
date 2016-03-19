ADDRESS = cowyo.com
PORT = 8003

CUR_DIR = $(shell bash -c 'pwd')
USERCUR = $(shell bash -c 'whoami')

make:
	go build

install:
	rm -rf jinstall
	mkdir jinstall
	cp install/cowyo.nginx jinstall/cowyo.nginx
	sed -i 's/PORT/$(PORT)/g'  jinstall/cowyo.nginx
	sed -i 's/ADDRESS/$(ADDRESS)/g'  jinstall/cowyo.nginx
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  jinstall/cowyo.nginx
	cp install/cowyo.init jinstall/cowyo.init
	sed -i 's/EXT_ADDRESS/$(ADDRESS)/g'  jinstall/cowyo.init
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  jinstall/cowyo.init
	sed -i 's^USERCUR^$(USERCUR)^g'  jinstall/cowyo.init
	sed -i 's^PORT^$(PORT)^g'  jinstall/cowyo.init
	cp jinstall/cowyo.init /etc/init.d/cowyo.init
	chmod +x /etc/init.d/cowyo.init
	cp jinstall/cowyo.nginx /etc/nginx/sites-available/cowyo.nginx
	ln -fs /etc/nginx/sites-available/cowyo.nginx /etc/nginx/sites-enabled/cowyo.nginx
	/etc/init.d/nginx reload
	/etc/init.d/nginx restart
	/etc/init.d/cowyo.init restart
	rm -rf jinstall

binaries:
	rm -rf binaries
	rm -f cowyo
	mkdir binaries
	env GOOS=linux GOARCH=amd64 go build -o cowyo -v *.go
	zip -9 -r cowyo-linux-64bit.zip cowyo static/* templates/*
	rm -f cowyo
	env GOOS=windows GOARCH=amd64 go build -o cowyo.exe -v *.go
	zip -9 -r cowyo-windows-64bit.zip cowyo.exe static/* templates/*
	rm -f cowyo.exe
	env GOOS=linux GOARCH=arm go build -o cowyo -v *.go
	zip -9 -r cowyo-raspberrypi.zip cowyo static/* templates/*
	rm -f cowyo
	env GOOS=darwin GOARCH=amd64 go build -o cowyo -v *.go
	zip -9 -r cowyo-macosx-64bit.zip cowyo static/* templates/*
	rm -f cowyo
	mv *.zip binaries/


.PHONY: install
.PHONY: binaries
