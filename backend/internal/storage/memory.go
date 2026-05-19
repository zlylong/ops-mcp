package storage

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

type ExecutionStore struct {
	mu    sync.RWMutex
	items []domain.Execution
}

func NewExecutionStore() *ExecutionStore {
	return &ExecutionStore{items: make([]domain.Execution, 0, 64)}
}
func (s *ExecutionStore) Add(e domain.Execution) domain.Execution {
	if e.ID == "" {
		e.ID = newID("exe")
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	s.items = append([]domain.Execution{e}, s.items...)
	s.mu.Unlock()
	return e
}
func (s *ExecutionStore) List() []domain.Execution {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Execution, len(s.items))
	copy(out, s.items)
	return out
}
func (s *ExecutionStore) Get(id string) (domain.Execution, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}
	return domain.Execution{}, false
}

type ApprovalStore struct {
	mu    sync.RWMutex
	items []domain.Approval
}

func NewApprovalStore() *ApprovalStore { return &ApprovalStore{items: make([]domain.Approval, 0, 16)} }
func (s *ApprovalStore) Add(a domain.Approval) domain.Approval {
	if a.ID == "" {
		a.ID = newID("app")
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	s.items = append([]domain.Approval{a}, s.items...)
	s.mu.Unlock()
	return a
}
func (s *ApprovalStore) List() []domain.Approval {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Approval, len(s.items))
	copy(out, s.items)
	return out
}
func (s *ApprovalStore) Decide(id string, status domain.ApprovalStatus) (domain.Approval, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			now := time.Now().UTC()
			s.items[i].Status = status
			s.items[i].DecidedAt = &now
			return s.items[i], nil
		}
	}
	return domain.Approval{}, errors.New("approval not found")
}

func newID(prefix string) string {
	return prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}
