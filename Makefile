ADDRESS = awwkoala.com
PORT = 8003

CUR_DIR = $(shell bash -c 'pwd')
USERCUR = $(shell bash -c 'whoami')

make:
	go build

install:
	rm -rf jinstall
	mkdir jinstall
	cp install/awwkoala.nginx jinstall/awwkoala.nginx
	sed -i 's/PORT/$(PORT)/g'  jinstall/awwkoala.nginx
	sed -i 's/ADDRESS/$(ADDRESS)/g'  jinstall/awwkoala.nginx
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  jinstall/awwkoala.nginx
	cp install/awwkoala.init jinstall/awwkoala.init
	sed -i 's/EXT_ADDRESS/$(ADDRESS)/g'  jinstall/awwkoala.init
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  jinstall/awwkoala.init
	sed -i 's^USERCUR^$(USERCUR)^g'  jinstall/awwkoala.init
	sed -i 's^PORT^$(PORT)^g'  jinstall/awwkoala.init
	cp jinstall/awwkoala.init /etc/init.d/awwkoala.init
	chmod +x /etc/init.d/awwkoala.init
	cp jinstall/awwkoala.nginx /etc/nginx/sites-available/awwkoala.nginx
	ln -fs /etc/nginx/sites-available/awwkoala.nginx /etc/nginx/sites-enabled/awwkoala.nginx
	/etc/init.d/nginx reload
	/etc/init.d/nginx restart
	/etc/init.d/awwkoala.init restart
	rm -rf jinstall

binaries:
	rm -rf binaries
	rm -f awwkoala
	mkdir binaries
	env GOOS=linux GOARCH=amd64 go build -o awwkoala -v *.go
	zip -9 -r awwkoala-linux-amd64.zip awwkoala static/* templates/*
	rm -f awwkoala


.PHONY: install
.PHONY: binaries