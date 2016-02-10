ADDRESS = awwkoala.com
PORT = 8002

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
	mv jinstall/awwkoala.init /etc/init.d/
	chmod +x /etc/init.d/awwkoala.init
	mv jinstall/awwkoala.nginx /etc/nginx/sites-available/awwkoala.nginx
	rm /etc/nginx/sites-enabled/awwkoala.nginx
	ln -fs /etc/nginx/sites-available/awwkoala.nginx /etc/nginx/sites-enabled/awwkoala.nginx
	/etc/init.d/nginx reload
	/etc/init.d/nginx restart
	/etc/init.d/awwkoala.init restart
	rm -rf jinstall

.PHONY: install
