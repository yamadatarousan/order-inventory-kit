package domain

import "order-inventory-kit/internal/usecase"

func Leak() {
	usecase.Run()
}
