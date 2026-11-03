# MOVARA — Tahap 4H: Rekomendasi Kebijakan Database dan Migration

Tanggal audit: 27 Oktober 2026  
Mode: audit/rekomendasi read-only.

## 1. Ringkasan keputusan terbuka

- **TERBUKTI:** DSN contoh aplikasi menggunakan charset `utf8mb4` (`.env.example:9`).
- **TERBUKTI:** schema saat ini dibuat melalui GORM AutoMigrate dan kompatibilitas legacy (`internal/database/database.go:14-49`).
- **TERBUKTI:** query MySQL memakai `LIKE`, `ORDER BY`, dan handler memakai `LOWER`; memory repository menormalisasi sebagian pencarian dengan `strings.ToLower`.
- **BELUM TERVERIFIKASI:** versi MySQL/MariaDB lokal; executable `mysql` tidak ditemukan pada PATH.
- **BELUM TERVERIFIKASI:** database, user, privilege, dan collation aktual.
- **NEEDS APPROVAL:** collation final, aturan case/accent sensitivity, migration tool, akun DDL, dan strategi backfill.

## 2. Kebijakan charset/collation

### Bukti dan dampak

`utf8mb4` adalah satu-satunya charset yang terbukti dari konfigurasi. Model memiliki unique username, category name/slug, outlet code, dan unique gabungan product outlet+barcode (`internal/models/models.go:13-59`). Collation dapat memengaruhi apakah nilai berbeda casing/aksen dianggap sama, hasil `LIKE`, `LOWER`, unique index, dan sorting.

Authentication menurunkan username dengan `strings.ToLower` (`internal/service/auth.go:19-24`), sedangkan repository MySQL memakai perbandingan database (`internal/repository/mysql/mutations.go:57-63`). Karena memory dan MySQL dapat berbeda, regression harus menguji casing, aksen, uniqueness, LIKE, dan ORDER BY.

### Rekomendasi

- **REKOMENDASI:** gunakan `utf8mb4` pada database test karena didukung konfigurasi aktual.
- **NEEDS APPROVAL:** jangan menetapkan collation sebelum versi server dan kebutuhan case/accent sensitivity disetujui.
- **REKOMENDASI alternatif:** pilih collation case-insensitive hanya jika kebijakan bisnis memang menganggap nilai casing berbeda sebagai sama; pilih collation binary/case-sensitive hanya jika identifier harus membedakan casing. Keduanya harus diuji terhadap username, kategori, slug, pencarian, dan sorting.

Tidak ada klaim kandidat collation kompatibel dengan server lokal karena versinya belum terbukti.

## 3. Strategi migration berversi

### Kondisi saat ini

`AutoMigrate` masih menjadi mekanisme schema. Kompatibilitas legacy juga melakukan `ALTER TABLE`, `DROP INDEX` tertentu, dan update backfill. Ini bukan migration versioned yang dapat dilacak secara eksplisit.

### Rekomendasi transisi

- **REKOMENDASI:** gunakan migration SQL berversi dengan tabel metadata versi, command deployment terpisah dari startup server, dan akun DDL terpisah.
- **KANDIDAT:** tool Go/MySQL seperti `golang-migrate`, karena komentar `database.go:15` sendiri menyebutkannya; keputusan final tetap **NEEDS APPROVAL**.
- Setiap migration immutable dan diuji pada database kosong serta baseline lama.
- Verifikasi nama tabel, kolom, index, nullability, default, dan foreign key melalui schema metadata setelah migration; jangan mengasumsikan hasil AutoMigrate.
- Runtime user hanya DML yang diperlukan; akun migration memiliki DDL minimum selama migration dan tidak dipakai aplikasi.

Jika migration gagal, hentikan versi berikutnya dan tandai kondisi dirty. Rollback destructive tidak boleh otomatis menghapus data operasional/chat; gunakan forward-fix atau restore terkontrol.

## 4. Backfill legacy `products.outlet_id`

`internal/database/database.go:26-40` menambah kolom nullable, memilih outlet pertama berdasarkan `id`, lalu mengisi baris kosong. **RISIKO:** outlet pertama belum tentu tujuan yang benar.

Rencana aman:

1. Pada database test terisolasi, buat dataset sintetis: products tanpa outlet, satu outlet, beberapa outlet, dan data ambigu.
2. Hitung baris NULL/0 dan distribusi product sebelum perubahan.
3. Validasi outlet tujuan melalui mapping bisnis eksplisit; jangan memakai “outlet pertama” sebagai asumsi.
4. Hentikan proses jika tidak ada mapping, ada lebih dari satu kandidat, atau jumlah baris tidak sesuai.
5. Jalankan backfill sebagai migration terpisah dalam transaction bila operasi mendukung.
6. Verifikasi jumlah NULL/0 sesudah, jumlah total product, unique outlet+barcode, dan sample mapping.
7. Baru setelah validasi, pasang NOT NULL/index/constraint.

Tidak ada backfill atau SQL yang dibuat/dijalankan pada audit ini.

## 5. Fixture dan perlindungan database

Bedakan secara tegas:

- database runtime: target aplikasi, tidak boleh disentuh test;
- database test: `movara_chat_test` sebagai target rencana, belum diverifikasi;
- akun runtime: akun aplikasi, bukan akun migration;
- akun migration: DDL terpisah dan terkontrol;
- fixture: data sintetis deterministik untuk test;
- seeder aplikasi: `database.Seed`, bukan fixture integration.

Fixture regression harus mencakup outlet aktif/nonaktif, kategori, PM/Admin/Manager, product barcode/code berawalan nol, mutation dua kategori pada outlet sama, edit log, audit, dan kasus backfill. Test hanya boleh memperoleh `TEST_DATABASE_URL`, `APP_ENV=test`, dan policy approved. Tidak boleh fallback ke `DATABASE_URL` atau menjalankan `cmd/server/main.go`.

## 6. Risiko, prasyarat, dan langkah berikutnya

Risiko utama: collation salah, schema hasil AutoMigrate berbeda dari migration SQL, backfill salah outlet, privilege DDL terlalu luas, dan fixture mengarah ke database aplikasi.

Prasyarat berikutnya:

1. Verifikasi versi server tanpa koneksi jika executable/config lokal tersedia.
2. Setujui charset/collation dan kebijakan case/accent.
3. Setujui tool migration dan akun DDL.
4. Siapkan database/user/privilege melalui prosedur Tahap 4F setelah approval.
5. Tulis baseline migration dan uji schema kosong.
6. Uji backfill pada data sintetis.
7. Buat fixture dan integration test setelah guardrail lulus.

## 7. Keputusan yang memerlukan persetujuan pengguna

- **NEEDS APPROVAL:** collation final.
- **NEEDS APPROVAL:** migration tool dan format version table.
- **NEEDS APPROVAL:** akun/privilege DDL dan cara pencabutannya.
- **NEEDS APPROVAL:** mapping legacy product ke outlet.
- **NEEDS APPROVAL:** strategi rollback/forward-fix.
- **NEEDS APPROVAL:** fixture reset dan retention.
- **NEEDS APPROVAL:** kapan database/user/privilege test boleh diverifikasi atau dibuat.

## 8. Pernyataan perubahan

Tidak ada source code, konfigurasi, dependency, schema, database, user, password, privilege, migration, seeder, atau laporan lama yang diubah. Tidak ada koneksi MySQL, SQL, server, AutoMigrate, Seed, migration, maupun integration test yang dijalankan. Satu-satunya file baru adalah laporan ini.
