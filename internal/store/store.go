// Package store is the DynamoDB persistence layer for share records.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ErrNotFound is returned by Get when no share exists for the slug.
var ErrNotFound = errors.New("share not found")

// Share is one OTP-gated link to an S3 folder.
type Share struct {
	Slug           string `dynamodbav:"slug"`
	Prefix         string `dynamodbav:"prefix"`  // access ceiling; children reachable, parent not
	OTPHash        string `dynamodbav:"otpHash"` // hex(PBKDF2-HMAC-SHA256(otp, salt))
	OTPSalt        string `dynamodbav:"otpSalt"` // hex random salt
	Label          string `dynamodbav:"label,omitempty"`
	CreatedAt      int64  `dynamodbav:"createdAt"` // epoch seconds
	ExpiresAt      int64  `dynamodbav:"expiresAt"` // epoch seconds; the real expiry check
	FailedAttempts int    `dynamodbav:"failedAttempts"`
	LockUntil      int64  `dynamodbav:"lockUntil"` // epoch seconds; 0 = not locked
	TTL            int64  `dynamodbav:"ttl"`       // = ExpiresAt; DynamoDB TTL cleanup
}

// Expired reports whether the share is past its expiry as of now.
func (s *Share) Expired(now time.Time) bool { return now.Unix() >= s.ExpiresAt }

// Locked reports whether OTP verification is currently locked out.
func (s *Share) Locked(now time.Time) bool { return s.LockUntil > now.Unix() }

type Store struct {
	db    *dynamodb.Client
	table string
}

func New(db *dynamodb.Client, table string) *Store {
	return &Store{db: db, table: table}
}

func (s *Store) Put(ctx context.Context, sh *Share) error {
	item, err := attributevalue.MarshalMap(sh)
	if err != nil {
		return err
	}
	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item:      item,
	})
	return err
}

func (s *Store) Get(ctx context.Context, slug string) (*Share, error) {
	out, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key:       map[string]types.AttributeValue{"slug": &types.AttributeValueMemberS{Value: slug}},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var sh Share
	if err := attributevalue.UnmarshalMap(out.Item, &sh); err != nil {
		return nil, err
	}
	return &sh, nil
}

func (s *Store) Delete(ctx context.Context, slug string) error {
	_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.table),
		Key:       map[string]types.AttributeValue{"slug": &types.AttributeValueMemberS{Value: slug}},
	})
	return err
}

// List returns all shares. Admin-only, low volume, so a Scan is fine.
func (s *Store) List(ctx context.Context) ([]Share, error) {
	var shares []Share
	p := dynamodb.NewScanPaginator(s.db, &dynamodb.ScanInput{TableName: aws.String(s.table)})
	for p.HasMorePages() {
		out, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		var batch []Share
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &batch); err != nil {
			return nil, err
		}
		shares = append(shares, batch...)
	}
	return shares, nil
}

// IncrementFailure atomically bumps failedAttempts and returns the new count.
func (s *Store) IncrementFailure(ctx context.Context, slug string) (int, error) {
	out, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        aws.String(s.table),
		Key:              map[string]types.AttributeValue{"slug": &types.AttributeValueMemberS{Value: slug}},
		UpdateExpression: aws.String("ADD failedAttempts :one"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, err
	}
	var res struct {
		FailedAttempts int `dynamodbav:"failedAttempts"`
	}
	if err := attributevalue.UnmarshalMap(out.Attributes, &res); err != nil {
		return 0, err
	}
	return res.FailedAttempts, nil
}

// SetLock locks the slug until the given epoch second.
func (s *Store) SetLock(ctx context.Context, slug string, until int64) error {
	return s.setNumbers(ctx, slug, map[string]int64{"lockUntil": until})
}

// ClearFailures resets the brute-force counters after a successful verify.
func (s *Store) ClearFailures(ctx context.Context, slug string) error {
	return s.setNumbers(ctx, slug, map[string]int64{"failedAttempts": 0, "lockUntil": 0})
}

func (s *Store) setNumbers(ctx context.Context, slug string, fields map[string]int64) error {
	names := map[string]string{}
	values := map[string]types.AttributeValue{}
	expr := "SET "
	i := 0
	for k, v := range fields {
		nk, vk := "#f"+itoa(i), ":v"+itoa(i)
		names[nk] = k
		values[vk] = &types.AttributeValueMemberN{Value: itoa64(v)}
		if i > 0 {
			expr += ", "
		}
		expr += nk + " = " + vk
		i++
	}
	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(s.table),
		Key:                       map[string]types.AttributeValue{"slug": &types.AttributeValueMemberS{Value: slug}},
		UpdateExpression:          aws.String(expr),
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
	})
	return err
}

func itoa(i int) string { return itoa64(int64(i)) }
func itoa64(i int64) string {
	// small helper to avoid importing strconv just for this
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
