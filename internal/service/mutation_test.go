package service

import (
	"testing"
	"time"

	"movara/internal/clock"
	"movara/internal/models"
	"movara/internal/repository/memory"
)

func TestByDateFiltersPMCategoryWithinOutlet(t *testing.T) {
	repos, err := memory.New()
	if err != nil {
		t.Fatal(err)
	}
	now := clock.Now().In(clock.WIB)
	catObat, catKosmetik := uint(101), uint(202)
	for i, category := range []*uint{&catObat, &catKosmetik} {
		qty := i + 1
		if err := repos.Mutations.Create(&models.Mutation{
			Barcode: "barcode-" + string(rune('A'+i)), ProductName: "produk", CategoryID: category,
			OutletID: 2, UserID: 4, Direction: "counter_to_market", Qty: qty,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewMutationService(repos.Products, repos.Mutations, repos.MutationEdits, nil)
	result, err := svc.ByDate(2, &catObat, "", now.Format("2006-01-02"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0].CategoryID == nil || *result.Rows[0].CategoryID != catObat {
		t.Fatalf("expected only obat category, got %#v", result.Rows)
	}
}
