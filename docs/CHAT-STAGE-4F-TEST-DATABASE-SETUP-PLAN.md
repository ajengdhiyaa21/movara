# MOVARA — Tahap 4F: Rencana Setup Database Test MySQL

Tanggal audit: 27 Oktober 2026  
Mode: read-only terhadap database  
Target rencana: MySQL lokal XAMPP pada `127.0.0.1:3306`.

## 1. Temuan aktual

- `internal/config/testdb_policy.go` mengunci target non-secret: database `movara_chat_test`, host `127.0.0.1`, port `3306`, user `movara_test`.
- `internal/config/testdb.go` hanya memvalidasi `TEST_DATABASE_URL` terhadap policy tersebut; helper tidak membuka koneksi.
- `.env.example:9` memakai charset `utf8mb4` pada DSN aplikasi. Collation tidak ditentukan di kode/dokumentasi yang diperiksa.
- `internal/database/database.go:14-49` masih menggunakan AutoMigrate dan operasi kompatibilitas; prosedur setup test tidak boleh menjalankan startup aplikasi.
- `cmd/server/main.go:24-32` menghubungkan database lalu menjalankan migration dan seed; file ini tidak boleh dipakai untuk setup test.
- Tidak ditemukan bukti database `movara_chat_test`, akun `movara_test`, atau privilege-nya sudah ada. Tidak ada koneksi yang dicoba.

Target database/user adalah rencana yang disetujui, bukan bukti objek sudah dibuat atau terisolasi.

## 2. Keputusan NEEDS APPROVAL

- **NEEDS APPROVAL:** collation final. `utf8mb4` sudah terlihat pada DSN, tetapi collation belum dibuktikan. `utf8mb4_unicode_ci` di bawah hanya kandidat yang harus ditinjau.
- **NEEDS APPROVAL:** apakah database test dibuat oleh operator XAMPP secara manual atau provisioning terpisah.
- **NEEDS APPROVAL:** privilege DDL sementara untuk migration dan cara mencabutnya.
- **NEEDS APPROVAL:** strategi migration berversi yang belum tersedia di project.
- **NEEDS APPROVAL:** apakah fixture membutuhkan `DELETE` penuh atau memakai database/schema baru per run.
- **NEEDS APPROVAL:** password test dibuat operator dan disimpan hanya di environment lokal.

## 3. Pemeriksaan sebelum eksekusi

Operator harus terlebih dahulu memakai akun administrator MySQL yang sah, tanpa menaruh password di laporan/source code. Pemeriksaan berikut hanya boleh dilakukan setelah prosedur ini disetujui dan dengan target yang dikonfirmasi.

### Cek database dan user tanpa membuat/menghapus objek

```sql
SELECT SCHEMA_NAME
FROM INFORMATION_SCHEMA.SCHEMATA
WHERE SCHEMA_NAME = 'movara_chat_test';

SELECT User, Host
FROM mysql.user
WHERE User = 'movara_test' AND Host = '127.0.0.1';
```

Jika database/user sudah ada, jangan menganggapnya aman. Lanjutkan dengan `SHOW GRANTS` dan pemeriksaan schema/privilege. Jangan menjalankan `DROP`, `TRUNCATE`, atau reset.

## 4. SQL yang diusulkan — belum boleh dijalankan

### Membuat database

Collation berikut adalah kandidat, bukan keputusan final:

```sql
CREATE DATABASE IF NOT EXISTS `movara_chat_test`
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;
```

Alasan: DSN aplikasi menunjukkan `utf8mb4`; penggunaan collation tertentu masih **NEEDS APPROVAL** dan harus disesuaikan dengan standar MySQL project.

### Membuat akun test dari host lokal saja

Password harus dibuat manual oleh operator menggunakan password acak, lalu dimasukkan melalui prompt/secret environment. Placeholder tidak boleh diganti di source code atau laporan.

```sql
CREATE USER IF NOT EXISTS 'movara_test'@'127.0.0.1'
  IDENTIFIED BY '<PASSWORD-DIBUAT-OPERATOR-SECARA-MANUAL>';
```

Jika user sudah ada, jangan mengganti password atau privilege tanpa persetujuan. Audit dahulu dengan `SHOW GRANTS`.

### Privilege runtime test minimum

Untuk test CRUD runtime pada schema yang sudah dimigrasikan, kandidat minimum:

```sql
GRANT SELECT, INSERT, UPDATE, DELETE
ON `movara_chat_test`.*
TO 'movara_test'@'127.0.0.1';
```

Tidak ada `GRANT OPTION`, tidak ada `*.*`, dan tidak ada privilege pada schema aplikasi. `CREATE`, `ALTER`, `DROP`, dan privilege DDL tidak diberikan permanen kepada akun runtime test.

### Migration privilege terpisah

Migration memerlukan keputusan dan akun/role operator terpisah. Jangan memberikan DDL kepada `movara_test` secara otomatis. Jika migration perlu DDL sementara, gunakan akun migration khusus atau grant sementara yang terdokumentasi, diverifikasi, dan dicabut setelah migration; prosedur tersebut **NEEDS APPROVAL** dan tidak dirancang sebagai perintah otomatis di tahap ini.

## 5. Verifikasi setelah eksekusi oleh operator

### Verifikasi grant akun test

```sql
SHOW GRANTS FOR 'movara_test'@'127.0.0.1';
```

Pastikan hasil hanya menyebut `movara_chat_test` dan privilege runtime yang disetujui. Pastikan tidak ada `ON *.*`, `GRANT OPTION`, atau schema aplikasi.

### Verifikasi database dan default charset/collation

```sql
SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME
FROM INFORMATION_SCHEMA.SCHEMATA
WHERE SCHEMA_NAME = 'movara_chat_test';
```

Bandingkan hasil dengan keputusan collation yang disetujui. Jangan mengubah collation existing tanpa review.

### Verifikasi objek user

```sql
SELECT User, Host
FROM mysql.user
WHERE User = 'movara_test' AND Host = '127.0.0.1';
```

Keberadaan row tersebut tidak membuktikan privilege aman; `SHOW GRANTS` adalah pemeriksaan wajib.

### Verifikasi isolasi

Gunakan sesi terpisah dengan akun test setelah password disediakan melalui environment operator. Verifikasi hanya target test dan jangan membaca/menulis schema aplikasi. Jika hasil grant mencurigakan atau target tidak sesuai, hentikan setup dan jangan melakukan cleanup destruktif.

## 6. Cara membuat dan menyimpan password

Operator membuat password acak secara manual menggunakan password manager atau generator lokal yang disetujui. Password:

- tidak ditulis pada source code, Git, laporan, command history, atau template SQL yang disimpan;
- dimasukkan ke environment lokal, misalnya `TEST_DATABASE_URL`, tanpa menampilkan nilainya;
- tidak boleh sama dengan password aplikasi/production;
- dibatasi akses file/environment sesuai akun operator.

Format DSN hanya disusun pada environment lokal setelah database, user, host, port, dan privilege diverifikasi. Jangan menambahkan password ke `.env.example`.

## 7. Pemulihan aman

- Jika database sudah ada: hentikan pembuatan, audit charset/collation, objek, dan grants; jangan drop atau reset.
- Jika user sudah ada: audit host dan grants; jangan `DROP USER` atau mengganti credential otomatis.
- Jika setup gagal: simpan error non-secret, hentikan proses, dan minta operator memperbaiki target/privilege.
- Jika collation berbeda: minta keputusan kompatibilitas; jangan menjalankan `ALTER DATABASE` tanpa review.
- Jika grant terlalu luas: jangan melakukan revocation otomatis pada akun yang mungkin dipakai pihak lain; minta operator melakukan perubahan terkontrol.
- Jika migration gagal di masa depan: hentikan versi berikutnya dan gunakan forward-fix/restore terkontrol; jangan menghapus data chat otomatis.

## 8. Risiko dan mitigasi

| Risiko | Mitigasi |
|---|---|
| XAMPP tidak aktif atau port berbeda | Verifikasi operator terpisah; jangan mengasumsikan koneksi |
| Database suffix `_test` dianggap cukup | Wajib verifikasi host, user, grants, dan schema |
| Akun test membaca database aplikasi | Grant hanya `movara_chat_test.*`; cek `SHOW GRANTS` |
| Privilege global | Tolak hasil yang memuat `*.*` atau `GRANT OPTION` |
| Password masuk Git/log | Placeholder saja; password melalui secret environment |
| Runtime test memiliki DDL | Pisahkan akun runtime dan migration |
| SQL idempotent disalahgunakan untuk objek lama | Cek dulu, jangan drop/reset |
| Charset/collation tidak kompatibel | Tandai collation sebagai NEEDS APPROVAL dan verifikasi sebelum create |

## 9. Daftar perintah yang belum boleh dijalankan

Pada tahap ini belum boleh menjalankan:

- seluruh blok `CREATE DATABASE`, `CREATE USER`, `GRANT`, atau perubahan privilege di atas;
- `SHOW GRANTS` atau query pemeriksaan melalui koneksi MySQL;
- `database.Connect`, `gorm.Open`, server MOVARA, `go run`, atau executable aplikasi;
- AutoMigrate, Seed, migration, integration test, reset, truncate, drop, atau cleanup database;
- pembuatan database/user XAMPP atau perubahan privilege.

SQL di dokumen hanya proposal untuk ditinjau dan disetujui. Prosedur ini tidak boleh dijalankan pada Tahap 4F.

## 10. Konfirmasi perubahan

File yang dibuat: `docs/CHAT-STAGE-4F-TEST-DATABASE-SETUP-PLAN.md`.

Tidak ada koneksi MySQL, SQL, database, schema, user, privilege, server, migration, seeder, AutoMigrate, source code, konfigurasi, `.env`, `DATABASE_URL`, atau dependency yang diubah. Perubahan lokal yang sudah ada dipertahankan.
