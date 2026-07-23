package cleaninghistory

import (
	"context"

	appctx "ai-disk-cleanner/backend/ctx"
)

// Service performs startup maintenance for persisted cleaning history.
type Service struct {
	ctx   context.Context
	store store
}

// NewService creates the cleaning history service for the central service manager.
func NewService(store store) *Service {
	return newService(appctx.GetContext(), store)
}

func newService(ctx context.Context, store store) *Service {
	return &Service{ctx: ctx, store: store}
}

// CleanupOnStartup repairs interrupted tasks and enforces the configured history limit.
func (service *Service) CleanupOnStartup() error {
	if err := service.store.MarkInterruptedCleaningRecords(service.ctx); err != nil {
		return err
	}
	maxCount, err := service.recordMaxCount()
	if err != nil {
		return err
	}
	return service.store.DeleteOldCleaningRecords(service.ctx, maxCount)
}
