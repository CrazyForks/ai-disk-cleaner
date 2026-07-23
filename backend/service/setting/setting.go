package setting

import (
	"context"

	appctx "ai-disk-cleanner/backend/ctx"
	settingmodel "ai-disk-cleanner/backend/data/models/setting"
)

// Service exposes system configuration operations.
type Service struct {
	ctx   context.Context
	store settingStore
}

// NewService creates the setting service for the central service manager.
func NewService(store settingStore) *Service {
	return newService(appctx.GetContext(), store)
}

func newService(ctx context.Context, store settingStore) *Service {
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
