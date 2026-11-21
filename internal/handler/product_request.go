package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"movara/internal/clock"
	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/service"
)

func (h *PageHandler) ProductRequests(c *gin.Context) {
	var requests []models.ProductRequest
	var categories []models.ProductCategory
	h.db.Preload("Outlet").Preload("User").Preload("User.Category").Preload("Category").Where("status = ?", "pending").Order("requested_at desc").Find(&requests)
	for i := range requests {
		if requests[i].Category == nil && requests[i].User.Category != nil {
			requests[i].Category = requests[i].User.Category
			categoryID := requests[i].User.Category.ID
			requests[i].CategoryID = &categoryID
		}
	}
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	h.r.HTML(c, http.StatusOK, "product_requests.html", gin.H{
		"Title": "Produk Baru", "Active": "produk-baru", "Requests": requests,
		"Categories": categories, "Message": c.Query("message"), "MessageTone": c.DefaultQuery("tone", "blue"),
	})
}

func (h *PageHandler) ProductRequestNotification(c *gin.Context) {
	var pending int64
	h.db.Model(&models.ProductRequest{}).Where("status = ?", "pending").Count(&pending)
	c.JSON(http.StatusOK, gin.H{"pending": pending})
}

func (h *PageHandler) ApproveProductRequest(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req models.ProductRequest
	if h.db.Preload("Outlet").Preload("User").First(&req, uint(id)).Error != nil || req.Status != "pending" {
		h.redirectProductRequest(c, "Pengajuan+produk+tidak+ditemukan+atau+sudah+diproses", "blue")
		return
	}
	productName := strings.TrimSpace(c.PostForm("product_name"))
	barcode := strings.TrimSpace(c.PostForm("barcode"))
	code := strings.TrimSpace(c.PostForm("code"))
	qty, _ := strconv.Atoi(strings.TrimSpace(c.PostForm("qty")))
	categoryID64, _ := strconv.ParseUint(c.PostForm("category_id"), 10, 64)
	categoryID := uint(categoryID64)
	if productName == "" || barcode == "" || len(barcode) > 20 || len(code) < 6 || qty < 1 || qty > service.MaxQty || categoryID == 0 {
		h.redirectProductRequest(c, "Nama%2C+barcode%2C+qty%2C+PLU%2FSKU%2C+dan+kategori+wajib+valid", "blue")
		return
	}
	now := clock.Now()
	admin := middleware.CurrentUser(c)
	usedExistingProduct := false
	var existingProduct models.Product
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// Jika salah satu identitas sudah ada di master, gunakan produk tersebut
		// untuk jurnal mutasi. Jangan membuat master product kedua.
		lookup := tx.Where("outlet_id = ? AND (barcode = ? OR code = ? OR name = ?)", req.OutletID, barcode, code, productName).First(&existingProduct)
		if lookup.Error == nil {
			usedExistingProduct = true
		} else if !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
			return lookup.Error
		}
		product := existingProduct
		if !usedExistingProduct {
			product = models.Product{Barcode: barcode, Code: code, Name: productName, IsActive: true, OutletID: req.OutletID, CategoryID: &categoryID}
			if err := tx.Create(&product).Error; err != nil {
				return err
			}
		}
		if req.MutationID != nil {
			// Pengajuan lama bisa memiliki status jurnal selain pending_code.
			// Selama jurnalnya masih terkait pengajuan ini, selesaikan jurnal tersebut.
			mutationUpdate := tx.Model(&models.Mutation{}).Where("id = ?", *req.MutationID).Updates(map[string]interface{}{"product_id": product.ID, "barcode": product.Barcode, "code": product.Code, "product_name": product.Name, "category_id": product.CategoryID, "status": "approved", "qty": qty})
			if mutationUpdate.Error != nil {
				return mutationUpdate.Error
			}
			if mutationUpdate.RowsAffected == 0 {
				// Fallback untuk data lama yang memiliki MutationID tidak valid.
				mutation := models.Mutation{ProductID: &product.ID, Barcode: product.Barcode, Code: product.Code, ProductName: product.Name, CategoryID: product.CategoryID, Direction: req.Direction, Status: "approved", Qty: qty, BAName: req.BAName, OutletID: req.OutletID, UserID: req.UserID, CreatedAt: req.RequestedAt}
				if err := tx.Create(&mutation).Error; err != nil {
					return err
				}
			}
		} else {
			mutation := models.Mutation{ProductID: &product.ID, Barcode: product.Barcode, Code: product.Code, ProductName: product.Name, CategoryID: product.CategoryID, Direction: req.Direction, Status: "approved", Qty: qty, BAName: req.BAName, OutletID: req.OutletID, UserID: req.UserID, CreatedAt: req.RequestedAt}
			if err := tx.Create(&mutation).Error; err != nil {
				return err
			}
		}
		return tx.Model(&req).Updates(map[string]interface{}{"product_name": productName, "barcode": barcode, "code": code, "category_id": categoryID, "qty": qty, "status": "approved", "reviewed_at": now, "reviewed_by": admin.ID}).Error
	})
	if err != nil {
		h.redirectProductRequest(c, "Produk+gagal+disetujui%3A+"+url.QueryEscape(err.Error()), "blue")
		return
	}
	h.audit(c, "product_request_approve", "Produk: "+productName+" | Barcode: "+barcode+" | Outlet: "+req.Outlet.Name+" | Pengaju: "+req.User.Name+" | Jumlah: "+strconv.Itoa(qty)+" pcs | PLU/SKU: "+code)
	if usedExistingProduct {
		h.redirectProductRequest(c, "Produk+disetujui.+Data+sudah+ada+di+master+dan+dicatat+ke+jurnal+mutasi+tanpa+duplikasi+master.", "orange")
		return
	}
	h.redirectProductRequest(c, "Produk+berhasil+disetujui+dan+ditambahkan+ke+master+data+serta+jurnal+mutasi", "blue")
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
	h.redirectProductRequest(c, message, "blue")
}

func (h *PageHandler) redirectProductRequest(c *gin.Context, message, tone string) {
	c.Redirect(http.StatusFound, "/kelola/produk-baru?message="+message+"&tone="+tone)
}
