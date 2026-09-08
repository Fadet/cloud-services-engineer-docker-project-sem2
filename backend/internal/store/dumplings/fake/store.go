package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"gitlab.praktikum-services.ru/Stasyan/momo-store/internal/store/dumplings"
)

type IDGenerator interface {
	Next() string
}

type orderRecord struct {
	ID        string                `json:"id"`
	CreatedAt string                `json:"created_at"`
	Items     []dumplings.OrderItem `json:"items"`
}

type Store struct {
	ids               IDGenerator
	availableProducts []dumplings.Product

	mu      sync.Mutex
	journal io.Writer
}

func NewStore(ids IDGenerator, journal io.Writer) *Store {
	return &Store{ids: ids, journal: journal}
}

func (s *Store) SetAvailablePacks(products ...dumplings.Product) {
	s.availableProducts = products
}

func (s *Store) ListProducts(_ context.Context) ([]dumplings.Product, error) {
	return s.availableProducts, nil
}

func (s *Store) CreateOrder(_ context.Context, items ...dumplings.OrderItem) (id string, err error) {
	id = s.ids.Next()

	if s.journal == nil {
		return id, nil
	}

	line, err := json.Marshal(orderRecord{
		ID:        id,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     items,
	})
	if err != nil {
		return "", fmt.Errorf("cannot encode order %s: %w", id, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.journal.Write(append(line, '\n')); err != nil {
		return "", fmt.Errorf("cannot persist order %s: %w", id, err)
	}

	return id, nil
}
