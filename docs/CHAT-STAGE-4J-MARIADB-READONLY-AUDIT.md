# MOVARA — Tahap 4J: Audit Metadata MariaDB Read-only

Tanggal audit: 27 Oktober 2026  
Scope: pemeriksaan service/proses/port lokal dan kesiapan metadata SQL tanpa autentikasi database.

## 1. Ruang lingkup dan keselamatan

Target yang diperiksa secara lokal adalah MariaDB XAMPP pada `127.0.0.1:3306`. Tidak ada credential yang dibaca dari `.env`, tidak ada percobaan login, dan tidak ada query SQL yang dijalankan karena sumber credential aman belum tersedia.

## 2. Status service, proses, dan port

| Item | Status | Bukti |
|---|---|---|
| Service Windows MySQL/MariaDB | TIDAK DAPAT DIVERIFIKASI | Tidak ada service dengan nama/display name MySQL atau MariaDB yang cocok |
| Proses server | TERVERIFIKASI | `mysqld.exe` berjalan dari `C:\xampp\mysql\bin\mysqld.exe`, PID 15640 |
| TCP listener port 3306 | TERVERIFIKASI | Listener pada `0.0.0.0:3306` dan `[::]:3306`, owner PID 15640 |
| Koneksi aktif | TERVERIFIKASI sebagai metadata OS | Terlihat koneksi TCP lokal established; pihak aplikasi dan tujuan pengguna tidak disimpulkan |

Proses aktif dan port listen berarti pemeriksaan SQL mungkin dapat dilanjutkan, tetapi hanya dengan credential administrator/read-only yang sudah tersedia secara aman.

## 3. Metadata server/database/user/grants

Status seluruh metadata SQL berikut: **TIDAK DAPAT DIVERIFIKASI**.

Tidak ada credential aman yang tersedia untuk melakukan koneksi ke `127.0.0.1:3306`. Sesuai batasan, tidak ada password default yang dicoba, tidak ada credential dari `.env` yang dibaca, dan tidak ada percobaan alternatif.

Karena itu belum dapat dipastikan:

- versi server aktif dan produk server dari SQL `VERSION()`;
- charset/collation server;
- keberadaan database `movara_chat_test`;
- charset/collation database test;
- keberadaan akun `movara_test` pada host `127.0.0.1`;
- `SHOW GRANTS` akun target;
- adanya DDL, `GRANT OPTION`, akses global, atau akses schema lain;
- tabel dan collation yang sudah ada pada database test.

Tidak ada status `TIDAK DITEMUKAN` yang dapat diberikan karena query metadata belum dijalankan.

## 4. Evaluasi keselamatan

Listener port 3306 tidak membuktikan database test terisolasi. Proses `mysqld.exe`, nama database, username, atau port tidak membuktikan privilege aman. Verifikasi berikut tetap memerlukan persetujuan/kredensial terpisah:

1. Query identitas server dan charset/collation.
2. Query keberadaan database target.
3. Query user/host tanpa kolom credential.
4. `SHOW GRANTS` hanya jika user target ditemukan pada host yang tepat.
5. Query tabel/collation hanya jika database target terlihat oleh akun pemeriksa.

## 5. Risiko yang ditemukan

- Tidak ada service Windows terdaftar yang dapat mengonfirmasi lifecycle XAMPP; hanya proses server yang terlihat.
- Listener terbuka pada wildcard IPv4/IPv6 berdasarkan metadata OS, bukan hanya `127.0.0.1`; bind policy server belum diverifikasi melalui SQL/config lengkap.
- Ada koneksi TCP lokal established yang identitas aplikasinya belum diketahui.
- Database, user, dan privilege target belum diverifikasi.
- Belum ada bukti target test tidak dapat mengakses database aplikasi.

## 6. Langkah berikutnya yang membutuhkan persetujuan terpisah

1. Sediakan credential administrator/read-only secara aman tanpa menempelkannya ke chat atau repository.
2. Jalankan hanya query metadata Tahap 4J pada host `127.0.0.1` port `3306`.
3. Simpan hanya hasil non-rahasia: versi, charset/collation, keberadaan database, user/host, dan ringkasan grants.
4. Jika autentikasi gagal, berhenti; jangan mencoba credential lain.
5. Jangan membuat database/user, mengubah grant, menjalankan migration, atau menjalankan aplikasi sebagai tindak lanjut otomatis.

## 7. Konfirmasi akhir

File dibuat: `docs/CHAT-STAGE-4J-MARIADB-READONLY-AUDIT.md`.

Tidak ada koneksi database, SQL, database, user, password, privilege, schema, tabel, data, migration, seeder, AutoMigrate, server MOVARA, atau integration test yang dijalankan/diubah. Pemeriksaan hanya menggunakan metadata proses dan port OS.
