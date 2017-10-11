# First build step
FROM golang:1.9-alpine as builder

WORKDIR /go/src/cowyo
COPY . .
# Disable crosscompiling
ENV CGO_ENABLED=0

# Install git and make, compile and cleanup
RUN apk add --no-cache git make \
	&& go get -u github.com/schollz/cowyo \
    && go get -u github.com/jteeuwen/go-bindata/... \
    && make \
    && apk del --purge git make \
    && rm -rf /var/cache/apk*

# Second build step uses the minimal scratch Docker image
FROM scratch
# Copy the binary from the first step
COPY --from=builder /go/src/cowyo/cowyo /usr/local/bin/cowyo
# Expose data folder
VOLUME /data
EXPOSE 8050
# Start cowyo listening on any host
CMD ["cowyo", "--host", "0.0.0.0"]