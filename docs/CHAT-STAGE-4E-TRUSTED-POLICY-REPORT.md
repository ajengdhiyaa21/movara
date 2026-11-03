# MOVARA — Tahap 4E: Laporan Sumber Policy Tepercaya

Tanggal: 27 Oktober 2026  
Scope: policy target database test lokal tanpa koneksi MySQL.

## 1. File yang berubah

- `internal/config/testdb_policy.go` — sumber policy approved non-secret.
- `internal/config/testdb_test.go` — test target approved.
- `docs/CHAT-STAGE-4E-TRUSTED-POLICY-REPORT.md` — laporan ini.

File lain dan perubahan lokal sebelumnya dipertahankan.

## 2. Sumber policy yang dipilih

`ApprovedTestDatabasePolicy()` di `internal/config/testdb_policy.go` menggunakan target yang sudah disetujui: database `movara_chat_test`, host `127.0.0.1`, port `3306`, dan user `movara_test`.

Policy hanya berisi identifier non-secret, tanpa password atau DSN, dan tidak digunakan oleh `cmd/server/main.go` maupun startup aplikasi. Function mengembalikan map baru setiap pemanggilan agar state policy tidak dibagi antar pemanggil.

`ValidateTestDatabase` tetap menerima policy terpisah dari environment target. Allowlist tidak dibangun dari `TEST_DATABASE_URL` yang sedang divalidasi.

## 3. Validasi dan keamanan

Validasi mensyaratkan `APP_ENV=test`, `TEST_DATABASE_URL`, policy database/host/port/user yang valid, dan kecocokan target DSN dengan policy approved. Tidak ada fallback ke `DATABASE_URL`, `.env`, atau default DSN. Error tidak memuat DSN, username, password, atau nama database input.

Helper hanya mem-parse dan memvalidasi DSN; tidak ada fungsi yang membuka koneksi database. Privilege user belum diverifikasi karena database tidak disentuh.

## 4. Unit test

`internal/config/testdb_test.go` menguji environment salah, `TEST_DATABASE_URL` kosong, tidak ada fallback `DATABASE_URL`, DSN invalid, target database/host/port/user di luar policy, policy kosong/invalid, policy approved yang menolak target lain, konfigurasi valid, dan error tanpa credential/target input.

Seluruh test menggunakan nilai sintetis dan tidak memanggil connector.

## 5. Verifikasi aktual

Test diperiksa sebelum dijalankan; tidak ditemukan `database.Connect`, `gorm.Open`, SQL, migration, seeder, atau startup aplikasi pada jalur test.

```text
gofmt -w internal/config/testdb_policy.go internal/config/testdb_test.go
go test ./...
go vet ./...
go build ./...
```

Hasil: `go test ./...` berhasil, `go vet ./...` berhasil, dan `go build ./...` berhasil.

## 6. Keterbatasan dan NEEDS APPROVAL

- Database `movara_chat_test` belum diverifikasi ada.
- Port XAMPP `3306` belum diverifikasi.
- User `movara_test` dan privilege-nya belum diverifikasi.
- Policy approved berada di source code, tetapi pemanggil integration test dan provider policy CI belum dibuat.
- **NEEDS APPROVAL:** prosedur pembuatan database/user dan verifikasi privilege.
- **NEEDS APPROVAL:** apakah policy lokal dipakai CI atau CI memiliki provider terpisah.
- **NEEDS APPROVAL:** strategi fixture dan migration.

Username atau suffix database tidak dianggap bukti isolasi privilege.

## 7. Konfirmasi keselamatan

Tidak ada koneksi MySQL, `gorm.Open`, SQL, migration, AutoMigrate, Seed, server, executable aplikasi, database, schema, credential, `DATABASE_URL`, atau `.env` yang diubah/disentuh. Tidak ada dependency baru, chat, atau CSRF yang diimplementasikan.
