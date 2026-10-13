# Task Management API - Test Case GDCPay

REST API multi-user untuk mengelola task. Backend menggunakan Go dan Gin, dengan PostgreSQL sebagai database.

LIVE URL: `https://gdcpay-test-project.onrender.com`

## Menjalankan dengan Docker

Compose adalah environment development lokal. File `.env` hanya untuk nilai lokal dan tidak boleh di-commit.

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

Atur `DATABASE_URL` agar mengarah ke PostgreSQL yang sedang berjalan. Migrasi tabel users dan tasks dijalankan otomatis saat aplikasi mulai.

## Konfigurasi

| Environment variable | Default | Keterangan |
| --- | --- | --- |
| `APP_ENV` | `development` | Nama env aplikasi |
| `HTTP_ADDR` | `:8080` | Alamat bind HTTP server |
| `PORT` | `8080` di Compose; diberikan platform saat deploy | Port aplikasi; `HTTP_ADDR` jika disetel memiliki prioritas |
| `DATABASE_URL` | wajib | PostgreSQL connection URL |
| `JWT_SECRET` | wajib, minimum 32 byte | Secret untuk menandatangani JWT |
| `JWT_TTL` | `15m` | Masa berlaku access token, format Go duration |
| `POSTGRES_PORT` | `5432` | Port PostgreSQL yang dipetakan ke host |
| `POSTGRES_DB` | `taskapp` | Nama database Compose |
| `POSTGRES_USER` | `taskapp` | User PostgreSQL Compose |
| `POSTGRES_PASSWORD` | `taskapp` | Password PostgreSQL Compose |

### Pemisahan environment

| Environment | Branch | Cara menjalankan | Sumber konfigurasi |
| --- | --- | --- | --- |
| Development | `development` dan branch fitur | Jalankan Compose di komputer lokal untuk pengujian manual | `.env` lokal dari `.env.example` |
| Production | `main` | GitHub Actions memicu deployment setelah pemeriksaan berhasil | Environment variables dan secrets pada platform hosting |

Jangan menyimpan `DATABASE_URL` produksi atau `JWT_SECRET` produksi di repository. Pada production, set `APP_ENV=production`; aplikasi akan memakai Gin release mode. Jika platform menyediakan `PORT`, aplikasi akan bind ke port tersebut. `HTTP_ADDR` dapat dipakai untuk override eksplisit.

### Branch dan GitHub Actions

`development` dibuat dari `main` sebagai branch dasar pengembangan. Untuk setiap perubahan, buat branch `feature/<nama>` atau `fix/<nama>` dari `development`, uji di lokal, lalu gabungkan ke `development`. Setelah siap dirilis, gabungkan `development` ke `main`.

Workflow di `.github/workflows/ci.yml` menjalankan `go test ./...` dan membangun Docker image pada push ke `development`, `main`, `feature/*`, atau `fix/*`, serta pull request ke `development`/`main`. Deployment hanya dipicu oleh push ke `main`, setelah job verifikasi berhasil, dan hanya jika variable repository `RENDER_DEPLOY_ENABLED` bernilai `true`. Pull request ke `main` menjalankan verifikasi, tetapi belum men-deploy.

Untuk mengaktifkan deployment Render:

1. Buat Web Service berbasis Docker dari repository ini, pilih branch `main`, dan matikan auto deploy agar deployment dikendalikan GitHub Actions.
2. Tambahkan GitHub Environment bernama `production`.
3. Tambahkan secret `RENDER_DEPLOY_HOOK_URL` pada environment `production` menggunakan URL deploy hook dari pengaturan Render.
4. Tambahkan repository variable `RENDER_DEPLOY_ENABLED=true`.
5. Atur `APP_ENV=production`, `DATABASE_URL`, dan `JWT_SECRET` pada environment Render. Render menyediakan `PORT` secara otomatis.

### Test live melalui production

Lakukan smoke test pada URL `https://gdcpay-test-project.onrender.com`.

```sh
API_URL="https://gdcpay-test-project.onrender.com"
curl -i "$API_URL/health"
curl -i "$API_URL/ready"
curl -i "$API_URL/openapi.yaml"
```

Untuk mencoba alur register/login dan seluruh CRUD task, siapkan `jq` dan `uuidgen`, lalu jalankan di shell lokal. Email dan idempotency key dibuat unik setiap kali skrip dijalankan.

```sh
API_URL="https://gdcpay-test-project.onrender.com"
TEST_EMAIL="gdcpay-live-$(date +%s)@example.com"
TEST_PASSWORD="replace-with-a-test-only-password"
IDEMPOTENCY_KEY="$(uuidgen | tr '[:upper:]' '[:lower:]')"

curl -fsS -X POST "$API_URL/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"

API_TOKEN="$(curl -fsS -X POST "$API_URL/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" | jq -r '.access_token')"

CREATE_RESPONSE="$(curl -fsS -X POST "$API_URL/tasks" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Idempotency-Key: $IDEMPOTENCY_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Production smoke test","description":"Temporary test task"}')"
TASK_ID="$(printf '%s' "$CREATE_RESPONSE" | jq -r '.id')"
printf '%s\n' "$CREATE_RESPONSE"

curl -fsS "$API_URL/tasks?page=1&limit=20" -H "Authorization: Bearer $API_TOKEN"
curl -fsS "$API_URL/tasks/$TASK_ID" -H "Authorization: Bearer $API_TOKEN"
curl -fsS -X PUT "$API_URL/tasks/$TASK_ID" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Production smoke test updated","description":"Temporary test task","status":"in_progress"}'
curl -i -X DELETE "$API_URL/tasks/$TASK_ID" -H "Authorization: Bearer $API_TOKEN"
```

Variabel `API_TOKEN` hanya berada di shell lokal; jangan tampilkan di log, commit, atau bagikan. Contoh tersebut hanya menghapus task yang dibuat oleh smoke test ini. Periksa status CI di GitHub Actions dan status akhir deployment di dashboard Render, karena respons deploy hook hanya menandakan bahwa deployment diminta.

## Endpoint yang tersedia

| Method | Path | Fungsi |
| --- | --- | --- |
| `GET` | `/health` | Liveness check; memeriksa proses aplikasi |
| `GET` | `/ready` | Readiness check; memeriksa koneksi PostgreSQL |
| `POST` | `/auth/register` | Membuat akun dan mengembalikan JWT |
| `POST` | `/auth/login` | Memvalidasi akun dan mengembalikan JWT |
| `POST` | `/tasks` | Membuat task secara idempotent; wajib menyertakan header `Idempotency-Key` berformat UUID |
| `GET` | `/tasks` | Daftar task milik pengguna; mendukung filter status, pencarian judul, dan pagination |
| `GET` | `/tasks/{id}` | Detail task milik pengguna |
| `PUT` | `/tasks/{id}` | Memperbarui judul, deskripsi, dan status task |
| `DELETE` | `/tasks/{id}` | Menghapus task milik pengguna |
| `GET` | `/openapi.yaml` | Spesifikasi OpenAPI untuk endpoint yang sudah tersedia |
| `GET` | `/swagger/` | Swagger UI interaktif |

Swagger UI menggunakan aset dari unpkg CDN. Spesifikasi OpenAPI disediakan oleh aplikasi di `/openapi.yaml`.

Contoh:

```sh
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
curl -i -X POST http://localhost:8080/auth/register -H 'Content-Type: application/json' -d '{"email":"user@example.com","password":"correct-horse-battery"}'
curl -i -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"email":"user@example.com","password":"correct-horse-battery"}'
IDEMPOTENCY_KEY="$(uuidgen | tr '[:upper:]' '[:lower:]')"
curl -i -X POST http://localhost:8080/tasks -H 'Authorization: Bearer <token>' -H "Idempotency-Key: $IDEMPOTENCY_KEY" -H 'Content-Type: application/json' -d '{"title":"Prepare report","description":"Quarterly summary"}'
curl -i 'http://localhost:8080/tasks?page=1&limit=20&status=pending&search=report' -H 'Authorization: Bearer <token>'
curl -i http://localhost:8080/openapi.yaml
```

Task baru berstatus `pending`. Status yang tersedia adalah `pending`, `in_progress`, dan `completed`. Query list menerima `page` (default 1), `limit` (default 20, maksimum 100), `status`, dan `search` (pencarian substring judul, tidak peka huruf besar/kecil). Semua endpoint `/tasks` membutuhkan bearer token. Pengguna hanya dapat membaca dan mengubah task miliknya; task pengguna lain diperlakukan sebagai tidak ditemukan.

`POST /tasks` mewajibkan header `Idempotency-Key` berupa UUID. Request berulang dengan key dan payload yang sama dalam 24 jam mengembalikan task yang sama tanpa membuat duplikat. Key yang sama dengan payload berbeda menghasilkan `409 IDEMPOTENCY_KEY_REUSED`. Scope key adalah per pengguna.

Migration tabel users berjalan otomatis ketika API mulai. Password disimpan dalam bentuk bcrypt hash; email dinormalisasi ke huruf kecil. Register mensyaratkan email valid dan password sepanjang 8–72 byte. JWT ditandatangani dengan HS256 dan masa berlakunya diatur oleh `JWT_TTL`.

## Arsitektur singkat

- `cmd/api` memuat konfigurasi, menyiapkan koneksi PostgreSQL dan logger, lalu menjalankan HTTP server.
- `internal/config` membaca konfigurasi environment.
- `internal/auth` menangani aturan register/login, hashing password, dan penerbitan/verifikasi JWT.
- `internal/task` berisi aturan bisnis dan validasi task.
- `internal/repository/postgres` mengimplementasikan akses data PostgreSQL untuk autentikasi dan task.
- `internal/database/migrations` menyimpan migrasi SQL yang di-embed ke binary dan dijalankan saat startup.
- `internal/httpapi` merangkai routes dan handler HTTP.
- `internal/middleware` menyediakan request ID, structured access log, panic recovery, dan JWT bearer authentication reusable.
- `internal/response` menyediakan format error JSON bersama.
- `compose.yaml` menjalankan API dan PostgreSQL untuk development lokal.
- `.github/workflows/ci.yml` menjalankan pemeriksaan CI dan memicu deployment production dari `main`.

Alur fitur akan mengikuti `handler → service → repository → PostgreSQL`. Layer service dan repository akan ditambahkan bersama fitur agar interface dan abstraksinya mengikuti kebutuhan nyata.

## Checklist requirement

Status **Belum** berarti belum diimplementasikan; item tersebut sengaja dicatat agar progres bisa ditinjau dan diverifikasi.

| Requirement | Status | Lokasi atau bukti verifikasi |
| --- | --- | --- |
| Go backend dan framework Gin | Selesai | `go.mod`, `cmd/api`, `internal/httpapi` |
| PostgreSQL | Sebagian | Pool, migrasi, tabel users/tasks, repository autentikasi dan task, serta readiness tersedia; fitur idempotency dan assign menyusul |
| Docker deployment | Sebagian | `Dockerfile`, `compose.yaml`; deployment image perlu dikonfigurasi sesuai platform |
| Branch development dan production terpisah | Selesai | Branch `development` untuk basis kerja; `main` untuk rilis production |
| CI/CD GitHub Actions | Selesai | `.github/workflows/ci.yml`; verifikasi pada perubahan branch dan deploy hook hanya dari `main` jika diaktifkan |
| README: panduan test live di production | Selesai | Bagian “Test live melalui production” untuk health, readiness, auth, dan CRUD |
| Health check | Selesai | `GET /health` |
| Readiness check termasuk PostgreSQL | Selesai | `GET /ready` |
| Dokumentasi Swagger/OpenAPI untuk semua API | Sebagian | `/swagger/` dan `/openapi.yaml` mencakup health, auth, dan CRUD task; dokumentasi API fitur berikutnya menyusul |
| README: cara menjalankan, endpoint, arsitektur | Selesai | Bagian di atas |
| README: checklist keterpenuhan seluruh requirement | Selesai | Tabel checklist ini |
| Register dan login | Selesai | `POST /auth/register`, `POST /auth/login`; service dan repository auth |
| JWT authentication | Selesai | Token HS256 dan middleware reusable; seluruh route task memakai bearer authentication |
| Create, list, detail, update, delete task | Selesai | `POST/GET /tasks`, `GET/PUT/DELETE /tasks/{id}`; service dan repository PostgreSQL |
| Filter status, search title, pagination | Selesai | `GET /tasks` menerima status, search, page, dan limit |
| Idempotency key UUID selama 24 jam | Selesai | Header wajib pada `POST /tasks`; hasil dan hash request tersimpan 24 jam dengan unique constraint `(owner_id, idempotency_key)` |
| Error JSON konsisten, 4xx/5xx, tanpa detail internal | Selesai | Handler API, route 404/405, dan error Swagger memakai format bersama; middleware menangani unhandled error dan panic tanpa mengirim detail internal |
| Assign dalam satu tim dan transaksi DB | Belum | Tahap assign; model tim/membership dan task logs belum dibuat |
| Rollback bila langkah assign gagal | Belum | Akan diverifikasi pada implementasi transaksi assign |
| Notifikasi assign (mock/log) | Belum | Tahap assign |
| Structured request logging | Selesai | Middleware access log: request ID, method, path, status, latency, level |
| Unit test tanpa database/service eksternal | Selesai | Test idempotency sequential dan concurrent memakai repository in-memory; test autentikasi juga memakai stub |

Unit test service autentikasi tersedia dan menggunakan repository stub. Test service task memakai repository in-memory untuk memeriksa replay sequential, 64 request bersamaan, penggunaan ulang key dengan payload berbeda, dan expiry dengan clock test yang dapat dimajukan tanpa menunggu 24 jam. Unit test middleware dan routing juga memeriksa format error, panic recovery, dan agar detail internal tidak bocor. Jalankan seluruh unit test dengan `go test ./...`.

## Pengembangan

Jalankan server lokal dengan `make run`, atau gunakan `make compose-up` dan `make compose-down` untuk Docker Compose. Perubahan fitur berikutnya harus memperbarui OpenAPI dan checklist ini bersama implementasinya.
