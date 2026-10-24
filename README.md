# MOVARA (Go + Gin)

Tahap saat ini: **tampilan** (data contoh di memori, belum ada database).

## Menjalankan
1. `go mod tidy`
2. `make run` lalu buka http://localhost:8080
3. Login dengan akun demo (password semua `123`):
   - `puji` Pramuniaga MK 2 (halaman Input Mutasi)
   - `manager` Manager
   - `admin.a` Admin
   - `admin.b` Admin

Login `manager` atau `admin.a` untuk membuka **Rekap & Laporan** (data contoh ±45 hari, semua outlet).
Export Excel hanya tampil untuk Admin.

Barkode contoh untuk dicoba: `8992222075220`, `8997208072606`, `8997232230157`.
Barkode lain akan memunculkan pesan "belum terdaftar" + tombol WhatsApp.

## Struktur
- `internal/repository`: kontrak akses data. `repository/memory` = data contoh.
  Tahap database nanti cukup menambah implementasi PostgreSQL.
- `internal/service`: logika bisnis. `internal/handler`: HTTP + template.
- `web/templates`: `layout.html`, `icons.html`, satu file per halaman.
