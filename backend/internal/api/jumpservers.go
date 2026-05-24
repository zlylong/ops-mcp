package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zlylong/darwin-ops-mcp/backend/internal/domain"
)

// ssrfBlocklist hosts and IP ranges that are forbidden as JumpServer targets
// to prevent SSRF attacks against cloud metadata services and internal networks.
var ssrfBlocklist = []net.IPNet{
	// Loopback
	{Mask: net.CIDRMask(8, 32), IP: net.IP{127, 0, 0, 0}},
	// Link-local (169.254.x.x) — used by cloud metadata services
	{Mask: net.CIDRMask(16, 32), IP: net.IP{169, 254, 0, 0}},
	// Private 10.0.0.0/8
	{Mask: net.CIDRMask(8, 32), IP: net.IP{10, 0, 0, 0}},
	// Private 172.16.0.0/12
	{Mask: net.CIDRMask(12, 32), IP: net.IP{172, 16, 0, 0}},
	// Private 192.168.0.0/16
	{Mask: net.CIDRMask(16, 32), IP: net.IP{192, 168, 0, 0}},
	// CGNAT 100.64.0.0/10 (carrier-grade NAT)
	{Mask: net.CIDRMask(10, 32), IP: net.IP{100, 64, 0, 0}},
	// Any IP (0.0.0.0/0 already blocked by being non-routable, kept for explicitness)
}

// isSSRFHost returns true if the host is known to be an internal/metadata address.
func isSSRFHost(host string) bool {
	// Check raw hostname strings used by cloud metadata services.
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "localhost", "metadata.google.internal", "metadata.internal":
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, block := range ssrfBlocklist {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// isAllowedProbePort returns true if the port is allowed for connectivity probes.
// Only 80 (HTTP) and 443 (HTTPS) are permitted.
func isAllowedProbePort(port int) bool {
	return port == 80 || port == 443
}

// validateProbeURL checks a full URL for SSRF readiness.
// It ensures the scheme is http/https, the port is 80 or 443, and the host
// is not in the SSRF blocklist. Returns an error describing the violation.
func validateProbeURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("invalid probe URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("probe URL scheme must be http or https")
	}
	host, port, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		// If no port in URL, net.SplitHostPort returns an error and host=parsed.Host.
		host = parsed.Host
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if host == "" {
		return errors.New("probe URL has no host")
	}
	if !isAllowedProbePort(portFromString(port)) {
		return errors.New("probe URL port must be 80 or 443")
	}
	if isSSRFHost(host) {
		return errors.New("probe URL host is in the SSRF blocklist")
	}
	return nil
}

func portFromString(s string) int {
	var p int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		p = p*10 + int(c-'0')
	}
	return p
}

func normalizeJumpServerURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("baseUrl is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("baseUrl must be a valid absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("baseUrl scheme must be http or https")
	}
	// Enforce SSRF protection on the stored BaseURL as well.
	if err := validateProbeURL(raw); err != nil {
		return "", errors.New("baseUrl: " + err.Error())
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func validateJumpServerRequest(req domain.JumpServerInstanceRequest, create bool) (domain.JumpServerInstanceRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Version = strings.TrimSpace(req.Version)
	req.Status = strings.TrimSpace(req.Status)
	req.Description = strings.TrimSpace(req.Description)
	if create && req.Name == "" {
		return req, errors.New("name is required")
	}
	if create || strings.TrimSpace(req.BaseURL) != "" {
		baseURL, err := normalizeJumpServerURL(req.BaseURL)
		if err != nil {
			return req, err
		}
		req.BaseURL = baseURL
	}
	if req.AuthType == "" {
		req.AuthType = domain.JumpServerAuthToken
	}
	switch req.AuthType {
	case domain.JumpServerAuthSession, domain.JumpServerAuthToken, domain.JumpServerAuthPrivateToken, domain.JumpServerAuthAccessKey:
	default:
		return req, errors.New("authType must be session, token, private_token, or access_key")
	}
	if req.Status == "" {
		req.Status = "active"
	}
	switch req.Status {
	case "active", "inactive", "unreachable":
	default:
		return req, errors.New("status must be active, inactive, or unreachable")
	}
	return req, nil
}

func (s *Server) listJumpServers(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	c.JSON(http.StatusOK, s.registry.JumpServers().List())
}

func (s *Server) getJumpServer(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	item, ok := s.registry.JumpServers().Get(strings.TrimSpace(c.Param("id")))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "jumpserver instance not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) createJumpServer(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	var req domain.JumpServerInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	req, err := validateJumpServerRequest(req, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := s.registry.JumpServers().Add(domain.JumpServerInstance{Name: req.Name, BaseURL: req.BaseURL, Version: req.Version, AuthType: req.AuthType, Status: req.Status, Description: req.Description}, strings.TrimSpace(req.Credential), strings.TrimSpace(req.AccessKeyID), strings.TrimSpace(req.AccessKeySecret))
	c.JSON(http.StatusCreated, item)
}

func (s *Server) updateJumpServer(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	var req domain.JumpServerInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	req, err := validateJumpServerRequest(req, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.registry.JumpServers().Update(strings.TrimSpace(c.Param("id")), func(j *domain.JumpServerInstance) {
		if req.Name != "" {
			j.Name = req.Name
		}
		if req.BaseURL != "" {
			j.BaseURL = req.BaseURL
		}
		if req.Version != "" {
			j.Version = req.Version
		}
		j.AuthType = req.AuthType
		j.Status = req.Status
		j.Description = req.Description
	}, strings.TrimSpace(req.Credential), strings.TrimSpace(req.AccessKeyID), strings.TrimSpace(req.AccessKeySecret))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) deleteJumpServer(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	if err := s.registry.JumpServers().Delete(strings.TrimSpace(c.Param("id"))); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// testJumpServer probes the JumpServer base URL for connectivity.
//
// SECURITY: The URL was already validated at creation/update time by
// normalizeJumpServerURL which enforces the SSRF blocklist and port restrictions.
// An additional runtime check is added here as a defence-in-depth measure.
func (s *Server) testJumpServer(c *gin.Context) {
	if !s.requireAdminRole(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	item, ok := s.registry.JumpServers().Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "jumpserver instance not found"})
		return
	}
	checkedAt := time.Now().UTC()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	probeURL := strings.TrimRight(item.BaseURL, "/") + "/api/docs/"

	// Defence-in-depth: validate the URL before probing.
	if err := validateProbeURL(probeURL); err != nil {
		_, _ = s.registry.JumpServers().MarkChecked(id, "unreachable", checkedAt)
		c.JSON(http.StatusOK, domain.JumpServerConnectionCheck{ID: item.ID, Name: item.Name, BaseURL: item.BaseURL, Reachable: false, Status: "unreachable", Message: "ssrf blocked: " + err.Error(), CheckedAt: checkedAt})
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		_, _ = s.registry.JumpServers().MarkChecked(id, "unreachable", checkedAt)
		c.JSON(http.StatusOK, domain.JumpServerConnectionCheck{ID: item.ID, Name: item.Name, BaseURL: item.BaseURL, Reachable: false, Status: "unreachable", Message: err.Error(), CheckedAt: checkedAt})
		return
	}
	_ = res.Body.Close()
	reachable := res.StatusCode < 500
	status := "active"
	message := "JumpServer API docs endpoint reachable"
	if !reachable {
		status = "unreachable"
		message = res.Status
	}
	_, _ = s.registry.JumpServers().MarkChecked(id, status, checkedAt)
	c.JSON(http.StatusOK, domain.JumpServerConnectionCheck{ID: item.ID, Name: item.Name, BaseURL: item.BaseURL, Reachable: reachable, Status: status, Message: message, CheckedAt: checkedAt})
}
