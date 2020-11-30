# builder stage first
FROM golang:1.15.5-buster as builder
COPY . /opt/petkeep-server
WORKDIR /opt/petkeep-server
ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w"

# final stage, just copying in the binary
FROM alpine:latest
COPY --from=builder /opt/petkeep-server/petkeep-server /opt/petkeep-server/petkeep-server
COPY ./docs /opt/petkeep-server/docs
WORKDIR /opt/petkeep-server
ENTRYPOINT ["./petkeep-server"]