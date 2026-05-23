package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

// authUserID is the context key for the authenticated user ID.
const authUserID = "auth.userID"

// listUsers returns all users (admin only).
//
// @Summary List Users
// @Description Returns the full user list. Only available to admin role.
// @Tags users
// @Produce json
// @Success 200 {array} object
// @Failure 403 {object} map[string]string
// @Router /api/v1/users [get]
func (s *Server) listUsers(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	c.JSON(http.StatusOK, s.registry.Users().List())
}

// getUser returns a user by ID (admin only).
func (s *Server) getUser(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	user, found := s.registry.Users().Get(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// getMe returns the currently authenticated user's profile.
//
// @Summary Get My Profile
// @Description Returns the profile of the currently logged-in user.
// @Tags users
// @Produce json
// @Success 200 {object} object
// @Failure 401 {object} map[string]string
// @Router /api/v1/users/me [get]
func (s *Server) getMe(c *gin.Context) {
	var userID string
	if v, ok := c.Get(authUserID); ok {
		userID = v.(string)
	} else if v, ok := c.Get(authIsMasterKey); ok && v.(bool) {
		// Master token: use the first admin user as the current user
		users := s.registry.Users().List()
		for _, u := range users {
			if u.Role == "admin" && u.Status == "active" {
				userID = u.ID
				break
			}
		}
		if userID == "" && len(users) > 0 {
			userID = users[0].ID
		}
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	user, found := s.registry.Users().Get(userID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// resolveUserID returns the user ID for the current request.
// Handles both user token auth and master token auth (falls back to first admin).
func (s *Server) resolveUserID(c *gin.Context) string {
	if v, ok := c.Get(authUserID); ok {
		return v.(string)
	}
	if v, ok := c.Get(authIsMasterKey); ok && v.(bool) {
		users := s.registry.Users().List()
		for _, u := range users {
			if u.Role == "admin" && u.Status == "active" {
				return u.ID
			}
		}
		if len(users) > 0 {
			return users[0].ID
		}
	}
	return ""
}

// updateMe updates the authenticated user's own nickname and email.
//
// @Summary Update My Profile
// @Description Updates the authenticated user's nickname and email.
// @Tags users
// @Accept json
// @Produce json
// @Param request body domain.UserUpdateRequest true "Profile update"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/users/me [put]
func (s *Server) updateMe(c *gin.Context) {
	userID := s.resolveUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	var req domain.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	err := s.registry.Users().Update(userID, func(u *domain.User) {
		if s := strings.TrimSpace(req.Nickname); s != "" {
			u.Nickname = s
		}
		if strings.Contains(req.Email, "@") {
			u.Email = strings.TrimSpace(req.Email)
		}
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	user, _ := s.registry.Users().Get(userID)
	c.JSON(http.StatusOK, user)
}

// changeMyPassword changes the authenticated user's password after verifying the old one.
//
// @Summary Change My Password
// @Description Verifies old password then updates to the new one.
// @Tags users
// @Accept json
// @Produce json
// @Param request body domain.ChangePasswordRequest true "Password change"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/users/me/password [put]
func (s *Server) changeMyPassword(c *gin.Context) {
		userID := s.resolveUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	var req domain.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if len(strings.TrimSpace(req.NewPassword)) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new password must be at least 8 characters"})
		return
	}
	user, found := s.registry.Users().Get(userID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	_, hash, ok := s.registry.Users().GetByUsername(user.Username)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "old password is incorrect"})
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	if err := s.registry.Users().SetPassword(userID, newHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// createUser creates a new user account (admin only).
//
// @Summary Create User
// @Description Creates a new user account. Admin role required.
// @Tags users
// @Accept json
// @Produce json
// @Param request body domain.UserCreateRequest true "User creation"
// @Success 201 {object} object
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/users [post]
func (s *Server) createUser(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	var req domain.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}
	if len(req.Username) < 3 || len(req.Username) > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username must be 3-32 characters"})
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}
	switch req.Role {
	case domain.RoleViewer, domain.RoleOperator, domain.RoleAdmin:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be viewer, operator, or admin"})
		return
	}
	if _, _, found := s.registry.Users().GetByUsername(req.Username); found {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username already taken"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	now := time.Now().UTC()
	u := domain.User{
		ID:        "usr-" + strconv.FormatInt(now.UnixNano(), 36),
		Username:  req.Username,
		Nickname:  strings.TrimSpace(req.Nickname),
		Email:     strings.TrimSpace(req.Email),
		Role:      req.Role,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.registry.Users().Add(u, req.Password, hash)
	c.JSON(http.StatusCreated, u)
}

// updateUser updates an existing user (admin only).
//
// @Summary Update User
// @Description Updates an existing user's profile, role, or status. Admin only.
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body domain.UserUpdateRequest true "User update"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [put]
func (s *Server) updateUser(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	var req domain.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if req.Role != "" {
		switch req.Role {
		case domain.RoleViewer, domain.RoleOperator, domain.RoleAdmin:
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be viewer, operator, or admin"})
			return
		}
	}
	if req.Status != "" && req.Status != "active" && req.Status != "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be active or inactive"})
		return
	}
	err := s.registry.Users().Update(id, func(u *domain.User) {
		if s := strings.TrimSpace(req.Nickname); s != "" {
			u.Nickname = s
		}
		if strings.Contains(req.Email, "@") {
			u.Email = strings.TrimSpace(req.Email)
		}
		if req.Role != "" {
			u.Role = req.Role
		}
		if req.Status != "" {
			u.Status = req.Status
		}
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	user, _ := s.registry.Users().Get(id)
	c.JSON(http.StatusOK, user)
}

// deleteUser removes a user (admin only).
//
// @Summary Delete User
// @Description Permanently deletes a user account. Admin only.
// @Tags users
// @Param id path string true "User ID"
// @Success 204
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id} [delete]
func (s *Server) deleteUser(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	if err := s.registry.Users().Delete(strings.TrimSpace(c.Param("id"))); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// changeUserPassword allows an admin to reset any user's password.
//
// @Summary Reset User Password
// @Description Admin resets password for a specific user without knowing the old one.
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body domain.ChangePasswordByAdminRequest true "New password"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/users/{id}/password [put]
func (s *Server) changeUserPassword(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	var req domain.ChangePasswordByAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if len(strings.TrimSpace(req.NewPassword)) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	if err := s.registry.Users().SetPassword(id, hash); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password reset successfully"})
}

// login authenticates a user with username and password, returning a token.
//
// @Summary Login
// @Description Authenticates with username and password, returns a session token.
// @Tags users
// @Accept json
// @Produce json
// @Param request body map[string]string true "Credentials"
// @Success 200 {object} object
// @Failure 401 {object} map[string]string
// @Router /api/v1/users/login [post]
func (s *Server) login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password are required"})
		return
	}
	user, hash, found := s.registry.Users().GetByUsername(strings.TrimSpace(req.Username))
	if !found {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is " + user.Status})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":     "user:" + user.ID,
		"user":      user,
		"expiresIn": 86400 * 7,
	})
}

// requireAdminRole returns true if the authenticated user has admin role.
// Master-key callers are always admin.
func (s *Server) requireAdminRole(c *gin.Context) bool {
	if isMaster, ok := c.Get(authIsMasterKey); ok {
		if is, ok := isMaster.(bool); ok && is {
			return true
		}
	}
	uid, ok := c.Get(authUserID)
	if !ok {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return false
	}
	user, found := s.registry.Users().Get(uid.(string))
	if !found || user.Role != domain.RoleAdmin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
		return false
	}
	return true
}