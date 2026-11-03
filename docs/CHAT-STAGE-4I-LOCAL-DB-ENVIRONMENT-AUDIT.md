# MOVARA — Tahap 4I: Audit Lingkungan MySQL/MariaDB Lokal

Tanggal audit: 27 Oktober 2026  
Mode: read-only tanpa koneksi database.

## 1. Bukti lokasi instalasi

Struktur `C:\xampp` tersedia dan memiliki direktori `mysql`. Pemeriksaan lokal menemukan:

- `C:\xampp\mysql\bin\mysql.exe` tersedia.
- `C:\xampp\mysql\bin\mysqld.exe` tersedia.
- `C:\xampp\mysql\bin\my.ini` tersedia.
- `C:\xampp\mysql\data` tersedia.
- `C:\xampp\xampp-control.exe` tersedia.

Tidak ada database atau user yang disimpulkan dari keberadaan file/direktori tersebut.

## 2. Hasil pemeriksaan versi

Executable dijalankan hanya dengan opsi versi, tanpa host, user, password, atau perintah koneksi:

```text
C:\xampp\mysql\bin\mysql.exe  Ver 15.1 Distrib 10.4.32-MariaDB, for Win64 (AMD64)
C:\xampp\mysql\bin\mysqld.exe  Ver 10.4.32-MariaDB for Win64 on AMD64
```

Produk dan versi binary lokal **TERBUKTI** sebagai MariaDB 10.4.32. Ini tidak membuktikan service sedang berjalan atau port dapat diakses.

## 3. Konfigurasi non-rahasia

Dari `C:\xampp\mysql\bin\my.ini` ditemukan:

```text
port=3306
socket="C:/xampp/mysql/mysql.sock"
basedir="C:/xampp/mysql"
datadir="C:/xampp/mysql/data"
```

Port 3306 dan lokasi data directory terkonfigurasi pada file lokal. `.env.example` project juga menyebut charset `utf8mb4`, tetapi tidak menentukan collation.

## 4. Informasi yang belum terverifikasi

- Service MariaDB/XAMPP sedang berjalan atau tidak.
- Port 3306 benar-benar listen atau tidak.
- Database `movara_chat_test` sudah ada atau belum.
- User `movara_test` sudah ada atau belum.
- Password dan privilege user tersebut.
- Charset/collation aktual database.
- Schema, tabel, migration, dan foreign key aktual.
- Akun test tidak memiliki akses ke database aplikasi.

Tidak ada pemeriksaan yang memerlukan koneksi dilakukan pada tahap ini.

## 5. Rekomendasi langkah berikutnya

1. Dapatkan persetujuan eksplisit untuk menjalankan pemeriksaan SQL read-only terhadap server lokal.
2. Verifikasi service/port secara terpisah tanpa mencetak credential.
3. Dengan akun administrator yang sah, periksa database, user/host, dan `SHOW GRANTS` tanpa membuat atau menghapus objek.
4. Pastikan target hanya `movara_chat_test`, host `127.0.0.1`, port 3306, dan user `movara_test`.
5. Setelah privilege disetujui, ikuti prosedur setup Tahap 4F; jangan menjalankan startup MOVARA karena startup melakukan AutoMigrate dan Seed.
6. Verifikasi versi server yang benar-benar aktif sebelum memilih collation atau migration tool.

Jangan menganggap target database/user sudah tersedia atau aman hanya karena binary dan port konfigurasi ditemukan.

## 6. Konfirmasi keselamatan

File dibuat: `docs/CHAT-STAGE-4I-LOCAL-DB-ENVIRONMENT-AUDIT.md`.

Tidak ada koneksi database, SQL, server database, server MOVARA, database/user/privilege, konfigurasi project, source code, migration, seeder, AutoMigrate, atau integration test yang dijalankan/diubah. Pemeriksaan executable hanya menggunakan opsi versi.
