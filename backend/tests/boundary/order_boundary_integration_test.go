package boundary_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// このテストは 境界一貫性統合テスト の受け皿を固定する。
// 仕様対象: 実Router+実UseCase+実DB を通した境界観測の検証。
// 根拠: Stub前提の単体境界テストと統合境界テストを明確に分離するため。
func TestIntegration_OrderBoundary_受け皿(t *testing.T) {
	kit := new境界統合Testkit(t)

	payload, _ := json.Marshal(map[string]any{
		"customerId": "c-1",
		"items":      []map[string]any{{"sku": "sku-1", "quantity": 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	kit.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
