# syntax=docker/dockerfile:1

FROM alpine:3.20

ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache \
    ca-certificates \
    git \
    openssh-client

COPY ${TARGETOS}/${TARGETARCH}/terraci /usr/local/bin/terraci

RUN chmod +x /usr/local/bin/terraci

WORKDIR /workspace

ENTRYPOINT ["terraci"]
CMD ["--help"]
