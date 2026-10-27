# ---------- build stage ----------
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---------- runtime stage ----------
# distroless/static sudah membawa CA certs (untuk HTTPS ke n8n) dan tzdata
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
COPY migrations /migrations
COPY static /static
ENV PORT=8080 GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["/server"]
