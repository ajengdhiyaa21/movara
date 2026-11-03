# MOVARA — Tahap 4D: Laporan Penguncian Policy Database Test

Tanggal: 27 Oktober 2026  
Scope: penguatan validasi konfigurasi test tanpa koneksi MySQL.

## 1. File yang berubah

- `internal/config/testdb.go`
- `internal/config/testdb_test.go`
- `docs/CHAT-STAGE-4D-TEST-POLICY-REPORT.md` (laporan ini)

Perubahan lokal lain dipertahankan dan tidak ditimpa. Startup aplikasi, `DATABASE_URL`, `.env`, dan `cmd/server/main.go` tidak diubah.

## 2. Penguatan policy

`TestDatabasePolicy` sekarang mewajibkan allowlist eksplisit untuk database, host, port, dan user database test. Policy kosong, tidak lengkap, memiliki nilai kosong/spasi, wildcard, newline, atau port di luar rentang valid ditolak.

Helper tidak membuat nilai policy dari `TEST_DATABASE_URL`; policy harus diinjeksi pemanggil dari sumber yang disetujui. Sumber policy deployment final belum ada di repository, sehingga target lokal, username, host, dan port final tetap **NEEDS APPROVAL**.

## 3. DSN dan kebocoran credential

`TestDatabaseConfig` tidak lagi menyimpan DSN mentah. Hasil validasi hanya menyimpan database, host, port, dan user yang sudah diparse. Helper tidak mencetak DSN, password, atau username ke error.

Parser MySQL digunakan untuk membaca DSN, tetapi helper tidak membuka connector. Username hanya metadata terstruktur untuk pemanggil dan tidak boleh ditulis ke log.

## 4. Unit test guardrail

`internal/config/testdb_test.go` mencakup:

1. Environment kosong atau `APP_ENV` selain `test` ditolak.
2. `TEST_DATABASE_URL` kosong ditolak.
3. `DATABASE_URL` tidak menjadi fallback.
4. DSN tidak valid ditolak.
5. Database, host, dan port di luar policy ditolak.
6. Policy kosong/tidak lengkap atau port invalid ditolak.
7. Konfigurasi valid berdasarkan policy yang diinjeksi diterima.
8. Error tidak mengandung password, username, atau nama database dari input.

Tidak ada pemanggilan `gorm.Open`, `database.Connect`, SQL, migration, atau seeder pada helper maupun unit test.

## 5. Verifikasi aktual

Test yang ada diperiksa terlebih dahulu: middleware memakai `httptest`, service memakai memory repository, dan test config hanya memakai parser/validator. Tidak ditemukan koneksi database langsung maupun tidak langsung.

Perintah yang dijalankan:

```text
gofmt -w internal/config/testdb.go internal/config/testdb_test.go
go test ./...
go vet ./...
go build ./...
```

Hasil:

- `go test ./...` — berhasil.
- `go vet ./...` — berhasil.
- `go build ./...` — berhasil.

## 6. Keterbatasan dan NEEDS APPROVAL

- Belum ada target database/host/port/user lokal yang disetujui.
- Policy masih diinjeksi pemanggil; belum ada provider policy tepercaya untuk CI/lokal.
- Privilege MySQL belum diverifikasi karena tidak ada koneksi/database yang boleh dibuka.
- Belum ada integration test MySQL, migration berversi, atau fixture test.
- **NEEDS APPROVAL:** sumber resmi allowlist dan format secret test.
- **NEEDS APPROVAL:** apakah metadata `User` tetap diperlukan atau dibuat opaque pada tahap koneksi berikutnya.

Tidak boleh mengklaim akun test terisolasi atau tidak memiliki akses production sebelum privilege diverifikasi pada setup MySQL terpisah.

## 7. Konfirmasi keselamatan

Tidak ada koneksi MySQL, `gorm.Open`, SQL, AutoMigrate, Seed, migration, server, atau executable aplikasi yang dijalankan. Tidak ada database, schema, credential, `DATABASE_URL`, `.env`, startup, atau dependency yang diubah. Fitur chat dan CSRF tidak diimplementasikan.
