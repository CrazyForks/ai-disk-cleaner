package cleaninghistory

import (
	"context"
	"reflect"
	"testing"

	settingmodel "ai-disk-cleanner/backend/data/models/setting"
)

type fakeStore struct {
	settings   []settingmodel.Setting
	calls      []string
	deletedMax int
}

func (store *fakeStore) ListSettings(context.Context) ([]settingmodel.Setting, error) {
	store.calls = append(store.calls, "list")
	return store.settings, nil
}

func (store *fakeStore) MarkInterruptedCleaningRecords(context.Context) error {
	store.calls = append(store.calls, "mark")
	return nil
}

func (store *fakeStore) DeleteOldCleaningRecords(_ context.Context, maxCount int) error {
	store.calls = append(store.calls, "delete")
	store.deletedMax = maxCount
	return nil
}

func TestCleanupOnStartupUsesConfiguredRecordLimit(t *testing.T) {
	store := &fakeStore{settings: []settingmodel.Setting{
		{Key: recordMaxCountKey, Value: "7"},
	}}
	service := newService(context.Background(), store)

	if err := service.CleanupOnStartup(); err != nil {
		t.Fatalf("CleanupOnStartup() error = %v", err)
	}
	if store.deletedMax != 7 {
		t.Fatalf("deleted max = %d, want 7", store.deletedMax)
	}
	if want := []string{"mark", "list", "delete"}; !reflect.DeepEqual(store.calls, want) {
		t.Fatalf("calls = %#v, want %#v", store.calls, want)
	}
}

func TestCleanupOnStartupRejectsInvalidRecordLimit(t *testing.T) {
	store := &fakeStore{settings: []settingmodel.Setting{
		{Key: recordMaxCountKey, Value: "invalid"},
	}}
	service := newService(context.Background(), store)

	if err := service.CleanupOnStartup(); err == nil {
		t.Fatal("CleanupOnStartup() error = nil, want validation error")
	}
	if store.deletedMax != 0 {
		t.Fatalf("DeleteOldCleaningRecords() called with %d", store.deletedMax)
	}
}
