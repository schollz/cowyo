# First build step
FROM golang:1.10-alpine as builder

#WORKDIR /go/src/github.com/schollz/cowyo
#COPY . .
# Disable crosscompiling
ENV CGO_ENABLED=0

# Install git and make, compile and cleanup
RUN apk add --no-cache git make \
    && go get -u -v github.com/jteeuwen/go-bindata/... \
    && go get -u -v -d github.com/schollz/cowyo \
	&& cd /go/src/github.com/schollz/cowyo \
    && make \
    && apk del --purge git make \
    && rm -rf /var/cache/apk*

# Second build step uses the minimal scratch Docker image
FROM scratch
# Copy the binary from the first step
COPY --from=builder /go/src/github.com/schollz/cowyo/cowyo /usr/local/bin/cowyo
# Expose data folder
VOLUME /data
EXPOSE 8050
# Start cowyo listening on any host
CMD ["cowyo", "--host", "0.0.0.0"]