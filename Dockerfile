# sudo docker build -t cowyo .
# sudo docker run -it -p 8003:8003 -v `pwd`/data:/data cowyo bash
FROM ubuntu:16.04

# Get basics
RUN apt-get update
RUN apt-get -y upgrade
RUN apt-get install -y golang git wget curl vim
RUN mkdir /usr/local/work
ENV GOPATH /usr/local/work

# Install cowyo
WORKDIR "/root"
RUN go get github.com/schollz/cowyo
RUN git clone https://github.com/schollz/cowyo.git
WORKDIR "/root/cowyo"
RUN git pull
RUN go build

# Setup supervisor
RUN apt-get update && apt-get install -y supervisor

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Add Tini
ENV TINI_VERSION v0.9.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

# Startup
CMD ["/usr/bin/supervisord"]
