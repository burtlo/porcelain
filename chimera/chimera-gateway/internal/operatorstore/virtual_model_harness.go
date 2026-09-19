package operatorstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Harness module ids persisted per virtual model (v0.4 turn harness).
const (
	HarnessModuleRetrieval    = "retrieval"
	HarnessModuleIntent       = "intent"
	HarnessModuleEvaluator    = "evaluator"
	HarnessModuleEscalation   = "escalation"
	HarnessModuleToolExecutor = "tool_executor"
)

// HarnessModuleIDs is the stable operator-facing module order.
var HarnessModuleIDs = []string{
	HarnessModuleRetrieval,
	HarnessModuleIntent,
	HarnessModuleEvaluator,
	HarnessModuleEscalation,
	HarnessModuleToolExecutor,
}

// HarnessModule is one toggleable harness module on a virtual model.
type HarnessModule struct {
	ModuleID   string
	Enabled    bool
	ConfigJSON string
}

// DefaultHarnessModules returns the default profile for a new virtual model.
// retrievalEnabled should mirror gateway-global RAG (search.enabled) at create time.
func DefaultHarnessModules(retrievalEnabled bool) []HarnessModule {
	out := make([]HarnessModule, 0, len(HarnessModuleIDs))
	for _, id := range HarnessModuleIDs {
		en := false
		if id == HarnessModuleRetrieval {
			en = retrievalEnabled
		}
		out = append(out, HarnessModule{ModuleID: id, Enabled: en, ConfigJSON: "{}"})
	}
	return out
}

func normalizeHarnessConfigJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}"
	}
	if !json.Valid([]byte(raw)) {
		return "{}"
	}
	return raw
}

func (s *Store) loadVirtualModelHarness(ctx context.Context, vm *VirtualModel) error {
	rows, err := s.db.QueryContext(ctx, `
SELECT module_id, enabled, config_json
FROM assistant_harness_modules WHERE assistant_id = ?
ORDER BY module_id`, vm.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	byID := make(map[string]HarnessModule)
	for rows.Next() {
		var mid, cfg string
		var en int
		if err := rows.Scan(&mid, &en, &cfg); err != nil {
			return err
		}
		byID[mid] = HarnessModule{
			ModuleID:   mid,
			Enabled:    en != 0,
			ConfigJSON: normalizeHarnessConfigJSON(cfg),
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(byID) == 0 {
		// CreateVirtualModel always seeds rows; empty means incomplete row — defaults with retrieval off.
		vm.HarnessModules = DefaultHarnessModules(false)
		return nil
	}
	out := make([]HarnessModule, 0, len(HarnessModuleIDs))
	for _, id := range HarnessModuleIDs {
		if m, ok := byID[id]; ok {
			out = append(out, m)
			continue
		}
		out = append(out, HarnessModule{ModuleID: id, Enabled: false, ConfigJSON: "{}"})
	}
	vm.HarnessModules = out
	return nil
}

func (s *Store) insertDefaultHarnessModulesTx(ctx context.Context, tx *sql.Tx, vmID int64, retrievalEnabled bool, now string) error {
	for _, m := range DefaultHarnessModules(retrievalEnabled) {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO assistant_harness_modules (assistant_id, module_id, enabled, config_json, updated_at)
VALUES (?,?,?,?,?)`,
			vmID, m.ModuleID, boolToInt(m.Enabled), normalizeHarnessConfigJSON(m.ConfigJSON), now); err != nil {
			return err
		}
	}
	return nil
}

// SetVirtualModelHarness replaces all known harness modules for a virtual model.
func (s *Store) SetVirtualModelHarness(ctx context.Context, tenantID string, id int64, modules []HarnessModule) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("operator store unavailable")
	}
	w, err := s.GetVirtualModelByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if w == nil {
		return fmt.Errorf("virtual model not found")
	}
	byID := make(map[string]HarnessModule, len(modules))
	for _, m := range modules {
		mid := strings.TrimSpace(m.ModuleID)
		if mid == "" {
			continue
		}
		known := false
		for _, id := range HarnessModuleIDs {
			if id == mid {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("unknown harness module %q", mid)
		}
		byID[mid] = HarnessModule{
			ModuleID:   mid,
			Enabled:    m.Enabled,
			ConfigJSON: normalizeHarnessConfigJSON(m.ConfigJSON),
		}
	}
	now := s.nowRFC3339()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, moduleID := range HarnessModuleIDs {
		m, ok := byID[moduleID]
		if !ok {
			m = HarnessModule{ModuleID: moduleID, Enabled: false, ConfigJSON: "{}"}
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO assistant_harness_modules (assistant_id, module_id, enabled, config_json, updated_at)
VALUES (?,?,?,?,?)
ON CONFLICT(assistant_id, module_id) DO UPDATE SET
	enabled = excluded.enabled,
	config_json = excluded.config_json,
	updated_at = excluded.updated_at`,
			id, m.ModuleID, boolToInt(m.Enabled), m.ConfigJSON, now); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE assistants SET updated_at = ? WHERE id = ?`, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

// HarnessModuleEnabled reports whether moduleID is enabled on the model.
func (vm *VirtualModel) HarnessModuleEnabled(moduleID string) bool {
	if vm == nil {
		return false
	}
	for _, m := range vm.HarnessModules {
		if m.ModuleID == moduleID {
			return m.Enabled
		}
	}
	return false
}
