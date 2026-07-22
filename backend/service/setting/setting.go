package setting

import (
	"context"

	settingmodel "ai-disk-cleanner/backend/data/models/setting"
)

// Service exposes system configuration operations.
type Service struct {
	ctx   context.Context
	store settingStore
}

func NewService(ctx context.Context, store settingStore) *Service {
	return &Service{ctx: ctx, store: store}
}

func (service *Service) List() ([]settingmodel.Setting, error) {
	return service.store.ListSettings(service.ctx)
}

func (service *Service) Save(settings []settingmodel.Setting) error {
	if err := validateSettings(settings); err != nil {
		return err
	}
	return service.store.SaveSettings(service.ctx, settings)
}
