FROM golang:1.8 as builder

WORKDIR /polite
COPY . .

RUN go build -v


FROM alpine:3.10.2


RUN apk add --no-cache \
        libc6-compat

COPY --from=builder /polite/polite /entrypoint

ENTRYPOINT ["/entrypoint"]