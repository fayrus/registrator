FROM --platform=$BUILDPLATFORM cgr.dev/chainguard/go:latest AS builder
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /go/src/github.com/fayrus/registrator/
COPY . .
RUN GOARM=${TARGETVARIANT#v} CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
	-a -installsuffix cgo \
	-ldflags "-X main.Version=$(cat VERSION)" \
	-o bin/registrator \
	.

FROM cgr.dev/chainguard/static:latest
USER root
COPY --from=builder /go/src/github.com/fayrus/registrator/bin/registrator /bin/registrator

ENTRYPOINT ["/bin/registrator"]
