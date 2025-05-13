# --- Build Stage ---
FROM golang:1.21-alpine AS builder

# Label sebagai maintainer
LABEL maintainer="yourname@example.com"

# Set working directory di dalam container
WORKDIR /app

# Copy go.mod dan go.sum terlebih dahulu untuk caching layer
COPY go.mod go.sum ./
# Set GOSUMDB=off untuk menonaktifkan verifikasi checksum (hanya gunakan jika benar-benar diperlukan)
# ENV GOSUMDB=off
RUN go mod download

# Copy seluruh source code aplikasi Go
COPY . .

# Build aplikasi Go
# Ganti "your_main_package_path" dengan path ke main.go Anda, contoh: ./cmd/server
# CGO_ENABLED=0 untuk static linking, GOOS=linux untuk build ke linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main ./cmd/server/main.go

# --- Production Stage ---
FROM alpine:latest

# Set working directory
WORKDIR /app

# Copy binary yang sudah di-build dari builder stage
COPY --from=builder /app/main .

# Direktori untuk file frontend statis
RUN mkdir -p /app/ui/static
WORKDIR /app/ui/static
# Copy file HTML, CSS, JS ke direktori static
# Anda perlu menyesuaikan path ini jika struktur direktori frontend Anda berbeda
COPY ./ui/ /app/ui/static/
# Contoh jika file ada di ./frontend/dist maka COPY ./frontend/dist/ /app/ui/static/

# Kembali ke working directory utama aplikasi
WORKDIR /app

# Expose port yang digunakan oleh aplikasi Go (sesuaikan jika berbeda)
EXPOSE 8080

# Variabel lingkungan (opsional, bisa di-set saat run)
# ENV DB_HOST=mariadb
# ENV DB_PORT=3306
# ENV DB_USER=user
# ENV DB_PASSWORD=password
# ENV DB_NAME=db_wags
# ENV EXTERNAL_API_URL="https://wag.artakusuma.com/api/clients"
# ENV EXTERNAL_API_KEY="changeme"

# Command untuk menjalankan aplikasi
# Aplikasi Go akan melayani API dan juga file statis dari /app/ui/static
CMD ["/app/main"]

# Catatan:
# 1. Pastikan path ke main.go (`./cmd/server/main.go`) sudah benar.
# 2. Path untuk menyalin file frontend (`./ui/`) juga harus disesuaikan dengan struktur proyek Anda.
#    Jika frontend Anda memiliki proses build sendiri (misalnya dengan npm), Anda mungkin perlu stage build terpisah untuk frontend.
#    Untuk HTML + jQuery sederhana, menyalin file langsung seperti ini sudah cukup.
# 3. Aplikasi Go perlu dikonfigurasi untuk melayani file statis dari direktori `/app/ui/static` pada path tertentu (misalnya `/`).
# 4. Port yang di-expose (8080) harus sama dengan port yang didengarkan oleh server Go Anda.
# 5. Pertimbangkan untuk menambahkan user non-root untuk menjalankan aplikasi demi keamanan.
