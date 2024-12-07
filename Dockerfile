# Go için resmi base image kullanıyoruz
FROM golang:1.21 AS builder

# Çalışma dizinini belirliyoruz
WORKDIR /app

# Go modül dosyalarını kopyalıyoruz ve bağımlılıkları yüklüyoruz
COPY go.mod go.sum ./
RUN go mod download

# Kaynak kodları kopyalıyoruz
COPY . .

# Uygulamayı derliyoruz
RUN go build -o main ./cmd/api

# Hafif bir çalışma ortamı için minimal bir image kullanıyoruz
FROM debian:buster

# Çalışma dizinini ayarlıyoruz
WORKDIR /app

# Gerekli dosyaları kopyalıyoruz
COPY --from=builder /app/main /app/main
COPY .env /app/.env

# Uygulamanın dinlediği portu açıyoruz
EXPOSE 8080

# Uygulamayı çalıştırıyoruz
CMD ["./main"]
