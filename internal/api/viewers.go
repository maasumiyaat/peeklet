package api

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"peeklet/internal/otp"
	"peeklet/internal/s3list"
	"peeklet/internal/session"
	"peeklet/internal/store"
)

// POST /api/{slug}/verify  { "otp": "..." }
//
//	-> { "token", "root", "expiresAt" }
func (s *Server) handleVerify(ctx context.Context, req events.LambdaFunctionURLRequest, slug string) (events.LambdaFunctionURLResponse, error) {
	sh, err := s.store.Get(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return s.Error(404, "not found")
	}
	if err != nil {
		return s.Error(500, "store error")
	}

	now := time.Now()
	if sh.Expired(now) {
		return s.Error(410, "link expired")
	}
	if sh.Locked(now) {
		return s.Error(429, "too many attempts, try again later")
	}

	var body struct {
		OTP string `json:"otp"`
	}
	if err := s.decode(req, &body); err != nil {
		return s.Error(400, "invalid body")
	}

	if !otp.Verify(body.OTP, sh.OTPHash, sh.OTPSalt) {
		count, err := s.store.IncrementFailure(ctx, slug)
		if err == nil && count >= s.cfg.OTPMaxAttempts {
			_ = s.store.SetLock(ctx, slug, now.Add(s.cfg.OTPLockout).Unix())
		}
		return s.Error(401, "invalid code")
	}
	_ = s.store.ClearFailures(ctx, slug)

	// Viewer session expiry is capped at the share's own expiry.
	exp := now.Add(s.cfg.SessionTTL)
	if e := time.Unix(sh.ExpiresAt, 0); e.Before(exp) {
		exp = e
	}
	tok, err := s.sessions.IssueUntil(session.Claims{
		Role:   session.RoleViewer,
		Slug:   slug,
		Prefix: sh.Prefix,
	}, exp)
	if err != nil {
		return s.Error(500, "token error")
	}

	return s.JSON(200, map[string]any{
		"token":     tok,
		"root":      sh.Prefix,
		"expiresAt": sh.ExpiresAt,
	})
}

// GET /api/{slug}/list?path=...&token=...
//
//	-> { path, parent, folders:[{name,path}], files:[{name,url,type}], nextToken }
func (s *Server) handleList(ctx context.Context, req events.LambdaFunctionURLRequest, slug string) (events.LambdaFunctionURLResponse, error) {
	claims, err := s.requireViewer(req)
	if err != nil {
		return s.Error(401, "unauthorized")
	}
	if claims.Slug != slug {
		return s.Error(403, "forbidden")
	}

	// Re-check the share hasn't expired or been revoked since the token issued.
	sh, err := s.store.Get(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return s.Error(404, "not found")
	}
	if err != nil {
		return s.Error(500, "store error")
	}
	if sh.Expired(time.Now()) {
		return s.Error(410, "link expired")
	}

	current, err := s3list.Resolve(claims.Prefix, req.QueryStringParameters["path"])
	if err != nil {
		return s.Error(403, "outside granted folder")
	}

	res, err := s3list.List(ctx, s.s3, s.cfg.MediaBucket, current, req.QueryStringParameters["token"], s.cfg.ListPageSize)
	if err != nil {
		return s.Error(502, "listing error")
	}

	folders := make([]map[string]string, 0, len(res.Folders))
	for _, f := range res.Folders {
		folders = append(folders, map[string]string{"name": f.Name, "path": f.Path})
	}

	files := make([]map[string]string, 0, len(res.Files))
	for _, f := range res.Files {
		if !s.cfg.IsMediaAllowed(f.Name) {
			continue
		}
		signed, err := s.signer.SignKey(f.Key)
		if err != nil {
			return s.Error(500, "signing error")
		}
		files = append(files, map[string]string{
			"name": f.Name,
			"url":  signed,
			"type": mediaType(f.Name),
		})
	}

	return s.JSON(200, map[string]any{
		"path":      current,
		"parent":    s3list.Parent(claims.Prefix, current),
		"folders":   folders,
		"files":     files,
		"nextToken": res.NextToken,
	})
}
