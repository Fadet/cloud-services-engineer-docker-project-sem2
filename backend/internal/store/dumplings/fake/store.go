package fake

import (
	"context"

	"gitlab.praktikum-services.ru/Stasyan/momo-store/internal/store/dumplings"
)

// IDGenerator выдаёт идентификаторы заказов.
type IDGenerator interface {
	Next() string
}

// Store is a fake in-memory implementation of dumplings.Store
type Store struct {
	ids               IDGenerator
	availableProducts []dumplings.Product
}

func NewStore(ids IDGenerator) *Store {
	return &Store{ids: ids}
}

func (s *Store) SetAvailablePacks(products ...dumplings.Product) {
	s.availableProducts = products
}

func (s *Store) ListProducts(_ context.Context) ([]dumplings.Product, error) {
	return s.availableProducts, nil
}

// CreateOrder fakes order creation, returning an opaque unguessable id.
func (s *Store) CreateOrder(_ context.Context, _ ...dumplings.OrderItem) (id string, err error) {
	return s.ids.Next(), nil
}
