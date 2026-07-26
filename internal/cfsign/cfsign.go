// Package cfsign turns an S3 object key into a time-limited CloudFront signed
// URL. The RSA private key is pulled once from Secrets Manager at startup.
package cfsign

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/cloudfront/sign"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Signer struct {
	cdnDomain string
	urlSigner *sign.URLSigner
	ttl       time.Duration
}

// New loads the private key from Secrets Manager and builds a signer.
func New(ctx context.Context, sm *secretsmanager.Client, secretARN, keyPairID, cdnDomain string, ttl time.Duration) (*Signer, error) {
	out, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretARN),
	})
	if err != nil {
		return nil, err
	}
	privKey, err := sign.LoadPEMPrivKey(strings.NewReader(aws.ToString(out.SecretString)))
	if err != nil {
		return nil, err
	}
	return &Signer{
		cdnDomain: cdnDomain,
		urlSigner: sign.NewURLSigner(keyPairID, privKey),
		ttl:       ttl,
	}, nil
}

// SignKey builds https://<cdn>/<key> (path-escaped) and signs it with a canned
// policy expiring ttl from now.
func (s *Signer) SignKey(key string) (string, error) {
	segments := strings.Split(key, "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	raw := "https://" + s.cdnDomain + "/" + strings.Join(segments, "/")
	return s.urlSigner.Sign(raw, time.Now().Add(s.ttl))
}
