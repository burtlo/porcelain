package operatorstore

import (
	"context"
	"fmt"
)

// ResolveWorkspaceForHarness picks the workspace used to wire native tool roots.
// When workspaceRowID is set, that row wins (chat UI selection). Otherwise scope
// is matched on tenant/project/flavor with conservative fallbacks.
func (s *Store) ResolveWorkspaceForHarness(ctx context.Context, tenantID, projectID, flavorID string, workspaceRowID int64) (*Workspace, []int64, error) {
	if s == nil || s.db == nil {
		return nil, nil, fmt.Errorf("operator store unavailable")
	}
	if workspaceRowID > 0 {
		ws, err := s.GetWorkspace(ctx, tenantID, workspaceRowID)
		if err != nil {
			return nil, nil, err
		}
		if ws != nil {
			return ws, []int64{ws.ID}, nil
		}
	}
	ws, ids, err := s.ResolveWorkspaceScope(ctx, tenantID, projectID, flavorID)
	if err != nil || ws != nil {
		return ws, ids, err
	}
	if flavorID != "" {
		ws, ids, err = s.ResolveWorkspaceScope(ctx, tenantID, projectID, "")
		if err != nil || ws != nil {
			return ws, ids, err
		}
	}
	if projectID == "" {
		return nil, ids, nil
	}
	return s.resolveWorkspaceByProject(ctx, tenantID, projectID)
}

func (s *Store) resolveWorkspaceByProject(ctx context.Context, tenantID, projectID string) (*Workspace, []int64, error) {
	var ids []int64
	for _, tenant := range workspaceTenantCandidates(tenantID) {
		batch, err := s.listWorkspaceIDsByProject(ctx, tenant, projectID)
		if err != nil {
			return nil, nil, err
		}
		if len(batch) == 0 {
			continue
		}
		workspace, err := s.GetWorkspace(ctx, tenantID, batch[0])
		return workspace, batch, err
	}
	return nil, ids, nil
}

func (s *Store) listWorkspaceIDsByProject(ctx context.Context, tenantID, projectID string) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id FROM workspaces
WHERE tenant_id = ? AND project_id = ?
ORDER BY id`, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
