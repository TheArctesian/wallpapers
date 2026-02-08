FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY main.go .
RUN go mod tidy && go mod download
RUN go build -o /dither main.go

FROM alpine:3.21
COPY --from=build /dither /dither
CMD ["/dither"]
