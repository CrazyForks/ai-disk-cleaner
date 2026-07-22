package scanner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	modelscanner "ai-disk-cleanner/backend/model/scanner"

	"github.com/dundee/gdu/v5/build"
	"github.com/dundee/gdu/v5/pkg/analyze"
	"github.com/dundee/gdu/v5/pkg/fs"
)

func scanGDUContext(
	ctx context.Context,
	directoryPath string,
	onProgress func(modelscanner.ScanProgress),
) (*modelscanner.FileTree, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(directoryPath) == "" {
		return nil, errors.New("parse gdu directory: directory path is empty")
	}

	absoluteDirectoryPath, err := filepath.Abs(directoryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve directory path %q: %w", directoryPath, err)
	}
	info, err := os.Stat(absoluteDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("stat directory %q: %w", absoluteDirectoryPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("parse gdu directory %q: path is not a directory", absoluteDirectoryPath)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("analyze directory %q: %w", absoluteDirectoryPath, err)
	}

	analyzer := analyze.CreateAnalyzer()
	progressDone := make(chan struct{})
	if onProgress != nil {
		go func() {
			for {
				select {
				case progress := <-analyzer.GetProgressChan():
					onProgress(modelscanner.ScanProgress{
						CurrentPath: progress.CurrentItemName,
						ItemCount:   int64(progress.ItemCount),
						ScannedSize: progress.TotalSize,
					})
				case <-progressDone:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	directory := analyzer.AnalyzeDir(
		absoluteDirectoryPath,
		func(_, _ string) bool { return ctx.Err() != nil },
		func(_ string) bool { return ctx.Err() != nil },
	)
	close(progressDone)
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("analyze directory %q: %w", absoluteDirectoryPath, err)
	}
	if directory == nil {
		return nil, fmt.Errorf("analyze directory %q: gdu returned no directory", absoluteDirectoryPath)
	}
	directory.UpdateStats(make(fs.HardLinkedItems, 10))

	var reportBuffer bytes.Buffer
	fmt.Fprintf(
		&reportBuffer,
		`[1,2,{"progname":"gdu","progver":%q,"timestamp":%d},`+"\n",
		build.Version,
		time.Now().Unix(),
	)
	if err := directory.EncodeJSON(&reportBuffer, true); err != nil {
		return nil, fmt.Errorf("encode analysis for directory %q: %w", absoluteDirectoryPath, err)
	}
	if _, err := reportBuffer.WriteString("]\n"); err != nil {
		return nil, fmt.Errorf("encode analysis for directory %q: %w", absoluteDirectoryPath, err)
	}

	tree, err := parseGDUJSON(&reportBuffer)
	if err != nil {
		return nil, fmt.Errorf("parse analysis for directory %q: %w", absoluteDirectoryPath, err)
	}
	return tree, nil
}
