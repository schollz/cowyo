FROM golang:1.12-alpine as builder
RUN apk add --no-cache git make 
RUN go get -v github.com/jteeuwen/go-bindata/go-bindata
WORKDIR /go/cowyo
COPY . .
RUN make build

FROM alpine:latest 
VOLUME /data
EXPOSE 8050
COPY --from=builder /go/cowyo/cowyo /cowyo
ENTRYPOINT ["/cowyo"]
CMD ["--data","/data","--allow-file-uploads","--max-upload-mb","10","--host","0.0.0.0"]
