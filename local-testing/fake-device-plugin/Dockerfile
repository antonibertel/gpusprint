FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY local-testing/fake-device-plugin/ local-testing/fake-device-plugin/
RUN CGO_ENABLED=0 go build -o fake-device-plugin ./local-testing/fake-device-plugin

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/fake-device-plugin /fake-device-plugin
CMD ["/fake-device-plugin"]
