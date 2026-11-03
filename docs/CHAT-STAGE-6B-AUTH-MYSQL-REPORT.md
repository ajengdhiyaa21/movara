# MOVARA Chat — Stage 6B: Authorization dan MySQL Repository

Tanggal: 28 Oktober 2026  
Status: selesai untuk authorization service dan repository; API, WebSocket, migration, dan integration test belum dibuat.

## File berubah

File baru:

- `internal/repository/mysql/chat.go`
- `internal/service/chat_auth_test.go`
- `docs/CHAT-STAGE-6B-AUTH-MYSQL-REPORT.md`

File diubah:

- `internal/repository/chat.go`
- `internal/repository/memory/chat.go`
- `internal/repository/mysql/mysql.go`
- `internal/service/chat.go`
- `internal/repository/repository.go`
- `internal/repository/memory/memory.go`

`internal/models/models.go` tidak diubah pada Stage 6B; model chat Stage 6A digunakan apa adanya.

## Authorization yang diterapkan

`AuthorizedChatService` sekarang memvalidasi sebelum membaca atau mengubah data:

- hanya Admin dan PM aktif yang boleh menggunakan chat;
- Manager ditolak;
- Admin aktif dapat membaca conversation mana pun yang tersedia;
- PM hanya dapat membaca conversation dengan `PMUserID` miliknya;
- PM nonaktif ditolak sebelum membaca, mengirim, atau membuat read state;
- Admin tidak dapat mengirim pesan kepada PM nonaktif;
- Admin tetap dapat membaca histori PM nonaktif;
- sender diambil dari user terverifikasi yang diberikan ke service, bukan dari payload pesan;
- read state melalui jalur authorized tetap memeriksa conversation sebelum update.

Pemeriksaan service ini melengkapi middleware yang sudah ada. Middleware `LoadUser`, session, dan login tidak diubah.

## MySQL repository yang dibuat

Implementasi `internal/repository/mysql/chat.go` mencakup:

- lookup conversation berdasarkan PM dan ID;
- create conversation dengan timestamp server aplikasi;
- lookup message berdasarkan conversation dan `client_message_id`;
- lookup message berdasarkan ID;
- create message;
- pagination `before_id`/`after_id` dengan limit;
- lookup read state per conversation/user;
- update read state dalam transaction dengan perlindungan monotonic.

`mysql.New` sekarang mengekspos chat repository melalui `repository.Repos.Chat`.

Repository tidak menambahkan `AutoMigrate`, tidak menjalankan migration, dan tidak membuka koneksi sendiri. Unique conversation/message tetap bergantung pada constraint database saat migration nanti dibuat.

## Unit test yang ditambahkan

`internal/service/chat_auth_test.go` menguji:

- Manager ditolak;
- PM tidak dapat membaca conversation PM lain;
- Admin dapat membaca conversation;
- PM aktif dapat mengirim;
- PM nonaktif ditolak;
- Admin dapat membaca histori PM nonaktif;
- Admin ditolak mengirim kepada PM nonaktif.

Test memakai memory repository dan test double user; tidak membutuhkan database.

## Verifikasi yang dijalankan

```text
go test ./...
go vet ./...
go build ./...
```

Semua perintah **lulus**.

Test tidak menjalankan `cmd/server/main.go`, `database.Connect`, `gorm.Open`, `AutoMigrate`, `Seed`, SQL, atau koneksi MariaDB.

## Pekerjaan yang masih tertunda

- SQL migration berversi dan pembuatan constraint/FK `RESTRICT`.
- Verifikasi engine, FK, collation, dan privilege MariaDB.
- Integration test MySQL terhadap database test terisolasi.
- HTTP handler/API dan CSRF terpisah.
- WebSocket handshake, event, Origin validation, dan reconnect.
- Fixture database chat.
- Audit concurrency nyata pada unique constraint MariaDB.
- Validasi error duplicate-key spesifik driver database.

Tidak ada migration atau query database yang dijalankan pada tahap ini.

## Konfirmasi keselamatan

Tidak ada server, koneksi database, SQL, AutoMigrate, Seed, migration, perubahan `.env`, konfigurasi database, privilege, route, WebSocket, UI, fitur mutasi, atau pekerjaan CSRF yang dijalankan/diubah.
