FROM golang:1.19 AS build_deps

RUN apt-get update -y && apt-get install -y git

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o webhook .

FROM debian:bullseye-slim AS app

COPY --from=build /workspace/webhook /usr/local/bin/webhook

RUN apt-get update -y && apt-get install -y ca-certificates

ENTRYPOINT ["webhook"]
