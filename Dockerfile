ARG GO_VERSION
ARG ALPINE_VERSION

FROM golang:1.19-alpine3.15 as build

WORKDIR /app

COPY ./src .

RUN go build -mod vendor -o /app/dist/pusher .

FROM alpine

USER nobody

COPY --from=build --chown=nobody:nobody /app/dist /app

WORKDIR /app

ENTRYPOINT ["/app/pusher"]
