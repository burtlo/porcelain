package operatorstore

import "strings"

// workspaceTenantCandidates returns tenant_id values that should match one logical
// operator scope. Legacy UI-managed workspaces used "" while API keys use "default".
func workspaceTenantCandidates(tenantID string) []string {
	t := strings.TrimSpace(tenantID)
	switch t {
	case "":
		return []string{""}
	case "default":
		return []string{"default", ""}
	default:
		return []string{t}
	}
}
