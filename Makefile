ADDRESS = cowyo.com
PORT = 8001

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
	mv jinstall/cowyo.init /etc/init.d/
	chmod +x /etc/init.d/cowyo.init
	mv jinstall/cowyo.nginx /etc/nginx/sites-available/cowyo.nginx
	rm /etc/nginx/sites-enabled/cowyo.nginx
	ln -s /etc/nginx-sites-available/cowyo.nginx /etc/nginx/sites-enabled/cowyo.nginx
	/etc/init.d/nginx reload
	/etc/init.d/nginx restart
	/etc/init.d/cowyo.init restart
	rm -rf jinstall

.PHONY: install
