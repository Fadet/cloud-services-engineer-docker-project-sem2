package fake

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubIDs struct{ n int }

func (s *stubIDs) Next() string {
	s.n++
	return "id-" + string(rune('a'+s.n-1))
}

func TestCreateOrderWritesJournal(t *testing.T) {
	var journal bytes.Buffer
	store := NewStore(&stubIDs{}, &journal)

	id, err := store.CreateOrder(context.Background())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(journal.String()), "\n")
	require.Len(t, lines, 1)

	var rec map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &rec), "строка журнала должна быть валидным JSON")
	assert.Equal(t, id, rec["id"])
	assert.NotEmpty(t, rec["created_at"])
}

func TestCreateOrderAppendsLinePerOrder(t *testing.T) {
	var journal bytes.Buffer
	store := NewStore(&stubIDs{}, &journal)

	for i := 0; i < 5; i++ {
		_, err := store.CreateOrder(context.Background())
		require.NoError(t, err)
	}

	assert.Len(t, strings.Split(strings.TrimSpace(journal.String()), "\n"), 5)
}

// Без журнала стор обязан работать как раньше — на этом держатся тесты приложения.
func TestCreateOrderWithoutJournal(t *testing.T) {
	store := NewStore(&stubIDs{}, nil)

	id, err := store.CreateOrder(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}
