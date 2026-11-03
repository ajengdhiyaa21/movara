package migrations

import "embed"

// FS contains only the SQL migration files in this directory.
//
//go:embed *.sql
var FS embed.FS
