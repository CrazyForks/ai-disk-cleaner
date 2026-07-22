package scanner

import (
	"context"

	modelscanner "ai-disk-cleanner/backend/model/scanner"
)

// ParseGDUFile scans directoryPath with gdu and builds an in-memory tree.
// Its argument is a directory to scan; no JSON file is read or written.
func ParseGDUFile(directoryPath string) (*modelscanner.FileTree, error) {
	return scanGDUContext(context.Background(), directoryPath, nil)
}

// ParseGDU scans directoryPath with gdu, keeps the generated JSON report in
// memory, and parses it into a FileTree.
func ParseGDU(directoryPath string) (*modelscanner.FileTree, error) {
	return scanGDUContext(context.Background(), directoryPath, nil)
}

// ParseGDUContext scans a directory using gdu while observing cancellation and
// reporting the progress information exposed by gdu's analyzer.
func ParseGDUContext(
	ctx context.Context,
	directoryPath string,
	onProgress func(modelscanner.ScanProgress),
) (*modelscanner.FileTree, error) {
	return scanGDUContext(ctx, directoryPath, onProgress)
}
