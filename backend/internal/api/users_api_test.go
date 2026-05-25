package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/config"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

// mockRecorder implements the api.AuditRecorder interface for testing.
type mockRecorder struct{}

func (m *mockRecorder) Record(r domain.AuditRecord) domain.AuditRecord { return r }
func (m *mockRecorder) List() []domain.AuditRecord                     { return nil }

func jsonBody(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

// adminToken creates an admin user and returns a valid user token for them.
func adminToken(t *testing.T, r *app.Registry, cfg config.Config) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("admin-pass-8888"), bcrypt.DefaultCost)
	require.NoError(t, err)
	admin := domain.User{Username: "admin", Role: domain.RoleAdmin, Status: "active"}
	r.Users().Add(admin, "", hash)
	token := "user:" + admin.ID
	router := NewRouter(cfg, r, &mockRecorder{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
		jsonBody(map[string]string{"username": "admin", "password": "admin-pass-8888"}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct{ Token string }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Token
}

// ------------------------------------------------------------------
// listUsers
// ------------------------------------------------------------------

func TestListUsers_AuthenticatedAdmin_ReturnsUserList(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var users []domain.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &users))
	assert.NotEmpty(t, users)
}

func TestListUsers_ViewerRole_ReturnsForbidden(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	viewerHash, _ := bcrypt.GenerateFromPassword([]byte("viewer-pass-9999"), bcrypt.DefaultCost)
	viewer := domain.User{Username: "viewer", Role: domain.RoleViewer, Status: "active"}
	r.Users().Add(viewer, "", viewerHash)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
		jsonBody(map[string]string{"username": "viewer", "password": "viewer-pass-9999"}))
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)
	var loginResp struct{ Token string }
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestListUsers_Unauthenticated_ReturnsUnauthorized(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ------------------------------------------------------------------
// getUser
// ------------------------------------------------------------------

func TestGetUser_ValidID_ReturnsUser(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	var users []domain.User
	json.Unmarshal(listW.Body.Bytes(), &users)
	adminID := users[0].ID

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+adminID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var user domain.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &user))
	assert.Equal(t, adminID, user.ID)
}

func TestGetUser_NotFound_Returns404(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/nonexistent-id-xyz", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ------------------------------------------------------------------
// createUser
// ------------------------------------------------------------------

func TestCreateUser_ValidRequest_CreatesUser(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]any{"username": "newuser", "password": "newpass1234", "role": "viewer", "nickname": "New User"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var user domain.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &user))
	assert.Equal(t, "newuser", user.Username)
	assert.Equal(t, domain.RoleViewer, user.Role)
}

func TestCreateUser_DuplicateUsername_ReturnsBadRequest(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	// "admin" already exists from adminToken
	body := jsonBody(map[string]any{"username": "admin", "password": "somepass9999"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "already taken")
}

func TestCreateUser_InvalidRole_ReturnsBadRequest(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]any{"username": "roleuser", "password": "pass12345678", "role": "superadmin"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "role must be")
}

func TestCreateUser_PasswordTooShort_ReturnsBadRequest(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]any{"username": "shortpw", "password": "1234567"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ------------------------------------------------------------------
// getMe
// ------------------------------------------------------------------

func TestGetMe_ValidToken_ReturnsOwnProfile(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var user domain.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &user))
	assert.Equal(t, "admin", user.Username)
}

func TestGetMe_NoAuth_ReturnsUnauthorized(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ------------------------------------------------------------------
// updateMe
// ------------------------------------------------------------------

func TestUpdateMe_ValidUpdate_UpdatesNicknameAndEmail(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]any{"nickname": "Super Admin", "email": "admin@example.com"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var user domain.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &user))
	assert.Equal(t, "Super Admin", user.Nickname)
	assert.Equal(t, "admin@example.com", user.Email)
}

func TestUpdateMe_InvalidEmail_Ignored(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]any{"email": "not-an-email"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var user domain.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &user))
	assert.NotEqual(t, "not-an-email", user.Email)
}

// ------------------------------------------------------------------
// changeMyPassword
// ------------------------------------------------------------------

func TestChangeMyPassword_ValidRequest_ChangesPassword(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]string{"oldPassword": "admin-pass-8888", "newPassword": "newpass-87654321"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "successfully")

	// Verify old password no longer works
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
		jsonBody(map[string]string{"username": "admin", "password": "admin-pass-8888"}))
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)
	assert.Equal(t, http.StatusUnauthorized, loginW.Code)

	// Verify new password works
	loginReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
		jsonBody(map[string]string{"username": "admin", "password": "newpass-87654321"}))
	loginW2 := httptest.NewRecorder()
	router.ServeHTTP(loginW2, loginReq2)
	assert.Equal(t, http.StatusOK, loginW2.Code)
}

func TestChangeMyPassword_WrongOldPassword_ReturnsForbidden(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]string{"oldPassword": "wrong-old-pass", "newPassword": "newpass-87654321"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "incorrect")
}

func TestChangeMyPassword_NewPasswordTooShort_ReturnsBadRequest(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	token := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]string{"oldPassword": "admin-pass-8888", "newPassword": "short"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/password", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "at least 8 characters")
}

// ------------------------------------------------------------------
// changeUserPassword (admin resets any user's password)
// ------------------------------------------------------------------

func TestChangeUserPassword_ValidRequest_ResetsPassword(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	adminTokenVal := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	targetHash, _ := bcrypt.GenerateFromPassword([]byte("target-old-pass"), bcrypt.DefaultCost)
	target := domain.User{Username: "target", Role: domain.RoleViewer, Status: "active"}
	added := r.Users().Add(target, "", targetHash)

	body := jsonBody(map[string]string{"newPassword": "admin-reset-pass-123"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+added.ID+"/password", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminTokenVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "successfully")
}

func TestChangeUserPassword_UserNotFound_Returns404(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	adminTokenVal := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	body := jsonBody(map[string]string{"newPassword": "anypass-12345678"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/nonexistent-id-xyz/password", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminTokenVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestChangeUserPassword_PasswordTooShort_ReturnsBadRequest(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	adminTokenVal := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminTokenVal)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	var users []domain.User
	json.Unmarshal(listW.Body.Bytes(), &users)

	body := jsonBody(map[string]string{"newPassword": "short"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+users[0].ID+"/password", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminTokenVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ------------------------------------------------------------------
// updateUser
// ------------------------------------------------------------------

func TestUpdateUser_AdminUpdatesRoleAndStatus(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	adminTokenVal := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminTokenVal)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	var users []domain.User
	json.Unmarshal(listW.Body.Bytes(), &users)

	body := jsonBody(map[string]any{"role": "operator", "status": "inactive"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+users[0].ID, bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminTokenVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var updated domain.User
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, domain.RoleOperator, updated.Role)
	assert.Equal(t, "inactive", updated.Status)
}

func TestUpdateUser_InvalidStatus_ReturnsBadRequest(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	adminTokenVal := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listReq.Header.Set("Authorization", "Bearer "+adminTokenVal)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	var users []domain.User
	json.Unmarshal(listW.Body.Bytes(), &users)

	body := jsonBody(map[string]any{"status": "deleted"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+users[0].ID, bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminTokenVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ------------------------------------------------------------------
// deleteUser
// ------------------------------------------------------------------

func TestDeleteUser_AdminDeletesUser(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	adminTokenVal := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	newHash, _ := bcrypt.GenerateFromPassword([]byte("deleteme1234"), bcrypt.DefaultCost)
	newUser := domain.User{Username: "deleteme", Role: domain.RoleViewer, Status: "active"}
	added := r.Users().Add(newUser, "", newHash)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+added.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminTokenVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify user is gone
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+added.ID, nil)
	getReq.Header.Set("Authorization", "Bearer "+adminTokenVal)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusNotFound, getW.Code)
}

func TestDeleteUser_UserNotFound_Returns404(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	adminTokenVal := adminToken(t, r, cfg)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/nonexistent-id-xyz", nil)
	req.Header.Set("Authorization", "Bearer "+adminTokenVal)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteUser_ViewerRole_ReturnsForbidden(t *testing.T) {
	r := createTestRegistry(t)
	cfg := config.Config{}
	viewerHash, _ := bcrypt.GenerateFromPassword([]byte("viewer-pass-9999"), bcrypt.DefaultCost)
	viewer := domain.User{Username: "viewer2", Role: domain.RoleViewer, Status: "active"}
	r.Users().Add(viewer, "", viewerHash)
	router := NewRouter(cfg, r, &mockRecorder{}, nil)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/login",
		jsonBody(map[string]string{"username": "viewer2", "password": "viewer-pass-9999"}))
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)
	var loginResp struct{ Token string }
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/some-id", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
