FROM golang:1.27-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags="-s -w" -o /out/user-management ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=builder /out/user-management /app/user-management

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/user-management"]
