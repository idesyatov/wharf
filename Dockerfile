FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY wharf /usr/local/bin/wharf
ENTRYPOINT ["wharf"]
