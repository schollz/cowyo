ADDRESS = yourserver.com
PORT = 8003

CUR_DIR = $(shell bash -c 'pwd')
USERCUR = $(shell bash -c 'whoami')

make:
	go build

install:
	rm -rf jinstall
	mkdir jinstall
	cp awwkoala.ssl.nginx awwkoala.ssl.nginx.temp
	sed -i 's/PORT/$(PORT)/g'  awwkoala.ssl.nginx.temp
	sed -i 's/ADDRESS/$(ADDRESS)/g'  awwkoala.ssl.nginx.temp
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  awwkoala.ssl.nginx.temp
	cp awwkoala.init awwkoala.init.temp
	sed -i 's/EXT_ADDRESS/$(ADDRESS)/g'   awwkoala.init.temp
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  awwkoala.init.temp
	sed -i 's^USERCUR^$(USERCUR)^g'  awwkoala.init.temp
	sed -i 's^PORT^$(PORT)^g'  awwkoala.init.temp
	cp awwkoala.init.temp /etc/init.d/awwkoala.init
	chmod +x /etc/init.d/awwkoala.init
	cp awwkoala.ssl.nginx.temp /etc/nginx/sites-available/awwkoala.nginx
	ln -fs /etc/nginx/sites-available/awwkoala.nginx /etc/nginx/sites-enabled/awwkoala.nginx
	/etc/init.d/nginx reload
	/etc/init.d/nginx restart
	/etc/init.d/awwkoala.init restart
	rm -rf *.temp

.PHONY: install
