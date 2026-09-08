// Package orderid выдаёт непредсказуемые идентификаторы заказов, уникальные между
// репликами без общего хранилища и без координации.
package orderid

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

const (
	// defaultSecretPath — путь, куда Docker монтирует секрет внутри контейнера.
	defaultSecretPath = "/run/secrets/order_id_secret"
	// secretPathEnv переопределяет путь (нужно для локального запуска и тестов).
	secretPathEnv = "ORDER_ID_SECRET_PATH"

	// idBytes — сколько байт HMAC уходит в идентификатор. 16 байт => 32 hex-символа.
	idBytes = 16
	// generatedSecretBytes — размер временного секрета, если настоящий не передан.
	generatedSecretBytes = 32
)

// Generator выдаёт идентификаторы вида hex(HMAC(secret, instance||counter)).
//
// Уникальность структурная, а не вероятностная: instance — случайный нонс, свой у каждого
// процесса, counter — локальный счётчик. Пара (instance, counter) глобально уникальна,
// поэтому две реплики физически не могут подать на вход HMAC одно и то же значение.
type Generator struct {
	secret   []byte
	instance [8]byte
	counter  int64
}

// New создаёт генератор с заданным секретом и свежим случайным нонсом.
func New(secret []byte) (*Generator, error) {
	if len(secret) == 0 {
		return nil, fmt.Errorf("order id secret is empty")
	}

	g := &Generator{secret: secret}
	if _, err := rand.Read(g.instance[:]); err != nil {
		return nil, fmt.Errorf("cannot seed instance nonce: %w", err)
	}
	return g, nil
}

// Next возвращает следующий идентификатор. Безопасен для конкурентного вызова.
func (g *Generator) Next() string {
	n := atomic.AddInt64(&g.counter, 1)

	var payload [16]byte
	copy(payload[:8], g.instance[:])
	binary.BigEndian.PutUint64(payload[8:], uint64(n))

	mac := hmac.New(sha256.New, g.secret)
	mac.Write(payload[:])

	return hex.EncodeToString(mac.Sum(nil)[:idBytes])
}

// LoadSecret читает секрет из файла, смонтированного Docker'ом. Строки-комментарии и
// пробельные символы отбрасываются, поэтому файл-заглушка из репозитория читается как пустой.
//
// Если секрет не передан, возвращает случайный и generated=true: сервис остаётся рабочим,
// идентификаторы — непредсказуемыми и уникальными, но перестают быть воспроизводимыми
// между перезапусками и одинаковыми у реплик. Вызывающая сторона должна это залогировать.
func LoadSecret() (secret []byte, generated bool, err error) {
	path := os.Getenv(secretPathEnv)
	if path == "" {
		path = defaultSecretPath
	}

	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("cannot read %s: %w", path, err)
	}

	if s := stripComments(string(raw)); s != "" {
		return []byte(s), false, nil
	}

	secret = make([]byte, generatedSecretBytes)
	if _, err := rand.Read(secret); err != nil {
		return nil, false, fmt.Errorf("cannot generate temporary secret: %w", err)
	}
	return secret, true, nil
}

// stripComments убирает строки, начинающиеся с '#', и обрезает пробелы.
func stripComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		b.WriteString(line)
	}
	return b.String()
}
