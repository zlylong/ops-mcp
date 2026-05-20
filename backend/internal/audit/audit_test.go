package audit

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zlylong/ops-mcp/backend/internal/domain"
)

func TestStore_Record(t *testing.T) {
	store := NewStore(slog.Default())

	record := domain.AuditRecord{
		Actor:   "test-user",
		Role:    domain.RoleViewer,
		Action:  "k8s.list_pods",
		Target:  "local-dev",
		Allowed: true,
		Reason:  "allowed",
	}
	result := store.Record(record)

	assert.NotEmpty(t, result.ID)
	assert.NotEmpty(t, result.At)
	assert.Equal(t, "test-user", result.Actor)
	assert.Equal(t, "k8s.list_pods", result.Action)
}

func TestStore_Record_AutoID(t *testing.T) {
	store := NewStore(slog.Default())

	record := domain.AuditRecord{
		ID:      "",
		Actor:   "test-user",
		Role:    domain.RoleViewer,
		Action:  "test.action",
		Target:  "local-dev",
		Allowed: true,
		Reason:  "allowed",
	}
	result := store.Record(record)

	assert.NotEmpty(t, result.ID)
}

func TestStore_Record_AutoTimestamp(t *testing.T) {
	store := NewStore(slog.Default())

	before := domain.AuditRecord{
		Actor:   "test-user",
		Role:    domain.RoleViewer,
		Action:  "test.action",
		Target:  "local-dev",
		Allowed: true,
		Reason:  "allowed",
	}
	result1 := store.Record(before)

	after := domain.AuditRecord{
		Actor:   "test-user",
		Role:    domain.RoleViewer,
		Action:  "test.action2",
		Target:  "local-dev",
		Allowed: true,
		Reason:  "allowed",
	}
	result2 := store.Record(after)

	assert.NotEqual(t, result1.At.UnixNano(), result2.At.UnixNano())
}

func TestStore_List(t *testing.T) {
	store := NewStore(slog.Default())

	record1 := domain.AuditRecord{Actor: "user1", Role: domain.RoleViewer, Action: "action1", Target: "target", Allowed: true, Reason: "ok"}
	record2 := domain.AuditRecord{Actor: "user2", Role: domain.RoleOperator, Action: "action2", Target: "target", Allowed: false, Reason: "denied"}

	store.Record(record1)
	store.Record(record2)

	list := store.List()
	assert.Len(t, list, 2)
	assert.Equal(t, "action2", list[0].Action)
	assert.Equal(t, "action1", list[1].Action)
}

func TestMask_SensitiveKeys(t *testing.T) {
	input := map[string]any{
		"username":      "john",
		"password":      "secret123",
		"token":         "abc123",
		"api_key":       "key123",
		"normal_field":  "value",
		"nested":        map[string]any{"secret_key": "hidden"},
	}

	masked := Mask(input)

	assert.Equal(t, "john", masked["username"])
	assert.Equal(t, "***MASKED***", masked["password"])
	assert.Equal(t, "***MASKED***", masked["token"])
	assert.Equal(t, "***MASKED***", masked["api_key"])
	assert.Equal(t, "value", masked["normal_field"])
	nested, ok := masked["nested"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "***MASKED***", nested["secret_key"])
}

func TestMask_CaseInsensitive(t *testing.T) {
	input := map[string]any{
		"PASSWORD": "secret",
		"Api_Key":  "key",
		"TOKEN":    "tok",
	}

	masked := Mask(input)

	assert.Equal(t, "***MASKED***", masked["PASSWORD"])
	assert.Equal(t, "***MASKED***", masked["Api_Key"])
	assert.Equal(t, "***MASKED***", masked["TOKEN"])
}

func TestMask_NilInput(t *testing.T) {
	masked := Mask(nil)
	assert.Nil(t, masked)
}

func TestMask_EmptyInput(t *testing.T) {
	masked := Mask(map[string]any{})
	assert.Equal(t, 0, len(masked))
}

func TestIsSensitiveKey(t *testing.T) {
	assert.True(t, isSensitiveKey("password"))
	assert.True(t, isSensitiveKey("PASSWORD"))
	assert.True(t, isSensitiveKey("api_key"))
	assert.True(t, isSensitiveKey("apikey"))
	assert.True(t, isSensitiveKey("authorization"))
	assert.True(t, isSensitiveKey("credential"))

	assert.False(t, isSensitiveKey("username"))
	assert.False(t, isSensitiveKey("email"))
	assert.False(t, isSensitiveKey("normal_field"))
}
