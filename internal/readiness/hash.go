package readiness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

// fileDigest identifies observed on-disk bytes, not a loaded image, publisher,
// script interpreter or launcher target version. No program is executed.
type fileDigest struct {
	Path       string
	SHA256     string
	Size       int64
	Executable bool
}

func hashFile(ctx context.Context, path string, limit int64, allowRedirect bool) (fileDigest, error) {
	digest := sha256.New()
	resolved, info, err := scanRegular(ctx, path, limit, allowRedirect, digest)
	if err != nil {
		return fileDigest{}, err
	}
	return fileDigest{Path: resolved, SHA256: hex.EncodeToString(digest.Sum(nil)), Size: info.Size(), Executable: info.Mode()&0111 != 0}, nil
}
