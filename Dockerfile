FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o rifamaster .

FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/rifamaster .
COPY --from=builder /app/static ./static
COPY .env .
EXPOSE 3000
CMD ["./rifamaster"]
