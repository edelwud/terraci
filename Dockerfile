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
COPY ${TARGETOS}/${TARGETARCH}/xterraci /usr/local/bin/xterraci

RUN chmod +x /usr/local/bin/terraci /usr/local/bin/xterraci

WORKDIR /workspace

ENTRYPOINT ["terraci"]
CMD ["--help"]
