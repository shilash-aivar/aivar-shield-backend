FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /aivar-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY --from=build /aivar-server /app/server
COPY migrations /app/migrations
ENV AIVAR_MIGRATIONS_DIR=/app/migrations
EXPOSE 8080
CMD ["/app/server"]
