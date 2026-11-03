# MOVARA — Tahap 4N: Final Read-only Test Database Readiness Audit

Tanggal audit: 28 Oktober 2026  
Mode: audit statis; tidak ada autentikasi, koneksi database, atau SQL.

## 1. Kesimpulan readiness

Status database pengujian: **NOT READY**.

Integration test database belum boleh dijalankan karena database `movara_chat_test`, akun `movara_test`, dan privilege akun tersebut belum diverifikasi. Tidak terlihatnya database atau user pada phpMyAdmin tidak cukup untuk menyimpulkan objek tersebut pasti tidak ada; hasil dapat dipengaruhi instance, koneksi, refresh, atau hak akses akun pemeriksa.

## 2. File dan sumber yang diperiksa

Laporan:

- `docs/CHAT-STAGE-4F-TEST-DATABASE-SETUP-PLAN.md`
- `docs/CHAT-STAGE-4G-SCHEMA-COMPATIBILITY-AUDIT.md`
- `docs/CHAT-STAGE-4H-DATABASE-POLICY-RECOMMENDATION.md`
- `docs/CHAT-STAGE-4I-LOCAL-DB-ENVIRONMENT-AUDIT.md`
- `docs/CHAT-STAGE-4J-MARIADB-READONLY-AUDIT.md`
- `docs/CHAT-STAGE-4K-READONLY-ACCESS-OPTIONS.md`
- `docs/CHAT-STAGE-4L-READONLY-CONFIG-AUDIT.md`

Implementasi:

- `internal/config/testdb.go`
- `internal/config/testdb_policy.go`
- `internal/config/testdb_test.go`

Tidak ada `.env`, password, token, DSN rahasia, atau konfigurasi credential lokal yang dibaca atau ditampilkan.

## 3. Fakta yang sudah diketahui

### Lingkungan MariaDB

Berdasarkan pemeriksaan manual dan audit lokal sebelumnya:

- phpMyAdmin terhubung ke MariaDB `10.4.32`;
- koneksi ditampilkan sebagai `127.0.0.1` melalui TCP/IP;
- karakter server dilaporkan `utf8mb4`;
- binary lokal dan konfigurasi XAMPP sebelumnya menunjukkan target port `3306`.

Fakta tersebut belum membuktikan bahwa database test atau akun test sudah dibuat dan aman.

### Konfigurasi dan startup aplikasi

`internal/config/config.go` membaca `DATABASE_URL` sebagai konfigurasi aplikasi. `internal/database/database.go` menyediakan `Connect`, `AutoMigrate`, operasi kompatibilitas schema, dan seed terkait startup. `cmd/server/main.go` menghubungkan database, menjalankan migration, seed, kemudian memulai server.

Startup aplikasi karena itu bukan prosedur setup test yang aman dan tidak boleh dipakai sebagai pengganti migration test eksplisit.

### Guardrail test

`ValidateTestDatabase` di `internal/config/testdb.go` mensyaratkan `APP_ENV=test` dan `TEST_DATABASE_URL`, tidak fallback ke `DATABASE_URL`, memvalidasi policy sebelum menerima target, menolak target di luar allowlist, tidak membuka koneksi, dan tidak mengembalikan DSN mentah pada konfigurasi hasil validasi.

`ApprovedTestDatabasePolicy()` di `internal/config/testdb_policy.go` menyediakan allowlist non-rahasia untuk `movara_chat_test`, `127.0.0.1`, port `3306`, dan `movara_test`. Policy ini hanya membatasi target yang boleh diminta; policy bukan bukti objek database/user atau privilege telah ada.

`internal/config/testdb_test.go` menguji penolakan environment salah, URL test kosong, fallback aplikasi, policy invalid, target tidak sesuai, serta kebocoran credential pada error. Test tersebut bersifat unit dan tidak memerlukan koneksi MariaDB.

## 4. Fakta manual dan hal yang belum diketahui

| Item | Status | Catatan |
|---|---|---|
| Produk/versi server aktif | TERKONFIRMASI secara manual | MariaDB 10.4.32 pada phpMyAdmin |
| Host yang ditampilkan | TERKONFIRMASI secara manual | 127.0.0.1 via TCP/IP |
| Charset server | TERKONFIRMASI secara manual | utf8mb4 |
| Database `movara_chat_test` | BELUM TERVERIFIKASI | Tidak terlihat; bukan bukti pasti tidak ada |
| User `movara_test` | BELUM TERVERIFIKASI | Tidak terlihat; bukan bukti pasti tidak ada |
| `SHOW GRANTS` akun test | BELUM TERVERIFIKASI | Belum ada ringkasan grants yang disetujui |
| Akses akun test ke database aplikasi | BELUM TERVERIFIKASI | Wajib diperiksa dari grants aktual |
| Charset/collation database test | BELUM TERVERIFIKASI | Charset server tidak menetapkan default schema secara otomatis dalam laporan ini |
| Schema integration test | BELUM TERVERIFIKASI | Belum ada migration test yang disetujui/dijalankan |
| Fixture test | BELUM TERSEDIA | Belum boleh memakai seeder startup |

## 5. Mengapa integration test belum boleh dijalankan

Menjalankan integration test sekarang berisiko karena tidak ada bukti target database terpisah, tidak ada bukti privilege akun terbatas, helper hanya memvalidasi alamat/identitas target bukan grants aktual, schema belum dibuat melalui migration eksplisit, dan startup aplikasi melakukan AutoMigrate serta Seed. Test dapat gagal karena schema/fixture belum tersedia atau menulis ke target yang salah.

Sebelum test database dijalankan harus ada bukti non-rahasia tentang target, grants, schema, dan prosedur migration yang disetujui.

## 6. Rencana setup terpisah dan aman

Rencana berikut hanya untuk tahap setelah persetujuan; tidak dijalankan pada Tahap 4N.

### Fase A — Konfirmasi operator

- Tetapkan operator/admin database yang berwenang.
- Pastikan phpMyAdmin/client benar-benar terhubung ke instance target.
- Simpan credential hanya secara lokal; jangan memasukkannya ke repository, laporan, chat, atau command history.

### Fase B — Pemeriksaan read-only

Operator memeriksa keberadaan schema, user/host, charset/collation, dan grants tanpa membuat atau menghapus objek. Jika objek sudah ada, audit grants dan schema terlebih dahulu; jangan melakukan reset, drop, atau truncate.

Hasil yang dibagikan cukup ringkasan non-rahasia. Jika pemeriksaan user/grants membutuhkan hak administratif, pemeriksaan dilakukan oleh admin sah atau akun inspector read-only yang telah disetujui.

### Fase C — Persetujuan target

Setujui secara eksplisit schema `movara_chat_test`, host `127.0.0.1`/port `3306`, akun runtime test terpisah, charset `utf8mb4`, collation, aturan case/accent sensitivity, migration tool, serta fixture dan strategi cleanup.

### Fase D — Provisioning terpisah

Jika objek belum ada setelah pemeriksaan, operator menjalankan prosedur provisioning yang disetujui pada database test saja. Database aplikasi yang ada tidak boleh disentuh.

Privilege runtime test dibatasi pada DML yang diperlukan di `movara_chat_test` dan tidak mencakup `GRANT OPTION`, akses global, atau schema aplikasi. Jangan memberikan `CREATE`, `ALTER`, atau `DROP` secara permanen kepada runtime test.

### Fase E — Migration dan fixture

- Akun migration/DDL harus berbeda dari akun runtime test.
- Migration dijalankan eksplisit di database test, bukan melalui startup aplikasi.
- Fixture test dibuat deterministik dan terpisah dari `database.Seed`.
- Kegagalan migration dihentikan dan ditangani dengan forward-fix atau pemulihan yang disetujui; tidak ada cleanup destruktif otomatis.

### Fase F — Validasi akhir

Validasi ulang target melalui `ValidateTestDatabase` dengan `APP_ENV=test`, `TEST_DATABASE_URL`, dan `ApprovedTestDatabasePolicy()`. Setelah itu baru lakukan pemeriksaan koneksi melalui helper test khusus yang disetujui, bukan `cmd/server/main.go`.

## 7. Pemisahan akun

### Akun runtime test

Digunakan test untuk membaca dan menulis fixture. Kandidat minimum adalah DML pada schema test saja. Tidak boleh memiliki DDL permanen, akses database aplikasi, `ON *.*`, atau `GRANT OPTION`.

### Akun migration

Digunakan hanya untuk migration terkontrol. Hak DDL dibatasi sesuai prosedur dan tidak dipakai sebagai credential integration test.

### Akun pemeriksa

Jika diperlukan, akun pemeriksa metadata harus berbeda dari runtime dan migration. Haknya ditentukan operator/admin dan hanya digunakan untuk pemeriksaan yang disetujui.

## 8. Tindakan yang membutuhkan persetujuan

Status **NEEDS APPROVAL**:

- pembuatan database `movara_chat_test`;
- pembuatan/perubahan akun `movara_test`;
- penetapan password dan penyimpanan `TEST_DATABASE_URL`;
- pemberian privilege runtime;
- pembuatan akun migration dan privilege DDL;
- pemilihan collation final;
- pemilihan tool migration berversi;
- pembuatan schema chat dan migration fitur chat;
- strategi fixture, reset, cleanup, dan pelaksanaan integration test pertama.

Tidak satu pun tindakan tersebut dilakukan pada tahap ini.

## 9. Verifikasi read-only yang masih diperlukan

Setelah ada persetujuan dan operator berwenang, verifikasi harus menghasilkan ringkasan:

1. versi server dan instance/host/port;
2. keberadaan atau visibility schema test;
3. user dan host akun runtime;
4. grants akun runtime;
5. tidak adanya `ON *.*`, `GRANT OPTION`, atau schema aplikasi pada grants runtime;
6. charset/collation schema test;
7. status schema/migration yang diterapkan.

Jika salah satu poin keamanan utama tidak dapat diverifikasi, status tetap **NOT READY**.

## 10. Kriteria siap pada tahap berikutnya

Database test dapat dinyatakan READY apabila target lolos `ValidateTestDatabase` dengan policy tepercaya, schema/instance terpisah terkonfirmasi, grants runtime terbatas pada schema test, runtime dan migration account terpisah, migration eksplisit berhasil, fixture deterministik tersedia, serta jalur test tidak menjalankan startup app, AutoMigrate, Seed, atau fallback `DATABASE_URL`.

## 11. Konfirmasi batasan tahap ini

Laporan ini adalah rencana readiness saja. Tidak ada koneksi database, autentikasi, SQL, server MOVARA, migration, AutoMigrate, seeder, integration test, pembuatan/perubahan akun, privilege, database, schema, data, konfigurasi, dependency, atau source code yang dijalankan/diubah. Laporan Tahap 4F–4L tidak diubah.
