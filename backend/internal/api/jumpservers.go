package api

import (
	"context"
	"errors"
	"fmt"
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
	// IPv4 loopback, link-local, private, CGNAT, and unspecified ranges.
	{Mask: net.CIDRMask(8, 32), IP: net.IP{127, 0, 0, 0}},
	{Mask: net.CIDRMask(16, 32), IP: net.IP{169, 254, 0, 0}},
	{Mask: net.CIDRMask(8, 32), IP: net.IP{10, 0, 0, 0}},
	{Mask: net.CIDRMask(12, 32), IP: net.IP{172, 16, 0, 0}},
	{Mask: net.CIDRMask(16, 32), IP: net.IP{192, 168, 0, 0}},
	{Mask: net.CIDRMask(10, 32), IP: net.IP{100, 64, 0, 0}},
	{Mask: net.CIDRMask(8, 32), IP: net.IP{0, 0, 0, 0}},
	// IPv6 loopback, unspecified, unique-local, and link-local ranges.
	{Mask: net.CIDRMask(128, 128), IP: net.ParseIP("::1")},
	{Mask: net.CIDRMask(128, 128), IP: net.ParseIP("::")},
	{Mask: net.CIDRMask(7, 128), IP: net.ParseIP("fc00::")},
	{Mask: net.CIDRMask(10, 128), IP: net.ParseIP("fe80::")},
}

func isBlockedProbeIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, block := range ssrfBlocklist {
		if block.Contains(ip) {
			return true
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// isSSRFHost returns true if the literal host is known to be internal/metadata.
func isSSRFHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "localhost", "metadata.google.internal", "metadata.internal":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && isBlockedProbeIP(ip)
}

func resolvePublicProbeHost(ctx context.Context, host string) ([]net.IPAddr, error) {
	if isSSRFHost(host) {
		return nil, errors.New("probe URL host is in the SSRF blocklist")
	}
	if ip := net.ParseIP(host); ip != nil {
		return []net.IPAddr{{IP: ip}}, nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("probe URL host could not be resolved")
	}
	for _, ip := range ips {
		if isBlockedProbeIP(ip.IP) {
			return nil, fmt.Errorf("probe URL host resolves to blocked address %s", ip.IP.String())
		}
	}
	return ips, nil
}

// isAllowedProbePort returns true if the port is allowed for connectivity probes.
// Only 80 (HTTP) and 443 (HTTPS) are permitted.
func isAllowedProbePort(port int) bool {
	return port == 80 || port == 443
}

func splitProbeHostPort(parsed *url.URL) (string, string, error) {
	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if host == "" {
		return "", "", errors.New("probe URL has no host")
	}
	if !isAllowedProbePort(portFromString(port)) {
		return "", "", errors.New("probe URL port must be 80 or 443")
	}
	return host, port, nil
}

// validateProbeURL checks a full URL for SSRF readiness.
// It ensures the scheme is http/https, the port is 80 or 443, and the host
// is neither blocked directly nor resolvable to blocked IP ranges.
func validateProbeURL(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("invalid probe URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("probe URL scheme must be http or https")
	}
	host, _, err := splitProbeHostPort(parsed)
	if err != nil {
		return err
	}
	_, err = resolvePublicProbeHost(ctx, host)
	return err
}

func safeProbeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := resolvePublicProbeHost(ctx, host)
			if err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return validateProbeURL(req.Context(), req.URL.String())
		},
	}
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
	if err := validateProbeURL(context.Background(), raw); err != nil {
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
	if err := validateProbeURL(ctx, probeURL); err != nil {
		_, _ = s.registry.JumpServers().MarkChecked(id, "unreachable", checkedAt)
		c.JSON(http.StatusOK, domain.JumpServerConnectionCheck{ID: item.ID, Name: item.Name, BaseURL: item.BaseURL, Reachable: false, Status: "unreachable", Message: "ssrf blocked: " + err.Error(), CheckedAt: checkedAt})
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := safeProbeHTTPClient().Do(req)
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
