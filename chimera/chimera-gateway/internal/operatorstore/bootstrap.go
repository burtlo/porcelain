package operatorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/internal/naming"
)

var nonSlugRE = regexp.MustCompile(`[^a-z0-9]+`)

// RuleNameToSlug converts a policy rule name to a stable catalog slug.
func RuleNameToSlug(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = nonSlugRE.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "routing.rule.unnamed"
	}
	return "routing.rule." + name
}

func seedRoutingRuleCatalog(ctx context.Context, s *Store) error {
	defaultCfg, _ := json.Marshal(map[string]any{
		"when":   map[string]any{},
		"models": []string{},
	})
	longCfg, _ := json.Marshal(map[string]any{
		"when":   map[string]any{"min_message_chars": 8000},
		"models": []string{},
	})
	if err := s.UpsertRoutingRuleDefinition(ctx, "default", "routing.rule.default", string(defaultCfg),
		"Default rule (empty when clause)"); err != nil {
		return err
	}
	return s.UpsertRoutingRuleDefinition(ctx, "long-user-turn", "routing.rule.long_user_turn", string(longCfg),
		"Route long user turns via min_message_chars")
}

// BootstrapAssistants seeds operator SQLite on first open. Legacy file-based routing
// import was removed; operators create assistants in settings (or tests seed rows explicitly).
func BootstrapAssistants(ctx context.Context, s *Store, res *config.Resolved, log *slog.Logger) error {
	if s == nil {
		return nil
	}
	has, err := s.HasAssistants(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap assistants count: %w", err)
	}
	if has {
		return nil
	}
	if err := seedRoutingRuleCatalog(ctx, s); err != nil {
		return fmt.Errorf("bootstrap routing rule catalog: %w", err)
	}
	return nil
}

// ChimeraSeed returns a Chimera-<semver> assistant definition for tests and explicit seeding.
func ChimeraSeed(semver string, fallbackChain []string, policyDefaultModel string) Assistant {
	if semver == "" {
		semver = "0.1.0"
	}
	if len(fallbackChain) == 0 {
		fallbackChain = []string{"groq/a", "groq/b"}
	}
	if policyDefaultModel == "" {
		policyDefaultModel = fallbackChain[0]
	}
	policyYAML := fmt.Sprintf(`ambiguous_default_model: %s
rules:
  - name: default
    when: {}
    models:
      - %s
`, policyDefaultModel, policyDefaultModel)
	return Assistant{
		ModelID:              "Chimera-" + semver,
		Name:                 "Chimera",
		Version:              semver,
		Description:          "Test Chimera assistant",
		Enabled:              true,
		Visibility:           VisibilityPublic,
		FallbackChain:        append([]string(nil), fallbackChain...),
		RoutingPolicyYAML:    policyYAML,
		RoutingPolicyEnabled: true,
		ToolRouterEnabled:    false,
		RouterModels:         nil,
		ToolRouterConfidence: 0.5,
	}
}

// Gemini010Seed returns the Gemini-0.1.0 assistant definition (gemini provider only).
func Gemini010Seed(geminiModels []string) Assistant {
	if len(geminiModels) == 0 {
		geminiModels = []string{
			"gemini/gemini-2.5-flash",
			"gemini/gemini-2.5-flash-lite",
		}
	}
	defaultModel := geminiModels[0]
	policyYAML := fmt.Sprintf(`ambiguous_default_model: %s
rules:
  - name: default
    when: {}
    models:
      - %s
  - name: long-user-turn
    when:
      min_message_chars: 4000
    models:
      - %s
`, defaultModel, defaultModel, defaultModel)
	return Assistant{
		ModelID:              "Gemini-0.1.0",
		Name:                 "Gemini",
		Version:              "0.1.0",
		Description:          "Gemini-only assistant; routes exclusively through gemini provider models",
		Enabled:              true,
		Visibility:           VisibilityPublic,
		FallbackChain:        append([]string(nil), geminiModels...),
		RoutingPolicyYAML:    policyYAML,
		RoutingPolicyEnabled: true,
		ToolRouterEnabled:    false,
		RouterModels:         nil,
		ToolRouterConfidence: 0.5,
	}
}

// EnsureGeminiAssistant creates Gemini-0.1.0 when absent (used after bootstrap in dev/tests).
func EnsureGeminiAssistant(ctx context.Context, s *Store, geminiModels []string, log *slog.Logger) error {
	if s == nil {
		return nil
	}
	existing, err := s.GetAssistantByModelID(ctx, "Gemini-0.1.0")
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	vm := Gemini010Seed(geminiModels)
	if _, err := s.InsertAssistantFull(ctx, vm); err != nil {
		return fmt.Errorf("seed gemini assistant: %w", err)
	}
	if log != nil {
		log.Info("gemini assistant seeded", "msg", naming.MsgGatewayAssistantGeminiSeeded, "assistant_id", vm.ModelID)
	}
	return nil
}
