package operatorstore

import "testing"

func TestParseEvaluatorConfigDefaultsAndNormalizes(t *testing.T) {
	got := ParseEvaluatorConfig(`{"mode":"other","stream_policy":"not-a-policy","min_confidence":2}`)
	if got.Mode != "single_pass" || got.StreamPolicy != StreamPolicyImmediate || got.MinConfidence != 0 {
		t.Fatalf("unexpected normalized config: %#v", got)
	}
}

func TestParseEscalationConfigBoundsAndFiltersActions(t *testing.T) {
	got := ParseEscalationConfig(`{"max_rounds":8,"on_fail":["re_retrieve","unknown","re_retrieve","human"]}`)
	if got.MaxRounds != 2 || len(got.OnFail) != 2 || got.OnFail[0] != "re_retrieve" || got.OnFail[1] != "human" {
		t.Fatalf("unexpected normalized config: %#v", got)
	}
}

func TestParseEvaluatorConfigMultiDraft(t *testing.T) {
	got := ParseEvaluatorConfig(`{"mode":"multi_draft","draft_count":12,"synthesize_model_id":" groq/synth "}`)
	if got.Mode != "multi_draft" || got.DraftCount != 8 || got.SynthesizeModelID != "groq/synth" {
		t.Fatalf("unexpected multi-draft config: %#v", got)
	}
}

func TestParseEscalationConfigHumanFields(t *testing.T) {
	got := ParseEscalationConfig(`{"on_fail":["human"],"human_surfaces":[{"name":" Desk ","url":" https://example.test "}],"privacy_disclosure":" redact secrets ","paste_back_delimiter":" <<<ANSWER>>> "}`)
	if len(got.HumanSurfaces) != 1 || got.HumanSurfaces[0].Name != "Desk" || got.PasteBackDelimiter != "<<<ANSWER>>>" {
		t.Fatalf("unexpected human escalation config: %#v", got)
	}
}
