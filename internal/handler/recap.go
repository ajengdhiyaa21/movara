package handler

import (
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
	"movara/internal/export"
	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/service"
)

const dateLayout = "2006-01-02"

// recapForm: nilai filter yang ditampilkan kembali di form.
type recapForm struct {
	Q            string
	OutletID     uint // 0 = semua outlet
	CategoryID   uint // 0 = semua kategori
	Direction    string
	From, To     string
	PerPage      int
	AllData      bool
	ActivePreset string
	ScopeLabel   string
}

func (f recapForm) values() url.Values {
	v := url.Values{}
	if f.Q != "" {
		v.Set("q", f.Q)
	}
	if f.OutletID != 0 {
		v.Set("outlet", strconv.FormatUint(uint64(f.OutletID), 10))
	}
	if f.CategoryID != 0 {
		v.Set("category", strconv.FormatUint(uint64(f.CategoryID), 10))
	}
	if f.Direction != "" {
		v.Set("direction", f.Direction)
	}
	// from/to selalu disertakan (boleh kosong = semua periode),
	// supaya pagination tidak kembali ke default "hari ini".
	v.Set("from", f.From)
	v.Set("to", f.To)
	if f.PerPage > 0 {
		v.Set("per_page", strconv.Itoa(f.PerPage))
	}
	if f.AllData {
		v.Set("per_page", "all")
	}
	return v
}

type recapRow struct {
	No int
	models.Mutation
}

type summaryView struct {
	MonthQty  int
	HasGrowth bool
	GrowthUp  bool
	Growth    string
	TodayQty  int
	HasTop    bool
	TopName   string
	TopQty    int
	TopShare  string
}

type barRow struct {
	Label string
	Qty   int
	Pct   int
	Color string
}

type pageItem struct {
	Label   string
	URL     string
	Current bool
	Gap     bool
}

type RecapHandler struct {
	r     *Renderer
	svc   *service.RecapService
	audit *service.AuditService
	db    *gorm.DB
}

func NewRecapHandler(r *Renderer, svc *service.RecapService, audit *service.AuditService, db *gorm.DB) *RecapHandler {
	return &RecapHandler{r: r, svc: svc, audit: audit, db: db}
}

func (h *RecapHandler) Edit(c *gin.Context) {
	var m models.Mutation
	if h.db.Preload("Outlet").Preload("User").First(&m, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/rekap?message=Mutasi+tidak+ditemukan")
		return
	}
	h.r.HTML(c, http.StatusOK, "mutation_edit.html", gin.H{"Title": "Edit Mutasi", "Mutation": m})
}

func (h *RecapHandler) Update(c *gin.Context) {
	var m models.Mutation
	if h.db.First(&m, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/rekap?message=Mutasi+tidak+ditemukan")
		return
	}
	qty, err := strconv.Atoi(strings.TrimSpace(c.PostForm("qty")))
	name := strings.TrimSpace(c.PostForm("product_name"))
	ba := strings.TrimSpace(c.PostForm("ba_name"))
	direction := c.PostForm("direction")
	if direction != "market_to_counter" && direction != "counter_to_market" {
		c.Redirect(http.StatusFound, "/rekap?message=Arah+mutasi+tidak+valid")
		return
	}
	if qty < 1 || qty > 999 || name == "" {
		c.Redirect(http.StatusFound, "/rekap?message=Data+mutasi+tidak+valid")
		return
	}
	updates := map[string]interface{}{"qty": qty, "product_name": name, "direction": direction, "ba_name": nil}
	if ba != "" {
		updates["ba_name"] = strings.ToUpper(ba)
	}
	if err = h.db.Model(&m).Updates(updates).Error; err != nil {
		c.Redirect(http.StatusFound, "/rekap?message=Mutasi+gagal+diubah")
		return
	}
	if u := middleware.CurrentUser(c); u != nil {
		h.audit.Log(&u.ID, u.OutletID, "mutation_update", fmt.Sprintf("Edit mutasi #%d", m.ID))
	}
	c.Redirect(http.StatusFound, "/rekap?message=Mutasi+berhasil+diubah")
}

func (h *RecapHandler) Delete(c *gin.Context) {
	var m models.Mutation
	if h.db.First(&m, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/rekap?message=Mutasi+tidak+ditemukan")
		return
	}
	if err := h.db.Delete(&m).Error; err != nil {
		c.Redirect(http.StatusFound, "/rekap?message=Mutasi+gagal+dihapus")
		return
	}
	if u := middleware.CurrentUser(c); u != nil {
		h.audit.Log(&u.ID, u.OutletID, "mutation_delete", fmt.Sprintf("Hapus mutasi #%d", m.ID))
	}
	c.Redirect(http.StatusFound, "/rekap?message=Mutasi+berhasil+dihapus")
}

// GET /rekap : Manager dan Admin.
func (h *RecapHandler) Index(c *gin.Context) {
	q, form := readRecapFilter(c)

	outlets, err := h.svc.Outlets()
	if err != nil {
		h.fail(c, err)
		return
	}
	var categories []models.ProductCategory
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	res, err := h.svc.Search(q)
	if err != nil {
		h.fail(c, err)
		return
	}
	sum, err := h.svc.Summary(q)
	if err != nil {
		h.fail(c, err)
		return
	}

	form.ScopeLabel = "Semua outlet"
	for _, o := range outlets {
		if o.ID == form.OutletID {
			form.ScopeLabel = o.Name
		}
	}

	offset := (res.Page - 1) * res.PerPage
	rows := make([]recapRow, len(res.Rows))
	for i, m := range res.Rows {
		rows[i] = recapRow{No: offset + i + 1, Mutation: m}
	}
	from, to := 0, 0
	if len(rows) > 0 {
		from, to = offset+1, offset+len(rows)
	}

	base := form.values()
	items, prev, next := buildPages(res.Page, res.Pages, base)

	h.r.HTML(c, http.StatusOK, "rekap.html", gin.H{
		"Title":      "Rekap & Laporan",
		"Active":     "rekap",
		"Form":       form,
		"Outlets":    outlets,
		"Categories": categories,
		"Summary":    newSummaryView(sum),
		"Rows":       rows,
		"Total":      res.Total,
		"TotalQty":   res.TotalQty,
		"ShowFrom":   from,
		"ShowTo":     to,
		"PageItems":  items,
		"PrevURL":    prev,
		"NextURL":    next,
		"Bars":       newBars(res),
		"ExportURL":  "/rekap/export?" + base.Encode(),
	})
}

// GET /rekap/export : hanya Admin. Memakai filter yang sama dengan halaman rekap.
func (h *RecapHandler) Export(c *gin.Context) {
	q, _ := readRecapFilter(c)
	rows, err := h.svc.ExportRows(q)
	if err != nil {
		h.fail(c, err)
		return
	}
	buf, err := export.RecapXLSX(rows, q.From, q.To)
	if err != nil {
		h.fail(c, err)
		return
	}
	if u := middleware.CurrentUser(c); u != nil {
		h.audit.Log(&u.ID, nil, "export_excel", fmt.Sprintf("Ekspor rekapitulasi mutasi (%d baris, .xlsx)", len(rows)))
	}
	name := "rekap-mutasi-" + clock.Now().Format("20060102-1504") + ".xlsx"
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// Print menampilkan seluruh jurnal sesuai periode yang dipilih, tanpa pagination.
func (h *RecapHandler) Print(c *gin.Context) {
	q, form := readRecapFilter(c)
	if q.From.IsZero() || q.To.IsZero() {
		c.String(http.StatusBadRequest, "Periode mulai dan sampai wajib dipilih sebelum mencetak.")
		return
	}
	rows, err := h.svc.ExportRows(q)
	if err != nil {
		h.fail(c, err)
		return
	}
	h.r.HTML(c, http.StatusOK, "rekap_print.html", gin.H{
		"Title":    "Cetak Rekapitulasi Mutasi",
		"Rows":     rows,
		"Form":     form,
		"Total":    len(rows),
		"TotalQty": totalMutationQty(rows),
	})
}

func totalMutationQty(rows []models.Mutation) int {
	total := 0
	for _, row := range rows {
		total += row.Qty
	}
	return total
}

func (h *RecapHandler) fail(c *gin.Context, err error) {
	log.Printf("rekap gagal: %v", err)
	c.String(http.StatusInternalServerError, "Gagal memuat data rekap.")
}

// readRecapFilter membaca query string. Tanpa parameter periode, defaultnya hari ini.
// from/to kosong tetapi ada di URL = semua periode.
func readRecapFilter(c *gin.Context) (service.RecapQuery, recapForm) {
	dayStart, dayEnd := clock.DayBounds(clock.Now())
	monthStart := time.Date(dayStart.Year(), dayStart.Month(), 1, 0, 0, 0, 0, clock.WIB)

	from, to := dayStart, dayEnd
	switch c.Query("preset") {
	case "today":
	case "7d":
		from = dayStart.AddDate(0, 0, -6)
	case "30d":
		from = dayStart.AddDate(0, 0, -29)
	case "month":
		from = monthStart
	case "all":
		from, to = time.Time{}, time.Time{}
	default:
		_, fok := c.GetQuery("from")
		_, tok := c.GetQuery("to")
		if fok || tok {
			from, to = parseDate(c.Query("from")), parseDate(c.Query("to"))
			if !to.IsZero() {
				to = to.AddDate(0, 0, 1)
			}
		}
	}
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		from, to = to, from
	}

	q := service.RecapQuery{OutletID: middleware.OutletScope(c), From: from, Q: strings.TrimSpace(c.Query("q"))}
	if direction := c.Query("direction"); direction == "counter_to_market" || direction == "market_to_counter" {
		q.Direction = direction
	}
	if category, parseErr := strconv.ParseUint(c.Query("category"), 10, 64); parseErr == nil && category > 0 {
		id := uint(category)
		q.CategoryID = &id
	}
	if !to.IsZero() {
		q.To = to // sudah eksklusif: awal hari berikutnya
	}
	q.Page, _ = strconv.Atoi(c.Query("page"))
	perPage := c.Query("per_page")
	q.All = perPage == "all"
	if !q.All {
		q.PerPage, _ = strconv.Atoi(perPage)
	}

	form := recapForm{Q: q.Q, PerPage: q.PerPage, AllData: q.All, Direction: q.Direction}
	if q.CategoryID != nil {
		form.CategoryID = *q.CategoryID
	}
	if q.OutletID != nil {
		form.OutletID = *q.OutletID
	}
	if !from.IsZero() {
		form.From = from.Format(dateLayout)
	}
	if !to.IsZero() {
		form.To = to.AddDate(0, 0, -1).Format(dateLayout)
	}
	switch {
	case from.IsZero() && to.IsZero():
		form.ActivePreset = "all"
	case to.Equal(dayEnd) && from.Equal(dayStart):
		form.ActivePreset = "today"
	case to.Equal(dayEnd) && from.Equal(dayStart.AddDate(0, 0, -6)):
		form.ActivePreset = "7d"
	case to.Equal(dayEnd) && from.Equal(dayStart.AddDate(0, 0, -29)):
		form.ActivePreset = "30d"
	case to.Equal(dayEnd) && from.Equal(monthStart):
		form.ActivePreset = "month"
	}
	return q, form
}

func parseDate(s string) time.Time {
	t, err := time.ParseInLocation(dateLayout, strings.TrimSpace(s), clock.WIB)
	if err != nil {
		return time.Time{}
	}
	return t
}

func newSummaryView(s *service.RecapSummary) summaryView {
	v := summaryView{MonthQty: s.MonthQty, TodayQty: s.TodayQty}
	if s.HasGrowth {
		v.HasGrowth = true
		v.GrowthUp = s.Growth >= 0
		v.Growth = strings.Replace(fmt.Sprintf("%+.1f%%", s.Growth), ".", ",", 1)
	}
	if s.TopOutlet != nil {
		v.HasTop = true
		v.TopName = shortOutlet(s.TopOutlet.Code)
		v.TopQty = s.TopOutlet.Count
		v.TopShare = strings.Replace(fmt.Sprintf("%.1f%%", s.TopShare), ".", ",", 1)
	}
	return v
}

func newBars(res *service.RecapResult) []barRow {
	max := 0
	for _, b := range res.ByOutlet {
		if b.Count > max {
			max = b.Count
		}
	}
	colors := []string{"bg-blue-600", "bg-sky-500", "bg-slate-400"}
	var bars []barRow
	for i, b := range res.ByOutlet {
		if i >= 5 {
			break
		}
		ci := i
		if ci > 2 {
			ci = 2
		}
		pct := 0
		if max > 0 {
			pct = b.Count * 100 / max
		}
		bars = append(bars, barRow{Label: shortOutlet(b.Code), Qty: b.Count, Pct: pct, Color: colors[ci]})
	}
	return bars
}

// buildPages membuat nomor halaman ringkas: 1 … 4 5 6 … 15.
func buildPages(cur, total int, base url.Values) (items []pageItem, prev, next string) {
	link := func(p int) string {
		v := url.Values{}
		for k, vals := range base {
			v[k] = append([]string(nil), vals...)
		}
		v.Set("page", strconv.Itoa(p))
		return "/rekap?" + v.Encode()
	}
	show := map[int]bool{1: true, total: true, cur: true, cur - 1: true, cur + 1: true}
	last := 0
	for p := 1; p <= total; p++ {
		if !show[p] {
			continue
		}
		if last != 0 && p-last > 1 {
			items = append(items, pageItem{Gap: true, Label: "…"})
		}
		items = append(items, pageItem{Label: strconv.Itoa(p), URL: link(p), Current: p == cur})
		last = p
	}
	if cur > 1 {
		prev = link(cur - 1)
	}
	if cur < total {
		next = link(cur + 1)
	}
	return items, prev, next
}
