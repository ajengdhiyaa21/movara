package service

import (
	"log"

	"movara/internal/models"
	"movara/internal/repository"
)

type AuditService struct{ repo repository.Audits }

func NewAuditService(repo repository.Audits) *AuditService { return &AuditService{repo: repo} }

// Log tidak mengembalikan error agar kegagalan audit tidak menggagalkan aksi utama.
func (s *AuditService) Log(userID, outletID *uint, action, detail string) {
	err := s.repo.Add(&models.AuditLog{UserID: userID, OutletID: outletID, Action: action, Detail: detail})
	if err != nil {
		log.Printf("audit log gagal: %v", err)
	}
}
