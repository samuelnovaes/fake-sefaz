FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/fake-sefaz ./cmd/fakesefaz

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/fake-sefaz /fake-sefaz
EXPOSE 8080
ENTRYPOINT ["/fake-sefaz"]
