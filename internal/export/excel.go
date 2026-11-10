package export

import (
	"bytes"
	"time"

	"github.com/xuri/excelize/v2"

	"movara/internal/clock"
	"movara/internal/models"
)

// RecapXLSX membuat file Excel rekap mutasi. Barkode dan kode ditulis sebagai teks
// sehingga angka nol di depan (mis. 070068) tidak hilang.
func RecapXLSX(rows []models.Mutation, from, to time.Time) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	const sheet = "Rekap Mutasi"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}

	set := func(col, row int, v interface{}) {
		cell, _ := excelize.CoordinatesToCellName(col, row)
		_ = f.SetCellValue(sheet, cell, v)
	}
	setText := func(col, row int, v string) {
		cell, _ := excelize.CoordinatesToCellName(col, row)
		_ = f.SetCellStr(sheet, cell, v)
	}

	period := "Semua periode"
	if !from.IsZero() && !to.IsZero() {
		period = from.In(clock.WIB).Format("02/01/2006") + " s/d " + to.AddDate(0, 0, -1).In(clock.WIB).Format("02/01/2006")
	} else if !from.IsZero() {
		period = "Mulai " + from.In(clock.WIB).Format("02/01/2006")
	} else if !to.IsZero() {
		period = "Sampai " + to.AddDate(0, 0, -1).In(clock.WIB).Format("02/01/2006")
	}
	setText(1, 1, "Periode Jurnal")
	setText(2, 1, period)
	headers := []string{"No", "Barkode", "SKU/PLU", "Nama Produk", "Kategori", "Qty", "Tgl & Jam", "Lokasi Perpindahan", "Outlet", "PM", "BA"}
	for i, h := range headers {
		set(i+1, 3, h)
	}
	if style, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E2E8F0"}},
	}); err == nil {
		_ = f.SetCellStyle(sheet, "A3", "K3", style)
	}

	total := 0
	for i, m := range rows {
		r := i + 4
		t := m.CreatedAt.In(clock.WIB)
		set(1, r, i+1)
		setText(2, r, m.Barcode)
		setText(3, r, m.Code)
		set(4, r, m.ProductName)
		categoryName := ""
		if m.Category != nil {
			categoryName = m.Category.Name
		}
		set(5, r, categoryName)
		set(6, r, m.Qty)
		direction := "Konter → Market"
		if m.Direction == "market_to_counter" {
			direction = "Market → Konter"
		}
		set(7, r, t.Format("02/01/2006 15:04"))
		set(8, r, direction)
		set(9, r, m.Outlet.Name)
		set(10, r, m.User.Name)
		set(11, r, m.BA())
		total += m.Qty
	}
	set(5, len(rows)+5, "TOTAL")
	set(6, len(rows)+5, total)

	widths := map[string]float64{"A": 6, "B": 16, "C": 12, "D": 30, "E": 18, "F": 8, "G": 18, "H": 20, "I": 14, "J": 18, "K": 14}
	for col, w := range widths {
		_ = f.SetColWidth(sheet, col, col, w)
	}
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, YSplit: 3, TopLeftCell: "A4", ActivePane: "bottomLeft",
	})

	return f.WriteToBuffer()
}
