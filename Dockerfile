FROM golang:alpine as builder

RUN go version
ENV GOPATH=/

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o stateful-server ./cmd/server/main.go

FROM alpine as runner

WORKDIR /app

EXPOSE 8000

ARG CONFIG_PATH

COPY --from=builder --chown=kira /build/stateful-server /build/${CONFIG_PATH} ./

CMD ["./stateful-server"]