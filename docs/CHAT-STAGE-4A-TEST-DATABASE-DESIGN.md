# MOVARA — Tahap 4A: Desain Database Test Fail-Closed

Tanggal audit: 27 Oktober 2026  
Mode: read-only  
Scope: persiapan database test MySQL terisolasi sebelum integration test dan chat.

## 1. Temuan kode aktual

### Konfigurasi koneksi

- `internal/config/config.go:17-29` membaca `.env` secara opsional melalui `godotenv.Load()` dan mengambil `DATABASE_URL` dari environment.
- `internal/config/config.go:31-38` hanya memvalidasi panjang `SESSION_SECRET`; belum ada validasi keberadaan/target aman untuk `DATABASE_URL`.
- `cmd/server/main.go:18-24` meneruskan `cfg.DatabaseURL` langsung ke `database.Connect`.
- `internal/database/database.go:9-12` membuka GORM MySQL dengan DSN yang diberikan.

Tidak ada `TEST_DATABASE_URL`, `APP_ENV=test`, allowlist database, atau pembanding yang mencegah test memakai `DATABASE_URL` aplikasi. Nilai environment rahasia tidak dibaca ke laporan.

### Startup yang dapat mengubah schema/data

`cmd/server/main.go:25-28` memanggil `database.Migrate` dan `database.Seed` sebelum server. `internal/database/database.go:14-51` memakai `AutoMigrate` dan SQL kompatibilitas yang dapat alter/update schema/data. Karena itu server startup bukan jalur yang aman untuk test database.

### Test saat ini

Test yang ditemukan:

- `internal/middleware/csrf_test.go`: Gin `httptest`, tidak memakai database.
- `internal/middleware/auth_test.go`: fake `repository.Outlets`, tidak memakai database.
- `internal/service/mutation_test.go`: memory repository melalui `memory.New()`, tidak memakai MySQL.

Tidak ditemukan test yang memanggil `database.Connect`, `database.Migrate`, `database.Seed`, atau repository MySQL. Berdasarkan kode yang tersedia, test saat ini aman dijalankan tanpa database. Belum ada integration test MySQL yang bisa diverifikasi.

## 2. Peta risiko salah target database

1. Test baru dapat membaca `DATABASE_URL` secara tidak sengaja jika helper memakai konfigurasi aplikasi.
2. `godotenv.Load()` dapat memuat `.env` dari working directory, sehingga test yang memanggil config loader bisa mengambil konfigurasi aplikasi.
3. `database.Connect` tidak mengetahui apakah DSN menunjuk schema test.
4. `AutoMigrate` dapat mengubah schema saat dipanggil.
5. `Seed` dapat menulis akun/data contoh saat startup.
6. Tidak ada pemeriksaan nama database, host, atau privilege sebelum operasi write.
7. Credential test belum memiliki jalur konfigurasi eksplisit.

Belum ada bukti data development/production pernah tersentuh oleh test; audit ini tidak menjalankan test atau koneksi database.

## 3. Desain fail-closed

Desain minimal yang direkomendasikan:

1. Helper test membaca `TEST_DATABASE_URL` secara eksplisit.
2. Jika variable kosong, test database berhenti sebelum `gorm.Open`.
3. Helper wajib memeriksa penanda environment eksplisit, misalnya `APP_ENV=test` atau flag test khusus.
4. DSN harus diparse dan database name harus cocok allowlist yang disetujui, misalnya suffix `_test`/prefix yang ditentukan organisasi.
5. Pemeriksaan tidak hanya nama database: validasi juga host/port yang diizinkan, user non-production, dan larangan nilai yang sama dengan konfigurasi production/development bila tersedia.
6. Tidak ada fallback ke `DATABASE_URL`, `.env` aplikasi, default DSN, atau secret development.
7. Kegagalan validasi menghasilkan error jelas dan berhenti sebelum koneksi/write.
8. User MySQL test memiliki privilege minimum pada schema test saja; tidak boleh memakai akun production.
9. Migration/fixture test dipanggil oleh command test/setup eksplisit, bukan `cmd/server/main.go`.
10. Seeder test terpisah dari `database.Seed`; fixture dibersihkan atau database dibuat ulang sesuai strategi yang disetujui.

Guardrail tetap perlu diuji dengan unit test konfigurasi: kosong, env salah, nama schema salah, host salah, user production, dan DSN valid.

## 4. Strategi isolasi lokal dan CI

| Strategi | Kelebihan | Risiko/kebutuhan | Penilaian |
|---|---|---|---|
| Schema/database terpisah pada MySQL lokal | Realistis untuk XAMPP, setup sederhana | Tetap berbagi instance; privilege dan target wajib diverifikasi | Pilihan awal lokal yang paling realistis |
| Instance MySQL khusus test | Isolasi lebih kuat dan target jelas | Perlu instance, port, credential, lifecycle | Baik bila tersedia |
| Container MySQL khusus test | Reproducible untuk CI, mudah reset | Memerlukan Docker/Podman; belum ada bukti tersedia di proyek | Pilihan CI setelah runtime disepakati |

Proyek tidak memiliki file Docker/Compose yang terdeteksi dan tidak ada bukti bahwa MySQL XAMPP aktif atau dapat diakses. Jangan mengasumsikan salah satunya tersedia. Tahap setup harus memeriksa layanan secara terpisah dengan persetujuan, bukan pada audit ini.

Rekomendasi: mulai dengan database/schema test khusus pada MySQL lokal yang user dan nama databasenya eksplisit, dilindungi guardrail fail-closed; gunakan instance/container khusus CI bila lingkungan CI tersedia.

## 5. Batas migration, startup, dan fixture

### Startup server

Startup aplikasi saat ini menghubungkan database, menjalankan AutoMigrate, seed, lalu server. Untuk keamanan test, startup tidak boleh dipakai sebagai test harness dan tidak boleh dijalankan selama test.

### Migration deployment

Migration production harus menjadi command deployment terpisah dengan target environment eksplisit. AutoMigrate yang ada perlu diperlakukan sebagai legacy baseline sampai strategi migration berversi disetujui. Jangan menambahkan schema chat ke AutoMigrate tanpa keputusan.

### Fixture test

Fixture chat harus memakai database test dan credential test. Fixture tidak boleh memanggil seeder production/development. Fixture harus deterministik, dapat di-reset, dan mencakup Admin, Manager, PM, outlet aktif/nonaktif, kategori, conversation, membership, unread state, dan pesan batas panjang.

Migration atau seeder tidak boleh berjalan otomatis dari startup server.

## 6. Rencana implementasi bertahap

1. Setujui nama database/schema test, host/port, user, privilege, dan penanda environment.
2. Implementasikan helper konfigurasi test yang hanya membaca `TEST_DATABASE_URL` dan fail-closed.
3. Tambahkan unit test guardrail tanpa membuka database.
4. Siapkan database/instance/container test secara eksplisit di luar aplikasi.
5. Tambahkan helper koneksi integration test setelah guardrail lulus.
6. Uji migration pada database test kosong dan fixture/snapshot yang disetujui.
7. Tambahkan integration test repository/service chat.
8. Jalankan test HTTP/WS dan regression mutasi dengan database test.
9. Dokumentasikan command CI dan pastikan tidak ada fallback ke konfigurasi aplikasi.

Tahap 4A ini tidak menjalankan langkah tersebut.

## 7. Test keamanan yang wajib dibuat

- `TEST_DATABASE_URL` kosong ditolak sebelum koneksi.
- Environment bukan `test` ditolak.
- Nama database tidak sesuai allowlist ditolak.
- Host/port tidak sesuai policy ditolak.
- Credential production atau DSN yang sama dengan app ditolak jika dapat dibandingkan.
- Tidak ada fallback ke `DATABASE_URL` atau `.env` aplikasi.
- Helper test tidak memanggil AutoMigrate/Seed otomatis.
- User database test tidak dapat mengakses schema production/development, diverifikasi melalui setup privilege terpisah.
- Migration gagal menghentikan proses dan tidak melanjutkan fixture.
- Fixture tidak berjalan melalui startup server.

Integration test chat yang nantinya wajib memakai database test: membership, unread per admin, pembuatan conversation idempoten, idempotency pesan, histori saat WebSocket putus, session expiry, dan audit metadata tanpa isi pesan.

## 8. Keputusan yang memerlukan persetujuan

1. Nama/schema database test dan allowlist namanya.
2. Apakah lokal memakai schema terpisah atau instance khusus.
3. Apakah CI memakai container MySQL.
4. User/privilege MySQL test.
5. Penanda environment test yang wajib.
6. Tool migration SQL berversi dan pemisahan command deployment.
7. Strategi fixture/reset dan apakah memakai snapshot atau database baru per run.
8. Apakah `database.Seed` dipertahankan hanya untuk development atau dipisahkan sepenuhnya.

## 9. Pernyataan perubahan

Tidak ada source code, test, konfigurasi, dependency, schema, migration, seeder, database, atau laporan lama yang diubah. Tidak ada server/executable aplikasi, test suite, migration, seeder, atau koneksi database yang dijalankan. Satu-satunya file baru adalah dokumen ini.
