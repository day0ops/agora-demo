# web and builder run with --platform=$BUILDPLATFORM (native, not emulated)
# since their RUN steps only need to produce platform-independent JS or
# cross-compiled Go output, not execute as the target arch. This also avoids
# a reproducible QEMU/Rosetta crash in esbuild's own Go binary when the vite
# build stage is emulated (e.g. building a linux/amd64 image on Apple
# Silicon). GOARCH is hardcoded rather than read from $TARGETARCH since this
# project only ever deploys to GKE's standard amd64 node pools, and some
# builders (e.g. podman's classic, non-buildx builder) don't reliably
# populate that automatic build arg.
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /web
COPY console/web/package.json console/web/package-lock.json* ./
RUN npm ci
COPY console/web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
WORKDIR /src
COPY fixtures/ ./fixtures/
COPY console/ ./console/
COPY --from=web /web/dist ./console/web/dist
WORKDIR /src/console
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/app /app
EXPOSE 8090
ENTRYPOINT ["/app"]
