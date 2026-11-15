FROM golang:1.23.12-alpine3.21 AS build

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY libs/platform ./libs/platform
COPY services/project-d-ckguesser-be-go ./services/project-d-ckguesser-be-go

WORKDIR /src/services/project-d-ckguesser-be-go
RUN GOWORK=off CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ckguesser-be ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ckguesser-be /ckguesser-be
USER nonroot:nonroot
ENTRYPOINT ["/ckguesser-be"]
