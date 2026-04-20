FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /go/src/github.com/fayrus/registrator/
COPY . .
RUN \
	apk add --no-cache git \
	&& GOARM=${TARGETVARIANT#v} CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
		-a -installsuffix cgo \
		-ldflags "-X main.Version=$(cat VERSION)" \
		-o bin/registrator \
		.
ENTRYPOINT ["/go/src/github.com/fayrus/registrator/bin/registrator"]

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/fayrus/registrator/bin/registrator /bin/registrator

ENTRYPOINT ["/bin/registrator"]
