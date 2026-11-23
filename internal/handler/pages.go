package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"movara/internal/clock"
	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/security"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// PageHandler: halaman yang masih berupa kerangka (diisi bertahap).
type PageHandler struct {
	r  *Renderer
	db *gorm.DB
}

func NewPageHandler(r *Renderer, db *gorm.DB) *PageHandler { return &PageHandler{r: r, db: db} }

// ForceLogout memutus sesi akun target. Endpoint ini dilindungi middleware super admin.
func (h *PageHandler) ForceLogout(c *gin.Context) {
	operator := middleware.CurrentUser(c)
	if operator == nil || !middleware.IsSuperAdmin(operator) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses hanya untuk super admin"})
		return
	}
	var target models.User
	if err := h.db.First(&target, c.Param("id")).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Akun tidak ditemukan"))
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", target.ID).Updates(map[string]interface{}{"active_session_token": "", "active_session_until": nil}).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Gagal mengeluarkan akun dari sistem"))
		return
	}
	h.db.Create(&models.AuditLog{UserID: &operator.ID, OutletID: operator.OutletID, Action: "force_logout", Detail: fmt.Sprintf("Super admin mengeluarkan akun %s (%s) dari sistem", target.Name, target.Username), CreatedAt: clock.Now()})
	c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Akun "+target.Name+" berhasil dikeluarkan dari sistem"))
}

func (h *PageHandler) CreateMaintenanceAnnouncement(c *gin.Context) {
	operator := middleware.CurrentUser(c)
	if operator == nil || operator.Username != "admin.c" {
		c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Fitur pengumuman hanya tersedia untuk admin.c"))
		return
	}
	title := strings.TrimSpace(c.PostForm("title"))
	message := strings.TrimSpace(c.PostForm("message"))
	start, startErr := time.ParseInLocation("2006-01-02T15:04", c.PostForm("starts_at"), clock.WIB)
	if title == "" || message == "" || startErr != nil {
		c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Judul, pesan, dan waktu mulai wajib diisi"))
		return
	}
	var end *time.Time
	if raw := strings.TrimSpace(c.PostForm("ends_at")); raw != "" {
		parsed, err := time.ParseInLocation("2006-01-02T15:04", raw, clock.WIB)
		if err != nil || !parsed.After(start) {
			c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Waktu selesai harus lebih besar dari waktu mulai"))
			return
		}
		end = &parsed
	}
	announcement := models.MaintenanceAnnouncement{Title: title, Message: message, StartsAt: start, EndsAt: end, IsActive: true, CreatedBy: operator.ID}
	if err := h.db.Create(&announcement).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Pengumuman gagal disimpan"))
		return
	}
	h.audit(c, "maintenance_create", title)
	c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Pengumuman maintenance berhasil dibuat"))
}

func (h *PageHandler) ToggleMaintenanceAnnouncement(c *gin.Context) {
	operator := middleware.CurrentUser(c)
	if operator == nil || operator.Username != "admin.c" {
		c.Redirect(http.StatusFound, "/kelola")
		return
	}
	var announcement models.MaintenanceAnnouncement
	if h.db.First(&announcement, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Pengumuman tidak ditemukan"))
		return
	}
	announcement.IsActive = !announcement.IsActive
	h.db.Model(&announcement).Update("is_active", announcement.IsActive)
	c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Status pengumuman diperbarui"))
}

// TODO: tabel produk + search + pagination + import Excel/CSV.
func (h *PageHandler) Master(c *gin.Context) {
	h.r.HTML(c, http.StatusOK, "master.html", gin.H{"Title": "Master Data", "Active": "master"})
}

func (h *PageHandler) Categories(c *gin.Context) {
	var categories []models.ProductCategory
	h.db.Order("name asc").Find(&categories)
	h.r.HTML(c, http.StatusOK, "categories.html", gin.H{
		"Title": "Kategori Produk", "Active": "master", "Categories": categories,
		"Message": c.Query("message"),
	})
}

func (h *PageHandler) AddCategory(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Nama+kategori+wajib+diisi")
		return
	}
	slug := strings.ToLower(strings.Join(strings.Fields(name), "-"))
	category := models.ProductCategory{Name: name, Slug: slug, IsActive: true}
	if err := h.db.Create(&category).Error; err != nil {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Kategori+sudah+ada+atau+tidak+valid")
		return
	}
	h.audit(c, "category_create", category.Name)
	c.Redirect(http.StatusFound, "/master-data/categories?message=Kategori+berhasil+ditambahkan")
}

func (h *PageHandler) UpdateCategory(c *gin.Context) {
	var category models.ProductCategory
	if h.db.First(&category, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Kategori+tidak+ditemukan")
		return
	}
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Nama+kategori+wajib+diisi")
		return
	}
	var duplicate models.ProductCategory
	if err := h.db.Where("lower(name) = lower(?) AND id <> ?", name, category.ID).First(&duplicate).Error; err == nil {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Nama+kategori+sudah+digunakan")
		return
	}
	slug := strings.ToLower(strings.Join(strings.Fields(name), "-"))
	if err := h.db.Model(&category).Updates(map[string]interface{}{"name": name, "slug": slug}).Error; err != nil {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Kategori+gagal+diperbarui")
		return
	}
	h.audit(c, "category_update", category.Name+" -> "+name)
	c.Redirect(http.StatusFound, "/master-data/categories?message=Kategori+berhasil+diperbarui")
}

func (h *PageHandler) ToggleCategory(c *gin.Context) {
	var category models.ProductCategory
	if h.db.First(&category, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Kategori+tidak+ditemukan")
		return
	}
	category.IsActive = !category.IsActive
	if err := h.db.Model(&category).Update("is_active", category.IsActive).Error; err != nil {
		c.Redirect(http.StatusFound, "/master-data/categories?message=Status+kategori+gagal+diubah")
		return
	}
	action := "category_deactivate"
	message := "Kategori+dinonaktifkan"
	if category.IsActive {
		action = "category_activate"
		message = "Kategori+diaktifkan+kembali"
	}
	h.audit(c, action, category.Name)
	c.Redirect(http.StatusFound, "/master-data/categories?message="+message)
}

// TODO: tab Outlet, Akun PM, Audit Log, persetujuan pindah outlet.
func (h *PageHandler) Kelola(c *gin.Context) {
	/* Halaman administrasi terpadu: akun, outlet, dan Aktivitas Aplikasi. */
	var outlets []models.Outlet
	var users []models.User
	var managers []models.User
	var superAdmins []models.User
	var transfers []models.OutletTransferRequest
	var logs []models.AuditLog
	var categories []models.ProductCategory
	var announcements []models.MaintenanceAnnouncement
	h.db.Order("CASE WHEN type = 'MK' THEN 0 ELSE 1 END ASC").Order("id ASC").Find(&outlets)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	userPage, _ := strconv.Atoi(c.Query("user_page"))
	if userPage < 1 {
		userPage = 1
	}
	userPerPage, _ := strconv.Atoi(c.Query("user_per_page"))
	if userPerPage != 10 && userPerPage != 30 && userPerPage != 50 {
		userPerPage = 10
	}
	var userTotal int64
	userDB := h.db.Model(&models.User{}).Where("role = ?", models.RolePM)
	pmQuery := strings.TrimSpace(c.Query("pm_q"))
	pmOutlet := strings.TrimSpace(c.Query("pm_outlet"))
	if pmQuery != "" {
		like := "%" + pmQuery + "%"
		userDB = userDB.Where("name LIKE ? OR username LIKE ?", like, like)
	}
	if pmOutlet != "" {
		userDB = userDB.Where("outlet_id = ?", pmOutlet)
	}
	userDB.Count(&userTotal)
	userPages := int((userTotal + int64(userPerPage) - 1) / int64(userPerPage))
	if userPages > 0 && userPage > userPages {
		userPage = userPages
	}
	userDB.Preload("Outlet").Preload("Category").Order("id DESC").Offset((userPage - 1) * userPerPage).Limit(userPerPage).Find(&users)
	managerPage, _ := strconv.Atoi(c.Query("manager_page"))
	if managerPage < 1 {
		managerPage = 1
	}
	managerPerPage, _ := strconv.Atoi(c.Query("manager_per_page"))
	if managerPerPage != 10 && managerPerPage != 30 && managerPerPage != 50 {
		managerPerPage = 10
	}
	managerDB := h.db.Model(&models.User{}).Where("role IN ?", []models.Role{models.RoleManager, models.RoleAdmin, models.RoleSuperAdmin})
	managerQuery := strings.TrimSpace(c.Query("manager_q"))
	if managerQuery != "" {
		like := "%" + managerQuery + "%"
		managerDB = managerDB.Where("name LIKE ? OR username LIKE ? OR user_code LIKE ?", like, like, like)
	}
	var managerTotal int64
	managerDB.Count(&managerTotal)
	managerPages := int((managerTotal + int64(managerPerPage) - 1) / int64(managerPerPage))
	if managerPages > 0 && managerPage > managerPages {
		managerPage = managerPages
	}
	managerDB.Preload("Outlet").Preload("Outlets", func(db *gorm.DB) *gorm.DB {
		return db.Order("CASE WHEN type = 'MK' THEN 0 ELSE 1 END ASC").Order("id ASC")
	}).Order("id DESC").Offset((managerPage - 1) * managerPerPage).Limit(managerPerPage).Find(&managers)
	h.db.Where("role = ?", models.RoleSuperAdmin).Order("id ASC").Find(&superAdmins)
	h.db.Order("starts_at desc").Limit(20).Find(&announcements)
	h.db.Where("status = ?", models.TransferPending).Order("requested_at desc").Find(&transfers)
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	var total int64
	auditDate := c.Query("audit_date")
	oneMonthAgo := clock.Now().AddDate(0, -1, 0)
	auditQuery := h.db.Model(&models.AuditLog{}).Where("created_at >= ?", oneMonthAgo)
	if q := strings.TrimSpace(c.Query("audit_q")); q != "" {
		like := "%" + q + "%"
		auditQuery = auditQuery.Where("detail LIKE ? OR action LIKE ?", like, like)
	}
	if role := strings.TrimSpace(c.Query("audit_role")); role != "" {
		auditQuery = auditQuery.Joins("JOIN users audit_users ON audit_users.id = audit_logs.user_id").Where("audit_users.role = ?", role)
	}
	if outlet := strings.TrimSpace(c.Query("audit_outlet")); outlet != "" {
		auditQuery = auditQuery.Where("audit_logs.outlet_id = ?", outlet)
	}
	if selected, err := time.ParseInLocation("2006-01-02", auditDate, clock.WIB); err == nil {
		nextDay := selected.AddDate(0, 0, 1)
		auditQuery = h.db.Model(&models.AuditLog{}).Where("created_at >= ? AND created_at < ?", selected, nextDay)
	}
	auditPerPage, _ := strconv.Atoi(c.Query("audit_per_page"))
	if auditPerPage != 10 && auditPerPage != 30 && auditPerPage != 50 {
		auditPerPage = 10
	}
	auditQuery.Count(&total)
	auditPages := (int(total) + auditPerPage - 1) / auditPerPage
	auditQuery.Preload("User").Preload("User.Outlet").Order("audit_logs.created_at desc").Offset((page - 1) * auditPerPage).Limit(auditPerPage).Find(&logs)
	auditPeriod := "1 bulan terakhir"
	if _, err := time.ParseInLocation("2006-01-02", auditDate, clock.WIB); err == nil {
		auditPeriod = "tanggal " + auditDate
	}
	revealAccountID, _ := strconv.ParseUint(c.Query("reveal_id"), 10, 64)
	userURL := func(p int) string {
		v := url.Values{"user_page": {strconv.Itoa(p)}, "user_per_page": {strconv.Itoa(userPerPage)}, "pm_q": {pmQuery}, "pm_outlet": {pmOutlet}}
		return "/kelola?" + v.Encode() + "#users"
	}
	auditURL := func(p int) string {
		v := url.Values{"page": {strconv.Itoa(p)}, "audit_per_page": {strconv.Itoa(auditPerPage)}, "audit_date": {auditDate}, "audit_q": {c.Query("audit_q")}, "audit_role": {c.Query("audit_role")}, "audit_outlet": {c.Query("audit_outlet")}}
		return "/kelola?" + v.Encode() + "#audit-log"
	}
	managerURL := func(p int) string {
		v := url.Values{"manager_page": {strconv.Itoa(p)}, "manager_per_page": {strconv.Itoa(managerPerPage)}, "manager_q": {managerQuery}}
		return "/kelola?" + v.Encode() + "#manager"
	}
	h.r.HTML(c, http.StatusOK, "kelola.html", gin.H{"Title": "Kelola Outlet & PM", "Active": "kelola", "Outlets": outlets, "Categories": categories, "Users": users, "Managers": managers, "ManagerPage": managerPage, "ManagerPages": managerPages, "ManagerPageItems": buildProductPageItems(managerPage, managerPages, managerURL), "ManagerPerPage": managerPerPage, "ManagerTotal": managerTotal, "Transfers": transfers, "Logs": logs, "Announcements": announcements, "UserPage": userPage, "UserPages": userPages, "UserPageItems": buildProductPageItems(userPage, userPages, userURL), "UserPerPage": userPerPage, "UserTotal": userTotal, "PMQuery": pmQuery, "PMOutlet": pmOutlet, "AuditPage": page, "AuditPages": auditPages, "AuditPageItems": buildProductPageItems(page, auditPages, auditURL), "AuditPerPage": auditPerPage, "AuditPeriod": auditPeriod, "AuditDate": auditDate, "AuditQuery": c.Query("audit_q"), "AuditRole": c.Query("audit_role"), "AuditOutlet": c.Query("audit_outlet"), "RevealAccountID": revealAccountID, "Message": c.Query("message")})
}

func (h *PageHandler) KelolaOutlet(c *gin.Context) {
	var outlets []models.Outlet
	h.db.Order("CASE WHEN type = 'MK' THEN 0 ELSE 1 END ASC").Order("id ASC").Find(&outlets)
	h.r.HTML(c, http.StatusOK, "outlet_manage.html", gin.H{"Title": "Kelola Outlet", "Active": "outlet", "Outlets": outlets, "Message": c.Query("message")})
}

func (h *PageHandler) KelolaPM(c *gin.Context) {
	redirect := "/kelola"
	if message := c.Query("message"); message != "" {
		redirect += "?message=" + url.QueryEscape(message)
	}
	c.Redirect(http.StatusFound, redirect)
}

func (h *PageHandler) EditOutlet(c *gin.Context) {
	var outlet models.Outlet
	if h.db.First(&outlet, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/kelola/outlet?message=Outlet+tidak+ditemukan")
		return
	}
	h.r.HTML(c, http.StatusOK, "outlet_edit.html", gin.H{"Title": "Edit Outlet", "Active": "outlet", "Outlet": outlet, "Message": c.Query("message")})
}

func (h *PageHandler) UpdateOutlet(c *gin.Context) {
	var outlet models.Outlet
	if h.db.First(&outlet, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/kelola/outlet?message=Outlet+tidak+ditemukan")
		return
	}
	code := strings.TrimSpace(c.PostForm("code"))
	name := strings.TrimSpace(c.PostForm("name"))
	typ := strings.TrimSpace(c.PostForm("type"))
	if code == "" || name == "" || (typ != "MK" && typ != "MINI") {
		c.Redirect(http.StatusFound, "/kelola/outlet/"+c.Param("id")+"/edit?message=Data+outlet+belum+lengkap")
		return
	}
	outlet.Code, outlet.Name, outlet.Type = code, name, typ
	if err := h.db.Save(&outlet).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola/outlet/"+c.Param("id")+"/edit?message=Kode+outlet+sudah+digunakan")
		return
	}
	h.audit(c, "outlet_update", outlet.Code+" - "+outlet.Name)
	c.Redirect(http.StatusFound, "/kelola/outlet?message=Outlet+berhasil+diperbarui")
}

func (h *PageHandler) ToggleOutlet(c *gin.Context) {
	var outlet models.Outlet
	if h.db.First(&outlet, c.Param("id")).Error != nil {
		c.Redirect(http.StatusFound, "/kelola/outlet?message=Outlet+tidak+ditemukan")
		return
	}
	outlet.IsActive = !outlet.IsActive
	if err := h.db.Model(&outlet).Update("is_active", outlet.IsActive).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola/outlet?message=Status+outlet+gagal+diubah")
		return
	}
	action, message := "outlet_deactivate", "Outlet+dinonaktifkan"
	if outlet.IsActive {
		action, message = "outlet_activate", "Outlet+diaktifkan+kembali"
	}
	h.audit(c, action, outlet.Code+" - "+outlet.Name)
	c.Redirect(http.StatusFound, "/kelola/outlet?message="+message)
}

func (h *PageHandler) AddOutletForm(c *gin.Context) {
	h.r.HTML(c, http.StatusOK, "outlet_create.html", gin.H{"Title": "Tambah Outlet", "Active": "outlet"})
}

func (h *PageHandler) AddPMForm(c *gin.Context) {
	c.Redirect(http.StatusFound, "/kelola")
}

func (h *PageHandler) audit(c *gin.Context, action, detail string) {
	u := middleware.CurrentUser(c)
	if u == nil {
		return
	}
	h.db.Create(&models.AuditLog{UserID: &u.ID, OutletID: u.OutletID, Action: action, Detail: detail, CreatedAt: clock.Now()})
}

func (h *PageHandler) AddOutlet(c *gin.Context) {
	o := models.Outlet{Code: c.PostForm("code"), Name: c.PostForm("name"), Type: c.PostForm("type"), IsActive: true}
	if o.Code != "" && o.Name != "" {
		if err := h.db.Create(&o).Error; err != nil {
			c.Redirect(302, "/kelola/outlet?message=Outlet+gagal+disimpan")
			return
		}
		h.audit(c, "outlet_create", o.Code+" - "+o.Name)
	}
	c.Redirect(302, "/kelola/outlet?message=Outlet+diproses")
}
func (h *PageHandler) AddPM(c *gin.Context) {
	outlet, _ := strconv.ParseUint(c.PostForm("outlet_id"), 10, 64)
	role := models.Role(strings.TrimSpace(c.PostForm("role")))
	if role != models.RoleManager {
		role = models.RolePM
	}
	pass := c.PostForm("password")
	if pass == "" {
		pass = "admin123!"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	cipherText, _ := security.EncryptPassword(os.Getenv("SESSION_SECRET"), pass)
	username := strings.TrimSpace(c.PostForm("username"))
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		name = username
	}
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	if userCode == "" {
		userCode = username
	}
	u := models.User{Name: name, UserCode: userCode, Username: username, PasswordHash: string(hash), Role: role, IsActive: true, PasswordCiphertext: cipherText}
	if role == models.RolePM {
		if outlet == 0 {
			c.Redirect(302, "/kelola/pm?message=Outlet+wajib+untuk+pramuniaga")
			return
		}
		x := uint(outlet)
		u.OutletID = &x
		if category, _ := strconv.ParseUint(c.PostForm("category_id"), 10, 64); category > 0 {
			id := uint(category)
			u.CategoryID = &id
		}
	}
	if u.Name == "" || u.Username == "" || (role == models.RolePM && u.CategoryID == nil) {
		c.Redirect(302, "/kelola/pm?message=Data+akun+belum+lengkap")
		return
	}
	if err := h.db.Create(&u).Error; err != nil {
		c.Redirect(302, "/kelola/pm?message=Username+sudah+digunakan")
		return
	}
	h.audit(c, "user_create", u.Username)
	c.Redirect(302, "/kelola?message=Akun+berhasil+ditambahkan")
}

func (h *PageHandler) EditPM(c *gin.Context) {
	var u models.User
	if h.db.Preload("Outlet").Preload("Category").Where("id = ? AND role = ?", c.Param("id"), models.RolePM).First(&u).Error != nil {
		c.Redirect(302, "/kelola/pm?message=Akun+PM+tidak+ditemukan")
		return
	}
	var outlets []models.Outlet
	var categories []models.ProductCategory
	h.db.Where("is_active = ?", true).Order("id").Find(&outlets)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	var pmOutlet uint
	if u.OutletID != nil {
		pmOutlet = *u.OutletID
	}
	h.r.HTML(c, http.StatusOK, "pm_edit.html", gin.H{"Title": "Edit Akun PM", "UserPM": u, "PMOutletID": pmOutlet, "Outlets": outlets, "Categories": categories, "Message": c.Query("message"), "Active": "pm"})
}

func (h *PageHandler) UpdatePM(c *gin.Context) {
	var u models.User
	if h.db.Where("id = ? AND role = ?", c.Param("id"), models.RolePM).First(&u).Error != nil {
		c.Redirect(302, "/kelola/pm?message=Akun+PM+tidak+ditemukan")
		return
	}
	oldUserCode, oldUsername := u.UserCode, u.Username
	outlet, _ := strconv.ParseUint(c.PostForm("outlet_id"), 10, 64)
	name, username := c.PostForm("name"), c.PostForm("username")
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	if userCode == "" {
		userCode = username
	}
	if name == "" {
		name = username
	}
	if username == "" || outlet == 0 {
		c.Redirect(302, "/kelola/pm/"+c.Param("id")+"/edit?message=Data+PM+belum+lengkap")
		return
	}
	u.Name, u.UserCode, u.Username = name, userCode, username
	u.OutletID = func() *uint { v := uint(outlet); return &v }()
	if category, _ := strconv.ParseUint(c.PostForm("category_id"), 10, 64); category > 0 {
		id := uint(category)
		u.CategoryID = &id
	}
	if u.CategoryID == nil {
		c.Redirect(302, "/kelola/pm/"+c.Param("id")+"/edit?message=Kategori+PM+wajib+dipilih")
		return
	}
	if err := h.db.Save(&u).Error; err != nil {
		c.Redirect(302, "/kelola/pm/"+c.Param("id")+"/edit?message=Username+sudah+digunakan")
		return
	}
	h.audit(c, "pm_update", fmt.Sprintf("ID Pengguna: %s -> %s | Username: %s -> %s", oldUserCode, u.UserCode, oldUsername, u.Username))
	c.Redirect(302, "/kelola/pm?message=Akun+PM+berhasil+diperbarui")
}
func (h *PageHandler) DecideTransfer(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	u := middleware.CurrentUser(c)
	status := models.TransferStatus(c.PostForm("status"))
	if status != models.TransferApproved && status != models.TransferRejected {
		c.Redirect(302, "/kelola")
		return
	}
	var tr models.OutletTransferRequest
	if h.db.First(&tr, id).Error == nil && tr.Status == models.TransferPending {
		now := clock.Now()
		tr.Status = status
		tr.DecidedAt = &now
		tr.DecidedBy = &u.ID
		if err := h.db.Save(&tr).Error; err != nil {
			c.Redirect(302, "/kelola/pm?message=Pengajuan+gagal+diproses")
			return
		}
		h.audit(c, "transfer_"+string(status), "user "+strconv.FormatUint(uint64(tr.UserID), 10))
		if status == models.TransferApproved {
			if err := h.db.Model(&models.User{}).Where("id = ?", tr.UserID).Update("outlet_id", tr.ToOutletID).Error; err != nil {
				c.Redirect(302, "/kelola/pm?message=Outlet+PM+gagal+diperbarui")
				return
			}
		}
	}
	c.Redirect(302, "/kelola/pm?message=Pengajuan+diproses")
}
func (h *PageHandler) RequestTransfer(c *gin.Context) {
	u := middleware.CurrentUser(c)
	to, _ := strconv.ParseUint(c.PostForm("to_outlet"), 10, 64)
	if u != nil && u.OutletID != nil && to > 0 && uint(to) != *u.OutletID {
		h.db.Create(&models.OutletTransferRequest{UserID: u.ID, FromOutletID: *u.OutletID, ToOutletID: uint(to), Status: models.TransferPending, RequestedAt: clock.Now()})
	}
	c.Redirect(302, "/mutasi?message=Pengajuan+pindah+outlet+terkirim")
}
func (h *PageHandler) TransferForm(c *gin.Context) {
	var outlets []models.Outlet
	h.db.Where("is_active = ?", true).Order("id").Find(&outlets)
	h.r.HTML(c, http.StatusOK, "transfer.html", gin.H{"Title": "Pengajuan Pindah Outlet", "Active": "mutasi", "Outlets": outlets})
}
func (h *PageHandler) DeactivatePM(c *gin.Context) {
	var u models.User
	if h.db.Where("id = ? AND role = ?", c.Param("id"), models.RolePM).First(&u).Error != nil {
		c.Redirect(302, "/kelola/pm?message=Akun+PM+tidak+ditemukan")
		return
	}
	u.IsActive = !u.IsActive
	if err := h.db.Model(&u).Update("is_active", u.IsActive).Error; err != nil {
		c.Redirect(302, "/kelola/pm?message=Status+akun+gagal+diubah")
		return
	}
	action := "pm_deactivate"
	message := "Akun+PM+dinonaktifkan"
	if u.IsActive {
		action = "pm_activate"
		message = "Akun+PM+diaktifkan+kembali"
	}
	h.audit(c, action, c.Param("id"))
	c.Redirect(302, "/kelola/pm?message="+message)
}
func (h *PageHandler) PasswordForm(c *gin.Context) {
	h.r.HTML(c, http.StatusOK, "password.html", gin.H{"Title": "Ganti Password", "Message": c.Query("message")})
}
func (h *PageHandler) ChangePassword(c *gin.Context) {
	u := middleware.CurrentUser(c)
	old := c.PostForm("old_password")
	next := c.PostForm("password")
	if u == nil || len(next) < 8 || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(old)) != nil {
		c.Redirect(302, "/password?message=Password+lama+salah+atau+password+baru+minimal+8+karakter")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err := h.db.Model(&models.User{}).Where("id = ?", u.ID).Update("password_hash", string(hash)).Error; err != nil {
		c.Redirect(302, "/password?message=Password+gagal+diubah")
		return
	}
	c.Redirect(302, "/password?message=Password+berhasil+diubah")
}

func (h *PageHandler) ProfileForm(c *gin.Context) {
	u := middleware.CurrentUser(c)
	h.r.HTML(c, http.StatusOK, "profile.html", gin.H{"Title": "Edit Profil", "UserProfile": u, "Message": c.Query("message")})
}

func (h *PageHandler) UpdateProfile(c *gin.Context) {
	u := middleware.CurrentUser(c)
	if u == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	name := c.PostForm("name")
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	username := c.PostForm("username")
	if name == "" {
		name = username
	}
	if userCode == "" || username == "" {
		c.Redirect(http.StatusFound, "/profile?message=Nama+dan+username+wajib+diisi")
		return
	}
	var duplicate models.User
	if err := h.db.Where("username = ? AND id <> ?", username, u.ID).First(&duplicate).Error; err == nil {
		c.Redirect(http.StatusFound, "/profile?message=Username+sudah+digunakan")
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{"user_code": userCode, "name": name, "username": username}).Error; err != nil {
		c.Redirect(http.StatusFound, "/profile?message=Profil+gagal+diperbarui")
		return
	}
	h.audit(c, "profile_update", fmt.Sprintf("ID Pengguna: %s -> %s | Username: %s -> %s", u.UserCode, userCode, u.Username, username))
	c.Redirect(http.StatusFound, "/profile?message=Profil+berhasil+diperbarui")
}

func (h *PageHandler) AddManager(c *gin.Context) {
	name, username, password := strings.TrimSpace(c.PostForm("name")), strings.TrimSpace(c.PostForm("username")), c.PostForm("password")
	if name == "" || username == "" || password == "" {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Data+Manajemen+belum+lengkap+atau+password+wajib+diisi")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Password+gagal+diproses")
		return
	}
	cipherText, err := security.EncryptPassword(os.Getenv("SESSION_SECRET"), password)
	if err != nil {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Password+gagal+dienkripsi")
		return
	}
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	if userCode == "" {
		userCode = username
	}
	outletIDs := managerOutletIDs(c)
	if len(outletIDs) == 0 {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Outlet+Manajemen+wajib+dipilih")
		return
	}
	var outlets []models.Outlet
	if h.db.Where("id IN ? AND is_active = ?", outletIDs, true).Find(&outlets).Error != nil || len(outlets) != len(outletIDs) {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Outlet+Manajemen+tidak+ditemukan")
		return
	}
	outletValue := outletIDs[0]
	u := models.User{Name: name, UserCode: userCode, Username: username, PasswordHash: string(hash), PasswordCiphertext: cipherText, Role: models.RoleManager, OutletID: &outletValue, Outlets: outlets, IsActive: true}
	if err = h.db.Create(&u).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Username+Manajemen+sudah+digunakan")
		return
	}
	h.audit(c, "manager_create", username)
	c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Akun+Manajemen+berhasil+ditambahkan")
}

func (h *PageHandler) ResetManagerPassword(c *gin.Context) {
	h.resetAccountPassword(c, models.RoleManager, "manager_password_reset", "Manajemen")
}

func (h *PageHandler) ResetPMPassword(c *gin.Context) {
	h.resetAccountPassword(c, models.RolePM, "pm_password_reset", "Pramuniaga")
}

func (h *PageHandler) resetAccountPassword(c *gin.Context, role models.Role, auditAction, accountLabel string) {
	var u models.User
	if h.db.Where("id = ? AND role = ?", c.Param("id"), role).First(&u).Error != nil {
		c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape(accountLabel+" tidak ditemukan"))
		return
	}
	password := c.PostForm("password")
	if password == "" {
		c.Redirect(http.StatusFound, "/kelola?message=Password+wajib+diisi")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.Redirect(http.StatusFound, "/kelola?message=Password+gagal+diproses")
		return
	}
	cipherText, err := security.EncryptPassword(os.Getenv("SESSION_SECRET"), password)
	if err != nil {
		c.Redirect(http.StatusFound, "/kelola?message=Password+gagal+dienkripsi")
		return
	}
	if err = h.db.Model(&u).Updates(map[string]interface{}{"password_hash": string(hash), "password_ciphertext": cipherText}).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message=Password+gagal+diubah")
		return
	}
	h.audit(c, auditAction, u.Username)
	c.Redirect(http.StatusFound, "/kelola?message="+url.QueryEscape("Password "+accountLabel+" berhasil direset")+"&reveal_id="+strconv.FormatUint(uint64(u.ID), 10))
}

func (h *PageHandler) ToggleManager(c *gin.Context) {
	var u models.User
	if h.db.Where("id = ? AND role = ?", c.Param("id"), models.RoleManager).First(&u).Error != nil {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Manajemen+tidak+ditemukan")
		return
	}
	u.IsActive = !u.IsActive
	if err := h.db.Model(&u).Update("is_active", u.IsActive).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message=Status+gagal+diubah")
		return
	}
	h.audit(c, "manager_status", u.Username)
	c.Redirect(http.StatusFound, "/kelola?message=Status+Manajemen+berhasil+diubah")
}

func (h *PageHandler) RevealLatestPassword(c *gin.Context) {
	var u models.User
	if h.db.Where("id = ? AND role IN ?", c.Param("id"), []models.Role{models.RolePM, models.RoleManager}).First(&u).Error != nil || u.PasswordCiphertext == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Password terbaru belum tersedia"})
		return
	}
	password, err := security.DecryptPassword(os.Getenv("SESSION_SECRET"), u.PasswordCiphertext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Password tidak dapat dibuka"})
		return
	}
	h.audit(c, "password_reveal", u.Username)
	c.JSON(http.StatusOK, gin.H{"password": password})
}

func (h *PageHandler) UpdateManager(c *gin.Context) {
	var u models.User
	if h.db.Where("id = ? AND role = ?", c.Param("id"), models.RoleManager).First(&u).Error != nil {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Manajemen+tidak+ditemukan")
		return
	}
	oldUserCode, oldUsername := u.UserCode, u.Username
	name, username := strings.TrimSpace(c.PostForm("name")), strings.TrimSpace(c.PostForm("username"))
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	if name == "" {
		name = username
	}
	if userCode == "" || username == "" {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Nama+dan+username+wajib+diisi")
		return
	}
	outletIDs := managerOutletIDs(c)
	if len(outletIDs) == 0 {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Outlet+Manajemen+wajib+dipilih")
		return
	}
	var outlets []models.Outlet
	if h.db.Where("id IN ? AND is_active = ?", outletIDs, true).Find(&outlets).Error != nil || len(outlets) != len(outletIDs) {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Outlet+Manajemen+tidak+ditemukan")
		return
	}
	u.Name, u.UserCode, u.Username = name, userCode, username
	outletValue := outletIDs[0]
	u.OutletID = &outletValue
	u.Outlets = outlets
	if err := h.db.Save(&u).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Username+Manajemen+sudah+digunakan")
		return
	}
	h.audit(c, "manager_update", fmt.Sprintf("ID Pengguna: %s -> %s | Username: %s -> %s", oldUserCode, u.UserCode, oldUsername, u.Username))
	c.Redirect(http.StatusFound, "/kelola?tab=manager&message=Akun+Manajemen+berhasil+diperbarui")
}

func managerOutletIDs(c *gin.Context) []uint {
	raw := c.PostFormArray("outlet_ids")
	if len(raw) == 0 && c.PostForm("outlet_id") != "" {
		raw = []string{c.PostForm("outlet_id")}
	}
	seen := map[uint]bool{}
	ids := make([]uint, 0, len(raw))
	for _, value := range raw {
		id, err := strconv.ParseUint(value, 10, 64)
		if err == nil && id > 0 && !seen[uint(id)] {
			seen[uint(id)] = true
			ids = append(ids, uint(id))
		}
	}
	return ids
}
