# MOVARA — Tahap 4B: Laporan Guardrail Database Test

Tanggal: 27 Oktober 2026  
Scope: helper validasi konfigurasi database test tanpa koneksi MySQL.

## 1. File yang diubah/dibuat

- `internal/config/testdb.go` — helper validasi fail-closed.
- `internal/config/testdb_test.go` — unit test guardrail tanpa koneksi.
- `docs/CHAT-STAGE-4B-TEST-GUARDRAIL-REPORT.md` — laporan ini.

Perubahan lokal lain yang sudah ada dipertahankan dan tidak ditimpa.

## 2. Perilaku helper

`ValidateTestDatabase` menerima map environment dan `TestDatabasePolicy` yang diinjeksi pemanggil. Helper:

- hanya membaca `APP_ENV` dan `TEST_DATABASE_URL`;
- mensyaratkan `APP_ENV=test`;
- menolak `TEST_DATABASE_URL` kosong;
- tidak membaca atau fallback ke `DATABASE_URL`, `.env`, default DSN, atau `config.Load()`;
- memakai parser MySQL yang sudah tersedia melalui dependency existing;
- memvalidasi database name, host, dan port terhadap allowlist eksplisit;
- menolak unix socket pada policy saat ini;
- mewajibkan user DSN tidak kosong;
- mengembalikan error generik tanpa mencetak DSN atau credential;
- tidak memanggil `gorm.Open`, `database.Connect`, SQL, migration, atau seeder.

Policy tidak memiliki default target deployment. Caller wajib memberikan allowlist database, host, dan port yang sudah disetujui. Ini mencegah target lokal/production ditebak dari suffix `_test` saja.

Helper belum memverifikasi privilege MySQL secara runtime; hal tersebut tetap membutuhkan setup akun test dan pemeriksaan terpisah.

## 3. Unit test yang ditambahkan

`internal/config/testdb_test.go` menguji:

1. Environment kosong atau bukan `test` ditolak.
2. `TEST_DATABASE_URL` kosong ditolak walaupun `DATABASE_URL` tersedia.
3. DSN tidak valid ditolak.
4. Nama database di luar allowlist ditolak.
5. Host dan port di luar policy ditolak.
6. DSN valid sesuai policy eksplisit diterima.
7. Error tidak memuat password test.

Semua test memakai DSN sintetis dan tidak membuka koneksi database.

## 4. Verifikasi keamanan sebelum test

Test yang tersedia diperiksa: middleware memakai `httptest`, service memakai memory repository, dan test config baru hanya memanggil helper parser/validator. Tidak ditemukan pemanggilan `database.Connect`, `database.Migrate`, `database.Seed`, atau `gorm.Open` dari test tersebut.

## 5. Perintah yang dijalankan

```text
gofmt -w internal/config/testdb.go internal/config/testdb_test.go
go test ./...
go vet ./...
go build ./...
```

Hasil aktual:

- `go test ./...` — berhasil; package `internal/config`, middleware, dan service lulus.
- `go vet ./...` — berhasil.
- `go build ./...` — berhasil.

Tidak ada server atau executable aplikasi yang dijalankan.

## 6. Keterbatasan dan pekerjaan berikutnya

- Nama database, host, port, dan user test final belum disetujui; policy saat ini hanya contoh eksplisit yang diinjeksi unit test.
- Belum ada koneksi MySQL test dan belum ada integration test.
- Privilege akun database belum dapat diverifikasi pada tahap tanpa koneksi.
- Belum ada perubahan pada `cmd/server/main.go`, `DATABASE_URL`, startup, migration, atau seeder.
- Helper belum dipakai oleh test integration karena setup database test belum disetujui.
- Belum ada migration tool berversi atau fixture/seeder khusus test.

Keputusan yang masih diperlukan: target database test, user/privilege, strategi lokal/CI, dan cara menjalankan migration/fixture secara eksplisit.

## 7. Konfirmasi keselamatan

Tidak ada koneksi database, SQL, migration, seeder, AutoMigrate, startup server, perubahan schema, perubahan credential, atau perubahan `DATABASE_URL` yang dilakukan. Tidak ada dependency baru yang ditambahkan. Fitur chat dan CSRF tidak diimplementasikan.
