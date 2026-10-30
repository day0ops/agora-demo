# Build from the repo root: docker build -f mcp-servers/pricing-mcp/Dockerfile .
FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY fixtures/ ./fixtures/
COPY mcp-servers/pricing-mcp/ ./mcp-servers/pricing-mcp/
WORKDIR /src/mcp-servers/pricing-mcp
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/app .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/app /app
EXPOSE 9121
ENTRYPOINT ["/app"]
