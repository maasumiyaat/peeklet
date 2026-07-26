// Package session issues and verifies the Bearer JWTs used for auth.
//
// Two roles: admin (the share creator) and viewer (an OTP-verified recipient).
// Viewer tokens carry the slug + granted prefix so the list handler can scope
// access without another DynamoDB read on every request.
package session

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleViewer Role = "viewer"
)

var ErrInvalid = errors.New("invalid or expired token")

type Claims struct {
	Role   Role   `json:"role"`
	Slug   string `json:"slug,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	jwt.RegisteredClaims
}

type Manager struct{ secret []byte }

func NewManager(secret string) *Manager { return &Manager{secret: []byte(secret)} }

// Issue signs a token that expires ttl from now.
func (m *Manager) Issue(c Claims, ttl time.Duration) (string, error) {
	return m.IssueUntil(c, time.Now().Add(ttl))
}

// IssueUntil signs a token that expires at an absolute time. Used for viewer
// tokens whose expiry is capped at the share's own expiry.
func (m *Manager) IssueUntil(c Claims, exp time.Time) (string, error) {
	now := time.Now()
	c.RegisteredClaims.IssuedAt = jwt.NewNumericDate(now)
	c.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(exp)
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// Parse validates the signature and expiry, returning the claims.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, ErrInvalid
	}
	var c Claims
	tok, err := jwt.ParseWithClaims(tokenStr, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalid
		}
		return m.secret, nil
	})
	if err != nil || !tok.Valid {
		return nil, ErrInvalid
	}
	return &c, nil
}
