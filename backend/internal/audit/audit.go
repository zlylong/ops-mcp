package audit

import (
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

type Recorder interface {
	Record(domain.AuditRecord) domain.AuditRecord
	List() []domain.AuditRecord
}

type Store struct {
	logger  *slog.Logger
	mu      sync.RWMutex
	records []domain.AuditRecord
}

func NewStore(logger *slog.Logger) *Store {
	return &Store{logger: logger, records: make([]domain.AuditRecord, 0, 64)}
}

func (s *Store) Record(record domain.AuditRecord) domain.AuditRecord {
	if record.ID == "" {
		record.ID = newID("aud")
	}
	if record.At.IsZero() {
		record.At = time.Now().UTC()
	}
	record.Parameters = Mask(record.Parameters)
	s.mu.Lock()
	s.records = append([]domain.AuditRecord{record}, s.records...)
	s.mu.Unlock()
	if s.logger != nil {
		s.logger.Info("audit", "id", record.ID, "executionId", record.ExecutionID, "actor", record.Actor, "role", record.Role, "action", record.Action, "target", record.Target, "allowed", record.Allowed, "reason", record.Reason)
	}
	return record
}

func (s *Store) List() []domain.AuditRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.AuditRecord, len(s.records))
	copy(out, s.records)
	return out
}

func Mask(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		if isSensitiveKey(key) {
			out[key] = "***MASKED***"
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			out[key] = Mask(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, marker := range []string{"password", "secret", "token", "api_key", "apikey", "authorization", "credential"} {
		if strings.Contains(k, marker) {
			return true
		}
	}
	return false
}

func newID(prefix string) string {
	return prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}
