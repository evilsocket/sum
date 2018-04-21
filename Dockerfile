# build stage
FROM golang:1.10-alpine AS build-env

ENV SRC_DIR $GOPATH/src/github.com/evilsocket/sum

RUN apk add --update ca-certificates 
RUN apk add --no-cache --update git make

WORKDIR $SRC_DIR
ADD . $SRC_DIR
RUN make sumd

# final stage
FROM alpine
RUN apk add --no-cache --update git make
RUN mkdir -p /var/lib/sumd/data
RUN mkdir -p /var/lib/sumd/oracles
COPY --from=build-env /go/src/github.com/evilsocket/sum/sumd /app/
WORKDIR /app
EXPOSE 50051
ENTRYPOINT ["/app/sumd"]
