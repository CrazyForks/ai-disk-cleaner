package migration

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Migration records a file or directory moved behind a symbolic link.
type Migration struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement:false" json:"id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Source    string    `gorm:"column:source;not null;uniqueIndex" json:"source"`
	Dest      string    `gorm:"column:dest;not null;uniqueIndex" json:"dest"`
	CreatedAt time.Time `gorm:"column:created_at;not null" json:"createdAt"`
}

// MigrationStore exposes migration persistence over the initialized GORM database.
type MigrationStore struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *MigrationStore {
	return &MigrationStore{db: db}
}

func (Migration) TableName() string {
	return "migration"
}

// BeforeCreate uses the current time as the record ID and creation time.
func (migration *Migration) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	if migration.ID == 0 {
		migration.ID = now.UnixMilli()
	}
	if migration.CreatedAt.IsZero() {
		migration.CreatedAt = now
	}
	return nil
}

func (store *MigrationStore) CreateMigration(ctx context.Context, migration *Migration) error {
	if err := store.db.WithContext(ctx).Create(migration).Error; err != nil {
		return fmt.Errorf("create migration: %w", err)
	}
	return nil
}

func (store *MigrationStore) GetMigration(ctx context.Context, id int64) (*Migration, error) {
	var migration Migration
	if err := store.db.WithContext(ctx).First(&migration, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("get migration %d: %w", id, err)
	}
	return &migration, nil
}

func (store *MigrationStore) ListMigrations(ctx context.Context) ([]Migration, error) {
	var migrations []Migration
	if err := store.db.WithContext(ctx).Order("id DESC").Find(&migrations).Error; err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	return migrations, nil
}

func (store *MigrationStore) DeleteMigration(ctx context.Context, id int64) error {
	result := store.db.WithContext(ctx).Delete(&Migration{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete migration %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete migration %d: %w", id, gorm.ErrRecordNotFound)
	}
	return nil
}
