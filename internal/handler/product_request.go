package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"movara/internal/clock"
	"movara/internal/middleware"
	"movara/internal/models"
)

func (h *PageHandler) ProductRequests(c *gin.Context) {
	var requests []models.ProductRequest
	var categories []models.ProductCategory
	h.db.Preload("Outlet").Preload("User").Preload("Category").Where("status = ?", "pending").Order("requested_at asc").Find(&requests)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	h.r.HTML(c, http.StatusOK, "product_requests.html", gin.H{
		"Title": "Produk Baru", "Active": "produk-baru", "Requests": requests,
		"Categories": categories, "Message": c.Query("message"),
	})
}

func (h *PageHandler) ProductRequestNotification(c *gin.Context) {
	var pending int64
	h.db.Model(&models.ProductRequest{}).Where("status = ?", "pending").Count(&pending)
	c.JSON(http.StatusOK, gin.H{"pending": pending})
}

func (h *PageHandler) ApproveProductRequest(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	code := strings.TrimSpace(c.PostForm("code"))
	categoryID64, _ := strconv.ParseUint(c.PostForm("category_id"), 10, 64)
	categoryID := uint(categoryID64)
	if len(code) < 6 || categoryID == 0 {
		h.redirectProductRequest(c, "PLU%2FSKU+minimal+6+karakter+dan+kategori+wajib+diisi")
		return
	}
	var req models.ProductRequest
	if h.db.Preload("Outlet").Preload("User").First(&req, uint(id)).Error != nil || req.Status != "pending" {
		h.redirectProductRequest(c, "Pengajuan+produk+tidak+ditemukan+atau+sudah+diproses")
		return
	}
	now := clock.Now()
	admin := middleware.CurrentUser(c)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var duplicate models.Product
		if err := tx.Where("outlet_id = ? AND (barcode = ? OR code = ? OR name = ?)", req.OutletID, req.Barcode, code, req.ProductName).First(&duplicate).Error; err == nil {
			return gorm.ErrDuplicatedKey
		}
		product := models.Product{Barcode: req.Barcode, Code: code, Name: req.ProductName, IsActive: true, OutletID: req.OutletID, CategoryID: &categoryID}
		if err := tx.Create(&product).Error; err != nil {
			return err
		}
		mutation := models.Mutation{ProductID: &product.ID, Barcode: product.Barcode, Code: product.Code, ProductName: product.Name, CategoryID: product.CategoryID, Direction: req.Direction, Status: "approved", Qty: req.Qty, BAName: req.BAName, OutletID: req.OutletID, UserID: req.UserID, CreatedAt: req.RequestedAt}
		if err := tx.Create(&mutation).Error; err != nil {
			return err
		}
		return tx.Model(&req).Updates(map[string]interface{}{"code": code, "category_id": categoryID, "status": "approved", "reviewed_at": now, "reviewed_by": admin.ID}).Error
	})
	if err != nil {
		h.redirectProductRequest(c, "Produk+gagal+disetujui%3A+barcode%2C+PLU%2FSKU%2C+atau+nama+sudah+ada")
		return
	}
	h.audit(c, "product_request_approve", "Produk: "+req.ProductName+" | Barcode: "+req.Barcode+" | Outlet: "+req.Outlet.Name+" | Pengaju: "+req.User.Name+" | Jumlah: "+strconv.Itoa(req.Qty)+" pcs | PLU/SKU: "+code)
	h.redirectProductRequest(c, "Produk+berhasil+disetujui+dan+ditambahkan+ke+master+data+serta+jurnal+mutasi")
}

func (h *PageHandler) RejectProductRequest(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	admin := middleware.CurrentUser(c)
	var req models.ProductRequest
	h.db.Preload("Outlet").Preload("User").Select("id, product_name, barcode, outlet_id, user_id, qty, direction").First(&req, uint(id))
	result := h.db.Model(&models.ProductRequest{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]interface{}{"status": "rejected", "reviewed_at": clock.Now(), "reviewed_by": admin.ID})
	message := "Pengajuan+produk+berhasil+ditolak"
	if result.RowsAffected == 0 {
		message = "Pengajuan+produk+tidak+ditemukan+atau+sudah+diproses"
	} else {
		h.audit(c, "product_request_reject", "Produk: "+req.ProductName+" | Barcode: "+req.Barcode+" | Outlet: "+req.Outlet.Name+" | Pengaju: "+req.User.Name+" | Jumlah: "+strconv.Itoa(req.Qty)+" pcs | Arah: "+req.Direction)
	}
	h.redirectProductRequest(c, message)
}

func (h *PageHandler) redirectProductRequest(c *gin.Context, message string) {
	c.Redirect(http.StatusFound, "/kelola/produk-baru?message="+message)
}
