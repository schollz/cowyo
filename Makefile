CUR_DIR = $(shell bash -c 'pwd')
CUR_USER = $(shell bash -c 'echo $USER')
PORT ?= $(shell bash -c 'read -p "Port (e.g. 8001): " pwd; echo $$pwd')
EXTERNAL_ADDRESS ?= $(shell bash -c 'read -p "External address (e.g. something.com): " pwd; echo $$pwd')

all:
	echo The password is $(EXTERNAL_ADDRESS)
	echo The port is $(PORT)
	echo The cwd is $(CUR_DIR)
