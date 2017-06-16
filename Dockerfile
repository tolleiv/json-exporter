FROM golang:1.8  as builder

WORKDIR /go/src/app
COPY . .
RUN go-wrapper download
RUN go-wrapper install

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from= as builder /go/app .
EXPOSE 9116
CMD ["./app"]  
