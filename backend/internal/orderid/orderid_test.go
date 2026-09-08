package orderid

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idFormat = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestOrderIDFormat(t *testing.T) {
	g, err := New([]byte("test-secret"))
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		assert.Regexp(t, idFormat, g.Next())
	}
}

func TestOrderIDNotSequential(t *testing.T) {
	g, err := New([]byte("test-secret"))
	require.NoError(t, err)

	// Счётчик внутри инкрементируется на 1, но наружу это просачиваться не должно:
	// соседние идентификаторы обязаны отличаться радикально, а не на единицу.
	first, second := g.Next(), g.Next()
	assert.NotEqual(t, first, second)

	diff := 0
	for i := range first {
		if first[i] != second[i] {
			diff++
		}
	}
	assert.Greater(t, diff, 8, "соседние id слишком похожи: %s vs %s", first, second)
}

func TestOrderIDUniqueWithinGenerator(t *testing.T) {
	g, err := New([]byte("test-secret"))
	require.NoError(t, err)

	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		seen[g.Next()] = struct{}{}
	}
	assert.Len(t, seen, n, "генератор выдал дубликаты")
}

// Главная проверка: две реплики с ОДНИМ И ТЕМ ЖЕ секретом не пересекаются по id.
// До изменений обе отдали бы 1, 2, 3... и совпали бы полностью.
func TestOrderIDDisjointAcrossReplicas(t *testing.T) {
	secret := []byte("shared-secret")

	replicaA, err := New(secret)
	require.NoError(t, err)
	replicaB, err := New(secret)
	require.NoError(t, err)

	const n = 5000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		seen[replicaA.Next()] = struct{}{}
	}
	for i := 0; i < n; i++ {
		id := replicaB.Next()
		_, collision := seen[id]
		require.False(t, collision, "реплики выдали одинаковый id: %s", id)
	}
}

func TestOrderIDConcurrent(t *testing.T) {
	g, err := New([]byte("test-secret"))
	require.NoError(t, err)

	const workers, perWorker = 16, 500
	ids := make(chan string, workers*perWorker)
	done := make(chan struct{})

	for w := 0; w < workers; w++ {
		go func() {
			for i := 0; i < perWorker; i++ {
				ids <- g.Next()
			}
			done <- struct{}{}
		}()
	}
	for w := 0; w < workers; w++ {
		<-done
	}
	close(ids)

	seen := make(map[string]struct{}, workers*perWorker)
	for id := range ids {
		seen[id] = struct{}{}
	}
	assert.Len(t, seen, workers*perWorker, "конкурентные вызовы дали дубликаты")
}

func TestLoadSecretFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order_id_secret")
	require.NoError(t, os.WriteFile(path, []byte("# комментарий\n  real-secret-value  \n"), 0o600))

	t.Setenv(secretPathEnv, path)

	secret, generated, err := LoadSecret()
	require.NoError(t, err)
	assert.False(t, generated, "секрет из файла не должен считаться сгенерированным")
	assert.Equal(t, "real-secret-value", string(secret), "комментарии и пробелы должны отбрасываться")
}

func TestLoadSecretGeneratesWhenAbsent(t *testing.T) {
	t.Setenv(secretPathEnv, filepath.Join(t.TempDir(), "does-not-exist"))

	secret, generated, err := LoadSecret()
	require.NoError(t, err)
	assert.True(t, generated)
	assert.Len(t, secret, generatedSecretBytes)
}

// Файл-заглушка из репозитория состоит из одних комментариев: она должна читаться
// как отсутствующий секрет, иначе в прод уехал бы известный всем ключ.
func TestLoadSecretTreatsCommentOnlyFileAsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "order_id_secret")
	require.NoError(t, os.WriteFile(path, []byte("# только комментарий\n#  и ещё один\n\n"), 0o600))

	t.Setenv(secretPathEnv, path)

	_, generated, err := LoadSecret()
	require.NoError(t, err)
	assert.True(t, generated)
}
