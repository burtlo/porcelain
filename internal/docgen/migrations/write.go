package migrations

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DefaultOutputPath is the repo-relative path for generated migration documentation.
const DefaultOutputPath = "docs/generated/operator-sqlite-migrations.md"

const operatorMigrationsDir = "migrations/chimera-gateway/operator"
const metricsMigrationsDir = "migrations/chimera-gateway/metrics"

const operatorMigrateGo = "chimera/chimera-gateway/internal/operatorstore/migrate.go"

var migrationFileRE = regexp.MustCompile(`^(\d{6})_(.+)\.sql$`)

// WriteOperatorSQLiteMigrationsMD emits migration reference docs from SQL files and migrate.go notes.
func WriteOperatorSQLiteMigrationsMD(w io.Writer, repoRoot string) error {
	notes, err := parseDocgenNotes(filepath.Join(repoRoot, operatorMigrateGo))
	if err != nil {
		return err
	}
	opMigs, err := loadMigrations(filepath.Join(repoRoot, operatorMigrationsDir))
	if err != nil {
		return fmt.Errorf("operator migrations: %w", err)
	}
	metMigs, err := loadMigrations(filepath.Join(repoRoot, metricsMigrationsDir))
	if err != nil {
		return fmt.Errorf("metrics migrations: %w", err)
	}

	var b strings.Builder
	b.WriteString("# Operator SQLite migrations (generated)\n\n")
	b.WriteString("DO NOT EDIT; run: `make contracts-generate`.\n\n")
	b.WriteString("Gateway applies versioned SQL under `migrations/chimera-gateway/` at startup. ")
	b.WriteString("The **operator store** (`operator.sqlite`, table `operator_migrations`) and **gateway metrics** DB ")
	b.WriteString("(`gateway_migrations`) use separate directories and migrators.\n\n")

	if len(notes) > 0 {
		b.WriteString("## Operator migrator notes\n\n")
		b.WriteString("From `internal/operatorstore/migrate.go` (`//docgen:migrations`):\n\n")
		for _, line := range notes {
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	writeSuiteOverview(&b, "Operator store", operatorMigrationsDir, "operator_migrations", "internal/operatorstore/migrate.go")
	writeMigrationTable(&b, opMigs)
	for _, m := range opMigs {
		writeMigrationDetail(&b, m, operatorMigrationsDir, m.Version == 11)
	}

	writeSuiteOverview(&b, "Gateway metrics", metricsMigrationsDir, "gateway_migrations", "internal/gatewaymetrics/migrate.go")
	writeMigrationTable(&b, metMigs)
	for _, m := range metMigs {
		writeMigrationDetail(&b, m, metricsMigrationsDir, false)
	}

	_, err = io.WriteString(w, b.String())
	return err
}

func writeSuiteOverview(b *strings.Builder, title, dir, migrationsTable, migrateGo string) {
	b.WriteString(fmt.Sprintf("## %s\n\n", title))
	b.WriteString(fmt.Sprintf("- **SQL directory:** `%s`\n", dir))
	b.WriteString(fmt.Sprintf("- **Applied versions table:** `%s`\n", migrationsTable))
	b.WriteString(fmt.Sprintf("- **Go:** `chimera/chimera-gateway/%s`\n\n", migrateGo))
}

func writeMigrationTable(b *strings.Builder, migs []migration) {
	b.WriteString("| Version | File | Summary |\n")
	b.WriteString("|---------|------|--------|\n")
	for _, m := range migs {
		b.WriteString(fmt.Sprintf("| %d | `%s` | %s |\n", m.Version, m.Filename, escapeTableCell(m.Summary)))
	}
	b.WriteString("\n")
}

func writeMigrationDetail(b *strings.Builder, m migration, dir string, goApplied bool) {
	b.WriteString(fmt.Sprintf("### Version %d — `%s`\n\n", m.Version, m.Filename))
	if goApplied {
		b.WriteString("**Runtime:** `ApplyMigrations` does **not** execute this file as raw SQL; it runs ")
		b.WriteString("`applyRenameVirtualModelsToAssistants` when legacy `virtual_models` exists ")
		b.WriteString("(see `migrate_rename.go`). The `.sql` file documents the upgrade path.\n\n")
	}
	b.WriteString("<details>\n<summary>SQL</summary>\n\n")
	b.WriteString("```sql\n")
	body := m.Body
	if body == "" {
		body = "-- (empty)\n"
	}
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("```\n\n")
	b.WriteString("</details>\n\n")
}

func escapeTableCell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

type migration struct {
	Version  int
	Filename string
	Slug     string
	Summary  string
	Body     string
}

func loadMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}
		sub := migrationFileRE.FindStringSubmatch(name)
		if len(sub) != 3 {
			return nil, fmt.Errorf("invalid migration filename %q (want 000001_name.sql)", name)
		}
		v, err := strconv.Atoi(sub[1])
		if err != nil {
			return nil, fmt.Errorf("migration version %q: %w", name, err)
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		out = append(out, migration{
			Version:  v,
			Filename: name,
			Slug:     sub[2],
			Summary:  firstCommentSummary(string(body)),
			Body:     string(body),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func firstCommentSummary(sql string) string {
	for _, line := range strings.Split(sql, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "--" {
			continue
		}
		if strings.HasPrefix(line, "--") {
			s := strings.TrimSpace(strings.TrimPrefix(line, "--"))
			if s != "" {
				return s
			}
		}
		break
	}
	return ""
}

func parseDocgenNotes(migrateGoPath string) ([]string, error) {
	data, err := os.ReadFile(migrateGoPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", migrateGoPath, err)
	}
	var notes []string
	inBlock := false
	for _, line := range strings.Split(string(data), "\n") {
		trim := strings.TrimSpace(line)
		if strings.Contains(trim, "//docgen:migrations") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if !strings.HasPrefix(trim, "//") {
			break
		}
		text := strings.TrimSpace(strings.TrimPrefix(trim, "//"))
		if text == "" {
			continue
		}
		notes = append(notes, text)
	}
	return notes, nil
}
