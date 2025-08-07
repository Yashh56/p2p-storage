FROM golang:1.23.8-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server


FROM alpine:latest

RUN adduser -D nonroot

RUN mkdir /data

RUN chown nonroot:nonroot /data

USER nonroot

COPY --from=builder /bin/server /bin/server

WORKDIR /data

EXPOSE 50051

EXPOSE 4001/tcp

EXPOSE 4001/udp


ENTRYPOINT ["/bin/server"]
