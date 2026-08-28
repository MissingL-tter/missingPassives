package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteArtifacts builds every script document (all of them when want is
// empty, else the named subset), plus conquertables.json and the
// hand-maintained foulborn map, and writes them into outDir — the complete
// data/raw artifact set except tree_<version>.json (sourced from GGG's
// published tree JSON, see BuildTreeDoc) and modcache.jsonl. nodeIDs feeds
// BuildConquerTables (data.NodeGraphIDs). Returns the written file names.
func WriteArtifacts(x *Ctx, outDir string, want map[string]bool, nodeIDs []int64) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	var written []string
	for _, s := range Scripts {
		if len(want) > 0 && !want[s.Name] {
			continue
		}
		data, err := s.Build(x)
		if err != nil {
			return written, fmt.Errorf("%s: %w", s.Name, err)
		}
		raw, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			return written, fmt.Errorf("%s: %w", s.Name, err)
		}
		outName := s.Name
		if s.OutName != "" {
			outName = s.OutName
		}
		outName = strings.ToLower(outName) + ".json" // raw artifacts are all-lowercase
		if err := os.WriteFile(filepath.Join(outDir, outName), append(raw, '\n'), 0o644); err != nil {
			return written, fmt.Errorf("%s: %w", s.Name, err)
		}
		written = append(written, outName)
	}
	if len(want) == 0 || want["conquerTables"] {
		doc, err := BuildConquerTables(x, nodeIDs)
		if err != nil {
			return written, fmt.Errorf("conquerTables: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "conquertables.json"), doc, 0o644); err != nil {
			return written, fmt.Errorf("conquerTables: %w", err)
		}
		written = append(written, "conquertables.json")
	}
	if len(want) == 0 {
		// The hand-maintained foulborn map rides along so raw/ is the
		// complete data artifact.
		fb, err := os.ReadFile(filepath.Join(x.TplDir, "Data", "ModFoulbornMap.jsonc"))
		if err != nil {
			return written, fmt.Errorf("ModFoulbornMap: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "modfoulbornmap.jsonc"), fb, 0o644); err != nil {
			return written, fmt.Errorf("ModFoulbornMap: %w", err)
		}
		written = append(written, "modfoulbornmap.jsonc")
	}
	return written, nil
}
