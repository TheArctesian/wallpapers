FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -trimpath -o /dither ./cmd/dither

FROM alpine:3.21
# libheif-tools provides heif-convert, for .heic sources.
RUN apk add --no-cache libheif-tools
COPY --from=build /dither /usr/local/bin/dither
ENTRYPOINT ["dither"]
