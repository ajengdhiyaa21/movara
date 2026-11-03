package mysql

import (
	"gorm.io/gorm"
	"movara/internal/repository"
)

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *repository.Repos {
	s := &Store{db: db}
	return &repository.Repos{Users: s, Chat: s, Products: s, Outlets: s, Mutations: s, MutationEdits: s, Audits: s}
}
