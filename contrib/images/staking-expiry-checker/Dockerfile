FROM golang:1.21-alpine AS builder

ARG VERSION="HEAD"

RUN apk add --no-cache  \
    make \
    git \
    build-base \
    linux-headers \
    libc-dev \
    pkgconfig \
    alpine-sdk \
    libsodium-dev \
    libsodium-static \
    openssh

# Build
WORKDIR /go/src/github.com/babylonchain/staking-expiry-checker
# Cache dependencies
COPY go.mod go.sum /go/src/github.com/babylonchain/staking-expiry-checker/

# Copy the rest of the files
COPY ./ /go/src/github.com/babylonchain/staking-expiry-checker/


RUN LDFLAGS='-extldflags "-static" -v' \
    CGO_ENABLED=1 \
    CGO_LDFLAGS="$CGO_LDFLAGS -lstdc++ -lm -lsodium" \
    LINK_STATICALLY=true \
    BUILD_TAGS="muslc" \
    make build

# Final minimal image with binary only
FROM alpine:3.16 as run

RUN addgroup --gid 1138 -S staking-expiry-checker && adduser --uid 1138 -S staking-expiry-checker -G staking-expiry-checker
RUN apk add bash curl jq

# Label should match your github repo
LABEL org.opencontainers.image.source="https://github.com/babylonchain/staking-expiry-checker:${VERSION}"


# Copy over binaries from the build-env
COPY --from=builder /go/src/github.com/babylonchain/staking-expiry-checker/build/staking-expiry-checker /bin/staking-expiry-checker
COPY --from=builder /go/src/github.com/babylonchain/staking-expiry-checker/contrib/images/staking-expiry-checker/entrypoint.sh /bin/entrypoint.sh

WORKDIR /home/staking-expiry-checker
RUN chown -R staking-expiry-checker /home/staking-expiry-checker
USER staking-expiry-checker

ENTRYPOINT ["/bin/entrypoint.sh"]
CMD []
STOPSIGNAL SIGTERM