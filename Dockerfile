FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/odsquiz-initiatives ./cmd/api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& addgroup -S app \
	&& adduser -S -G app app

WORKDIR /app

COPY --from=builder /out/odsquiz-initiatives /app/odsquiz-initiatives

ENV PORT=8080

EXPOSE 8080

USER app

ENTRYPOINT ["/app/odsquiz-initiatives"]
