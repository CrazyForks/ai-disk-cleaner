package main

import (
	"ai-disk-cleanner/backend/model"
	"ai-disk-cleanner/backend/util"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ai-disk-cleanner/backend/data"
	"ai-disk-cleanner/backend/data/models/cleaningrecord"
	"ai-disk-cleanner/backend/data/models/migration"
	"ai-disk-cleanner/backend/data/models/setting"
	"ai-disk-cleanner/backend/service/analyzer"
	"ai-disk-cleanner/backend/service/cleaner"
	"ai-disk-cleanner/backend/service/cleaninghistory"
	migrationservice "ai-disk-cleanner/backend/service/migration"
	settingservice "ai-disk-cleanner/backend/service/setting"

	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx              context.Context
	cleaningService  *cleaner.Service
	migrationService *migrationservice.Service
	settingService   *settingservice.Service
	startupError     error
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	store, err := data.GetStore()
	if err != nil {
		a.startupError = err
		return
	}
	if err := cleaninghistory.NewService(ctx, store).CleanupOnStartup(); err != nil {
		a.startupError = err
		return
	}
	a.cleaningService = cleaner.NewService(
		ctx,
		store.Store,
		analyzer.NewAnalyzer(store),
		func(eventName string, payload any) {
			runtime.EventsEmit(ctx, eventName, payload)
		},
	)
	a.migrationService = migrationservice.NewService(ctx, store)
	a.settingService = settingservice.NewService(ctx, store)
}

func (a *App) shutdown(_ context.Context) {
	if a.cleaningService == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = a.cleaningService.Close(ctx)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// SelectDirectory opens the operating system directory picker.
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择要扫描的目录",
	})
}

// SelectMigrationDirectory opens the directory picker for a migration target.
func (a *App) SelectMigrationDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择迁移目标文件夹",
	})
}

// GetDisks returns the disks currently available to scan.
func (a *App) GetDisks() ([]model.DiskInfo, error) {
	return util.ListDisks()
}

// IsRunningAsAdministrator reports whether the current process is elevated.
func (a *App) IsRunningAsAdministrator() bool {
	return util.IsRunningAsAdministrator()
}

// OpenTrashFileDirectory opens the directory containing a trash file.
func (a *App) OpenTrashFileDirectory(path string) error {
	target := filepath.Clean(path)
	info, err := os.Stat(target)
	if err == nil && info.IsDir() {
		return browser.OpenFile(target)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect path: %w", err)
	}
	return browser.OpenFile(filepath.Dir(target))
}

func (a *App) StartCleaning(path string, language string) (*cleaner.CleaningTaskSnapshot, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.cleaningService.StartCleaning(path, language)
}

func (a *App) StopCleaning(recordID int64) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.cleaningService.StopCleaning(recordID)
}

func (a *App) GetActiveCleaning() (*cleaner.CleaningTaskSnapshot, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.cleaningService.GetActiveCleaning(), nil
}

func (a *App) ListCleaningRecords() ([]cleaningrecord.CleaningRecord, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.cleaningService.ListCleaningRecords()
}

func (a *App) DeleteTrashFiles(recordID int64, paths []string, keepOriginalDirectories bool) ([]cleaner.DeleteFailure, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.cleaningService.DeleteTrashFiles(recordID, paths, keepOriginalDirectories)
}

func (a *App) CreateMigration(
	source string,
	destinationDirectory string,
	name string,
) (*migration.Migration, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.migrationService.Create(source, destinationDirectory, name)
}

func (a *App) CopyMigrationSource(
	source string,
	destinationDirectory string,
	name string,
) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	return a.migrationService.CopySource(source, destinationDirectory, name)
}

func (a *App) DeleteMigrationSource(source string, dest string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.migrationService.DeleteSource(source, dest)
}

func (a *App) CreateMigrationLink(
	source string,
	dest string,
	name string,
) (*migration.Migration, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.migrationService.CreateLink(source, dest, name)
}

func (a *App) ListMigrations() ([]migration.Migration, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.migrationService.List()
}

func (a *App) RestoreMigration(id int64) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.migrationService.Restore(id)
}

func (a *App) ListSettings() ([]setting.Setting, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.settingService.List()
}

func (a *App) SaveSettings(settings []setting.Setting) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.settingService.Save(settings)
}

func (a *App) ready() error {
	if a.startupError != nil {
		return fmt.Errorf("application startup: %w", a.startupError)
	}
	if a.cleaningService == nil || a.migrationService == nil || a.settingService == nil {
		return errors.New("application is not ready")
	}
	return nil
}
