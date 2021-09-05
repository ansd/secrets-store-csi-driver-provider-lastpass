FROM golang:1.17-alpine as builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY main.go ./
COPY server/ ./server/
RUN go build -o /secrets-store-csi-driver-provider-lastpass


FROM gcr.io/distroless/static

COPY --from=builder /secrets-store-csi-driver-provider-lastpass /bin/
ENTRYPOINT ["/bin/secrets-store-csi-driver-provider-lastpass"]
