ARG BUILDKIT_INLINE_CACHE=1

FROM golang:1.21.9-alpine as builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN ./build.sh


FROM alpine:3.15

RUN set -ex \
  && apk add --no-cache \
    ffmpeg

WORKDIR /home/app

RUN set -ex \
 && adduser -D -s /sbin/nologin app \
 && chown app:app /home/app

COPY --from=builder --chown=app:app /build/bin/vibenator ./

ENTRYPOINT ["/home/app/vibenator"]
