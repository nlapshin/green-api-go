FROM golang:1.24-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/app /app/app
COPY --from=build /src/web /app/web

ENV APP_HOST=0.0.0.0
ENV APP_PORT=8080

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/app"]

