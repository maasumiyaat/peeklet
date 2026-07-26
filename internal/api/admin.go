package api

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"golang.org/x/crypto/bcrypt"

	"peeklet/internal/otp"
	"peeklet/internal/session"
	"peeklet/internal/store"
)

// POST /admin/login  { "password": "..." } -> { "token": "..." }
func (s *Server) handleLogin(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	var body struct {
		Password string `json:"password"`
	}
	if err := s.decode(req, &body); err != nil {
		return s.Error(400, "invalid body")
	}
	if bcrypt.CompareHashAndPassword([]byte(s.cfg.OwnerPasswordHash), []byte(body.Password)) != nil {
		return s.Error(401, "invalid password")
	}
	tok, err := s.sessions.Issue(session.Claims{Role: session.RoleAdmin}, s.cfg.SessionTTL)
	if err != nil {
		return s.Error(500, "token error")
	}
	return s.JSON(200, map[string]string{"token": tok})
}

// POST /admin/shares  { "prefix": "...", "label"?: "...", "ttlOverride"?: "72h" }
//
//	-> { "slug", "url", "otp", "expiresAt" }   (otp shown once, not stored)
func (s *Server) handleCreateShare(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return s.Error(401, "unauthorized")
	}
	var body struct {
		Prefix      string `json:"prefix"`
		Label       string `json:"label"`
		TTLOverride string `json:"ttlOverride"`
	}
	if err := s.decode(req, &body); err != nil {
		return s.Error(400, "invalid body")
	}
	prefix := normalizePrefix(body.Prefix)
	if prefix == "" {
		return s.Error(400, "a valid prefix is required")
	}

	ttl := s.cfg.LinkTTL
	if body.TTLOverride != "" {
		d, err := time.ParseDuration(body.TTLOverride)
		if err != nil || d <= 0 {
			return s.Error(400, "invalid ttlOverride")
		}
		ttl = d
	}

	code, err := otp.Generate(s.cfg.OTPLength, s.cfg.OTPAlphabet)
	if err != nil {
		return s.Error(500, "otp error")
	}
	hash, salt, err := otp.Hash(code)
	if err != nil {
		return s.Error(500, "otp error")
	}
	slug, err := generateSlug(8)
	if err != nil {
		return s.Error(500, "slug error")
	}

	now := time.Now()
	exp := now.Add(ttl).Unix()
	sh := &store.Share{
		Slug:      slug,
		Prefix:    prefix,
		OTPHash:   hash,
		OTPSalt:   salt,
		Label:     body.Label,
		CreatedAt: now.Unix(),
		ExpiresAt: exp,
		TTL:       exp,
	}
	if err := s.store.Put(ctx, sh); err != nil {
		return s.Error(500, "store error")
	}

	url := trimRightSlash(s.cfg.AllowedOrigin) + "/s/" + slug
	return s.JSON(200, map[string]any{
		"slug":      slug,
		"url":       url,
		"otp":       code,
		"expiresAt": exp,
	})
}

// GET /admin/shares -> [ { slug, prefix, label, createdAt, expiresAt } ]
func (s *Server) handleListShares(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return s.Error(401, "unauthorized")
	}
	shares, err := s.store.List(ctx)
	if err != nil {
		return s.Error(500, "store error")
	}
	views := make([]map[string]any, 0, len(shares))
	for _, sh := range shares {
		views = append(views, map[string]any{
			"slug":      sh.Slug,
			"prefix":    sh.Prefix,
			"label":     sh.Label,
			"createdAt": sh.CreatedAt,
			"expiresAt": sh.ExpiresAt,
		})
	}
	return s.JSON(200, views)
}

// DELETE /admin/shares/{slug} -> { "deleted": slug }
func (s *Server) handleDeleteShare(ctx context.Context, req events.LambdaFunctionURLRequest, slug string) (events.LambdaFunctionURLResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return s.Error(401, "unauthorized")
	}
	if err := s.store.Delete(ctx, slug); err != nil {
		return s.Error(500, "store error")
	}
	return s.JSON(200, map[string]string{"deleted": slug})
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
