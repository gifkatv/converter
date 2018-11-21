FROM alpine:3.8
MAINTAINER <terranisu@gmail.com>

EXPOSE 8080

ADD converter /app/converter

CMD ["/app/converter"]
