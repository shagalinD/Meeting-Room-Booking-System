FROM golang:1.26.1-alpine AS build

WORKDIR /src

ARG APP=api

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/${APP}

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
WORKDIR /app

COPY --from=build /out/server ./server

USER app
EXPOSE 8081 50051 50052 50053

ENTRYPOINT ["./server"]
