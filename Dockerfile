FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
# NVIDIA NVML requires CGO enabled to link against headers cleanly
RUN CGO_ENABLED=1 go build -a -o gpusprint ./cmd/gpusprint

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/gpusprint .
CMD ["./gpusprint"]
