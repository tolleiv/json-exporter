FROM golang
LABEL stage=tempbuilder

WORKDIR /workspace

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /json_exporter
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

FROM scratch

COPY --from=0 /json_exporter /json_exporter
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /etc_passwd /etc/passwd
USER nobody

