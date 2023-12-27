FROM golang:1.21 AS build_deps

RUN apt-get update -y && apt-get install -y git

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM build_deps AS build

COPY . .

RUN go build -o webhook .

FROM golang:1.21 AS runtime

RUN apt-get update -y && apt-get install -y ca-certificates

COPY --from=build /workspace/webhook /usr/local/bin/webhook

ENTRYPOINT ["webhook"]