FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/server ./cmd/migrate ./cmd/account-deletion

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /out/migrate /out/account-deletion /
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]
