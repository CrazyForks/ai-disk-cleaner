package migration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	migrationmodel "ai-disk-cleanner/backend/data/models/migration"
)

type memoryStore struct {
	records         map[int64]*migrationmodel.Migration
	migratedSources []string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{records: make(map[int64]*migrationmodel.Migration)}
}

func (store *memoryStore) CreateMigration(_ context.Context, record *migrationmodel.Migration) error {
	if record.ID == 0 {
		record.ID = time.Now().UnixMilli()
		record.CreatedAt = time.Now()
	}
	copy := *record
	store.records[record.ID] = &copy
	return nil
}

func (store *memoryStore) GetMigration(_ context.Context, id int64) (*migrationmodel.Migration, error) {
	record, ok := store.records[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copy := *record
	return &copy, nil
}

func (store *memoryStore) ListMigrations(context.Context) ([]migrationmodel.Migration, error) {
	records := make([]migrationmodel.Migration, 0, len(store.records))
	for _, record := range store.records {
		records = append(records, *record)
	}
	return records, nil
}

func (store *memoryStore) DeleteMigration(_ context.Context, id int64) error {
	delete(store.records, id)
	return nil
}

func (store *memoryStore) MarkTrashFileMigrated(_ context.Context, source string) error {
	store.migratedSources = append(store.migratedSources, source)
	return nil
}

func TestCreateRestoresSourceWhenSymlinkFails(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destinationDirectory := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("important"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destinationDirectory, 0o700); err != nil {
		t.Fatal(err)
	}

	service := newService(context.Background(), newMemoryStore())
	service.symlink = func(string, string) error {
		return errors.New("simulated symlink failure")
	}

	if _, err := service.Create(source, destinationDirectory, "moved.txt"); err == nil {
		t.Fatal("Create() error = nil, want symlink failure")
	}
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("source was not restored: %v", err)
	}
	if string(content) != "important" {
		t.Fatalf("restored content = %q", content)
	}
	if _, err := os.Lstat(filepath.Join(destinationDirectory, "moved.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("destination still exists, error = %v", err)
	}
}

func TestCreateAndRestoreFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destinationDirectory := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("important"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destinationDirectory, 0o700); err != nil {
		t.Fatal(err)
	}

	store := newMemoryStore()
	service := newService(context.Background(), store)
	record, err := service.Create(source, destinationDirectory, "moved.txt")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symbolic links are not enabled: %v", err)
		}
		t.Fatalf("Create() error = %v", err)
	}
	info, err := os.Lstat(source)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("source is not a symbolic link: info=%v error=%v", info, err)
	}
	content, err := os.ReadFile(record.Dest)
	if err != nil || string(content) != "important" {
		t.Fatalf("destination content = %q, error=%v", content, err)
	}

	if err := service.Restore(record.ID); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	info, err = os.Lstat(source)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("source was not restored as a regular file: info=%v error=%v", info, err)
	}
	if _, err := os.Lstat(record.Dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("destination still exists, error = %v", err)
	}
	if len(store.records) != 0 {
		t.Fatalf("migration record was not deleted: %#v", store.records)
	}
}

func TestMigrationStepsSupportRetry(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destinationDirectory := filepath.Join(root, "destination")
	if err := os.WriteFile(source, []byte("important"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destinationDirectory, 0o700); err != nil {
		t.Fatal(err)
	}

	store := newMemoryStore()
	service := newService(context.Background(), store)
	dest, err := service.CopySource(source, destinationDirectory, "moved.txt")
	if err != nil {
		t.Fatalf("CopySource() error = %v", err)
	}
	if err := service.DeleteSource(source, dest); err != nil {
		t.Fatalf("DeleteSource() error = %v", err)
	}
	if err := service.DeleteSource(source, dest); err != nil {
		t.Fatalf("DeleteSource() retry error = %v", err)
	}

	realSymlink := service.symlink
	service.symlink = func(string, string) error {
		return errors.New("simulated symlink failure")
	}
	if _, err := service.CreateLink(source, dest, "moved.txt"); err == nil {
		t.Fatal("CreateLink() error = nil, want symlink failure")
	}
	service.symlink = realSymlink

	record, err := service.CreateLink(source, dest, "moved.txt")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symbolic links are not enabled: %v", err)
		}
		t.Fatalf("CreateLink() retry error = %v", err)
	}
	retriedRecord, err := service.CreateLink(source, dest, "moved.txt")
	if err != nil {
		t.Fatalf("CreateLink() completed-step retry error = %v", err)
	}
	if retriedRecord.ID != record.ID || len(store.records) != 1 {
		t.Fatalf("CreateLink() retry created duplicate records: first=%#v retry=%#v all=%#v", record, retriedRecord, store.records)
	}
	if len(store.migratedSources) != 2 ||
		!samePath(store.migratedSources[0], source) ||
		!samePath(store.migratedSources[1], source) {
		t.Fatalf("migrated sources = %#v, want source marked on completion and retry", store.migratedSources)
	}
}

func TestDeleteSourceRequiresCompletedCopyStep(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	dest := filepath.Join(root, "unrelated.txt")
	if err := os.WriteFile(source, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("other"), 0o600); err != nil {
		t.Fatal(err)
	}

	service := newService(context.Background(), newMemoryStore())
	if err := service.DeleteSource(source, dest); err == nil {
		t.Fatal("DeleteSource() error = nil without a completed copy step")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source was changed by rejected deletion: %v", err)
	}
}
