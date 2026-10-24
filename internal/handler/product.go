package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"movara/internal/middleware"
	"movara/internal/models"
)

type ProductHandler struct {
	db *gorm.DB
	r  *Renderer
}

func NewProductHandler(db *gorm.DB) *ProductHandler { return &ProductHandler{db: db} }
func (h *ProductHandler) SetRenderer(r *Renderer)   { h.r = r }
func (h *ProductHandler) audit(c *gin.Context, action, detail string) {
	if u := middleware.CurrentUser(c); u != nil {
		h.db.Create(&models.AuditLog{UserID: &u.ID, OutletID: u.OutletID, Action: action, Detail: detail})
	}
}

func (h *ProductHandler) resolvePending(c *gin.Context, p *models.Product) {
	if p == nil || p.OutletID == 0 {
		return
	}
	var pending []models.Mutation
	if h.db.Where("barcode = ? AND outlet_id = ? AND status = ?", p.Barcode, p.OutletID, "pending_code").Find(&pending).Error != nil {
		return
	}
	for i := range pending {
		if h.db.Model(&pending[i]).Updates(map[string]interface{}{"product_id": p.ID, "code": p.Code, "product_name": p.Name, "status": "approved_by_admin"}).Error == nil {
			h.audit(c, "mutation_approved", fmt.Sprintf("Pengajuan mutasi #%d disetujui melalui produk %s", pending[i].ID, p.Code))
		}
	}
}

func (h *ProductHandler) Index(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	var rows []models.Product
	var outlets []models.Outlet
	var categories []models.ProductCategory
	h.db.Where("is_active = ?", true).Order("id").Find(&outlets)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	db := h.db.Model(&models.Product{})
	if v := c.Query("outlet_id"); v != "" {
		db = db.Where("outlet_id = ?", v)
	}
	categoryID := c.Query("category_id")
	if categoryID != "" {
		db = db.Where("category_id = ?", categoryID)
	}
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("barcode LIKE ? OR code LIKE ? OR name LIKE ?", like, like, like)
	}
	var total int64
	db.Count(&total)
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage != 10 && perPage != 30 && perPage != 50 {
		perPage = 10
	}
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pages := int((total + int64(perPage) - 1) / int64(perPage))
	if pages > 0 && page > pages {
		page = pages
	}
	db.Preload("Outlet").Preload("Category").Order("id DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&rows)
	pageNumbers := make([]int, pages)
	for i := range pageNumbers {
		pageNumbers[i] = i + 1
	}
	base := url.Values{"q": []string{q}, "outlet_id": []string{c.Query("outlet_id")}, "category_id": []string{categoryID}, "per_page": []string{strconv.Itoa(perPage)}}
	pageURL := func(n int) string {
		v := url.Values{}
		for k, values := range base {
			v[k] = values
		}
		v.Set("page", strconv.Itoa(n))
		return "/master-data?" + v.Encode()
	}
	h.r.HTML(c, http.StatusOK, "master_clean.html", gin.H{"Title": "Master Data", "Active": "master", "Products": rows, "Total": total, "Query": q, "Outlets": outlets, "Categories": categories, "OutletID": c.Query("outlet_id"), "CategoryID": categoryID, "Message": c.Query("message"), "ProductPage": page, "ProductPages": pages, "ProductPerPage": perPage, "ProductPageNumbers": pageNumbers, "ProductPrevURL": pageURL(page - 1), "ProductNextURL": pageURL(page + 1), "ProductPageURL": pageURL})
}

func (h *ProductHandler) CreateForm(c *gin.Context) {
	var outlets []models.Outlet
	var categories []models.ProductCategory
	h.db.Where("is_active = ?", true).Order("id").Find(&outlets)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	h.r.HTML(c, http.StatusOK, "product_create.html", gin.H{"Title": "Tambah Produk", "Active": "master", "Outlets": outlets, "Categories": categories})
}

func (h *ProductHandler) ImportForm(c *gin.Context) {
	var outlets []models.Outlet
	var categories []models.ProductCategory
	h.db.Where("is_active = ?", true).Order("id").Find(&outlets)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	h.r.HTML(c, http.StatusOK, "product_import.html", gin.H{"Title": "Import Produk", "Active": "master", "Outlets": outlets, "Categories": categories})
}
func (h *ProductHandler) Create(c *gin.Context) {
	outlet, _ := strconv.ParseUint(c.PostForm("outlet_id"), 10, 64)
	category, _ := strconv.ParseUint(c.PostForm("category_id"), 10, 64)
	if outlet == 0 {
		c.Redirect(302, "/master-data?message=Pilih+outlet+terlebih+dahulu")
		return
	}
	p := models.Product{Barcode: strings.TrimSpace(c.PostForm("barcode")), Code: strings.TrimSpace(c.PostForm("code")), Name: strings.TrimSpace(c.PostForm("name")), IsActive: true, OutletID: uint(outlet)}
	if category > 0 {
		id := uint(category)
		p.CategoryID = &id
	}
	if p.Barcode == "" || p.Code == "" || p.Name == "" {
		c.Redirect(302, "/master-data?message=Semua+field+wajib+diisi")
		return
	}
	if err := h.db.Create(&p).Error; err != nil {
		c.Redirect(302, "/master-data?message=Barcode+atau+kode+sudah+terdaftar")
		return
	}
	h.audit(c, "product_create", p.Barcode)
	h.resolvePending(c, &p)
	c.Redirect(302, "/master-data?message=Produk+berhasil+ditambahkan")
}

func (h *ProductHandler) Template(c *gin.Context) {
	x := excelize.NewFile()
	sheet := x.GetSheetName(0)
	_ = x.SetSheetRow(sheet, "A1", &[]interface{}{"barcode", "plu_sku", "nama_produk", "kategori"})
	_ = x.SetCellStr(sheet, "A2", "8990000000000")
	_ = x.SetCellStr(sheet, "B2", "000001")
	_ = x.SetCellStr(sheet, "C2", "Contoh Nama Produk")
	_ = x.SetCellStr(sheet, "D2", "Kosmetik")
	// Barcode dan kode selalu diperlakukan sebagai teks agar nol di depan
	// seperti 00123 tidak dihapus oleh Excel saat user mengisi template.
	if textStyle, err := x.NewStyle(&excelize.Style{NumFmt: 49}); err == nil {
		_ = x.SetCellStyle(sheet, "A2", "A1000", textStyle)
		_ = x.SetCellStyle(sheet, "B2", "B1000", textStyle)
	}
	_ = x.SetColWidth(sheet, "A", "A", 20)
	_ = x.SetColWidth(sheet, "B", "B", 14)
	_ = x.SetColWidth(sheet, "C", "C", 32)
	_ = x.SetColWidth(sheet, "D", "D", 18)
	c.Header("Content-Disposition", `attachment; filename="template_produk_movara.xlsx"`)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	_ = x.Write(c.Writer)
}
func (h *ProductHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var p models.Product
	if h.db.First(&p, id).Error != nil {
		c.Redirect(302, "/master-data?message=Produk+tidak+ditemukan")
		return
	}
	p.Barcode = strings.TrimSpace(c.PostForm("barcode"))
	p.Code = strings.TrimSpace(c.PostForm("code"))
	p.Name = strings.TrimSpace(c.PostForm("name"))
	if category, _ := strconv.ParseUint(c.PostForm("category_id"), 10, 64); category > 0 {
		id := uint(category)
		p.CategoryID = &id
	}
	outlet, _ := strconv.ParseUint(c.PostForm("outlet_id"), 10, 64)
	if outlet > 0 {
		p.OutletID = uint(outlet)
	}
	if p.Barcode == "" || p.Code == "" || p.Name == "" {
		c.Redirect(302, "/master-data?message=Semua+field+wajib+diisi")
		return
	}
	h.db.Save(&p)
	h.audit(c, "product_update", p.Barcode)
	h.resolvePending(c, &p)
	c.Redirect(302, "/master-data?message=Produk+berhasil+diubah")
}

func (h *ProductHandler) Edit(c *gin.Context) {
	var p models.Product
	if h.db.Preload("Category").First(&p, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/master-data?message=Produk+tidak+ditemukan")
		return
	}
	var outlets []models.Outlet
	var categories []models.ProductCategory
	h.db.Where("is_active = ?", true).Order("id").Find(&outlets)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	h.r.HTML(c, http.StatusOK, "product_edit.html", gin.H{"Title": "Edit Produk", "Product": p, "Outlets": outlets, "Categories": categories})
}
func (h *ProductHandler) Deactivate(c *gin.Context) {
	var p models.Product
	if h.db.First(&p, c.Param("id")).Error != nil {
		c.Redirect(302, "/master-data?message=Produk+tidak+ditemukan")
		return
	}
	p.IsActive = !p.IsActive
	h.db.Model(&p).Update("is_active", p.IsActive)
	action := "product_deactivate"
	message := "Produk+dinonaktifkan"
	if p.IsActive {
		action = "product_activate"
		message = "Produk+diaktifkan+kembali"
	}
	h.audit(c, action, c.Param("id"))
	c.Redirect(302, "/master-data?message="+message)
}
func (h *ProductHandler) Import(c *gin.Context) {
	outlet, _ := strconv.ParseUint(c.PostForm("outlet_id"), 10, 64)
	if outlet == 0 {
		c.Redirect(302, "/master-data?message=Pilih+outlet+untuk+import")
		return
	}
	f, e := c.FormFile("file")
	if e != nil {
		c.Redirect(302, "/master-data?message=File+tidak+valid")
		return
	}
	if !strings.EqualFold(filepath.Ext(f.Filename), ".xlsx") {
		c.Redirect(302, "/master-data?message=File+harus+berformat+.xlsx")
		return
	}
	src, e := f.Open()
	if e != nil {
		c.Redirect(302, "/master-data?message=File+tidak+bisa+dibuka")
		return
	}
	defer src.Close()
	x, err := excelize.OpenReader(src)
	if err != nil {
		c.Redirect(302, "/master-data?message=Excel+tidak+valid")
		return
	}
	defer x.Close()
	rows, err := x.GetRows(x.GetSheetName(0))
	if err != nil {
		c.Redirect(302, "/master-data?message=Excel+tidak+bisa+dibaca")
		return
	}
	n := 0
	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}
		p := models.Product{Barcode: strings.TrimSpace(row[0]), Code: strings.TrimSpace(row[1]), Name: strings.TrimSpace(row[2]), IsActive: true, OutletID: uint(outlet)}
		var categoryID uint
		categoryValue := ""
		if len(row) >= 4 && strings.TrimSpace(row[3]) != "" {
			var rowCategory models.ProductCategory
			categoryValue = strings.TrimSpace(row[3])
			lookup := h.db.Where("is_active = ?", true)
			if parsed, parseErr := strconv.ParseUint(categoryValue, 10, 64); parseErr == nil && parsed > 0 {
				lookup = lookup.Where("id = ?", parsed)
			} else {
				lookup = lookup.Where("LOWER(name) = LOWER(?)", categoryValue)
			}
			if lookup.First(&rowCategory).Error == nil {
				categoryID = rowCategory.ID
			}
		}
		if p.Barcode == "" || p.Code == "" || p.Name == "" || categoryID == 0 {
			continue
		}
		p.CategoryID = &categoryID
		if err = h.db.Where("barcode = ? AND outlet_id = ?", p.Barcode, outlet).Assign(map[string]interface{}{"code": p.Code, "name": p.Name, "category_id": categoryID, "is_active": true, "outlet_id": outlet}).FirstOrCreate(&p).Error; err == nil {
			n++
			h.resolvePending(c, &p)
		}
	}
	c.Redirect(302, "/master-data?message="+url.QueryEscape(fmt.Sprintf("%d produk berhasil diimport", n)))
}

// Lookup dipanggil saat scanner menekan Enter di kolom barkode.
// GET /products/lookup?barcode=8992222075220
func (h *ProductHandler) Lookup(c *gin.Context) {
	barcode := strings.TrimSpace(c.Query("barcode"))
	if barcode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"found": false, "message": "barkode kosong"})
		return
	}

	var p models.Product
	u := middleware.CurrentUser(c)
	var outlet uint
	if u != nil && u.OutletID != nil {
		outlet = *u.OutletID
	}
	err := h.db.Where("barcode = ? AND outlet_id = ? AND is_active = TRUE", barcode, outlet).First(&p).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"found": false, "message": "Produk belum terdaftar. Hubungi admin via WhatsApp."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"found": true, "code": p.Code, "name": p.Name})
}
