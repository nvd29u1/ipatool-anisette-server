# NVD-Anisette-Server (Docker)
# Anisette server for personal use, built on ipatool's modern ADI engine.
# Same warning as the exe: first provisioning downloads ~11 GB of Apple
# restore data from Apple's CDN into /data (cache) — keep that volume.
# The machine identity (adi.json) also lives in /data: DO NOT SHARE it.
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/anisette-server ./cmd/anisette-server

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/anisette-server /usr/local/bin/anisette-server
VOLUME ["/data"]
EXPOSE 6969
ENTRYPOINT ["anisette-server", "--listen", "0.0.0.0:6969", "--data-dir", "/data"]
