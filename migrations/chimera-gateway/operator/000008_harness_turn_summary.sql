-- Per-turn harness envelope summary (redacted JSON) on assistant/error rows.

ALTER TABLE conversation_turns ADD COLUMN harness_summary_json TEXT NOT NULL DEFAULT '';
