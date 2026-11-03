# MOVARA Chat — Stage 6A: Foundation Report

Tanggal: 28 Oktober 2026  
Status: selesai untuk fondasi backend; API, WebSocket, migration, dan UI belum dibuat.

## File dibuat/diubah

File baru:

- `internal/repository/chat.go`
- `internal/repository/memory/chat.go`
- `internal/service/chat.go`
- `internal/service/chat_test.go`
- `docs/CHAT-STAGE-6A-FOUNDATION-REPORT.md`

File diubah:

- `internal/models/models.go`
- `internal/repository/repository.go`
- `internal/repository/memory/memory.go`

Tidak ada perubahan pada autentikasi, middleware, mutasi barang, UI, CSRF, router, startup, konfigurasi database, atau dependency.

## Implementasi yang selesai

- Model `ChatConversation`, `ChatMessage`, dan `ChatReadState`.
- Unique conversation per `PMUserID`.
- Composite unique message key `(ConversationID, ClientMessageID)`.
- Index histori `(ConversationID, ID)`.
- Read state per user dan conversation.
- Repository abstraction `ChatRepository`.
- Memory repository dengan lock terpisah dan ID generator untuk test.
- Pencarian conversation berdasarkan PM.
- Pencarian message berdasarkan client ID dan message ID.
- Pagination `before_id` dan `after_id`.
- Validasi body kosong dan maksimum 3.000 karakter.
- Validasi UUID client message ID tanpa dependency baru.
- Idempotent retry jika client ID dan payload sama.
- Penolakan konflik jika client ID yang sama dipakai untuk payload berbeda.
- Read state monotonic dan validasi bahwa pesan berasal dari conversation yang sama.
- Default page size 30 dan maksimum page size 100.

## Unit test yang ditambahkan

Test tanpa database mencakup:

- validasi UUID;
- batas panjang pesan;
- satu conversation per PM;
- idempotent retry;
- konflik client ID;
- pagination histori;
- read state tidak dapat mundur;
- read state menolak pesan dari conversation lain.

## Verifikasi yang dijalankan

Perintah yang dijalankan:

```text
go test ./...
go vet ./...
go build ./...
```

Hasil ketiganya: **lulus**.

Test yang ada tidak menjalankan `cmd/server/main.go`, tidak memanggil `database.Connect`, tidak membuka GORM, dan hanya menggunakan memory repository atau unit helper.

## Keterbatasan yang tersisa

- Belum ada implementasi MySQL repository.
- Belum ada pemeriksaan role/status aktif pada ChatService; itu akan diterapkan pada service authorization/HTTP tahap berikutnya.
- Belum ada route HTTP, handler, CSRF, WebSocket, event adapter, atau UI.
- Belum ada SQL migration berversi.
- FK `RESTRICT`, engine tabel, collation, dan privilege database belum diverifikasi karena belum ada koneksi database.
- Belum ada integration test.
- Repository memory mensimulasikan duplicate-key, tetapi perilaku error database MariaDB belum diuji.

## Konfirmasi keselamatan

Tidak ada database, akun, privilege, schema, migration, AutoMigrate, Seed, server, SQL, atau koneksi database yang dijalankan. Tidak ada dependency atau konfigurasi environment yang diubah.
