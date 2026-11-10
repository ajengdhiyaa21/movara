package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"movara/internal/clock"
	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/service"
)

// scanForm: state form scan di halaman Input Mutasi.
type scanForm struct {
	Barcode     string
	Product     *models.Product // hasil lookup (hanya untuk tampilan)
	ProductName string
	Qty         int
	BA          string
	Direction   string
	Error       string
	Success     string
	Warning     string
	WAURL       string // link WhatsApp "produk belum terdaftar"
}

type historyView struct {
	Rows      []models.Mutation
	TotalQty  int
	Total     int
	Page      int
	Pages     int
	PerPage   int
	PrevURL   string
	NextURL   string
	Query     string
	Date      string
	Direction string
}

type MutationHandler struct {
	r   *Renderer
	svc *service.MutationService
	db  *gorm.DB
}

func NewMutationHandler(r *Renderer, svc *service.MutationService, db *gorm.DB) *MutationHandler {
	return &MutationHandler{r: r, svc: svc, db: db}
}

// GET /mutasi : halaman penuh.
func (h *MutationHandler) Index(c *gin.Context) {
	u := middleware.CurrentUser(c)
	counterHistory, ok := h.history(c, u, "", "", "counter_to_market")
	if !ok {
		return
	}
	marketHistory, ok := h.history(c, u, "", "", "market_to_counter")
	if !ok {
		return
	}
	direction := normalizeDirection(c.Query("direction"))
	h.r.HTML(c, http.StatusOK, "mutasi.html", gin.H{
		"Title":          "Input Mutasi",
		"Active":         "mutasi",
		"Form":           scanForm{Qty: 1, Direction: direction},
		"CounterHistory": counterHistory,
		"MarketHistory":  marketHistory,
		"WAURL": h.r.WhatsAppURL(fmt.Sprintf(
			"Halo Admin, saya %s (%s). Saya ingin menambahkan produk baru ke master data.",
			u.Name, outletName(u))),
	})
}

// GET /mutasi/form : form kosong (tombol Reset / Esc).
func (h *MutationHandler) Form(c *gin.Context) {
	h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": scanForm{Qty: 0, Direction: normalizeDirection(c.Query("direction"))}})
}

// GET /mutasi/input : halaman khusus untuk scan dan menyimpan mutasi.
func (h *MutationHandler) Input(c *gin.Context) {
	direction := normalizeDirection(c.Query("direction"))
	if direction == "" {
		c.Redirect(http.StatusFound, "/mutasi")
		return
	}
	u := middleware.CurrentUser(c)
	hist, ok := h.history(c, u, "", "", direction)
	if !ok {
		return
	}
	yesterday := clock.Now().AddDate(0, 0, -1).Format("2006-01-02")
	previous, ok := h.historyPage(c, u, "", yesterday, direction, 1, 10)
	if !ok {
		return
	}
	h.r.HTML(c, http.StatusOK, "mutasi.html", gin.H{
		"Title":           "Input Mutasi Barang",
		"Active":          "mutasi",
		"InputOnly":       true,
		"Form":            scanForm{Qty: 0, Direction: direction},
		"History":         hist,
		"PreviousHistory": previous,
		"WAURL": h.r.WhatsAppURL(fmt.Sprintf(
			"Halo Admin, saya %s (%s). Saya ingin menambahkan produk baru ke master data.",
			u.Name, outletName(u))),
	})
}

// GET /mutasi/history : riwayat mutasi PM berdasarkan bulan.
func (h *MutationHandler) HistoryPage(c *gin.Context) {
	u := middleware.CurrentUser(c)
	date := c.Query("date")
	now := clock.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, clock.WIB)
	if parsed, err := time.ParseInLocation("2006-01-02", date, clock.WIB); err == nil {
		start = parsed
	} else {
		date = start.Format("2006-01-02")
	}
	end := start.AddDate(0, 1, 0)
	direction := normalizeDirection(c.Query("direction"))
	q := strings.TrimSpace(c.Query("q"))
	rows, err := h.svc.Period(*u.OutletID, u.CategoryID, q, direction, start, end)
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat riwayat.")
		return
	}
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage != 10 && perPage != 30 && perPage != 50 {
		perPage = 10
	}
	view := paginateHistory(rows, page, perPage)
	view.Query, view.Direction = q, direction
	view.Date = date
	if page > 1 {
		view.PrevURL = historyPageURL(date, direction, q, page-1, perPage)
	}
	if page < view.Pages {
		view.NextURL = historyPageURL(date, direction, q, page+1, perPage)
	}
	if c.GetHeader("HX-Request") == "true" {
		h.r.Partial(c, http.StatusOK, "mutasi.html", "history_inner", gin.H{"History": view, "Target": "history-full"})
		return
	}
	h.r.HTML(c, http.StatusOK, "mutasi.html", gin.H{"Title": "Riwayat Mutasi", "Active": "mutasi", "HistoryPage": true, "History": view, "Date": date, "Direction": direction, "Query": q})
}

// GET /mutasi/lookup?barcode=... : isi otomatis kode & nama produk.
func (h *MutationHandler) Lookup(c *gin.Context) {
	u := middleware.CurrentUser(c)
	barcode := strings.TrimSpace(c.Query("barcode"))
	qty, _ := strconv.Atoi(strings.TrimSpace(c.Query("qty")))
	if qty < 0 {
		qty = 0
	}
	form := scanForm{Barcode: barcode, ProductName: strings.TrimSpace(c.Query("product_name")), Qty: qty, BA: c.Query("ba_name"), Direction: normalizeDirection(c.Query("direction"))}

	p, err := h.svc.FindProductForUser(barcode, u)
	if err != nil {
		form.Error, form.WAURL = h.explain(u, barcode, err)
		h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
		return
	}
	form.Product = p
	if n, qty := h.svc.DuplicateTodayByDirection(*u.OutletID, barcode, form.Direction); n > 0 {
		form.Warning = fmt.Sprintf(
			"Produk ini sudah diinput %d kali hari ini di outlet ini (total %d pcs). Pastikan bukan input ganda dari shift lain.",
			n, qty)
	}
	h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
}

// GET /mutasi/riwayat?q=... : pencarian riwayat hari ini.
func (h *MutationHandler) History(c *gin.Context) {
	u := middleware.CurrentUser(c)
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage != 10 && perPage != 30 && perPage != 50 {
		perPage = 10
	}
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	hist, ok := h.historyPage(c, u, strings.TrimSpace(c.Query("q")), strings.TrimSpace(c.Query("date")), normalizeDirection(c.Query("direction")), page, perPage)
	if !ok {
		return
	}
	target := "history-counter"
	if hist.Direction == "market_to_counter" {
		target = "history-market"
	}
	h.r.Partial(c, http.StatusOK, "mutasi.html", "history_inner", gin.H{"History": hist, "Target": target})
}

// POST /mutasi : simpan. Hanya barcode, qty, dan ba_name yang dibaca dari form;
// outlet dari akun login, kode & nama produk dari master data.
func (h *MutationHandler) Create(c *gin.Context) {
	u := middleware.CurrentUser(c)
	barcode := strings.TrimSpace(c.PostForm("barcode"))
	productName := strings.TrimSpace(c.PostForm("product_name"))
	direction := normalizeDirection(c.PostForm("direction"))
	qty, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("qty")))
	ba := c.PostForm("ba_name")
	product, lookupErr := h.svc.FindProductForUser(barcode, u)
	if qty == 0 {
		form := scanForm{Barcode: barcode, ProductName: productName, Qty: qty, BA: ba, Direction: direction, Error: "Jumlah mutasi tidak boleh 0."}
		if lookupErr == nil {
			form.Product = product
		}
		h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
		return
	}
	if errors.Is(lookupErr, service.ErrProductNotFound) && c.PostForm("request_new_product") != "1" {
		form := scanForm{Barcode: barcode, ProductName: productName, Qty: qty, BA: ba, Direction: direction, Error: service.ErrProductRequest.Error()}
		h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
		return
	}
	if errors.Is(lookupErr, service.ErrProductNotFound) && productName == "" {
		form := scanForm{Barcode: barcode, ProductName: productName, Qty: qty, BA: ba, Direction: direction, Error: "Nama produk wajib diisi sebelum mengajukan produk baru."}
		h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
		return
	}
	if errors.Is(lookupErr, service.ErrProductNotFound) {
		ba = strings.ToUpper(strings.TrimSpace(ba))
		req := &models.ProductRequest{Barcode: barcode, ProductName: productName, Qty: qty, Direction: direction, OutletID: *u.OutletID, UserID: u.ID, Status: "pending", RequestedAt: clock.Now()}
		if ba != "" {
			req.BAName = &ba
		}
		if qty < 1 || qty > service.MaxQty {
			form := scanForm{Barcode: barcode, ProductName: productName, Qty: qty, BA: ba, Direction: direction, Error: service.ErrQtyInvalid.Error()}
			h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
			return
		}
		if err := h.db.Create(req).Error; err != nil {
			form := scanForm{Barcode: barcode, ProductName: productName, Qty: qty, BA: ba, Direction: direction, Error: "Pengajuan produk gagal disimpan."}
			h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
			return
		}
		h.db.Create(&models.AuditLog{UserID: &u.ID, OutletID: u.OutletID, Action: "product_request_create", Detail: fmt.Sprintf("Produk: %s | Barcode: %s | Outlet: %s | Pengaju: %s | Jumlah: %d pcs | Arah: %s", req.ProductName, req.Barcode, outletName(u), u.Name, req.Qty, req.Direction), CreatedAt: clock.Now()})
		form := scanForm{Qty: 0, Direction: direction, Success: fmt.Sprintf("Pengajuan produk berhasil dikirim: %d pcs. Menunggu Super Admin melengkapi data.", qty)}
		h.r.Partial(c, http.StatusOK, "mutasi.html", "mutasi_saved", gin.H{"Form": form})
		return
	}

	m, err := h.svc.Add(u, barcode, productName, direction, qty, ba)
	if err != nil {
		form := scanForm{Barcode: barcode, Qty: qty, BA: ba, Direction: direction}
		if form.Qty < 1 {
			form.Qty = 1
		}
		form.Error, form.WAURL = h.explain(u, barcode, err)
		if p, e := h.svc.FindProductForUser(barcode, u); e == nil {
			form.Product = p
		}
		form.ProductName = productName
		h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
		return
	}
	direction = m.Direction

	// Sukses: form baru dengan input BA yang sudah dikosongkan.
	form := scanForm{
		Qty: 0, Direction: direction,
		Success: func() string {
			if m.Status == "pending_code" {
				return fmt.Sprintf("Pengajuan produk berhasil dikirim: %d pcs. Menunggu Admin melengkapi data produk.", m.Qty)
			}
			return fmt.Sprintf("Tersimpan: %d pcs %s", m.Qty, m.ProductName)
		}(),
	}
	hist, ok := h.history(nil, u, "", "", direction)
	if !ok {
		h.r.Partial(c, http.StatusOK, "mutasi.html", "scan_form", gin.H{"Form": form})
		return
	}
	h.r.Partial(c, http.StatusOK, "mutasi.html", "mutasi_saved", gin.H{"Form": form, "History": hist})
}

func (h *MutationHandler) UpdateQty(c *gin.Context) {
	u := middleware.CurrentUser(c)
	id, _ := strconv.ParseUint(c.PostForm("mutation_id"), 10, 64)
	qty, _ := strconv.ParseUint(c.PostForm("qty"), 10, 64)
	var before models.Mutation
	oldQty := -1
	if u != nil && u.OutletID != nil && h.db.Preload("Outlet").Where("id = ? AND outlet_id = ?", uint(id), *u.OutletID).First(&before).Error == nil {
		oldQty = before.Qty
	}
	if err := h.svc.UpdateQtyForPM(u, uint(id), uint(qty)); err != nil {
		c.String(http.StatusBadRequest, "Gagal menyimpan perubahan qty.")
		return
	}
	if oldQty >= 0 && oldQty != int(qty) {
		h.db.Create(&models.AuditLog{
			UserID: &u.ID, OutletID: u.OutletID, Action: "mutation_qty_update",
			Detail: fmt.Sprintf("Produk: %s | Barcode: %s | Outlet: %s | Dari %d pcs menjadi %d pcs | Arah: %s", before.ProductName, before.Barcode, before.Outlet.Name, oldQty, qty, before.Direction), CreatedAt: clock.Now(),
		})
	}
	hist, ok := h.history(c, u, "", "", normalizeDirection(c.PostForm("direction")))
	if !ok {
		return
	}
	c.Header("HX-Trigger", "mutation-edit-saved")
	h.r.Partial(c, http.StatusOK, "mutasi.html", "history_inner", gin.H{"History": hist, "Target": "history-input", "Actions": true})
}

func (h *MutationHandler) EditHistory(c *gin.Context) {
	u := middleware.CurrentUser(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rows, err := h.svc.EditHistory(u, uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Riwayat edit tidak ditemukan.")
		return
	}
	h.r.Partial(c, http.StatusOK, "mutasi.html", "mutation_edit_history", gin.H{"Rows": rows})
}

// history memuat riwayat hari ini. c boleh nil (tidak menulis respons error).
func (h *MutationHandler) history(c *gin.Context, u *models.User, q, date, direction string) (historyView, bool) {
	return h.historyPage(c, u, q, date, direction, 1, 10)
}

func (h *MutationHandler) historyPage(c *gin.Context, u *models.User, q, date, direction string, page, perPage int) (historyView, bool) {
	res, err := h.svc.ByDate(*u.OutletID, u.CategoryID, q, date)
	if err != nil {
		log.Printf("riwayat mutasi gagal: %v", err)
		if c != nil {
			c.String(http.StatusInternalServerError, "Gagal memuat riwayat.")
		}
		return historyView{}, false
	}
	rows := make([]models.Mutation, 0, len(res.Rows))
	for _, row := range res.Rows {
		if row.Direction == direction {
			rows = append(rows, row)
		}
	}
	// Pengajuan produk baru dicatat atas nama PM, lalu kategorinya dapat
	// dilengkapi admin saat approval. Jika kategori approval berbeda dari
	// kategori akun PM, tetap tampilkan mutasi milik PM tersebut berdasarkan
	// outlet, arah, dan tanggal pengajuan.
	hasOwnMutation := false
	for _, row := range rows {
		if row.UserID == u.ID {
			hasOwnMutation = true
			break
		}
	}
	if !hasOwnMutation && u.Role == models.RolePM && u.CategoryID != nil {
		all, fallbackErr := h.svc.ByDate(*u.OutletID, nil, q, date)
		if fallbackErr == nil {
			for _, row := range all.Rows {
				if row.Direction == direction && row.UserID == u.ID {
					rows = append(rows, row)
				}
			}
		}
	}
	totalQty := 0
	for _, row := range rows {
		totalQty += row.Qty
	}
	if perPage < 1 {
		perPage = 10
	}
	pages := (len(rows) + perPage - 1) / perPage
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * perPage
	end := start + perPage
	if start > len(rows) {
		start = len(rows)
	}
	if end > len(rows) {
		end = len(rows)
	}
	view := historyView{Rows: rows[start:end], TotalQty: totalQty, Total: len(rows), Page: page, Pages: pages, PerPage: perPage, Query: q, Date: res.Date, Direction: direction}
	if page > 1 {
		view.PrevURL = historyURL(direction, q, date, page-1, perPage)
	}
	if page < pages {
		view.NextURL = historyURL(direction, q, date, page+1, perPage)
	}
	return view, true
}

func historyURL(direction, q, date string, page, perPage int) string {
	v := url.Values{}
	v.Set("direction", direction)
	v.Set("q", q)
	if date != "" {
		v.Set("date", date)
	}
	v.Set("page", strconv.Itoa(page))
	v.Set("per_page", strconv.Itoa(perPage))
	return "/mutasi/riwayat?" + v.Encode()
}

func historyPageURL(date, direction, q string, page, perPage int) string {
	v := url.Values{"date": []string{date}, "direction": []string{direction}, "q": []string{q}, "page": []string{strconv.Itoa(page)}, "per_page": []string{strconv.Itoa(perPage)}}
	return "/mutasi/history?" + v.Encode()
}

func paginateHistory(rows []models.Mutation, page, perPage int) historyView {
	if perPage < 1 {
		perPage = 10
	}
	pages := (len(rows) + perPage - 1) / perPage
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * perPage
	if start > len(rows) {
		start = len(rows)
	}
	end := start + perPage
	if end > len(rows) {
		end = len(rows)
	}
	totalQty := 0
	for _, row := range rows {
		totalQty += row.Qty
	}
	return historyView{Rows: rows[start:end], Total: len(rows), TotalQty: totalQty, Page: page, Pages: pages, PerPage: perPage}
}

func normalizeDirection(direction string) string {
	switch direction {
	case "market_to_counter", "counter_to_market":
		return direction
	}
	return ""
}

// explain mengubah error service menjadi pesan untuk PM (+ link WA bila produk belum terdaftar).
func (h *MutationHandler) explain(u *models.User, barcode string, err error) (msg, waURL string) {
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		msg = fmt.Sprintf("Barkode %s belum terdaftar di master data.", barcode)
		waURL = h.r.WhatsAppURL(fmt.Sprintf(
			"Halo Admin, saya %s (%s). Mohon tambahkan produk baru ke master data.\nBarkode: %s",
			u.Name, outletName(u), barcode))
	case errors.Is(err, service.ErrBarcodeEmpty),
		errors.Is(err, service.ErrQtyInvalid),
		errors.Is(err, service.ErrBATooLong),
		errors.Is(err, service.ErrNoOutlet),
		errors.Is(err, service.ErrProductRequest):
		msg = err.Error()
	case strings.Contains(err.Error(), "Nama produk wajib"):
		msg = err.Error()
	default:
		log.Printf("mutasi gagal: %v", err)
		msg = "Terjadi kesalahan saat menyimpan. Coba lagi."
	}
	return msg, waURL
}

func outletName(u *models.User) string {
	if u.Outlet != nil {
		return u.Outlet.Name
	}
	return ""
}
