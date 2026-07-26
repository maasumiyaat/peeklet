// Package s3list lists a folder's immediate subfolders and files, and enforces
// that navigation stays within the granted prefix (children yes, parent no).
package s3list

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ErrOutsidePrefix means the requested path escaped the granted folder.
var ErrOutsidePrefix = errors.New("path outside granted folder")

// Resolve normalizes a client-supplied path and guarantees it stays within
// granted. granted must be a "foo/bar/" style prefix (trailing slash, no
// leading slash). An empty request resolves to the granted root.
func Resolve(granted, requested string) (string, error) {
	if requested == "" {
		return granted, nil
	}
	// path.Clean on an absolute path collapses ".." and can never climb above
	// root, so traversal attacks become harmless before the prefix check.
	p := strings.TrimPrefix(path.Clean("/"+requested), "/")
	if p == "" {
		return granted, nil
	}
	p += "/"
	if p != granted && !strings.HasPrefix(p, granted) {
		return "", ErrOutsidePrefix
	}
	return p, nil
}

// Parent returns the parent prefix, or "" when current is at (or somehow above)
// the granted ceiling — which is how "cannot go to parent" is enforced.
func Parent(granted, current string) string {
	if current == granted || !strings.HasPrefix(current, granted) {
		return ""
	}
	trimmed := strings.TrimSuffix(current, "/")
	idx := strings.LastIndex(trimmed, "/")
	if idx < 0 {
		return ""
	}
	parent := trimmed[:idx+1]
	if len(parent) < len(granted) {
		return ""
	}
	return parent
}

type Folder struct {
	Name string
	Path string
}

type File struct {
	Name string
	Key  string
}

type Result struct {
	Folders   []Folder
	Files     []File
	NextToken string
}

// List returns one page of immediate children under prefix.
func List(ctx context.Context, client *s3.Client, bucket, prefix, token string, maxKeys int32) (*Result, error) {
	in := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(maxKeys),
	}
	if token != "" {
		in.ContinuationToken = aws.String(token)
	}
	out, err := client.ListObjectsV2(ctx, in)
	if err != nil {
		return nil, err
	}

	res := &Result{}
	for _, cp := range out.CommonPrefixes {
		full := aws.ToString(cp.Prefix)
		name := strings.TrimSuffix(strings.TrimPrefix(full, prefix), "/")
		res.Folders = append(res.Folders, Folder{Name: name, Path: full})
	}
	for _, obj := range out.Contents {
		key := aws.ToString(obj.Key)
		if key == prefix || strings.HasSuffix(key, "/") {
			continue // the folder's own placeholder object, if any
		}
		res.Files = append(res.Files, File{Name: strings.TrimPrefix(key, prefix), Key: key})
	}
	if aws.ToBool(out.IsTruncated) {
		res.NextToken = aws.ToString(out.NextContinuationToken)
	}
	return res, nil
}
