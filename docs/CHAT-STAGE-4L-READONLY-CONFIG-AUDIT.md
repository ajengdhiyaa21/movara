# MOVARA — Tahap 4L: Audit Read-only Konfigurasi Database

Tanggal audit: 28 Oktober 2026  
Mode: audit statis dan read-only; tanpa autentikasi database, koneksi, atau SQL.

## 1. Scope dan file yang diperiksa

File/laporan yang diperiksa:

- `docs/CHAT-STAGE-4I-LOCAL-DB-ENVIRONMENT-AUDIT.md`
- `docs/CHAT-STAGE-4J-MARIADB-READONLY-AUDIT.md`
- `docs/CHAT-STAGE-4K-READONLY-ACCESS-OPTIONS.md`
- `internal/config/config.go`
- `internal/config/testdb.go`
- `internal/config/testdb_policy.go`
- `internal/config/testdb_test.go`
- `internal/database/database.go`
- `cmd/server/main.go`
- referensi konfigurasi dan dokumentasi project yang menyebut `DATABASE_URL`, `TEST_DATABASE_URL`, charset, MariaDB, atau target database test.

Tidak ada `.env` atau nilai credential yang dibaca atau dicetak.

## 2. Fakta yang sudah terbukti

### Lingkungan lokal

Laporan Tahap 4I membuktikan adanya binary MariaDB XAMPP berikut:

- `C:\xampp\mysql\bin\mysql.exe`
- `C:\xampp\mysql\bin\mysqld.exe`
- `C:\xampp\mysql\bin\my.ini`

Pemeriksaan versi binary dengan opsi versi saja menunjukkan MariaDB `10.4.32`. Laporan Tahap 4J juga mencatat proses `mysqld.exe` dan listener TCP port `3306` pada saat audit tersebut. Fakta ini membuktikan binary/proses dan metadata OS pada waktu audit, bukan keberadaan schema, user, atau privilege tertentu.

### Konfigurasi aplikasi

`internal/config/config.go`:

- memanggil `godotenv.Load()` secara opsional;
- membaca `DATABASE_URL` ke `Config.DatabaseURL`;
- `config.Load()` adalah konfigurasi aplikasi umum, bukan konfigurasi khusus integration test;
- tidak ada pemeriksaan target database aman pada `DATABASE_URL`.

`internal/database/database.go`:

- `Connect(dsn string)` meneruskan DSN ke `gorm.Open(mysql.Open(dsn), ...)`;
- `Migrate` menjalankan `AutoMigrate` dan operasi kompatibilitas pada schema lama;
- fungsi tersebut bukan prosedur pemeriksaan read-only.

`cmd/server/main.go`:

1. memuat konfigurasi aplikasi;
2. membuka koneksi melalui `database.Connect`;
3. menjalankan `database.Migrate`;
4. menjalankan `database.Seed`;
5. memulai server HTTP.

Karena itu startup aplikasi tidak boleh digunakan untuk audit konfigurasi atau menyiapkan database test.

### Guardrail konfigurasi test

`internal/config/testdb.go` hanya menerima `APP_ENV` dan `TEST_DATABASE_URL` dari map environment yang diberikan. Guardrail yang tersedia menolak environment selain `test`, URL test kosong/tidak valid, policy kosong/tidak valid, serta target DSN yang tidak cocok dengan policy.

`internal/config/testdb_policy.go` menyediakan policy non-rahasia terpisah untuk target yang telah direncanakan: host `127.0.0.1`, port `3306`, database `movara_chat_test`, dan user `movara_test`. Policy tersebut bukan bukti bahwa objek-objek itu sudah dibuat di MariaDB.

`internal/config/testdb_test.go` menguji guardrail secara unit tanpa koneksi database, termasuk tidak adanya fallback ke `DATABASE_URL` dan error yang tidak memuat credential.

## 3. Hal yang belum dapat diverifikasi

Tanpa login atau SQL, hal-hal berikut tetap **BELUM TERVERIFIKASI**:

- versi server aktif dari metadata SQL, meskipun versi binary lokal sudah diketahui;
- database `movara_chat_test` ada atau tidak;
- akun `movara_test` ada atau tidak;
- host account yang tepat dan password-nya;
- grants akun test, termasuk kemungkinan akses global atau ke database aplikasi;
- charset/collation aktual server dan database;
- schema, tabel, migration, dan fixture yang sudah ada;
- apakah database yang terlihat di phpMyAdmin adalah target yang sama dengan instance pada port `3306`.

Database test tidak boleh dinyatakan tidak ada hanya karena tidak terlihat pada daftar phpMyAdmin. Visibility dapat dipengaruhi instance, koneksi, hak akses, atau refresh UI.

## 4. Cara mengetahui versi server dan konfigurasi test tanpa credential aplikasi

### Versi binary lokal

Ini sudah dilakukan secara aman pada Tahap 4I dengan menjalankan `mysql.exe --version` dan `mysqld.exe --version`. Hasilnya hanya membuktikan versi binary MariaDB lokal, bukan versi server yang sedang menerima koneksi.

### Versi server dan metadata database

Cara yang benar adalah operator berwenang menggunakan sesi phpMyAdmin atau client MariaDB yang sudah dikonfigurasi secara resmi, dengan akun pemeriksa terpisah dari akun aplikasi. Credential tidak perlu diberikan kepada agent atau ditempel di repository. Hasil yang dibagikan cukup berupa:

- versi server;
- host/port instance yang diperiksa;
- apakah schema test terlihat;
- charset/collation schema test;
- user/host akun target;
- ringkasan grants tanpa password atau DSN.

Jangan menggunakan `DATABASE_URL` aplikasi sebagai sumber akses pemeriksaan. Jangan mencoba password default atau credential alternatif.

### Konfigurasi database test

Konfigurasi target non-rahasia yang sudah direncanakan ada di `ApprovedTestDatabasePolicy()`. Nilai password/DSN test harus tetap hanya berada di environment lokal atau secret manager yang disetujui. Keberadaan dan privilege target harus diverifikasi oleh operator sebelum `TEST_DATABASE_URL` dipakai untuk integration test.

## 5. Rekomendasi

1. Pertahankan pemisahan `DATABASE_URL` aplikasi dan `TEST_DATABASE_URL` test.
2. Jangan menjalankan `cmd/server/main.go` untuk setup test karena startup melakukan connect, AutoMigrate, dan seed.
3. Minta operator berwenang melakukan audit metadata read-only pada instance MariaDB yang benar.
4. Pastikan pemeriksaan grants menunjukkan hanya schema test dan privilege yang disetujui; suffix `_test` saja tidak cukup.
5. Gunakan akun runtime test terpisah dari akun pemeriksa dan akun migration.
6. Setelah metadata disahkan, barulah siapkan schema melalui prosedur migration eksplisit yang telah disetujui.

## 6. Langkah manual paling sederhana untuk operator pemula

1. Buka XAMPP Control Panel dan pastikan panel menunjukkan instance MySQL/MariaDB yang memang ingin diperiksa. Jangan mengubah konfigurasi.
2. Buka phpMyAdmin dari tombol instance tersebut, bukan dari bookmark yang mungkin menunjuk server lain.
3. Catat nama host/port yang tampil tanpa menyalin password atau DSN.
4. Di halaman informasi/server phpMyAdmin, catat versi server yang terlihat.
5. Cari daftar database dan lakukan refresh. Catat apakah `movara_chat_test` terlihat; jika tidak terlihat, catat sebagai “belum terlihat”, bukan “pasti tidak ada”.
6. Jangan membuat database, user, atau privilege pada langkah ini.
7. Untuk pemeriksaan user/grants, minta bantuan administrator MariaDB. Jangan mencoba menebak password dan jangan menggunakan akun aplikasi.
8. Kirim kembali hanya ringkasan non-rahasia: versi server, host/port, status terlihat/tidak terlihatnya database test, serta ringkasan grants jika administrator menyediakannya.

Jika phpMyAdmin tidak menyediakan informasi yang diperlukan, berhenti dan serahkan pemeriksaan kepada operator yang memiliki akses administratif resmi. Jangan mencari jalan pintas autentikasi.

## 7. Status dan keputusan yang masih diperlukan

### Terbukti

- Konfigurasi aplikasi memakai `DATABASE_URL` melalui `config.Load()`.
- Guardrail test memakai `TEST_DATABASE_URL` dan policy terpisah.
- Startup aplikasi melakukan koneksi, AutoMigrate, seed, lalu menjalankan server.
- Binary lokal adalah MariaDB 10.4.32; port konfigurasi lokal tercatat 3306.

### NEEDS APPROVAL

- Identitas operator dan akun pemeriksa read-only yang boleh digunakan.
- Apakah pemeriksaan metadata dilakukan melalui phpMyAdmin atau client MariaDB resmi.
- Verifikasi bahwa phpMyAdmin terhubung ke instance `127.0.0.1:3306` yang dimaksud.
- Privilege minimum akun pemeriksa dan akun runtime test.
- Charset/collation final database test.
- Prosedur migration dan fixture setelah metadata disahkan.

## 8. Konfirmasi keselamatan

Laporan ini dibuat sebagai satu-satunya perubahan untuk Tahap 4L. Tidak ada login MariaDB, koneksi database, SQL, server MOVARA, migration, AutoMigrate, seeder, integration test, pembuatan/perubahan akun, privilege, schema, data, konfigurasi, dependency, atau source code yang dijalankan/diubah.
