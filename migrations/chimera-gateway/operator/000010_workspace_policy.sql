-- Workspace policy used by the virtual-model harness meta-policy stage.
ALTER TABLE workspaces ADD COLUMN sensitivity TEXT NOT NULL DEFAULT 'internal';
ALTER TABLE workspaces ADD COLUMN allow_cloud INTEGER NOT NULL DEFAULT 1;
ALTER TABLE workspaces ADD COLUMN allow_cloud_summary_only INTEGER NOT NULL DEFAULT 0;
ALTER TABLE workspaces ADD COLUMN file_action_policy TEXT NOT NULL DEFAULT 'none';
