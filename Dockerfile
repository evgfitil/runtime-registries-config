FROM golang:alpine3.19 as builder
WORKDIR /app
COPY go.mod go.sum ./
COPY cmd/ /app/cmd
COPY internal/ /app/internal
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go mod download
RUN go build -o server ./cmd

FROM scratch
WORKDIR /app
COPY --from=builder /app/server /app/
ENTRYPOINT ["/app/server"]
