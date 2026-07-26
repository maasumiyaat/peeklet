// Package config loads all runtime tunables from environment variables.
// Everything here is set by the SAM template (see template.yaml Parameters).
// Change a value there, redeploy, and it takes effect on the next cold start.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Wiring (set from other CloudFormation resources)
	TableName   string
	MediaBucket string
	CDNDomain   string // CloudFront domain, e.g. dxxxx.cloudfront.net

	// CloudFront signed-URL signing material
	CFKeyPairID           string
	CFPrivateKeySecretARN string

	// App auth
	JWTSecret         string
	OwnerPasswordHash string
	AllowedOrigin     string

	// Lifetimes
	LinkTTL     time.Duration // default expiry stamped on new shares
	SessionTTL  time.Duration // viewer browsing session
	MediaURLTTL time.Duration // signed-URL validity (<= SessionTTL)
	CDNCacheTTL time.Duration // informational; enforced by CloudFront

	// OTP
	OTPLength      int
	OTPAlphabet    string
	OTPMaxAttempts int
	OTPLockout     time.Duration

	// Listing
	AllowedExt   map[string]bool
	ListPageSize int32
}

// Load reads and validates the environment. It returns an error if any
// required secret/wiring value is missing so the Lambda fails fast at startup.
func Load() (*Config, error) {
	c := &Config{
		TableName:             os.Getenv("TABLE_NAME"),
		MediaBucket:           os.Getenv("MEDIA_BUCKET"),
		CDNDomain:             os.Getenv("CDN_DOMAIN"),
		CFKeyPairID:           os.Getenv("CF_KEY_PAIR_ID"),
		CFPrivateKeySecretARN: os.Getenv("CF_PRIVATE_KEY_SECRET_ARN"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		OwnerPasswordHash:     os.Getenv("OWNER_PASSWORD_HASH"),
		AllowedOrigin:         envStr("ALLOWED_ORIGIN", "*"),
		OTPAlphabet:           envStr("OTP_ALPHABET", "ABCDEFGHJKMNPQRSTUVWXYZ23456789"),
	}

	var err error
	if c.LinkTTL, err = envDur("LINK_TTL", 360*time.Hour); err != nil {
		return nil, err
	}
	if c.SessionTTL, err = envDur("SESSION_TTL", 2*time.Hour); err != nil {
		return nil, err
	}
	if c.MediaURLTTL, err = envDur("MEDIA_URL_TTL", c.SessionTTL); err != nil {
		return nil, err
	}
	if c.OTPLockout, err = envDur("OTP_LOCKOUT", 15*time.Minute); err != nil {
		return nil, err
	}

	c.CDNCacheTTL = time.Duration(envInt("CDN_CACHE_TTL", 86400)) * time.Second
	c.OTPLength = envInt("OTP_LENGTH", 8)
	c.OTPMaxAttempts = envInt("OTP_MAX_ATTEMPTS", 5)
	c.ListPageSize = int32(envInt("LIST_PAGE_SIZE", 100))

	c.AllowedExt = map[string]bool{}
	for _, e := range strings.Split(envStr("ALLOWED_EXT", "jpg,jpeg,png,webp,gif,mp4,webm,mov"), ",") {
		if e = strings.TrimSpace(strings.ToLower(e)); e != "" {
			c.AllowedExt[e] = true
		}
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// IsMediaAllowed reports whether a key's extension is a viewable image/video.
func (c *Config) IsMediaAllowed(key string) bool {
	i := strings.LastIndexByte(key, '.')
	if i < 0 {
		return false
	}
	return c.AllowedExt[strings.ToLower(key[i+1:])]
}

func (c *Config) validate() error {
	required := map[string]string{
		"TABLE_NAME":                c.TableName,
		"MEDIA_BUCKET":              c.MediaBucket,
		"CDN_DOMAIN":                c.CDNDomain,
		"CF_KEY_PAIR_ID":            c.CFKeyPairID,
		"CF_PRIVATE_KEY_SECRET_ARN": c.CFPrivateKeySecretARN,
		"JWT_SECRET":                c.JWTSecret,
		"OWNER_PASSWORD_HASH":       c.OwnerPasswordHash,
	}
	for name, v := range required {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("config: required env %s is empty", name)
		}
	}
	if c.OTPLength < 4 {
		return fmt.Errorf("config: OTP_LENGTH must be >= 4")
	}
	if c.MediaURLTTL > c.SessionTTL {
		return fmt.Errorf("config: MEDIA_URL_TTL (%s) must be <= SESSION_TTL (%s)", c.MediaURLTTL, c.SessionTTL)
	}
	return nil
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func envDur(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("config: %s is not a valid duration: %w", key, err)
	}
	return d, nil
}
