FROM golang:1.8

WORKDIR /go/src/app
COPY . .
RUN go-wrapper download
RUN go-wrapper install
EXPOSE 9116
CMD ["go-wrapper", "run"] # ["app"]

# Once 17.05 has arrived
#FROM alpine:latest  
#RUN apk --no-cache add ca-certificates
#WORKDIR /root/
#COPY --from= as builder /go/app .
