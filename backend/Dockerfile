FROM mirror.gcr.io/library/golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cloud-cafe .

FROM mirror.gcr.io/library/alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/cloud-cafe /usr/local/bin/cloud-cafe
EXPOSE 3000
CMD ["cloud-cafe"]
