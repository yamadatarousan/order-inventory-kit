package domain_test

import (
	"testing"

	"order-inventory-kit/internal/domain"
)

// このテストは Inventory ドメイン不変条件を固定する。
// 仕様対象: OnHand/Reserved/Available の非負制約、過剰確保の失敗、戻し時の整合。
// 根拠: 標準在庫モデルへ移行後も、在庫の下限制約と状態整合が崩れないようにするため。
func TestNewInventory_不変条件_OnHandまたはReservedが負数なら失敗(t *testing.T) {
	tests := []struct {
		name     string
		onHand   int
		reserved int
	}{
		{name: "OnHandが負数", onHand: -1, reserved: 0},
		{name: "Reservedが負数", onHand: 0, reserved: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewInventory("sku-1", tt.onHand, tt.reserved)
			if err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestNewInventory_不変条件_SKUは必須(t *testing.T) {
	_, err := domain.NewInventory("", 10, 0)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestInventory_不変条件_過剰確保は失敗し状態が変化しない(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10, 2)
	if err := inv.Reserve(9); err == nil {
		t.Fatalf("expected error")
	}
	if inv.OnHand != 10 {
		t.Fatalf("expected onHand to remain 10, got %d", inv.OnHand)
	}
	if inv.Reserved != 2 {
		t.Fatalf("expected reserved to remain 2, got %d", inv.Reserved)
	}
	if inv.Available() != 8 {
		t.Fatalf("expected available to remain 8, got %d", inv.Available())
	}
}

func TestInventory_不変条件_戻し時整合_確保超過の戻しは禁止(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10, 3)
	if err := inv.Release(4); err == nil {
		t.Fatalf("expected error")
	}
	if inv.OnHand != 10 {
		t.Fatalf("expected onHand to remain 10, got %d", inv.OnHand)
	}
	if inv.Reserved != 3 {
		t.Fatalf("expected reserved to remain 3, got %d", inv.Reserved)
	}
	if inv.Available() != 7 {
		t.Fatalf("expected available to remain 7, got %d", inv.Available())
	}
}

func TestInventory_不変条件_旧Quantity互換生成でもReserved超過の戻しは禁止(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10)
	if err := inv.Release(1); err == nil {
		t.Fatalf("expected error")
	}
	if inv.OnHand != 10 {
		t.Fatalf("expected onHand to remain 10, got %d", inv.OnHand)
	}
	if inv.Reserved != 0 {
		t.Fatalf("expected reserved to remain 0, got %d", inv.Reserved)
	}
	if inv.Available() != 10 {
		t.Fatalf("expected available to remain 10, got %d", inv.Available())
	}
}

func TestInventory_不変条件_確保成功でReservedのみ増えAvailableが減る(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10, 3)
	if err := inv.Reserve(2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inv.OnHand != 10 {
		t.Fatalf("expected onHand to remain 10, got %d", inv.OnHand)
	}
	if inv.Reserved != 5 {
		t.Fatalf("expected reserved 5, got %d", inv.Reserved)
	}
	if inv.Available() != 5 {
		t.Fatalf("expected available 5, got %d", inv.Available())
	}
}

func TestInventory_不変条件_確保数量が0以下は失敗し状態が変化しない(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10, 3)
	tests := []int{0, -1}
	for _, q := range tests {
		err := inv.Reserve(q)
		if err == nil {
			t.Fatalf("expected error on reserve(%d)", q)
		}
	}
	if inv.OnHand != 10 {
		t.Fatalf("expected onHand to remain 10, got %d", inv.OnHand)
	}
	if inv.Reserved != 3 {
		t.Fatalf("expected reserved to remain 3, got %d", inv.Reserved)
	}
	if inv.Available() != 7 {
		t.Fatalf("expected available to remain 7, got %d", inv.Available())
	}
}

func TestInventory_不変条件_戻し数量が0以下は失敗し状態が変化しない(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10, 3)
	tests := []int{0, -1}
	for _, q := range tests {
		err := inv.Release(q)
		if err == nil {
			t.Fatalf("expected error on release(%d)", q)
		}
	}
	if inv.OnHand != 10 {
		t.Fatalf("expected onHand to remain 10, got %d", inv.OnHand)
	}
	if inv.Reserved != 3 {
		t.Fatalf("expected reserved to remain 3, got %d", inv.Reserved)
	}
	if inv.Available() != 7 {
		t.Fatalf("expected available to remain 7, got %d", inv.Available())
	}
}

func TestInventory_不変条件_戻し成功でReservedのみ減りAvailableが増える(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10, 3)
	if err := inv.Release(2); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if inv.OnHand != 10 {
		t.Fatalf("expected onHand to remain 10, got %d", inv.OnHand)
	}
	if inv.Reserved != 1 {
		t.Fatalf("expected reserved 1, got %d", inv.Reserved)
	}
	if inv.Available() != 9 {
		t.Fatalf("expected available 9, got %d", inv.Available())
	}
}

func TestInventory_不変条件_在庫恒等式_各操作後にavailableはonHandマイナスreservedと一致する(t *testing.T) {
	inv, _ := domain.NewInventory("sku-1", 10, 3)
	assert在庫恒等式(t, inv)

	if err := inv.Reserve(2); err != nil {
		t.Fatalf("reserve must succeed: %v", err)
	}
	assert在庫恒等式(t, inv)

	if err := inv.Reserve(100); err == nil {
		t.Fatalf("reserve over available must fail")
	}
	assert在庫恒等式(t, inv)

	if err := inv.Release(1); err != nil {
		t.Fatalf("release must succeed: %v", err)
	}
	assert在庫恒等式(t, inv)

	if err := inv.Release(100); err == nil {
		t.Fatalf("release over reserved must fail")
	}
	assert在庫恒等式(t, inv)
}

func assert在庫恒等式(t *testing.T, inv domain.Inventory) {
	t.Helper()
	want := inv.OnHand - inv.Reserved
	got := inv.Available()
	if got != want {
		t.Fatalf("available invariant broken: available=%d want=%d (onHand=%d reserved=%d)", got, want, inv.OnHand, inv.Reserved)
	}
}
