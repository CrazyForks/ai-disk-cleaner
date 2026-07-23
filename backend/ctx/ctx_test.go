package ctx

import (
	"context"
	"testing"
)

func TestSetAndGetContext(t *testing.T) {
	resetContextForTest(t)
	want := context.WithValue(context.Background(), struct{}{}, "value")

	SetContext(want)

	if got := GetContext(); got != want {
		t.Fatal("GetContext returned a different context")
	}
}

func TestGetContextPanicsBeforeInitialization(t *testing.T) {
	resetContextForTest(t)
	assertPanics(t, func() { GetContext() })
}

func TestSetContextPanicsWhenCalledTwice(t *testing.T) {
	resetContextForTest(t)
	SetContext(context.Background())
	assertPanics(t, func() { SetContext(context.Background()) })
}

func resetContextForTest(t *testing.T) {
	t.Helper()
	applicationContext = nil
	t.Cleanup(func() { applicationContext = nil })
}

func assertPanics(t *testing.T, action func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("action did not panic")
		}
	}()
	action()
}
