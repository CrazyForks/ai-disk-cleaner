package cleaninghistory

import "context"

// Service performs startup maintenance for persisted cleaning history.
type Service struct {
	ctx   context.Context
	store store
}

func NewService(ctx context.Context, store store) *Service {
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
