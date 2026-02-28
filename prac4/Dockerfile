FROM golang:1.22-alpine AS builder

WORKDIR /app

# Сначала копируем модульные файлы
COPY go.mod ./
RUN go mod download

# Потом копируем весь код5
COPY . .

# Ещё раз подтянем зависимости (создаст go.sum внутри контейнера)
RUN go mod tidy

# Собираем приложение
RUN CGO_ENABLED=0 go build -o myapp ./cmd/api/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/myapp .

EXPOSE 8080
ENTRYPOINT ["./myapp"]