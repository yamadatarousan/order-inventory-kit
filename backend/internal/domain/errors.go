package domain

import "errors"

var (
	// ErrPriceConflict はクライアント提示価格とサーバ価格の不一致を表す。
	ErrPriceConflict = errors.New("price conflict")
)
