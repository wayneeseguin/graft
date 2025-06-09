FROM golang:1.18-alpine AS go

FROM go AS build
RUN apk --no-cache add git
WORKDIR /go/src/github.com/wayneeseguin/graft
COPY . .
RUN CGOENABLED=0 go build \
       -o /usr/bin/graft \
       -ldflags "-s -w -extldflags '-static' -X main.Version=$( (git describe --tags 2>/dev/null || (git rev-parse HEAD | cut -c-8)) | sed 's/^v//' )" \
       cmd/graft/main.go

FROM alpine:3.22.0 AS certificates
RUN apk add --no-cache ca-certificates

FROM scratch
# Run as non-root user
USER 65534:65534
COPY --from=build /usr/bin/graft /graft
COPY --from=certificates /etc/ssl/ /etc/ssl/
ENV PATH=/
# Add health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/graft", "version"] || exit 1
CMD ["/graft"]
