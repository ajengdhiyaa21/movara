package handler

import (
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
	var transfers []models.OutletTransferRequest
	var logs []models.AuditLog
	var categories []models.ProductCategory
	h.db.Order("id").Find(&outlets)
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
	userDB.Preload("Outlet").Preload("Category").Order("id").Offset((userPage - 1) * userPerPage).Limit(userPerPage).Find(&users)
	h.db.Where("role = ?", models.RoleManager).Order("id").Find(&managers)
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
	h.r.HTML(c, http.StatusOK, "kelola.html", gin.H{"Title": "Kelola Outlet & PM", "Active": "kelola", "Outlets": outlets, "Categories": categories, "Users": users, "Managers": managers, "Transfers": transfers, "Logs": logs, "UserPage": userPage, "UserPages": userPages, "UserPerPage": userPerPage, "UserTotal": userTotal, "PMQuery": pmQuery, "PMOutlet": pmOutlet, "AuditPage": page, "AuditPages": auditPages, "AuditPerPage": auditPerPage, "AuditPeriod": auditPeriod, "AuditDate": auditDate, "AuditQuery": c.Query("audit_q"), "AuditRole": c.Query("audit_role"), "AuditOutlet": c.Query("audit_outlet"), "RevealAccountID": revealAccountID, "Message": c.Query("message")})
}

func (h *PageHandler) KelolaOutlet(c *gin.Context) {
	var outlets []models.Outlet
	h.db.Order("id").Find(&outlets)
	h.r.HTML(c, http.StatusOK, "outlet_manage.html", gin.H{"Title": "Kelola Outlet", "Active": "outlet", "Outlets": outlets, "Message": c.Query("message")})
}

func (h *PageHandler) KelolaPM(c *gin.Context) {
	var users []models.User
	h.db.Where("role = ?", models.RolePM).Preload("Outlet").Preload("Category").Order("id").Find(&users)
	h.r.HTML(c, http.StatusOK, "pm_manage.html", gin.H{"Title": "Kelola Akun PM", "Active": "pm", "Users": users, "Message": c.Query("message")})
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
	var outlets []models.Outlet
	var categories []models.ProductCategory
	h.db.Where("is_active = ?", true).Order("id").Find(&outlets)
	h.db.Where("is_active = ?", true).Order("name").Find(&categories)
	h.r.HTML(c, http.StatusOK, "pm_create.html", gin.H{"Title": "Tambah Akun", "Active": "kelola", "Outlets": outlets, "Categories": categories})
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
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	if userCode == "" {
		userCode = username
	}
	u := models.User{Name: c.PostForm("name"), UserCode: userCode, Username: username, PasswordHash: string(hash), Role: role, IsActive: true, PasswordCiphertext: cipherText}
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
	outlet, _ := strconv.ParseUint(c.PostForm("outlet_id"), 10, 64)
	name, username := c.PostForm("name"), c.PostForm("username")
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	if userCode == "" {
		userCode = username
	}
	if name == "" || username == "" || outlet == 0 {
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
	if pass := c.PostForm("password"); pass != "" {
		if len(pass) < 8 {
			c.Redirect(302, "/kelola/pm/"+c.Param("id")+"/edit?message=Password+minimal+8+karakter")
			return
		}
		hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		u.PasswordHash = string(hash)
		u.PasswordCiphertext, _ = security.EncryptPassword(os.Getenv("SESSION_SECRET"), pass)
	}
	if err := h.db.Save(&u).Error; err != nil {
		c.Redirect(302, "/kelola/pm/"+c.Param("id")+"/edit?message=Username+sudah+digunakan")
		return
	}
	h.audit(c, "pm_update", u.Username)
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
	username := c.PostForm("username")
	if name == "" || username == "" {
		c.Redirect(http.StatusFound, "/profile?message=Nama+dan+username+wajib+diisi")
		return
	}
	var duplicate models.User
	if err := h.db.Where("username = ? AND id <> ?", username, u.ID).First(&duplicate).Error; err == nil {
		c.Redirect(http.StatusFound, "/profile?message=Username+sudah+digunakan")
		return
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{"name": name, "username": username}).Error; err != nil {
		c.Redirect(http.StatusFound, "/profile?message=Profil+gagal+diperbarui")
		return
	}
	c.Redirect(http.StatusFound, "/profile?message=Profil+berhasil+diperbarui")
}

func (h *PageHandler) AddManager(c *gin.Context) {
	name, username, password := strings.TrimSpace(c.PostForm("name")), strings.TrimSpace(c.PostForm("username")), c.PostForm("password")
	if name == "" || username == "" || len(password) < 8 {
		c.Redirect(http.StatusFound, "/kelola?message=Data+Manager+belum+lengkap+atau+password+minimal+8+karakter")
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
	userCode := strings.TrimSpace(c.PostForm("user_code"))
	if userCode == "" {
		userCode = username
	}
	u := models.User{Name: name, UserCode: userCode, Username: username, PasswordHash: string(hash), PasswordCiphertext: cipherText, Role: models.RoleManager, IsActive: true}
	if err = h.db.Create(&u).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message=Username+Manager+sudah+digunakan")
		return
	}
	h.audit(c, "manager_create", username)
	c.Redirect(http.StatusFound, "/kelola?message=Akun+Manager+berhasil+ditambahkan")
}

func (h *PageHandler) ResetManagerPassword(c *gin.Context) {
	h.resetAccountPassword(c, models.RoleManager, "manager_password_reset", "Manager")
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
	if len(password) < 8 {
		c.Redirect(http.StatusFound, "/kelola?message=Password+minimal+8+karakter")
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
		c.Redirect(http.StatusFound, "/kelola?message=Manager+tidak+ditemukan")
		return
	}
	u.IsActive = !u.IsActive
	if err := h.db.Model(&u).Update("is_active", u.IsActive).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message=Status+gagal+diubah")
		return
	}
	h.audit(c, "manager_status", u.Username)
	c.Redirect(http.StatusFound, "/kelola?message=Status+Manager+berhasil+diubah")
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
		c.Redirect(http.StatusFound, "/kelola?message=Manager+tidak+ditemukan")
		return
	}
	name, username := strings.TrimSpace(c.PostForm("name")), strings.TrimSpace(c.PostForm("username"))
	if name == "" || username == "" {
		c.Redirect(http.StatusFound, "/kelola?message=Nama+dan+username+wajib+diisi")
		return
	}
	u.Name, u.Username = name, username
	if err := h.db.Save(&u).Error; err != nil {
		c.Redirect(http.StatusFound, "/kelola?message=Username+Manager+sudah+digunakan")
		return
	}
	h.audit(c, "manager_update", u.Username)
	c.Redirect(http.StatusFound, "/kelola?message=Akun+Manager+berhasil+diperbarui")
}
