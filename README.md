# Task Management API

REST API multi-user untuk mengelola task. Backend menggunakan Go dan Gin, dengan PostgreSQL sebagai database.

## Menjalankan dengan Docker

Pastikan Docker Engine dan Docker Compose tersedia, lalu jalankan:

```sh
docker compose up --build
```

API tersedia di `http://localhost:8080`. PostgreSQL berjalan di port `5432` dan datanya disimpan dalam named volume `postgres_data`.

Untuk menghentikan container:

```sh
docker compose down
```

Untuk menghapus data database lokal sekaligus:

```sh
docker compose down --volumes
```

Nilai default pada Compose ditujukan untuk development lokal. Untuk deployment, atur `DATABASE_URL`, port, dan kredensial database melalui environment platform.

## Menjalankan tanpa Docker

Persyaratan: Go 1.25+ dan PostgreSQL.

```sh
cp .env.example .env
set -a && source .env && set +a
go run ./cmd/api
```

Atur `DATABASE_URL` agar mengarah ke PostgreSQL yang sedang berjalan. Migrasi tabel users dijalankan otomatis saat aplikasi mulai.

## Konfigurasi

| Environment variable | Default | Keterangan |
| --- | --- | --- |
| `APP_ENV` | `development` | Nama lingkungan aplikasi |
| `HTTP_ADDR` | `:8080` | Alamat bind HTTP server |
| `DATABASE_URL` | wajib | PostgreSQL connection URL |
| `JWT_SECRET` | wajib, minimum 32 byte | Secret untuk menandatangani JWT |
| `JWT_TTL` | `15m` | Masa berlaku access token, format Go duration |
| `PORT` | `8080` | Port host yang dipetakan Compose |
| `POSTGRES_PORT` | `5432` | Port PostgreSQL yang dipetakan ke host |
| `POSTGRES_DB` | `taskapp` | Nama database Compose |
| `POSTGRES_USER` | `taskapp` | User PostgreSQL Compose |
| `POSTGRES_PASSWORD` | `taskapp` | Password PostgreSQL Compose |

## Endpoint yang tersedia

| Method | Path | Fungsi |
| --- | --- | --- |
| `GET` | `/health` | Liveness check; memeriksa proses aplikasi |
| `GET` | `/ready` | Readiness check; memeriksa koneksi PostgreSQL |
| `POST` | `/auth/register` | Membuat akun dan mengembalikan JWT |
| `POST` | `/auth/login` | Memvalidasi akun dan mengembalikan JWT |
| `GET` | `/openapi.yaml` | Spesifikasi OpenAPI untuk endpoint yang sudah tersedia |
| `GET` | `/swagger/` | Swagger UI interaktif |

Swagger UI menggunakan aset dari unpkg CDN. Spesifikasi OpenAPI disediakan oleh aplikasi di `/openapi.yaml`.

Contoh:

```sh
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
curl -i -X POST http://localhost:8080/auth/register -H 'Content-Type: application/json' -d '{"email":"user@example.com","password":"correct-horse-battery"}'
curl -i -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"email":"user@example.com","password":"correct-horse-battery"}'
curl -i http://localhost:8080/openapi.yaml
```

Migration tabel users berjalan otomatis ketika API mulai. Password disimpan dalam bentuk bcrypt hash; email dinormalisasi ke huruf kecil. Register mensyaratkan email valid dan password sepanjang 8–72 byte. JWT ditandatangani dengan HS256 dan masa berlakunya diatur oleh `JWT_TTL`.

## Arsitektur singkat

- `cmd/api` memuat konfigurasi, menyiapkan koneksi PostgreSQL dan logger, lalu menjalankan HTTP server.
- `internal/config` membaca konfigurasi environment.
- `internal/auth` menangani aturan register/login, hashing password, dan penerbitan/verifikasi JWT.
- `internal/repository/postgres` mengimplementasikan akses data PostgreSQL untuk autentikasi.
- `internal/database/migrations` menyimpan migrasi SQL yang di-embed ke binary dan dijalankan saat startup.
- `internal/httpapi` merangkai routes dan handler HTTP.
- `internal/middleware` menyediakan request ID, structured access log, panic recovery, dan JWT bearer authentication reusable.
- `internal/response` menyediakan format error JSON bersama.
- `compose.yaml` menjalankan API dan PostgreSQL untuk development lokal.

Alur fitur akan mengikuti `handler → service → repository → PostgreSQL`. Layer service dan repository akan ditambahkan bersama fitur agar interface dan abstraksinya mengikuti kebutuhan nyata.

## Checklist requirement

Status **Belum** berarti belum diimplementasikan; item tersebut sengaja dicatat agar progres bisa ditinjau dan diverifikasi.

| Requirement | Status | Lokasi atau bukti verifikasi |
| --- | --- | --- |
| Go backend dan framework Gin | Selesai | `go.mod`, `cmd/api`, `internal/httpapi` |
| PostgreSQL | Sebagian | Pool, migrasi, tabel users, repository autentikasi, dan readiness tersedia; skema task menyusul |
| Docker deployment | Sebagian | `Dockerfile`, `compose.yaml`; deployment image perlu dikonfigurasi sesuai platform |
| Health check | Selesai | `GET /health` |
| Readiness check termasuk PostgreSQL | Selesai | `GET /ready` |
| Dokumentasi Swagger/OpenAPI untuk semua API | Sebagian | `/swagger/` dan `/openapi.yaml` saat ini mencakup endpoint fondasi; API fitur akan ditambahkan saat dibuat |
| README: cara menjalankan, endpoint, arsitektur | Selesai | Bagian di atas |
| README: checklist keterpenuhan seluruh requirement | Selesai | Tabel checklist ini |
| Register dan login | Selesai | `POST /auth/register`, `POST /auth/login`; service dan repository auth |
| JWT authentication | Sebagian | Token HS256 diterbitkan dan middleware reusable memvalidasi bearer token; route task terlindungi menyusul |
| Create, list, detail, update, delete task | Belum | Tahap CRUD task |
| Filter status, search title, pagination | Belum | Tahap list task |
| Idempotency key UUID selama 24 jam | Belum | Tahap idempotency; perlu constraint database dan hasil respons tersimpan |
| Error JSON konsisten, 4xx/5xx, tanpa detail internal | Sebagian | Error routing, readiness, panic, register, dan login memakai format bersama; error fitur menyusul |
| Assign dalam satu tim dan transaksi DB | Belum | Tahap assign; model tim/membership dan task logs belum dibuat |
| Rollback bila langkah assign gagal | Belum | Akan diverifikasi pada implementasi transaksi assign |
| Notifikasi assign (mock/log) | Belum | Tahap assign |
| Structured request logging | Selesai | Middleware access log: request ID, method, path, status, latency, level |
| Unit test tanpa database/service eksternal | Sebagian | Test service autentikasi menggunakan stub; test idempotency sequential dan concurrent menyusul |

Unit test service autentikasi tersedia dan menggunakan repository stub. Jalankan seluruh unit test dengan `go test ./...`.

## Pengembangan

Jalankan server lokal dengan `make run`, atau gunakan `make compose-up` dan `make compose-down` untuk Docker Compose. Perubahan fitur berikutnya harus memperbarui OpenAPI dan checklist ini bersama implementasinya.
