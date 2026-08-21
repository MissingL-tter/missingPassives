package test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/MissingL-tter/missingPassives/export"
)

// The export differential test: runs every ported export script over the
// extracted GGPK and byte-compares each generated file against the checked-in
// copy under .archive/src (which the archive Lua exporter produced from the
// same game version). Fails on any disagreement.
//
// Requires the extracted GGPK at .archive/src/Export/ggpk (see that
// directory's README for bun_extract_file usage); skips when absent.
func TestExportAgainstReference(t *testing.T) {
	srcDir := filepath.Join("..", ".archive", "src", "Export", "ggpk")
	refDir := filepath.Join("..", ".archive", "src")
	if _, err := os.Stat(filepath.Join(srcDir, "Data", "stats.datc64")); err != nil {
		t.Skipf("extracted GGPK not present at %s: %v", srcDir, err)
	}

	if err := export.WriteEnumFiles(filepath.Join(srcDir, "Data")); err != nil {
		t.Fatalf("writing enum tables: %v", err)
	}
	dats, err := export.LoadDats(filepath.Join(srcDir, "Data"))
	if err != nil {
		t.Fatalf("loading dats: %v", err)
	}
	outDir := t.TempDir()
	ctx := &export.Ctx{Dats: dats, SrcDir: srcDir, OutDir: outDir, TplDir: refDir}

	var checked, disagree int
	for _, s := range export.Scripts {
		if err := s.Run(ctx); err != nil {
			t.Fatalf("%s: %v", s.Name, err)
		}
		for _, rel := range s.Outs {
			checked++
			got, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(rel)))
			if err != nil {
				disagree++
				t.Errorf("%s: output %s not written: %v", s.Name, rel, err)
				continue
			}
			want, err := os.ReadFile(filepath.Join(refDir, filepath.FromSlash(rel)))
			if err != nil {
				disagree++
				t.Errorf("%s: reference %s unreadable: %v", s.Name, rel, err)
				continue
			}
			if !bytes.Equal(got, want) {
				disagree++
				t.Errorf("%s: %s differs from the archive (%d vs %d bytes)", s.Name, rel, len(got), len(want))
			}
		}
	}
	t.Logf("export vs archive: %d files checked, %d disagreements", checked, disagree)
	if disagree > 0 {
		t.Fatalf("%d disagreements with the archive", disagree)
	}
}
