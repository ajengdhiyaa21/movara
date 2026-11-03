# MOVARA — Tahap 4K: Opsi Akses Metadata Read-only Lokal

Tanggal audit: 27 Oktober 2026  
Mode: audit lokal read-only.

## 1. Bukti yang tersedia

- Tahap 4I membuktikan executable MariaDB XAMPP berada di `C:\xampp\mysql\bin` dan versi binary `10.4.32-MariaDB`.
- Tahap 4I/4J membuktikan proses `mysqld.exe` aktif dan port TCP 3306 listen.
- Tahap 4J belum menjalankan SQL karena tidak ada credential pemeriksa yang aman dan tersedia.
- Dokumentasi project hanya mendeskripsikan kebutuhan akun test/read-only; tidak menyediakan credential pemeriksa atau prosedur autentikasi otomatis.
- File project memiliki konfigurasi aplikasi dan contoh DSN, tetapi tidak boleh dipakai untuk memperoleh credential pemeriksaan.

Tidak ada bukti bahwa database `movara_chat_test`, akun `movara_test`, atau grant-nya sudah tersedia/aman.

## 2. Status metode akses

| Opsi | Status | Alasan |
|---|---|---|
| Memakai credential dari `.env` aplikasi | Dilarang | Berisiko membuka database aplikasi dan melanggar batasan audit |
| Mencoba password default/alternatif | Dilarang | Tidak aman dan bukan prosedur terotorisasi |
| Memakai akun `movara_test` sebelum verifikasi | Belum dapat dilakukan | Keberadaan dan password belum diketahui |
| Akun pemeriksa read-only yang sudah tersedia | BELUM TERVERIFIKASI | Tidak ada sumber credential aman yang ditemukan |
| Operator berwenang menjalankan metadata read-only | REKOMENDASI | Operator dapat memakai credential resmi tanpa membagikannya ke chat/repository |
| Membuat akun pemeriksa baru | NEEDS APPROVAL | Memerlukan prosedur administratif, password, dan privilege terpisah |

Kesimpulan: tidak ada metode akses aman yang dapat dipakai oleh agent saat ini. Pemeriksaan SQL harus dilakukan manual oleh operator berwenang.

## 3. Privilege minimum yang diperlukan

### Metadata server/database

Untuk membaca versi server dan variabel charset/collation, operator memerlukan sesi yang diizinkan membaca metadata server. Untuk keberadaan schema/tabel dan collation, operator memerlukan akses metadata `INFORMATION_SCHEMA` yang sesuai dengan visibility akun.

### Daftar user/host dan grants

Membaca daftar user dari `mysql.user` dan menjalankan `SHOW GRANTS FOR` akun target dapat membutuhkan hak administratif tambahan. Akun aplikasi/runtime biasa tidak boleh diasumsikan dapat membaca metadata user atau grants seluruh akun.

Pemeriksaan grants target harus dilakukan hanya jika operator memiliki hak tersebut secara resmi. Jika tidak, status harus `TIDAK DAPAT DIVERIFIKASI`; jangan menaikkan privilege otomatis.

### Prinsip pemisahan

- Akun runtime test: hanya privilege DML pada schema test sesuai kebutuhan.
- Akun pemeriksa metadata: read-only/administratif minimum untuk audit, tidak dipakai aplikasi.
- Akun migration: DDL terpisah dan terkontrol, bukan akun runtime.

Nama user atau suffix `_test` tidak membuktikan privilege terisolasi.

## 4. Opsi aman yang tersedia

### Opsi A — Operator memakai akun administrator yang sudah sah

Operator menjalankan query metadata Tahap 4J secara lokal pada `127.0.0.1:3306`, tidak menyalin credential ke chat, lalu membagikan hanya ringkasan non-rahasia. Ini paling lengkap untuk memeriksa `mysql.user` dan `SHOW GRANTS`, tetapi memiliki risiko administratif sehingga harus dilakukan oleh operator yang memahami target.

### Opsi B — Operator memakai akun pemeriksa read-only khusus

Admin database membuat akun pemeriksa dengan privilege metadata minimum yang disetujui, hanya dari host lokal, tanpa `GRANT OPTION`, tanpa akses DML, dan tanpa akses schema aplikasi bila tidak dibutuhkan. Prosedur pembuatan akun belum dijalankan dan **NEEDS APPROVAL**.

### Opsi C — Pemeriksaan manual terbatas

Jika hanya metadata server/database yang tersedia, operator dapat memeriksa versi, port, charset/collation, dan visibility schema. Status user/grants tetap tidak lengkap. Jangan menyebut database aman sebelum grants diverifikasi.

## 5. Prosedur yang tidak boleh dilakukan

- Jangan memakai DSN aplikasi atau membaca password dari `.env`.
- Jangan mencoba password default atau credential alternatif.
- Jangan menjalankan `CREATE USER`, `GRANT`, `REVOKE`, `SET PASSWORD`, atau perubahan konfigurasi untuk membuka akses.
- Jangan menggunakan `mysql.exe` tanpa credential resmi.
- Jangan menjalankan server MOVARA, AutoMigrate, Seed, migration, atau integration test.

## 6. Langkah berikutnya dengan persetujuan

1. Pengguna/operator menentukan apakah audit akan memakai akun administrator sah atau akun pemeriksa khusus.
2. Credential dimasukkan hanya secara lokal, tidak ditempel di chat/repository.
3. Jalankan query metadata Tahap 4J satu per satu pada `127.0.0.1:3306`.
4. Simpan hanya versi, port, charset/collation, keberadaan schema, user/host, dan ringkasan grants tanpa credential.
5. Hentikan pemeriksaan jika autentikasi gagal atau hasil grants tidak lengkap.
6. Jangan membuat atau mengubah database/user/privilege sebagai efek samping audit.

## 7. Keputusan NEEDS APPROVAL

- Jenis akun yang boleh dipakai untuk audit metadata.
- Apakah operator memiliki credential administratif yang sah.
- Apakah akun pemeriksa read-only perlu dibuat oleh administrator.
- Privilege minimum akun pemeriksa dan batas aksesnya.
- Prosedur penyampaian hasil metadata tanpa secret.
- Apakah setelah audit metadata boleh dilanjutkan ke setup database test.

## 8. Konfirmasi perubahan

File dibuat: `docs/CHAT-STAGE-4K-READONLY-ACCESS-OPTIONS.md`.

Tidak ada koneksi database, percobaan autentikasi, SQL, perubahan konfigurasi, akun, privilege, schema, data, source code, migration, seeder, server, atau integration test yang dijalankan/diubah.
