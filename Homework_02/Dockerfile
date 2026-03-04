FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p generated && oapi-codegen --config oapi-codegen.yaml api/openapi.yaml

RUN CGO_ENABLED=0 go build -o /marketplace ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /marketplace /marketplace

EXPOSE 8080

ENTRYPOINT ["/marketplace"]
