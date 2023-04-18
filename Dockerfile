# Builder
FROM whatwewant/builder-go:v1.19-1 as builder

WORKDIR /build

COPY go.mod ./

COPY go.sum ./

RUN go mod download

COPY . .

RUN CGO_EABLE=0 \
  GOOS=linux \
  go build \
  -trimpath \
  -ldflags '-w -s -buildid=' \
  -v -o gzterminal

# Server
FROM whatwewant/alpine:v3.17-1

LABEL MAINTAINER="Zero<tobewhatwewant@gmail.com>"

LABEL org.opencontainers.image.source="https://github.com/go-zoox/gzterminal"

ARG VERSION=latest

ENV MODE=production

COPY --from=builder /build/gzterminal /bin

ENV VERSION=${VERSION}

CMD gzterminal server -c /conf/config.yml
