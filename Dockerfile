# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS server
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/amaquet ./cmd/amaquet \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/amaquet-cli ./cmd/amaquet-cli \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/amaquet-keygen ./cmd/amaquet-keygen \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/amaquet-restore ./cmd/amaquet-restore

FROM alpine:3.22
RUN addgroup -S amaquet && adduser -S -G amaquet amaquet
WORKDIR /app
COPY --from=server /out/amaquet /usr/local/bin/amaquet
COPY --from=server /out/amaquet-cli /usr/local/bin/amaquet-cli
COPY --from=server /out/amaquet-keygen /usr/local/bin/amaquet-keygen
COPY --from=server /out/amaquet-restore /usr/local/bin/amaquet-restore
COPY amaquet.example.json ./amaquet.example.json
COPY amaquet.example.json ./amaquet.json
COPY amaquet.production.example.json ./amaquet.production.example.json
RUN mkdir -p /app/data && chown -R amaquet:amaquet /app
USER amaquet
VOLUME ["/app/data"]
EXPOSE 13378 13379
ENTRYPOINT ["amaquet"]
CMD ["--config", "/app/amaquet.json"]
