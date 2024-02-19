FROM golang:1.22-alpine as builder

RUN apk --update add git

ENV GO111MODULE=on

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

ARG VERSION

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/root/.cache/go-build
RUN go build -ldflags="-w -s -X main.version=$VERSION" -o backend .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /app/backend .

ENTRYPOINT ["./backend"]