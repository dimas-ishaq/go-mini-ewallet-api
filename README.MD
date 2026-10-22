# Mini E-Wallet API

REST API sederhana untuk registrasi dan login pengguna, pengecekan saldo, serta transfer antar-dompet. Dibangun dengan Go, Gin, dan PostgreSQL.

> Status: proyek pembelajaran. Sebelum production, tinjau catatan keamanan dan keterbatasan di bawah.

## Fitur

- Registrasi pengguna dengan password hash bcrypt dan pembuatan wallet.
- Login dan token JWT.
- Endpoint terlindungi untuk melihat saldo dan melakukan transfer.
- Migrasi skema PostgreSQL otomatis saat aplikasi dimulai.

## Teknologi

- Go (versi minimum sesuai `go.mod`)
- PostgreSQL
- Gin
- golang-migrate

## Struktur proyek

```text
cmd/api/                 Entry point dan pendaftaran route
internal/delivery/http/  Handler HTTP, request, response, middleware
internal/domain/         Model dan interface repository
internal/repository/     Implementasi query PostgreSQL
internal/usecase/        Aturan bisnis user dan wallet
pkg/database/            Koneksi database dan runner migrasi
pkg/database/migrations/ Berkas migrasi SQL
```

## Prasyarat

- Go terpasang.
- PostgreSQL berjalan dan database sudah dibuat.
- Akses jaringan dari aplikasi ke PostgreSQL.

## Menjalankan lokal

1. Salin template konfigurasi:

   ```sh
   cp .env.example .env
   ```

   Windows PowerShell:

   ```powershell
   Copy-Item .env.example .env
   ```

2. Edit `.env`. Isi `DB_PASSWORD` dengan password lokal database. Jangan commit `.env`.
3. Buat database yang disebutkan pada `DB_NAME`, misalnya `ewallet_db`.
4. Jalankan aplikasi dari direktori proyek:

   ```sh
   go run ./cmd/api
   ```

Aplikasi membaca konfigurasi, membuka koneksi PostgreSQL, menjalankan migrasi yang tertanam dalam binary, lalu membuka HTTP server. Port default `8080`; ubah melalui `PORT`.

## Konfigurasi environment

| Variabel | Fungsi | Contoh lokal |
|---|---|---|
| `PORT` | Port HTTP server | `8080` |
| `DB_HOST` | Host PostgreSQL | `localhost` |
| `DB_PORT` | Port PostgreSQL | `5432` |
| `DB_USER` | User database | `postgres` |
| `DB_PASSWORD` | Password database | isi lokal, jangan commit |
| `DB_NAME` | Nama database yang sudah dibuat | `ewallet_db` |
| `DB_SSLMODE` | Mode TLS PostgreSQL | `disable` hanya untuk lokal |

## Endpoint

Base path endpoint aplikasi: `/api/v1`.

| Method | Path | Autentikasi | Keterangan |
|---|---|---|---|
| `GET` | `/api/health` | Tidak | Cek status API/database |
| `POST` | `/api/v1/register` | Tidak | Registrasi user dan wallet |
| `POST` | `/api/v1/login` | Tidak | Login dan memperoleh JWT |
| `GET` | `/api/v1/wallet/balance` | JWT | Membaca wallet pengguna |
| `POST` | `/api/v1/wallet/transfer` | JWT | Transfer ke nomor rekening tujuan |

Kirim token untuk route terlindungi:

```http
Authorization: Bearer <token>
```

### Contoh request

Registrasi:

```sh
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Budi Santoso","email":"budi@example.com","password":"password-lokal"}'
```

Login:

```sh
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"budi@example.com","password":"password-lokal"}'
```

Transfer:

```sh
curl -X POST http://localhost:8080/api/v1/wallet/transfer \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"destination_account":"<nomor-rekening-tujuan>","amount":1000}'
```

## Migrasi database

Migrasi berada di `pkg/database/migrations/` dan dikemas ke binary oleh `//go:embed`. Migrasi `*.up.sql` membuat skema; `*.down.sql` membatalkan migrasi. `RunMigrations` menjalankan migrasi ke versi terbaru saat startup. Pastikan user database memiliki hak membuat tabel dan mencatat versi migrasi.

## Pemeriksaan

```sh
go test ./...
go vet ./...
go build ./...
```

## Persiapan GitHub

- Pastikan `.env` tidak masuk commit; `.gitignore` mengecualikan `.env` dan `.env.*`, kecuali `.env.example`.
- Ganti password, secret JWT, serta kredensial contoh dengan secret aman di deployment. Jangan simpan secret di source control.
- Jika `.env` pernah di-commit, `.gitignore` tidak menghapusnya dari riwayat Git. Hapus dari index (`git rm --cached .env`) dan rotasi semua secret yang pernah tersimpan.
- Periksa `git diff` dan daftar file staged sebelum push agar tidak mengunggah secret.

## Catatan keamanan dan batasan

- `DB_SSLMODE=disable` hanya untuk pengembangan lokal; gunakan TLS pada lingkungan deployment.
- `jwtSecret` saat ini ditetapkan dalam kode. Jangan gunakan nilai contoh di production; pindahkan ke secret environment/configuration sebelum deployment publik.
- Transfer melibatkan perubahan saldo dalam transaksi database. Pengelolaan uang production perlu peninjauan tambahan, termasuk constraint unik, concurrency, format nominal, audit, dan integrasi transaksi user-wallet secara atomik.
- Endpoint health saat ini melaporkan status database sebagai `Connected`; jangan anggap ini sebagai pemeriksaan kesiapan database terkini.
