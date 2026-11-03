package router

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"movara/internal/chat"
	"movara/internal/config"
	"movara/internal/handler"
	"movara/internal/middleware"
	"movara/internal/models"
	"movara/internal/repository"
	"movara/internal/service"
	"movara/web"
)

func New(cfg *config.Config, repos *repository.Repos, db *gorm.DB) (*gin.Engine, error) {
	renderer, err := handler.NewRenderer(cfg.AdminWhatsApp)
	if err != nil {
		return nil, err
	}
	engine, _, err := NewWithRendererAndHub(cfg, repos, db, renderer, chat.NewDefaultHub())
	return engine, err
}

// NewWithHub returns the engine and its hub for graceful server shutdown.
func NewWithHub(cfg *config.Config, repos *repository.Repos, db *gorm.DB) (*gin.Engine, *chat.Hub, error) {
	renderer, err := handler.NewRenderer(cfg.AdminWhatsApp)
	if err != nil {
		return nil, nil, err
	}
	return NewWithRendererAndHub(cfg, repos, db, renderer, chat.NewDefaultHub())
}

// NewWithRenderer keeps route construction testable without parsing the full
// production template set.
func NewWithRenderer(cfg *config.Config, repos *repository.Repos, db *gorm.DB, renderer *handler.Renderer) (*gin.Engine, error) {
	engine, _, err := NewWithRendererAndHub(cfg, repos, db, renderer, chat.NewDefaultHub())
	return engine, err
}

// NewWithRendererAndHub is used by the server lifecycle and tests so the hub
// can be shut down together with the HTTP server.
func NewWithRendererAndHub(cfg *config.Config, repos *repository.Repos, db *gorm.DB, renderer *handler.Renderer, chatHub *chat.Hub) (*gin.Engine, *chat.Hub, error) {
	if renderer == nil {
		return nil, nil, fmt.Errorf("renderer wajib tersedia")
	}
	if chatHub == nil {
		return nil, nil, fmt.Errorf("chat hub wajib tersedia")
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.SameOrigin())

	// Session cookie (SameSite=Lax membantu mengurangi risiko CSRF).
	store := cookie.NewStore([]byte(cfg.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   8 * 60 * 60,
		HttpOnly: true,
		Secure:   cfg.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("movara", store))
	r.Use(middleware.LoadUser(repos.Users))

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		return nil, nil, err
	}
	r.StaticFS("/static", http.FS(staticFS))

	renderer.Demo = false

	audit := service.NewAuditService(repos.Audits)
	authSvc := service.NewAuthService(repos.Users)
	mutSvc := service.NewMutationService(repos.Products, repos.Mutations, repos.MutationEdits, audit)
	recapSvc := service.NewRecapService(repos.Mutations, repos.Outlets)
	chatSvc := service.NewAuthorizedChatService(repos.Chat, repos.Users, chat.NewHubNotifier(chatHub, repos.Users))

	authH := handler.NewAuthHandler(renderer, authSvc, audit)
	pageH := handler.NewPageHandler(renderer, db)
	productH := handler.NewProductHandler(db)
	productH.SetRenderer(renderer)
	mutH := handler.NewMutationHandler(renderer, mutSvc)
	recapH := handler.NewRecapHandler(renderer, recapSvc, audit, db)
	chatH := handler.NewChatHandler(chatSvc)
	chatWSH := handler.NewChatWebSocketHandler(chatHub, cfg.Port)

	// Publik
	r.GET("/login", authH.ShowLogin)
	r.POST("/login", authH.Login)

	// Wajib login
	authed := r.Group("/", middleware.RequireAuth())
	{
		authed.GET("/", handler.Home)
		authed.POST("/logout", authH.Logout)
		chat := authed.Group("/chat", middleware.RequireRole(models.RoleAdmin, models.RolePM))
		{
			chat.GET("/page", pageH.ChatPage)
			chat.GET("/ws", chatWSH.Serve)
			chat.GET("", chatH.Index)
			chat.POST("/messages", middleware.RequireCSRF(), chatH.SendMessage)
			chat.POST("/read", middleware.RequireCSRF(), chatH.MarkRead)
		}

		// PM: input mutasi di outlet sendiri
		pm := authed.Group("/", middleware.RequireRole(models.RolePM), middleware.RequireOutlet(repos.Outlets))
		{
			pm.GET("/mutasi", mutH.Index)
			pm.GET("/mutasi/input", mutH.Input)
			pm.GET("/mutasi/history", mutH.HistoryPage)
			pm.GET("/mutasi/form", mutH.Form)
			pm.GET("/mutasi/lookup", mutH.Lookup)
			pm.GET("/mutasi/riwayat", mutH.History)
			pm.POST("/mutasi", mutH.Create)
			pm.POST("/mutasi/today/qty", mutH.UpdateQty)
			pm.GET("/mutasi/today/:id/edit-history", mutH.EditHistory)
			pm.POST("/pindah-outlet", pageH.RequestTransfer)
			pm.GET("/pindah-outlet", pageH.TransferForm)
		}

		// Manager + Admin: lihat rekap (read-only)
		view := authed.Group("/", middleware.RequireRole(models.RoleManager, models.RoleAdmin))
		{
			view.GET("/rekap", recapH.Index)
			view.GET("/rekap/cetak", recapH.Print)
		}

		// Admin saja
		admin := authed.Group("/", middleware.RequireRole(models.RoleAdmin))
		{
			admin.GET("/password", pageH.PasswordForm)
			admin.POST("/password", pageH.ChangePassword)
			admin.GET("/profile", pageH.ProfileForm)
			admin.POST("/profile", pageH.UpdateProfile)
			admin.GET("/rekap/export", recapH.Export)
			admin.GET("/rekap/mutasi/:id/edit", recapH.Edit)
			admin.POST("/rekap/mutasi/:id/edit", recapH.Update)
			admin.POST("/rekap/mutasi/:id/delete", recapH.Delete)
			admin.GET("/master-data", productH.Index)
			admin.GET("/master-data/create", productH.CreateForm)
			admin.GET("/master-data/import", productH.ImportForm)
			admin.GET("/master-data/import/template", productH.Template)
			admin.GET("/master-data/categories", pageH.Categories)
			admin.POST("/master-data/categories", pageH.AddCategory)
			admin.POST("/master-data/categories/:id/toggle", pageH.ToggleCategory)
			admin.POST("/master-data", productH.Create)
			admin.POST("/master-data/import", productH.Import)
			admin.GET("/master-data/:id/edit", productH.Edit)
			admin.POST("/master-data/:id", productH.Update)
			admin.POST("/master-data/:id/deactivate", productH.Deactivate)
			admin.GET("/kelola", pageH.Kelola)
			admin.GET("/kelola/outlet", pageH.KelolaOutlet)
			admin.GET("/kelola/outlet/create", pageH.AddOutletForm)
			admin.GET("/kelola/outlet/:id/edit", pageH.EditOutlet)
			admin.POST("/kelola/outlet/:id/edit", pageH.UpdateOutlet)
			admin.POST("/kelola/outlet/:id/toggle", pageH.ToggleOutlet)
			admin.GET("/kelola/pm", pageH.KelolaPM)
			admin.GET("/kelola/pm/create", pageH.AddPMForm)
			admin.POST("/kelola/outlet", pageH.AddOutlet)
			admin.POST("/kelola/pm", pageH.AddPM)
			admin.GET("/kelola/pm/:id/edit", pageH.EditPM)
			admin.POST("/kelola/pm/:id/edit", pageH.UpdatePM)
			admin.POST("/kelola/pm/:id/deactivate", pageH.DeactivatePM)
			admin.POST("/kelola/transfer/:id", pageH.DecideTransfer)
		}
	}

	return r, chatHub, nil
}
