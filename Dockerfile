# syntax=docker/dockerfile:1.2
FROM golang:1.17-alpine AS builder
WORKDIR /app
COPY . .
RUN go install -v .

FROM alpine:3.8
COPY --from=builder /go/bin/too-many-requests-registry /usr/bin
EXPOSE 8080
CMD ["/usr/bin/too-many-requests-registry"]
