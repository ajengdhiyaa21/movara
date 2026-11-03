# MOVARA — Tahap 4C: Rencana Finalisasi Database Test

Tanggal audit: 27 Oktober 2026  
Mode: read-only  
Scope: rencana database test lokal yang fail-closed sebelum integration test MySQL.

## 1. Temuan aktual berdasarkan kode

### Konfigurasi dan startup

- `internal/config/config.go:17-29` membaca `.env` secara opsional dan mengambil `DATABASE_URL` untuk konfigurasi aplikasi.
- `internal/database/database.go:10-12` meneruskan DSN langsung ke GORM MySQL.
- `cmd/server/main.go:24-32` melakukan koneksi, `Migrate`, dan `Seed` sebelum server berjalan.
- `internal/database/database.go:14-49` memakai `AutoMigrate` dan operasi SQL kompatibilitas yang dapat mengubah schema/data.

Kesimpulan: startup aplikasi tidak boleh dipakai sebagai harness database test.

### Helper Tahap 4B

- `internal/config/testdb.go:12-18` mendefinisikan `TestDatabasePolicy` tanpa default target.
- `internal/config/testdb.go:28-61` hanya menerima `APP_ENV` dan `TEST_DATABASE_URL` dari map yang diberikan, lalu memvalidasi DSN, database, host, port, dan user.
- `internal/config/testdb.go:64-72` menolak unix socket dan memecah host/port.
- `internal/config/testdb_test.go:11-56` menguji konfigurasi kosong, environment salah, tidak ada fallback ke `DATABASE_URL`, DSN/target salah, konfigurasi valid berdasarkan policy yang diinjeksi, dan error tanpa password.

Helper tidak membuka koneksi database. Namun, policy dapat dibuat sembarang oleh pemanggil. Kode saat ini belum menyediakan sumber policy yang disetujui organisasi, belum memverifikasi privilege MySQL, dan belum mencegah pemanggil mencetak field `DSN` dari hasil validasi. Karena itu helper validasinya fail-closed terhadap input kosong/salah, tetapi keamanan final tetap bergantung pada bagaimana policy dan hasilnya dipakai.

## 2. Keputusan yang sudah diketahui dan NEEDS APPROVAL

### Sudah diketahui

- Test harus memakai `TEST_DATABASE_URL`, bukan `DATABASE_URL`.
- Environment harus eksplisit `APP_ENV=test`.
- Allowlist database, host, dan port wajib non-kosong.
- Startup server tidak boleh dijalankan untuk menyiapkan test.
- `AutoMigrate`/`Seed` aplikasi tidak boleh menjadi side effect integration test.

### NEEDS APPROVAL

- Nama database/schema test final.
- Host dan port MySQL lokal yang disetujui.
- Nama akun test dan privilege minimumnya.
- Apakah lokal memakai schema terpisah pada instance XAMPP atau instance terpisah.
- Sumber policy yang dipercaya: environment CI, file konfigurasi test di luar repository, atau secret manager.
- Tool migration SQL berversi.
- Strategi fixture dan reset database.
- Apakah raw DSN perlu dihapus dari `TestDatabaseConfig` untuk mencegah logging tidak sengaja.
- Prosedur backup dan forward-fix production.

Tidak ada nilai target yang ditebak dalam laporan ini.

## 3. Rancangan konfigurasi database test

Konfigurasi final yang disarankan:

```text
APP_ENV=test
TEST_DATABASE_URL=<credential test>@tcp(<host-test>:<port-test>)/<database-test>
```

Nilai di atas hanya bentuk kontrak, bukan konfigurasi aktual.

Policy yang disetujui harus menyediakan allowlist database, host, dan port dari sumber terpercaya. Pemanggil tidak boleh membangun allowlist dari nilai `TEST_DATABASE_URL` yang sama pada runtime; itu akan membuat validasi tidak bermakna. Policy sebaiknya dikunci oleh test helper/CI configuration dan direview sebagai bagian deployment/test setup.

Urutan aman:

1. Ambil environment test eksplisit.
2. Pastikan `APP_ENV=test`.
3. Pastikan `TEST_DATABASE_URL` ada; jangan membaca fallback.
4. Validasi DSN secara sintaksis.
5. Validasi database/host/port terhadap policy tepercaya.
6. Validasi user tidak kosong dan bukan akun production berdasarkan policy yang disetujui.
7. Hanya setelah semua validasi lulus, pemanggil boleh membuka koneksi.

Validasi nama database saja tidak cukup. Hak akses harus dibatasi pada schema test, tetapi pembuktian privilege memerlukan setup MySQL terpisah dan belum dilakukan.

## 4. Ancaman dan mitigasi

| Ancaman | Mitigasi |
|---|---|
| Fallback ke `DATABASE_URL` | Helper hanya membaca `TEST_DATABASE_URL`; test gagal jika kosong |
| `.env` aplikasi terbaca | Jangan memakai `config.Load()` dari integration helper; injeksikan env test |
| Policy dibuat terlalu longgar | Policy harus berasal dari konfigurasi yang direview, bukan dari DSN input |
| Suffix `_test` dianggap aman | Validasi database + host + port + user + privilege |
| Akun production dipakai | Credential test khusus dan policy melarang akun/target production |
| DSN tercetak ke log | Error generik; jangan log `TestDatabaseConfig.DSN`; pertimbangkan menghapus raw DSN dari hasil helper |
| Startup menjalankan seed/migration | Integration test tidak memanggil `cmd/server/main.go` |
| Test paralel saling menghapus data | Database/schema per run atau namespace fixture unik; keputusan reset masih NEEDS APPROVAL |
| Migration gagal parsial | Migration eksplisit, version table, stop on failure, recovery terkontrol |
| Cleanup menghapus data salah | Cleanup hanya setelah target tervalidasi dan hanya di schema test |

## 5. Rancangan migration dan fixture

### Migration

Gunakan migration SQL berversi yang dijalankan sebagai command terpisah dari server startup. Tool belum dipilih. Migration pertama harus diuji pada database kosong dan baseline schema yang merepresentasikan aplikasi saat ini.

AutoMigrate lama jangan diperluas diam-diam untuk tabel chat. Selama transisi, migration deployment harus memiliki target environment eksplisit dan menyimpan versi yang berhasil. Jika gagal, proses berhenti; jangan menjalankan versi berikutnya pada schema dirty.

Rollback destructive tidak direkomendasikan. Untuk tabel/pesan chat, gunakan forward-fix, kolom nullable/backfill, kompatibilitas sementara, dan restore terkontrol dari backup bila diperlukan. Jangan otomatis `DROP TABLE`, `DROP COLUMN`, atau menghapus pesan.

### Fixture test

Fixture khusus test harus terpisah dari `database.Seed`. Fixture perlu deterministik dan mencakup:

- Admin aktif, Manager, PM aktif/nonaktif;
- outlet aktif/nonaktif;
- kategori dan produk;
- conversation/membership;
- unread state antar-admin;
- pesan kosong, batas 3.000 karakter, script, dan retry/idempotency.

Fixture tidak boleh berjalan dari startup aplikasi. Pembersihan hanya boleh menarget database/schema test yang telah divalidasi; strategi database per-run lebih aman daripada cleanup luas, tetapi masih NEEDS APPROVAL.

## 6. Strategi lokal XAMPP dan CI

### Lokal saat ini

Pilihan paling realistis adalah schema/database test terpisah pada MySQL lokal, dengan akun test khusus dan privilege hanya pada target test. Namun belum ada bukti MySQL XAMPP aktif/terjangkau; jangan mencoba koneksi pada tahap ini.

Prasyarat lokal: target disetujui, database dibuat oleh operator berwenang, user test dibuat, privilege diverifikasi, dan policy disimpan di luar source code.

### CI masa depan

Instance/container MySQL khusus test lebih reproducible dan mengurangi risiko target lokal. CI harus menyuplai `TEST_DATABASE_URL` dan policy melalui secret/configuration CI, menjalankan migration eksplisit, fixture, test, lalu menghancurkan environment test. Docker/Podman belum terdeteksi/terverifikasi pada proyek sehingga belum boleh diasumsikan.

## 7. Urutan implementasi berikutnya

1. Dapatkan persetujuan semua item NEEDS APPROVAL.
2. Tetapkan policy tepercaya dan dokumentasikan tanpa secret.
3. Pisahkan helper koneksi integration dari `config.Load()` dan startup.
4. Tambahkan guardrail yang memastikan policy tidak dibangun dari target input.
5. Siapkan database/user test secara manual atau melalui platform yang disetujui.
6. Verifikasi privilege dan target secara terkontrol.
7. Pilih dan implementasikan migration berversi.
8. Jalankan migration hanya pada database test.
9. Tambahkan fixture/reset yang scoped.
10. Tambahkan integration test MySQL dan regression test chat/mutasi.

## 8. Kriteria penerimaan

- Test tanpa `TEST_DATABASE_URL` gagal sebelum koneksi.
- `APP_ENV` bukan `test` gagal.
- Policy kosong atau target di luar allowlist gagal.
- Tidak ada fallback ke `DATABASE_URL` atau `.env` aplikasi.
- Credential test bukan credential production dan privilege hanya target test.
- Startup aplikasi tidak dijalankan oleh integration test.
- Migration test eksplisit dan memiliki versi.
- Fixture tidak memanggil seeder production/development.
- Cleanup hanya menyentuh target test tervalidasi.
- Migration failure menghentikan proses tanpa destructive rollback otomatis.
- Test dapat dijalankan ulang tanpa menyentuh data non-test.

## 9. Perintah aman dan yang belum boleh dijalankan

### Aman pada tahap read-only ini

- Membaca file kode, laporan, dan konfigurasi tanpa nilai rahasia.
- `git status --short`.
- Pemeriksaan struktur file dan dependency tanpa menjalankan aplikasi.
- Menyusun/meninjau policy secara offline.

### Belum boleh dijalankan sebelum persetujuan dan setup

- `go run`, `server.exe`, atau startup aplikasi.
- `database.Connect`, `gorm.Open`, atau command yang membuka koneksi MySQL.
- `AutoMigrate`, `Seed`, migration, SQL, reset, truncate, drop, atau cleanup database.
- Integration test yang memakai MySQL.
- Pembuatan database/user/privilege XAMPP.
- Setup container/CI yang belum disetujui.

## 10. Pernyataan perubahan

File yang dibaca: enam laporan acuan yang disebutkan, `internal/config/testdb.go`, `internal/config/testdb_test.go`, `internal/database/database.go`, dan `cmd/server/main.go`.

File yang dibuat: `docs/CHAT-STAGE-4C-TEST-DATABASE-PLAN.md`.

Tidak ada kode, konfigurasi, dependency, database, schema, migration, seeder, atau laporan lain yang diubah. Tidak ada koneksi MySQL, SQL, server, AutoMigrate, Seed, atau integration test yang dijalankan.
