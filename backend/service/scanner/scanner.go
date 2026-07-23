package scanner

import (
	"context"

	modelscanner "ai-disk-cleanner/backend/model/scanner"
)

// Service exposes disk scanning operations.
type Service struct{}

// NewService creates the scanner service for the central service manager.
func NewService() *Service {
	return &Service{}
}

// ParseGDUFile scans directoryPath with gdu and builds an in-memory tree.
// Its argument is a directory to scan; no JSON file is read or written.
func (service *Service) ParseGDUFile(directoryPath string) (*modelscanner.FileTree, error) {
	return scanGDUContext(context.Background(), directoryPath, nil)
}

// ParseGDU scans directoryPath with gdu, keeps the generated JSON report in
// memory, and parses it into a FileTree.
func (service *Service) ParseGDU(directoryPath string) (*modelscanner.FileTree, error) {
	return scanGDUContext(context.Background(), directoryPath, nil)
}

// ParseGDUContext scans a directory using gdu while observing cancellation and
// reporting the progress information exposed by gdu's analyzer.
func (service *Service) ParseGDUContext(
	ctx context.Context,
	directoryPath string,
	onProgress func(modelscanner.ScanProgress),
) (*modelscanner.FileTree, error) {
	return scanGDUContext(ctx, directoryPath, onProgress)
}
