package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"movara/internal/models"
)

// OutletScope menentukan filter outlet untuk query data.
//   - PM           : SELALU outlet miliknya (query ?outlet= diabaikan).
//   - Manager/Admin: ?outlet=<id> untuk satu outlet, kosong/"all" untuk semua (nil).
//
// Jangan pernah mengambil outlet_id PM dari form/body.
func OutletScope(c *gin.Context) *uint {
	u := CurrentUser(c)
	if u == nil {
		return nil
	}
	if u.Role == models.RolePM {
		return u.OutletID
	}
	raw := c.Query("outlet")
	if raw == "" || raw == "all" {
		return nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil
	}
	v := uint(id)
	return &v
}
