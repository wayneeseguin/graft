FROM golang:1.18-alpine AS go

FROM go AS build
RUN apk --no-cache add git
COPY . /go/src/github.com/wayneeseguin/graft
RUN cd /go/src/github.com/wayneeseguin/graft && \
    CGOENABLED=0 go build \
       -o /usr/bin/graft \
       -ldflags "-s -w -extldflags '-static' -X main.Version=$( (git describe --tags 2>/dev/null || (git rev-parse HEAD | cut -c-8)) | sed 's/^v//' )" \
       cmd/graft/main.go

FROM alpine:latest AS certificates
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=build /usr/bin/graft /graft
COPY --from=certificates /etc/ssl/ /etc/ssl/
ENV PATH=/
CMD ["/graft"]
