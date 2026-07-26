// Package api holds the HTTP surface: the Server struct that owns all
// dependencies, the router, and shared auth/response helpers. Handler bodies
// live in admin.go and viewer.go as methods on *Server.
package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"peeklet/internal/cfsign"
	"peeklet/internal/config"
	"peeklet/internal/session"
	"peeklet/internal/store"
)

type Server struct {
	cfg      *config.Config
	store    *store.Store
	s3       *s3.Client
	signer   *cfsign.Signer
	sessions *session.Manager
}

// NewServer builds the AWS clients and wires up every dependency. Adding new
// dependencies here is the only reason to touch this constructor.
func NewServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	sm := secretsmanager.NewFromConfig(awsCfg)
	signer, err := cfsign.New(ctx, sm, cfg.CFPrivateKeySecretARN, cfg.CFKeyPairID, cfg.CDNDomain, cfg.MediaURLTTL)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:      cfg,
		store:    store.New(dynamodb.NewFromConfig(awsCfg), cfg.TableName),
		s3:       s3.NewFromConfig(awsCfg),
		signer:   signer,
		sessions: session.NewManager(cfg.JWTSecret),
	}, nil
}

// Start begins serving Lambda Function URL requests.
func (s *Server) Start() { lambda.Start(s.handle) }

// handle routes a request. CORS preflight (OPTIONS) is handled by the Function
// URL's own Cors config, so it never reaches here.
func (s *Server) handle(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RawPath
	parts := splitPath(path)

	switch {
	case method == "GET" && path == "/health":
		return s.JSON(200, map[string]string{"status": "ok"})

	// admin
	case method == "POST" && path == "/admin/login":
		return s.handleLogin(ctx, req)
	case method == "POST" && path == "/admin/shares":
		return s.handleCreateShare(ctx, req)
	case method == "GET" && path == "/admin/shares":
		return s.handleListShares(ctx, req)
	case method == "DELETE" && len(parts) == 3 && parts[0] == "admin" && parts[1] == "shares":
		return s.handleDeleteShare(ctx, req, parts[2])

	// viewer: /api/{slug}/verify and /api/{slug}/list
	case method == "POST" && len(parts) == 3 && parts[0] == "api" && parts[2] == "verify":
		return s.handleVerify(ctx, req, parts[1])
	case method == "GET" && len(parts) == 3 && parts[0] == "api" && parts[2] == "list":
		return s.handleList(ctx, req, parts[1])

	default:
		return s.Error(404, "not found")
	}
}

// ---- request helpers ----

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// decode unmarshals the (possibly base64-encoded) request body into v.
func (s *Server) decode(req events.LambdaFunctionURLRequest, v any) error {
	raw := []byte(req.Body)
	if req.IsBase64Encoded {
		dec, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return err
		}
		raw = dec
	}
	return json.Unmarshal(raw, v)
}

func bearer(req events.LambdaFunctionURLRequest) string {
	h := req.Headers["authorization"]
	if h == "" {
		h = req.Headers["Authorization"]
	}
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return h[7:]
	}
	return ""
}

func (s *Server) requireAdmin(req events.LambdaFunctionURLRequest) (*session.Claims, error) {
	c, err := s.sessions.Parse(bearer(req))
	if err != nil || c.Role != session.RoleAdmin {
		return nil, session.ErrInvalid
	}
	return c, nil
}

func (s *Server) requireViewer(req events.LambdaFunctionURLRequest) (*session.Claims, error) {
	c, err := s.sessions.Parse(bearer(req))
	if err != nil || c.Role != session.RoleViewer {
		return nil, session.ErrInvalid
	}
	return c, nil
}

// ---- response helpers ----

func (s *Server) JSON(status int, body any) (events.LambdaFunctionURLResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return s.Error(500, "internal error")
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(b),
	}, nil
}

func (s *Server) Error(status int, msg string) (events.LambdaFunctionURLResponse, error) {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"content-type": "application/json"},
		Body:       string(b),
	}, nil
}
