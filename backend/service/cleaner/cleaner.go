package cleaner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-disk-cleanner/backend/data/models/cleaningrecord"
	modelscanner "ai-disk-cleanner/backend/model/scanner"
	serviceScanner "ai-disk-cleanner/backend/service/scanner"
)

const (
	EventTaskUpdated = "cleaning:task-updated"
	EventLLMDelta    = "cleaning:llm-delta"
)

var (
	ErrTaskRunning  = errors.New("a cleaning task is already running")
	ErrNoActiveTask = errors.New("no active cleaning task")
)

// Analyzer performs the LLM phase. Implementations must stop when ctx is cancelled.
type Analyzer interface {
	Analyze(
		ctx context.Context,
		tree *modelscanner.FileTree,
		language string,
		onDelta func(string),
	) (*cleaningrecord.AnalysisResult, error)
}

type ScanFunc func(
	context.Context,
	string,
	func(modelscanner.ScanProgress),
) (*modelscanner.FileTree, error)

type EventEmitter func(string, any)

// CleaningTaskSnapshot is the frontend-safe in-memory view of the current task.
type CleaningTaskSnapshot struct {
	ID           int64                     `json:"id"`
	StartTime    time.Time                 `json:"startTime"`
	Path         string                    `json:"path"`
	State        string                    `json:"state"`
	ErrorMessage string                    `json:"errorMessage"`
	LLMOutput    string                    `json:"llmOutput"`
	ScanProgress modelscanner.ScanProgress `json:"scanProgress"`
	Stopping     bool                      `json:"stopping"`
}

// LLMDelta is emitted for each assistant text fragment.
type LLMDelta struct {
	RecordID int64  `json:"recordId"`
	Delta    string `json:"delta"`
}

// Service owns the process-wide cleaning task and the latest scanned tree.
type Service struct {
	ctx      context.Context
	store    *cleaningrecord.Store
	analyzer Analyzer
	scan     ScanFunc
	emit     EventEmitter

	mu           sync.RWMutex
	active       *activeTask
	tree         *modelscanner.FileTree
	treeSnapshot *CleaningTaskSnapshot
}

func NewService(
	ctx context.Context,
	store *cleaningrecord.Store,
	analyzer Analyzer,
	emit EventEmitter,
) *Service {
	return NewServiceWithScanner(ctx, store, analyzer, emit, serviceScanner.ParseGDUContext)
}

func NewServiceWithScanner(
	ctx context.Context,
	store *cleaningrecord.Store,
	analyzer Analyzer,
	emit EventEmitter,
	scan ScanFunc,
) *Service {
	if ctx == nil {
		ctx = context.Background()
	}
	if emit == nil {
		emit = func(string, any) {}
	}
	if scan == nil {
		scan = serviceScanner.ParseGDUContext
	}
	return &Service{
		ctx:      ctx,
		store:    store,
		analyzer: analyzer,
		scan:     scan,
		emit:     emit,
	}
}

// StartCleaning creates the record synchronously and starts the expensive work in the background.
func (service *Service) StartCleaning(directoryPath string, language string) (*CleaningTaskSnapshot, error) {
	if service.store == nil {
		return nil, errors.New("start cleaning: record store is nil")
	}
	if service.analyzer == nil {
		return nil, errors.New("start cleaning: analyzer is nil")
	}
	absolutePath, err := validateDirectory(directoryPath)
	if err != nil {
		return nil, err
	}

	service.mu.Lock()
	if service.active != nil {
		service.mu.Unlock()
		return nil, ErrTaskRunning
	}

	record := &cleaningrecord.CleaningRecord{
		StartTime:  time.Now(),
		TrashFiles: make([]cleaningrecord.TrashFile, 0),
		TopUsages:  make([]cleaningrecord.DiskUsage, 0),
		Path:       absolutePath,
		State:      cleaningrecord.CLEANING_STATE_SCANNING,
	}
	if err := service.store.CreateCleaningRecord(service.ctx, record); err != nil {
		service.mu.Unlock()
		return nil, err
	}

	taskContext, cancel := context.WithCancel(service.ctx)
	task := &activeTask{
		snapshot: CleaningTaskSnapshot{
			ID:        record.ID,
			StartTime: record.StartTime,
			Path:      absolutePath,
			State:     cleaningrecord.CLEANING_STATE_SCANNING,
		},
		ctx:      taskContext,
		cancel:   cancel,
		done:     make(chan struct{}),
		language: language,
	}
	service.active = task
	service.tree = nil
	service.treeSnapshot = nil
	snapshot := task.snapshot
	service.mu.Unlock()

	service.emit(EventTaskUpdated, snapshot)
	go service.run(task)
	return &snapshot, nil
}

// StopCleaning requests cancellation. The task remains active until its worker exits.
func (service *Service) StopCleaning(recordID int64) error {
	service.mu.Lock()
	if service.active == nil {
		service.mu.Unlock()
		return ErrNoActiveTask
	}
	if service.active.snapshot.ID != recordID {
		service.mu.Unlock()
		return fmt.Errorf(
			"stop cleaning record %d: active record is %d",
			recordID,
			service.active.snapshot.ID,
		)
	}
	service.active.snapshot.Stopping = true
	snapshot := service.active.snapshot
	cancel := service.active.cancel
	service.mu.Unlock()

	cancel()
	service.emit(EventTaskUpdated, snapshot)
	return nil
}

func (service *Service) GetActiveCleaning() *CleaningTaskSnapshot {
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.active != nil {
		snapshot := service.active.snapshot
		return &snapshot
	}
	if service.tree == nil || service.treeSnapshot == nil {
		return nil
	}
	snapshot := *service.treeSnapshot
	return &snapshot
}

func (service *Service) ListCleaningRecords() ([]cleaningrecord.CleaningRecord, error) {
	return service.store.ListCleaningRecords(service.ctx)
}

func (service *Service) DeleteTrashFiles(
	recordID int64,
	selectedPaths []string,
	keepOriginalDirectories bool,
) error {
	if len(selectedPaths) == 0 {
		return errors.New("delete trash files: no files selected")
	}
	service.mu.RLock()
	defer service.mu.RUnlock()

	record, err := service.store.GetCleaningRecord(service.ctx, recordID)
	if err != nil {
		return err
	}
	currentRootPath := record.Path

	if !samePath(record.Path, currentRootPath) {
		return fmt.Errorf("delete trash files for record %d: scan root does not match the current in-memory tree", recordID)
	}
	selected := make(map[string]struct{}, len(selectedPaths))
	for _, path := range selectedPaths {
		selected[path] = struct{}{}
	}
	knownCandidates := make(map[string]int, len(record.TrashFiles))
	for index, candidate := range record.TrashFiles {
		knownCandidates[candidate.Path] = index
	}
	for path := range selected {
		candidateIndex, ok := knownCandidates[path]
		if !ok {
			return fmt.Errorf("delete trash file: path %q is not a candidate in record %d", path, recordID)
		}
		if record.TrashFiles[candidateIndex].IsDeleted {
			return fmt.Errorf("delete trash file: path %q has already been deleted", path)
		}
		if strings.ContainsAny(path, "*?[") {
			return fmt.Errorf("delete trash file: glob path %q requires manual review", path)
		}
	}
	var freedSize int64
	for index := range record.TrashFiles {
		candidate := &record.TrashFiles[index]
		if _, ok := selected[candidate.Path]; !ok {
			continue
		}
		target, err := safeDeleteTarget(record.Path, candidate.Path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete trash file %q: %w", target, err)
		}
		if info != nil {
			freedSize += candidate.Size
			if err := removeTrashTarget(target, info, keepOriginalDirectories); err != nil {
				return fmt.Errorf("delete trash file %q: %w", target, err)
			}
		}
		candidate.IsDeleted = true
	}
	record.FreedSize += freedSize
	return service.store.SaveDeletedTrashFiles(service.ctx, record)
}

func removeTrashTarget(target string, info os.FileInfo, keepOriginalDirectory bool) error {
	if !keepOriginalDirectory || !info.IsDir() {
		return os.RemoveAll(target)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(target, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

// Tree returns the latest completed scan tree.
func (service *Service) Tree() *modelscanner.FileTree {
	service.mu.RLock()
	defer service.mu.RUnlock()
	return service.tree
}

// Close cancels the active task and waits for it to leave the single-task slot.
func (service *Service) Close(ctx context.Context) error {
	service.mu.RLock()
	if service.active == nil {
		service.mu.RUnlock()
		return nil
	}
	cancel := service.active.cancel
	done := service.active.done
	service.mu.RUnlock()

	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
