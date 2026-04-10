FROM golang:1.24-alpine AS build
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
WORKDIR /app
COPY --from=build /app/server /app/server
COPY --from=build /app/migrations /app/migrations
COPY --from=build /app/content /app/content
EXPOSE 8080
ENTRYPOINT ["/app/server"]
