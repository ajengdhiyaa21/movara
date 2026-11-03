# MOVARA Internal Chat — Technical Plan

Status: Rencana teknis, belum diimplementasikan.

## 1. Rekomendasi arsitektur

Gunakan pola yang sudah ada:

`model → repository → service → handler → router → template/JavaScript`

Komponen konseptual:

- `Conversation`: identitas percakapan dan status.
- `ConversationParticipant`: user yang boleh mengakses percakapan, jika model participant eksplisit dipilih.
- `Message`: pengirim, conversation, isi, waktu, status.
- `MessageRead`: opsional untuk read receipt/unread.
- Chat repository untuk create/list/read/update status.
- Chat service untuk authorization, validasi, deduplication, dan pagination.
- HTTP handler untuk halaman, daftar conversation, histori, dan kirim pesan.
- Real-time adapter terpisah dari business service.

## 2. Perbandingan mekanisme komunikasi

| Opsi | Kelebihan | Kekurangan | Penilaian |
|---|---|---|---|
| HTTP polling | Paling sederhana, mudah diuji, tidak perlu dependency besar | Tidak benar-benar real-time, request berulang | Cocok untuk MVP kecil |
| Long polling | Lebih responsif dari polling biasa | Handler dan timeout lebih kompleks | Alternatif transisi |
| WebSocket | Real-time, UX terbaik | Reconnect, concurrency, deployment, testing lebih kompleks | Cocok setelah kontrak API stabil |
| SSE + POST | Server push sederhana, browser native | Pengiriman client tetap HTTP, komunikasi satu arah | Kandidat baik untuk notifikasi pesan |

Rekomendasi: implementasikan persistence dan HTTP API terlebih dahulu. Untuk tahap awal, gunakan polling ringan atau SSE bila kebutuhan real-time mendesak. WebSocket dipilih hanya setelah jumlah user, proxy, timeout, dan kebutuhan online indicator disetujui.

## 3. Rancangan data konseptual

### conversations

- `id`
- `type` — `admin_pm`
- `created_at`
- `updated_at`
- `is_active`

### conversation_participants

- `id`
- `conversation_id`
- `user_id`
- `joined_at`
- `last_read_message_id` atau `last_read_at`

### messages

- `id`
- `conversation_id`
- `sender_id`
- `body`
- `client_message_id` untuk deduplication
- `created_at`
- `edited_at` jika edit disetujui
- `deleted_at` jika soft delete disetujui

Index minimal:

- `(conversation_id, created_at, id)`;
- `(sender_id, created_at)`;
- unique `(conversation_id, client_message_id)` jika client ID digunakan.

## 4. Kontrak endpoint konseptual

Endpoint final menunggu persetujuan scope.

- `GET /chat` — halaman chat.
- `GET /api/chat/conversations` — daftar percakapan yang user boleh lihat.
- `POST /api/chat/conversations` — membuat atau mendapatkan conversation Admin–PM.
- `GET /api/chat/conversations/:id/messages?before=...&limit=...` — histori ber-pagination.
- `POST /api/chat/conversations/:id/messages` — mengirim pesan.
- `POST /api/chat/conversations/:id/read` — menandai telah dibaca jika disetujui.
- `GET /api/chat/events` atau `GET /api/chat/stream` — SSE jika dipilih.
- `GET /api/chat/ws` — WebSocket jika dipilih.

Server tidak boleh menerima `sender_id` dari client sebagai sumber kebenaran. Pengirim harus berasal dari session.

## 5. Otorisasi

Setiap request harus:

1. Melewati `RequireAuth`.
2. Mengambil user dari session melalui `CurrentUser`.
3. Memastikan conversation memang dimiliki/diizinkan untuk user tersebut.
4. Memastikan PM hanya mengakses conversation miliknya.
5. Memastikan user nonaktif tidak dapat mengirim pesan baru.
6. Tidak mempercayai participant ID dari browser tanpa query authorization server-side.

Admin visibility masih membutuhkan keputusan bisnis: semua Admin atau berdasarkan outlet/tanggung jawab.

## 6. Keamanan

- Tambahkan CSRF token untuk POST/read state-changing request.
- Validasi panjang body dan content type.
- Simpan pesan sebagai teks dan render dengan escaping.
- Tolak HTML mentah, script, dan payload berukuran besar.
- Rate limit pengiriman pesan per user/conversation.
- Gunakan idempotency key untuk mencegah duplikasi saat reconnect.
- Jangan log isi pesan ke log aplikasi secara default.
- Jangan mengembalikan pesan dari conversation lain hanya karena ID valid.
- Jika lampiran disetujui, lakukan pembatasan MIME, ukuran, nama file, storage, dan antivirus scan.
- Saat production, gunakan Secure cookie dan TLS.
- WebSocket handshake harus tetap memvalidasi session dan Origin.

## 7. Rencana milestone

### M0 — Keputusan kebutuhan

Tujuan: menyelesaikan pertanyaan bisnis dan akses.  
Test: review stakeholder.  
Selesai jika: scope, role, retensi, dan fitur MVP disetujui.

### M1 — Kontrak dan desain data

Tujuan: finalisasi model conversation/message dan API.  
Komponen: model, migration, repository interface.  
Risiko: schema sulit diubah setelah data produksi.  
Selesai jika: migration rollback/forward dan query utama direview.

### M2 — Persistence backend

Tujuan: create/list conversation dan message.  
Test: repository integration test dengan database test.  
Selesai jika: pagination dan constraint bekerja.

### M3 — Service dan authorization

Tujuan: role, participant, validasi body, batas pesan, deduplication.  
Test: PM A/B, Admin, Manager, user nonaktif, ID palsu.  
Selesai jika: tidak ada cross-conversation access.

### M4 — HTTP API

Tujuan: endpoint chat berbasis HTTP.  
Test: status code, validation, CSRF, pagination, error response.  
Selesai jika: API dapat digunakan tanpa UI.

### M5 — UI chat

Tujuan: popup/halaman chat yang konsisten dengan layout MOVARA.  
Test: keyboard, mobile, empty state, loading, error.  
Selesai jika: PM dan Admin dapat menyelesaikan alur pesan dasar.

### M6 — Real-time adapter

Tujuan: polling, SSE, atau WebSocket sesuai keputusan M0.  
Test: reconnect, duplicate event, session expired, multi-tab.  
Selesai jika: pesan baru muncul tanpa refresh dan tidak dobel.

### M7 — QA dan security

Tujuan: regression, role matrix, load ringan, security testing.  
Stop condition: hentikan implementasi jika authorization test gagal atau pesan bisa dibaca lintas akun.

### M8 — Deployment

Tujuan: migration production, observability, backup, dokumentasi.  
Selesai jika: rollback plan, monitoring, dan runbook tersedia.

## 8. Risiko dan mitigasi

| Risiko | Mitigasi |
|---|---|
| Conversation ID dapat ditebak | Authorization server-side pada setiap query |
| Pesan dobel saat reconnect | client message ID dan unique constraint |
| Beban meningkat akibat polling | cursor, interval adaptif, SSE/WebSocket setelah diukur |
| Session expired saat koneksi aktif | tutup koneksi dan arahkan login ulang |
| Histori terlalu besar | cursor pagination dan index |
| Isi chat mengandung XSS | escape output dan validasi body |
| Lampiran berbahaya | tunda lampiran dari MVP atau gunakan pipeline upload aman |

## 9. Ketergantungan pada pekerjaan lama

- Perbaikan filter kategori PM diperlukan agar chat dan dashboard memiliki scope data yang konsisten.
- Validasi outlet aktif perlu diperkuat sebelum menentukan perilaku chat akun PM nonaktif.
- CSRF token sebaiknya diselesaikan sebelum menambah endpoint kirim pesan.
- Warning `go vet` dan template legacy sebaiknya dibereskan sebagai pekerjaan maintenance terpisah.
