# syntax=docker/dockerfile:1
# go-libsql は cgo を使うため CGO_ENABLED=1 でビルドする（glibc が必要）。

# ---- dev：compose から使う。ソースは bind mount し、air でホットリロードする ----
FROM golang:1.27-trixie AS dev
# bind mount したリポジトリは所有者が違うため、VCS 情報の埋め込みを止める
ENV CGO_ENABLED=1 GOFLAGS=-buildvcs=false
RUN go install github.com/air-verse/air@v1.67.4
WORKDIR /workspace/server
CMD ["air", "-c", ".air.toml"]

# ---- build：本番用バイナリ ----
FROM golang:1.27-trixie AS build
ENV CGO_ENABLED=1
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- prod：Cloud Run 用（雛形。デプロイ手順は別途） ----
FROM gcr.io/distroless/cc-debian13:nonroot AS prod
COPY --from=build /out/api /api
COPY --from=build /out/migrate /migrate
COPY migrations /migrations
ENV MIGRATIONS_URL=file:///migrations
EXPOSE 8080
ENTRYPOINT ["/api"]
