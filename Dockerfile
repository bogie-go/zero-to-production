# ONE IMAGE, several roles. `ecommerce` serves, `ecommerce worker` works the
# queue and `ecommerce migrate` moves the schema, so a web container and a
# job container are the same artifact started with different arguments,
# and cannot be built from different commits.
# go.mod decides the Go version, as .ruby-version does for a Rails image:
# with GOTOOLCHAIN=auto the builder fetches the toolchain go.mod names if
# the base image's is older, so bumping Go is a go.mod edit, not two.
FROM golang:1.27-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ecommerce .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 ecommerce
USER ecommerce

# The encrypted credentials travel with the image; the keys do NOT.
# .dockerignore keeps them out of the build context entirely, and the key
# arrives at runtime as ECOMMERCE_MASTER_KEY. An image carrying both halves
# leaks both. The whole config directory is copied because the layout varies:
# development lives in config/credentials.yml.enc, while the deployed
# environment has config/credentials/production.yml.enc.
COPY --from=builder /src/config /app/config
WORKDIR /app

COPY --from=builder /out/ecommerce /usr/local/bin/ecommerce
EXPOSE 8080

# No CMD: with no arguments this serves, reading ECOMMERCE_ENV and
# ECOMMERCE_CREDENTIALS from the environment and the rest from the
# credentials baked in above. The schema is compiled into the binary, so this
# image is also the tool that inspects and rolls it back; there is no separate
# goose binary to keep in step:
#
#   docker exec <container> ecommerce migrate status
#   docker exec <container> ecommerce migrate down --yes
ENTRYPOINT ["ecommerce"]
