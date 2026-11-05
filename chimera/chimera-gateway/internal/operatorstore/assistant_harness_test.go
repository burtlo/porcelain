package operatorstore

import (
	"context"
	"testing"
)

func TestAssistantHarness_DefaultsAndToggle(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	vm, err := s.CreateAssistant(ctx, CreateAssistantInput{
		Name: "Harness", Version: "1.0", Enabled: true, DefaultRetrievalEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !vm.HarnessModuleEnabled(HarnessModuleRetrieval) {
		t.Fatalf("expected retrieval on by default: %+v", vm.HarnessModules)
	}
	if vm.HarnessModuleEnabled(HarnessModuleIntent) {
		t.Fatalf("expected intent off: %+v", vm.HarnessModules)
	}

	mods := DefaultHarnessModules(false)
	for i := range mods {
		if mods[i].ModuleID == HarnessModuleIntent {
			mods[i].Enabled = true
		}
	}
	if err := s.SetAssistantHarness(ctx, "", vm.ID, mods); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetAssistantByID(ctx, "", vm.ID)
	if err != nil || got == nil {
		t.Fatalf("get: %v %+v", err, got)
	}
	if got.HarnessModuleEnabled(HarnessModuleRetrieval) {
		t.Fatalf("expected retrieval off after save: %+v", got.HarnessModules)
	}
	if !got.HarnessModuleEnabled(HarnessModuleIntent) {
		t.Fatalf("expected intent on: %+v", got.HarnessModules)
	}
}
