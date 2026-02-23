package domain

import "errors"

var (
	// ErrPriceConflict はクライアント提示価格とサーバ価格の不一致を表す。
	ErrPriceConflict = errors.New("price conflict")
	// ErrInvalidCustomer は注文で指定された顧客参照が不正であることを表す。
	ErrInvalidCustomer = errors.New("invalid customer")
)
