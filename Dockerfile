# syntax=docker/dockerfile:1

FROM golang:1.18 as builder

WORKDIR /blog

# Copy the local project files to the container's workspace.
COPY . ./

RUN go build -o /app

FROM debian

WORKDIR /

COPY --from=builder /app /app

EXPOSE 8080

ENTRYPOINT ["/app"]