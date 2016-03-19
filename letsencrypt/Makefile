ADDRESS = yourserver.com
PORT = 8003

CUR_DIR = $(shell bash -c 'pwd')
USERCUR = $(shell bash -c 'whoami')

make:
	go build

install:
	rm -rf jinstall
	mkdir jinstall
	cp cowyo.ssl.nginx cowyo.ssl.nginx.temp
	sed -i 's/PORT/$(PORT)/g'  cowyo.ssl.nginx.temp
	sed -i 's/ADDRESS/$(ADDRESS)/g'  cowyo.ssl.nginx.temp
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  cowyo.ssl.nginx.temp
	cp cowyo.init cowyo.init.temp
	sed -i 's/EXT_ADDRESS/$(ADDRESS)/g'   cowyo.init.temp
	sed -i 's^CUR_DIR^$(CUR_DIR)^g'  cowyo.init.temp
	sed -i 's^USERCUR^$(USERCUR)^g'  cowyo.init.temp
	sed -i 's^PORT^$(PORT)^g'  cowyo.init.temp
	cp cowyo.init.temp /etc/init.d/cowyo.init
	chmod +x /etc/init.d/cowyo.init
	cp cowyo.ssl.nginx.temp /etc/nginx/sites-available/cowyo.nginx
	ln -fs /etc/nginx/sites-available/cowyo.nginx /etc/nginx/sites-enabled/cowyo.nginx
	/etc/init.d/nginx reload
	/etc/init.d/nginx restart
	/etc/init.d/cowyo.init restart
	rm -rf *.temp

.PHONY: install
