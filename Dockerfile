# First stage of the build
FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/fileserver /app
RUN chmod +x /app/fileserver

# Second stage of the build
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/fileserver /app/fileserver

EXPOSE 8080

CMD ["/app/fileserver", "server"]
