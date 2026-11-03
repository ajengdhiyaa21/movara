# MOVARA — Tahap 2B: Test dan Foundation Plan

Tanggal audit: 27 Oktober 2026  
Mode: read-only  
Scope: database test terisolasi, migration berversi, dan fondasi sebelum chat Admin–PM.

## 1. Ringkasan eksekutif

Dokumen acuan berikut diverifikasi tersedia dan dibaca: `docs/PRD-MOVARA.md`, `docs/CHAT-PROJECT-AUDIT.md`, `docs/CHAT-REQUIREMENTS.md`, `docs/CHAT-TECHNICAL-PLAN.md`, `docs/CHAT-STAGE-1-BASELINE-REPORT.md`, dan `docs/CHAT-STAGE-2A-CSRF-DESIGN.md`.

Kondisi aktual: aplikasi Go/Gin memakai MySQL + GORM; `DATABASE_URL` dipakai langsung untuk koneksi; startup memanggil Connect, AutoMigrate, Seed, lalu server; belum ada migration SQL berversi atau database test terpisah. Test yang tersedia belum menulis ke database.

Rekomendasi: sediakan MySQL test eksplisit dan fail-closed, lalu pilih migration tool SQL berversi sebelum schema chat dibuat. AutoMigrate tidak boleh menjadi side effect deployment production setelah transisi migration dimulai.

## 2. Kondisi database dan migration

- `internal/config/config.go:11-29` membaca `DATABASE_URL` ke `Config.DatabaseURL`; nama database berasal dari DSN environment.
- `internal/database/database.go:9-12` memakai `gorm.Open(mysql.Open(dsn), ...)`.
- `cmd/server/main.go:18-24` memanggil Connect.
- `cmd/server/main.go:25-28` memanggil `database.Migrate` lalu `database.Seed` sebelum server aktif.
- `internal/database/database.go:14-51` menjalankan GORM `AutoMigrate` dan SQL kompatibilitas untuk `outlet_id`, index, dan backfill outlet.
- Tidak ditemukan folder migration SQL berversi atau migration tool pada struktur proyek/dependency.
- Tidak ditemukan konfigurasi database test terpisah, `TEST_DATABASE_URL`, atau mekanisme yang menolak fallback ke `DATABASE_URL` aplikasi.
- Tidak ada koneksi database, migration, seed, atau server yang dijalankan pada audit.

Startup normal dapat mengubah schema dan data. Seeder berada di `internal/database/seed.go`; isi credential tidak dicantumkan.

## 3. Risiko database salah target

Tanpa guardrail, test/command dapat memakai DSN aplikasi lalu memodifikasi development atau production, menjalankan migration pada schema yang salah, menjalankan seed, atau saling mengganggu saat paralel. Belum ada bukti risiko ini pernah terjadi karena audit tidak melakukan operasi database.

Guardrail wajib:

1. Test hanya membaca konfigurasi test eksplisit; tidak ada fallback ke `DATABASE_URL`.
2. Wajib ada environment test dan nama schema/database yang masuk allowlist.
3. Host, credential, dan database test harus berbeda dari production.
4. Test memverifikasi target sebelum koneksi/write dan gagal aman bila tidak valid.
5. Migration/seeder tidak berjalan tanpa command/flag eksplisit.
6. Seeder test terpisah, deterministik, dan memiliki privilege minimum.
7. CI menyuntikkan secret test; secret tidak disimpan di repository.

## 4. Opsi database test terisolasi

| Opsi | Setup | Keamanan | Kemudahan | Risiko salah target | Tooling |
|---|---|---|---|---|---|
| MySQL schema terpisah pada instance sama | Database + user test khusus | Sedang; privilege wajib dibatasi | Paling mudah untuk XAMPP | Sedang | Tidak wajib dependency baru |
| Instance/container MySQL khusus test | Container/instance, lifecycle, fixture | Tinggi | Baik untuk CI, setup lokal lebih besar | Rendah | Docker/Podman atau layanan CI |
| Konfigurasi test fail-closed | URL test eksplisit, penanda env, allowlist nama DB | Wajib untuk semua opsi | Sedikit boilerplate | Rendah bila diterapkan sebelum write | Tidak wajib dependency baru |

Rekomendasi: Opsi schema terpisah + fail-closed untuk lokal, lalu instance/container khusus + fail-closed untuk CI bila tersedia. Tidak ada opsi yang diimplementasikan sekarang.

## 5. Strategi migration SQL berversi dan rollback aman

Proyek belum memiliki migration tool. Setelah persetujuan, pilih satu tool SQL berversi yang kompatibel MySQL, simpan file immutable bernomor urut, dan catat versi pada tabel metadata tool. Uji migration pada database kosong dan snapshot schema lama sebelum deployment.

AutoMigrate lama sebaiknya diperlakukan sebagai baseline transisi. Jangan menambahkan model chat ke AutoMigrate production secara diam-diam. Migration production harus menjadi command deployment eksplisit, bukan side effect startup `server.exe`.

Migration gagal harus menghentikan deployment, mencatat versi terakhir, dan tidak menjalankan migration berikutnya pada schema dirty tanpa prosedur pemulihan. Gunakan transaction bila didukung operasi yang dipakai dan backup/snapshot sebelum production.

Rollback tidak boleh otomatis menghapus tabel atau pesan chat. Prioritaskan forward-fix, kolom nullable, backfill bertahap, kompatibilitas sementara, dan pemulihan terkontrol dari backup. `DROP TABLE`, `DROP COLUMN`, atau destructive down migration otomatis tidak direkomendasikan.

Dependency migration baru mungkin diperlukan, tetapi belum dipilih dan belum ditambahkan.

## 6. Test matrix chat

Semua test bertanda **DB** memerlukan database test terisolasi. **WS** memerlukan implementasi WebSocket.

| Skenario | Jenis | DB | Kriteria lulus |
|---|---|---:|---|
| Admin aktif baca/balas PM aktif | HTTP/WS | Ya | Role dan membership valid |
| Manager halaman/WebSocket chat | Authorization/WS | Opsional | Ditolak |
| PM akses conversation sendiri/lain | Authorization | Ya | Milik sendiri diterima, lain ditolak |
| PM nonaktif mengirim pesan | Security | Ya | Ditolak |
| Admin baca histori PM nonaktif | Integration | Ya | Read-only sesuai policy |
| Unread antar-admin independen | Integration | Ya | State A tidak mengubah B |
| Conversation dibuka pertama kali | Integration | Ya | Dibuat sekali, tidak duplikat |
| Pesan kosong atau >3.000 karakter | Validation | Opsional | Ditolak, tidak disimpan |
| HTML/script | Security | Ya | Ditampilkan sebagai teks biasa |
| Retry pengiriman | Integration | Ya | Tidak membuat duplikat |
| Session expired | WS/Security | Opsional | Akses diputus |
| Session, role, membership, Origin WS | WS/Security | Membership: Ya | Handshake tidak sah ditolak |
| HTTP POST CSRF | Security | Opsional | Token valid diterima, invalid ditolak |
| Histori saat WS putus | Integration | Ya | GET histori tetap tersedia |
| Audit tidak menyimpan isi pesan | Integration | Ya | Hanya metadata |

Fixture minimal: admin aktif, manager, PM Obat, PM Kosmetik, PM nonaktif, outlet aktif/nonaktif, conversation lintas membership, dan pesan batas panjang.

## 7. Bug lama dan prioritas terpisah

### P1 — Filter kategori riwayat PM: terkonfirmasi

Bukti: `internal/service/mutation.go:61-62` `Period` menerima `categoryID`; `:71-72` `ByDate` hanya menerima outlet/query/date. `internal/handler/mutation.go:131` memakai category untuk Period, sedangkan `:290-291` memanggil ByDate tanpa category.

Dampak: riwayat hari ini berpotensi mencampur kategori. Test: PM Obat/Kosmetik pada outlet sama dengan data silang. Harus diselesaikan sebelum chat karena memengaruhi scope konteks PM.

### P1 — Outlet nonaktif: terkonfirmasi sebagai gap validasi

Bukti: `internal/middleware/auth.go:89-99` hanya memeriksa `OutletID != nil`; `:35-37` hanya memeriksa user aktif. Middleware dipakai group PM di `internal/router/router.go:76`.

Dampak: PM pada outlet nonaktif berpotensi tetap mengakses route. Test: nonaktifkan outlet lalu uji GET dan POST seluruh route PM. Harus diselesaikan sebelum chat karena membership bergantung pada status outlet.

### P1 — Unreachable code: terkonfirmasi

Bukti: `internal/handler/pages.go:72-76`; `go vet ./...` melaporkan `internal/handler/pages.go:76:2: unreachable code` setelah redirect/return handler `Kelola`.

Dampak: maintenance dan static quality; bukan bukti data rusak. Test: static analysis dan route `/kelola`. Dapat dikerjakan terpisah, tetapi sebaiknya sebelum perubahan besar.

### P2 — Rate limiting login: belum ditemukan

Bukti: `internal/handler/auth.go:34-55` dan `internal/service/auth.go:19-28` tidak menunjukkan limiter; tidak ditemukan middleware limiter pada router. Dampak: brute-force risk. Test: request berulang dan policy response. Dapat terpisah, tetapi wajib sebelum production.

## 8. Ketergantungan sebelum chat

1. Persetujuan dan implementasi CSRF sesuai Tahap 2A.
2. Database test fail-closed.
3. Keputusan migration tool berversi.
4. Penyelesaian scope kategori dan outlet aktif.
5. Kontrak role/membership, histori PM nonaktif, Origin WS, expiry, reconnect, dan idempotency.
6. Fixture integration untuk role/outlet/kategori.

## 9. Pilihan tooling/dependency yang membutuhkan persetujuan

- Tool migration SQL berversi; belum dipilih.
- Client WebSocket; belum ada pada `go.mod`.
- Docker/Podman atau layanan CI MySQL.
- Strategi fixture/reset database.
- Keputusan memakai GORM untuk query chat dengan SQL migration terpisah.

Tidak ada dependency yang ditambahkan.

## 10. Keputusan terbuka

Nama database test, instance/container, user privilege minimum, migration tool, command deployment terpisah, backup/forward-fix, fixture reset, retensi chat, akses histori PM nonaktif, dan allowlist Origin WebSocket masih memerlukan persetujuan.

## 11. Pernyataan perubahan

Tidak ada source code, konfigurasi, dependency, schema, atau database yang berubah. Tidak ada koneksi database, server, migration, seeder, perbaikan bug, CSRF implementation, model/repository/handler/route/UI chat yang dijalankan atau dibuat. Satu-satunya file baru adalah laporan ini; audit berhenti dan menunggu persetujuan.
