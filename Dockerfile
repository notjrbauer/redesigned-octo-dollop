FROM golang:alpine as build
RUN apk add --no-cache ca-certificates
WORKDIR /build
ADD . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags '-extldflags "-static"' -o cli ./cmd/main.go

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt \
     /etc/ssl/certs/ca-certificates.crt
COPY --from=build /build/cli /cli
COPY --from=build /build/config.yml /config.yml


CMD ["/cli"]
