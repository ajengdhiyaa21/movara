# MOVARA — Tahap 4O: Review Rencana Provisioning Database Test

Tanggal review: 28 Oktober 2026  
Status: **DRAFT UNTUK REVIEW — BELUM DISETUJUI DAN BELUM DIJALANKAN**

## 1. Tujuan dan status readiness

Dokumen ini mendefinisikan prosedur provisioning database test `movara_chat_test` yang terpisah dari database aplikasi. Dokumen ini bukan instruksi eksekusi otomatis.

Status saat ini tetap **NOT READY**. Database, akun runtime, grants, schema, dan migration belum diverifikasi secara lengkap. Tidak terlihatnya database atau user di phpMyAdmin bukan bukti mutlak bahwa objek tersebut tidak ada.

### Label prosedur

- **[READ-ONLY]**: hanya memeriksa dan tidak mengubah database atau akun.
- **[MENGUBAH]**: membuat atau mengubah database, akun, privilege, schema, atau data.
- **[APPROVAL]**: keputusan yang harus disetujui sebelum tindakan dilakukan.

## 2. Target dan guardrail yang tersedia

Target non-rahasia pada `internal/config/testdb_policy.go` adalah database `movara_chat_test`, host `127.0.0.1`, port `3306`, dan user runtime yang direncanakan `movara_test`.

`ValidateTestDatabase` di `internal/config/testdb.go` hanya menerima `APP_ENV=test` dan `TEST_DATABASE_URL`, tidak fallback ke `DATABASE_URL`, memvalidasi policy sebelum target diterima, dan tidak membuka koneksi. Guardrail ini memvalidasi target konfigurasi; bukan grants aktual di MariaDB.

Startup aplikasi tidak boleh dipakai untuk provisioning karena `cmd/server/main.go` menjalankan koneksi, `AutoMigrate`, seed, dan server. Provisioning serta migration harus terpisah.

## 3. Urutan provisioning yang dapat ditinjau

### Langkah 0 — [APPROVAL] Tetapkan keputusan

Setujui target database/host/port, charset `utf8mb4`, collation, aturan case/accent sensitivity, akun runtime, akun migration, akun pemeriksa, privilege minimum, prosedur migration, strategi fixture/cleanup, dan penanggung jawab penghentian proses.

### Langkah 1 — [READ-ONLY] Pastikan instance yang benar

Operator memeriksa bahwa phpMyAdmin/client terhubung ke MariaDB `10.4.32` pada `127.0.0.1:3306` yang dimaksud. Catat hanya hasil non-rahasia.

### Langkah 2 — [READ-ONLY] Periksa objek sebelum membuat

Periksa apakah schema `movara_chat_test` terlihat, user `movara_test` pada host yang tepat terlihat, charset/collation schema tersedia, grants user dapat diperiksa, dan ada/tidaknya akses ke schema aplikasi.

Jika objek tidak terlihat, statusnya **BELUM TERVERIFIKASI**, bukan otomatis “tidak ada”. Periksa instance, refresh, dan hak akses sebelum mengambil keputusan.

### Langkah 3 — [APPROVAL] Tentukan jalur objek yang sudah ada

- Jika database sudah ada: jangan membuat ulang, mengubah collation, reset, drop, atau truncate. Audit schema dan minta keputusan kompatibilitas.
- Jika user sudah ada: jangan mengganti password atau grant otomatis. Audit host dan grants terlebih dahulu.
- Jika objek belum dapat diverifikasi: hentikan provisioning.
- Jika operator berwenang memastikan objek belum ada: lanjut hanya setelah approval provisioning.

### Langkah 4 — [MENGUBAH] Buat database test terpisah

Operator membuat hanya schema `movara_chat_test` dengan charset/collation yang telah disetujui. Database MOVARA yang sudah ada tidak boleh disentuh.

### Langkah 5 — [MENGUBAH] Buat akun terpisah

Buat akun runtime test dari host yang disetujui dengan password acak yang dibuat dan disimpan operator. Jangan memakai akun aplikasi atau `root`. Akun migration dan pemeriksa harus berbeda.

### Langkah 6 — [MENGUBAH] Berikan privilege minimum

- Runtime: kandidat `SELECT`, `INSERT`, `UPDATE`, dan `DELETE` pada `movara_chat_test`.
- Migration: DDL yang dibutuhkan selama migration melalui akun/prosedur terpisah.
- Pemeriksa: metadata read-only minimum; hak administratif hanya bila disetujui.

Jangan memberikan `GRANT OPTION`, `ON *.*`, akses schema aplikasi, atau DDL permanen kepada runtime test.

### Langkah 7 — [READ-ONLY] Verifikasi provisioning

Verifikasi schema aktif, user/host runtime, grants runtime, tidak adanya `GRANT OPTION`/akses global/schema aplikasi, charset/collation, pemisahan akun migration, dan tidak adanya password/DSN di log, Git, atau laporan. Jika hasil menyimpang, hentikan proses dan jangan cleanup otomatis.

## 4. Pemisahan akun dan tanggung jawab

| Akun | Fungsi | Hak minimum yang direncanakan | Batasan |
|---|---|---|---|
| Runtime test | Integration test dan fixture DML | DML pada `movara_chat_test` | Tanpa DDL, `GRANT OPTION`, global privilege, atau schema aplikasi |
| Migration | Schema test melalui migration eksplisit | DDL minimum selama migration | Tidak dipakai aplikasi atau integration test |
| Pemeriksa | Versi, schema, user, dan grants | Metadata minimum; admin hanya bila perlu | Tidak digunakan sebagai runtime atau migration |

Privilege aktual belum diverifikasi dan tidak boleh dianggap aman berdasarkan nama akun.

## 5. Charset dan collation

`utf8mb4` didukung oleh konfigurasi yang diperiksa. Collation belum ditetapkan oleh kode atau migration.

**[APPROVAL]** Tetapkan collation database test, aturan case/accent sensitivity, serta ekspektasi untuk unique index, `LIKE`, `LOWER`, dan sorting. Jangan mengubah collation schema existing tanpa review kompatibilitas.

## 6. Penyimpanan `TEST_DATABASE_URL`

Password dibuat manual oleh operator melalui password manager atau generator lokal yang disetujui. DSN disimpan hanya pada environment lokal/secret manager, tidak di chat, laporan, source code, `.env.example`, Git, command history, atau log. Password tidak boleh sama dengan aplikasi/production.

Test hanya boleh menggunakan `APP_ENV=test`, `TEST_DATABASE_URL`, dan policy approved. Tidak boleh ada fallback ke `DATABASE_URL` atau konfigurasi aplikasi untuk target test.

## 7. Migration eksplisit

**[APPROVAL]** Tool migration dan format version table belum ditetapkan. Setelah disetujui:

1. siapkan baseline migration untuk schema yang dibutuhkan test;
2. jalankan melalui command/proses migration khusus pada database test;
3. jangan menjalankan `cmd/server/main.go`, `AutoMigrate`, atau `Seed`;
4. verifikasi tabel, kolom, index, nullability, default, dan relasi;
5. buat fixture deterministik setelah migration berhasil;
6. catat versi migration tanpa credential.

Backfill legacy `products.outlet_id` harus menjadi langkah terpisah, diuji dengan data sintetis, dan tidak boleh memakai asumsi “outlet pertama” tanpa mapping bisnis yang disetujui.

## 8. Pemeriksaan keamanan sebelum integration test pertama

Integration test hanya boleh dipertimbangkan jika `ValidateTestDatabase` menerima target sesuai policy, instance/schema test terpisah terkonfirmasi, user/host runtime dan grants telah diverifikasi, tidak ada `ON *.*`/`GRANT OPTION`/akses schema aplikasi, runtime dan migration account berbeda, charset/collation disetujui, migration eksplisit berhasil, fixture tersedia, dan jalur test tidak memanggil startup app, AutoMigrate, Seed, atau fallback `DATABASE_URL`.

Jika satu syarat penting belum terbukti, status tetap **NOT READY**.

## 9. Pembatalan dan pemulihan aman

Jika pemeriksaan gagal, target berbeda, grants terlalu luas, collation tidak sesuai, atau migration gagal: hentikan langkah berikutnya, jangan jalankan test, jangan drop database/user, jangan truncate atau menghapus data existing, jangan mengubah akun pihak lain tanpa approval, simpan hanya error non-rahasia, dan minta review administrator untuk forward-fix/pemulihan terkontrol.

Pembatalan berarti tidak melanjutkan proses, bukan cleanup destruktif otomatis.

## 10. Keputusan yang masih harus disetujui

- provisioning manual XAMPP atau proses terpisah;
- collation dan aturan perbandingan string;
- akun migration dan privilege DDL;
- akun pemeriksa serta cakupan metadata;
- schema baseline dan tool migration berversi;
- strategi fixture, cleanup, dan retention;
- prosedur backfill legacy;
- kriteria dan penanggung jawab approval integration test pertama.

## 11. Konfirmasi tahap

Dokumen ini dibuat sebagai review prosedur. Tidak ada SQL, koneksi database, autentikasi, server MOVARA, migration, AutoMigrate, Seed, integration test, pembuatan/perubahan database, akun, privilege, schema, data, konfigurasi, dependency, atau source code yang dijalankan/diubah. Laporan tahap sebelumnya tidak diubah.
