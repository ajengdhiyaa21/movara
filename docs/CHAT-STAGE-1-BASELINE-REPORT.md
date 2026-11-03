# MOVARA Internal Chat — Stage 1 Baseline Report

Tanggal pemeriksaan: 27 Oktober 2026  
Mode: read-only baseline dan regression audit  
Scope: kondisi proyek sebelum implementasi chat internal Admin–PM

## 1. Ringkasan eksekutif

Baseline aplikasi dapat dikompilasi dan test yang tersedia berhasil dijalankan. Tidak ada database test terpisah yang terlihat dan tidak ada integration test database yang dapat membuktikan alur runtime terhadap MySQL tanpa risiko memakai database pengguna.

Temuan paling penting yang terkonfirmasi dari pemeriksaan kode:

1. Riwayat PM hari ini menggunakan `ByDate` yang belum menerima `CategoryID`, sehingga filter kategori PM belum konsisten.
2. `RequireOutlet` memeriksa keberadaan `OutletID`, tetapi belum terlihat memeriksa `Outlet.IsActive`.
3. Middleware `SameOrigin` tersedia, tetapi belum ada CSRF token.
4. Rate limiting login belum ditemukan.
5. `go vet ./...` menemukan unreachable code di `internal/handler/pages.go:76`.
6. Tidak ditemukan konfigurasi database test terpisah.

Tidak ada indikasi dari audit statis bahwa data mutasi berubah atau terhapus selama tahap ini. Tidak ada migration, seeder, server startup, atau operasi database yang dijalankan.

## 2. Identitas dan kondisi repository

- Root yang digunakan: `C:\xampp\htdocs\kosmemutasi\movara`.
- Branch aktif: `main`.
- Go: `go1.26.5 windows/amd64`.
- Framework: Go + Gin.
- ORM/driver: GORM + MySQL.
- Frontend: server-side templates, HTMX, JavaScript, dan CSS.
- Entry point: `cmd/server/main.go`.
- Direktori utama: `cmd`, `docs`, `internal`, `web`.

### Git status saat audit

Status lokal sebelum audit menunjukkan:

```text
M  docs/PRD-MOVARA.md
?? docs/CHAT-PROJECT-AUDIT.md
?? docs/CHAT-REQUIREMENTS.md
?? docs/CHAT-TECHNICAL-PLAN.md
```

Perubahan tersebut sudah ada sebelum audit Tahap 1 dan tidak ditimpa atau dibuang.

### Dependency dan migration

- Dependency utama tercatat di `go.mod` dan checksum di `go.sum`.
- Tidak ditemukan folder migration SQL berversi.
- `internal/database/database.go` menjalankan GORM `AutoMigrate` saat startup.
- `cmd/server/main.go` juga memanggil migration dan seed ketika server dijalankan.
- Tidak ditemukan konfigurasi database test terpisah. `.env.example` hanya mendokumentasikan konfigurasi aplikasi umum.

Server startup dan seeder sengaja tidak dijalankan karena dapat mengubah database.

## 3. Hasil perintah baseline

### `go test ./...`

Status: **berhasil**.

Hasil:

- Semua package selesai.
- Test yang terlihat tersedia terutama berada pada middleware.
- Tidak ada test service, repository MySQL, handler, route integration, atau browser E2E yang tersedia.

### `go vet ./...`

Status: **warning/gagal secara kualitas static analysis**.

Temuan:

```text
internal\handler\pages.go:76:2: unreachable code
```

Lokasi tersebut berkaitan dengan handler `Kelola` yang melakukan redirect lebih awal, sementara implementasi halaman legacy masih berada setelah `return`.

Temuan tidak diperbaiki pada Tahap 1.

### `go build ./...`

Status: **berhasil**.

Seluruh package berhasil dibuild. Build tidak membuktikan koneksi database, migration, atau fungsi runtime setiap route.

## 4. Matriks pemeriksaan regresi

| Area | Status | Bukti dan batasan |
|---|---|---|
| Login/logout | Lolos pemeriksaan kode | `internal/handler/auth.go`, `internal/service/auth.go`, session cookie, bcrypt; belum ada integration test login runtime |
| Session | Lolos pemeriksaan kode | `LoadUser` membaca user berdasarkan session ID pada setiap request; belum diuji dengan browser/session expiry |
| Role Admin/Manager/PM | Lolos pemeriksaan kode | Route group memakai `RequireRole`; belum ada test matrix HTTP |
| Scope outlet PM | Temuan potensial | Query mutasi memakai outlet user pada service, tetapi status outlet aktif belum divalidasi di `RequireOutlet` |
| Scope kategori PM saat lookup/input | Lolos pemeriksaan kode | `FindProductForUser` memvalidasi category; belum ada test service otomatis |
| Scope kategori PM pada riwayat hari ini | Temuan terkonfirmasi dari kode | `internal/handler/mutation.go:291` memanggil `ByDate` tanpa category; `internal/service/mutation.go:71` hanya filter outlet/tanggal/query |
| Input mutasi | Lolos pemeriksaan kode | `MutationService.Add` mengambil outlet/user dari session dan memvalidasi qty; belum ada integration test database |
| Edit qty PM | Lolos pemeriksaan kode | Service membatasi PM/outlet/hari ini dan repository menyimpan edit log; belum ada test authorization otomatis |
| Rekap dan filter Admin/Manager | Lolos pemeriksaan kode | `RecapHandler` dan `RecapService` memiliki filter outlet/kategori/direction/date/query; belum diuji kombinasi filter runtime |
| Import produk | Lolos pemeriksaan kode | Handler membaca Excel dan validasi dasar; belum ada fixture test Excel atau laporan error baris |
| Export Excel | Belum terverifikasi | Implementasi tersedia, tetapi belum dijalankan dengan data database test |
| Validasi produk/kategori/outlet | Temuan potensial | Beberapa handler menerima ID form dan belum selalu memvalidasi status aktif secara konsisten |
| Pengelolaan user | Lolos pemeriksaan kode | Admin route dan handler tersedia; belum ada test duplicate/disabled/session invalidation |
| Audit log | Lolos pemeriksaan kode | Banyak aksi memanggil audit service; kegagalan audit tidak menggagalkan aksi utama dan belum ada test persistensi |
| Dashboard/navigasi | Belum terverifikasi runtime | Template dan route tersedia; belum ada browser E2E |
| Tombol WhatsApp | Lolos pemeriksaan kode | `Renderer.WhatsAppURL` dan tombol di `mutasi.html` tersedia; belum diuji link dengan konfigurasi nomor aktual |

## 5. Verifikasi temuan audit sebelumnya

### 5.1 Filter kategori riwayat PM

Status: **terkonfirmasi**.

Bukti:

- `internal/handler/mutation.go:131` memakai `Period(..., u.CategoryID, ...)` untuk halaman history periodik.
- `internal/handler/mutation.go:291` memakai `ByDate(*u.OutletID, q, date)` untuk riwayat hari ini.
- `internal/service/mutation.go:71` `ByDate` tidak menerima atau menerapkan category ID.

Dampak: riwayat hari ini dan totalnya berpotensi mencampur produk kategori lain pada outlet yang sama.

Rekomendasi: tambahkan scope kategori pada query `ByDate` dan test PM Obat/PM Kosmetik.

### 5.2 Status outlet nonaktif

Status: **terkonfirmasi sebagai celah validasi potensial**.

Bukti:

- `internal/middleware/auth.go:89–98` hanya memeriksa `u.OutletID != nil`.
- `LoadUser` memeriksa `u.IsActive`, tetapi tidak terlihat memeriksa status outlet relasi.
- `Outlet` memiliki field `IsActive` di `internal/models/models.go`.

Dampak: PM aktif pada outlet yang dinonaktifkan berpotensi tetap lolos `RequireOutlet`.

Rekomendasi: validasi outlet aktif di middleware/service dan tambahkan test saat outlet dinonaktifkan.

### 5.3 CSRF token

Status: **terkonfirmasi belum tersedia**.

Bukti:

- `internal/router/router.go:27` memakai `middleware.SameOrigin()`.
- `internal/middleware/csrf.go` memvalidasi Origin/Referer jika tersedia.
- Tidak ditemukan token CSRF per session, hidden token, atau validasi token pada POST.

Dampak: Same-Origin check bukan pengganti penuh CSRF token. Implementasi chat POST dan WebSocket memerlukan desain keamanan tambahan.

Rekomendasi: rancang CSRF token kompatibel dengan form dan HTMX sebelum menambah endpoint chat yang mengubah state.

### 5.4 Rate limiting login

Status: **belum ditemukan**.

Bukti: tidak ditemukan middleware atau service rate limit pada login.

Dampak: endpoint login berpotensi menerima percobaan berulang tanpa pembatasan aplikasi.

Rekomendasi: rate limit berdasarkan IP dan username, tanpa membocorkan keberadaan username.

### 5.5 Database test terisolasi

Status: **tidak tersedia berdasarkan inspeksi konfigurasi**.

Bukti:

- Tidak ditemukan konfigurasi `DATABASE_TEST` atau file database test.
- `internal/database/database.go` menggunakan MySQL dan AutoMigrate.
- Test yang ada tidak menginisialisasi database MySQL.

Dampak: integration test database belum aman untuk dijalankan tanpa environment test khusus.

Rekomendasi: sediakan MySQL test database/container terpisah sebelum migration dan repository chat diuji.

### 5.6 Unreachable code

Status: **terkonfirmasi**.

Bukti:

- `go vet ./...` menghasilkan `internal/handler/pages.go:76:2: unreachable code`.
- `Kelola` melakukan redirect dan return, tetapi implementasi legacy berada setelahnya.

Dampak: maintenance dan static quality terganggu, tetapi tidak ada bukti langsung kerusakan data.

Rekomendasi: pisahkan handler baru dan hapus kode unreachable hanya pada tahap perbaikan yang disetujui.

## 6. Kesiapan fondasi chat

### Sudah tersedia

- Middleware autentikasi session.
- Pengambilan user dari session melalui `CurrentUser`.
- Middleware role Admin/Manager/PM.
- Pattern repository/service/handler/router.
- GORM transaction sudah digunakan pada update qty mutasi.
- Error response HTTP dasar dan partial HTMX.
- Audit service metadata.

### Belum siap atau perlu diselesaikan

- Belum ada CSRF token.
- Belum ada validasi Origin khusus WebSocket.
- Belum ada koneksi/session registry WebSocket.
- Belum ada batas frame, ping/pong, deadline, reconnect, atau cleanup connection.
- Belum ada database test.
- Belum ada model/repository/service chat.
- Belum ada test authorization lintas conversation.
- Status outlet aktif belum menjadi scope access check PM.

## 7. Risiko terhadap data mutasi

Tidak ada operasi database yang dijalankan pada baseline ini, sehingga tidak ada data mutasi yang berubah atau hilang akibat audit.

Risiko kode yang perlu ditindaklanjuti:

1. Riwayat PM berpotensi menampilkan kategori yang salah.
2. Outlet nonaktif berpotensi belum memblokir input PM.
3. Import dan operasi admin belum memiliki integration test database.

Tidak ada bukti pada baseline ini bahwa chat telah memengaruhi data mutasi karena chat belum diimplementasikan.

## 8. Rekomendasi prioritas

### Wajib sebelum implementasi chat

1. Sepakati dan implementasikan strategi CSRF token.
2. Sediakan database test terisolasi.
3. Pastikan outlet nonaktif memblokir PM.
4. Perjelas dan uji scope kategori pada seluruh riwayat PM.
5. Definisikan test authorization Admin, Manager, PM aktif, dan PM nonaktif.
6. Tentukan migration strategy chat yang tidak memakai database production secara langsung.

### Dapat dikerjakan terpisah

1. Perbaiki unreachable code.
2. Tambahkan rate limiting login.
3. Tambahkan integration test mutasi/import/export.
4. Rapikan template legacy.
5. Tambahkan browser E2E.

## 9. Pemeriksaan yang belum dapat dilakukan

- Login runtime dengan database: tidak dijalankan karena tidak ada database test terisolasi dan server startup menjalankan migration/seed.
- CRUD mutasi melalui HTTP: tidak dijalankan untuk menghindari perubahan data.
- Import/export Excel runtime: tidak dijalankan karena membutuhkan data/database test.
- Browser E2E: tidak tersedia dalam toolchain audit ini.
- WebSocket handshake: belum ada implementasi WebSocket.
- Load test: belum ada lingkungan deployment/test yang disediakan.

## 10. Pernyataan perubahan

Pada Tahap 1 ini:

- Tidak ada source code yang diubah.
- Tidak ada dependency yang diubah.
- Tidak ada konfigurasi yang diubah.
- Tidak ada migration yang dijalankan.
- Tidak ada seeder yang dijalankan.
- Tidak ada database atau data mutasi yang diubah.
- Satu-satunya file baru yang dibuat adalah dokumen laporan ini.
