package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/MissingL-tter/missingPassives/config"
	"github.com/MissingL-tter/missingPassives/test/luacanon"
)

// TestConfigOptionsAgainstReference exercises every configuration option,
// not only the ~32 the build corpus happens to set. tools/dump_config.lua
// sets each option in turn on an otherwise-default build - every value of
// a list option, several values of a numeric one - and records what
// BuildModList produced. This replays each case and compares.
func TestConfigOptionsAgainstReference(t *testing.T) {
	loadData(t)
	path := filepath.Join("testdata", "config_options.jsonl")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("option dump not present: run tools/dump_config.lua from .archive/src")
	}
	records := map[string]string{}
	forEachCalcRecord(t, path, func(k, c string) { records[k] = c })

	var cases []string
	for k := range records {
		if name, ok := strings.CutSuffix(k, ".value"); ok {
			cases = append(cases, name)
		}
	}
	if len(cases) == 0 {
		t.Fatal("the dump names no cases")
	}
	sort.Strings(cases)

	only := os.Getenv("MP_ONLY_OPTION")
	compared, options := 0, map[string]bool{}
	for _, name := range cases {
		if only != "" && !strings.HasPrefix(name, only) {
			continue
		}
		var chosen struct {
			Var   string `json:"var"`
			Value any    `json:"value"`
		}
		if err := json.Unmarshal([]byte(records[name+".value"]), &chosen); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		value := configValueOf(chosen.Value)
		if value == nil {
			t.Errorf("%s: cannot represent %#v", name, chosen.Value)
			continue
		}
		// The dump's build is a new one, whose level is 1.
		tab := config.NewTab(1)
		tab.Input[config.Var(chosen.Var)] = value
		tab.BuildModList()

		for _, part := range []struct {
			suffix string
			got    string
		}{
			{".mods", luacanon.Encode(tab.Mods.Mods)},
			{".enemyMods", luacanon.Encode(tab.EnemyMods.Mods)},
		} {
			want := records[name+part.suffix]
			if part.got != want {
				t.Errorf("%s%s (%s = %v) diverged\n%s",
					name, part.suffix, chosen.Var, chosen.Value, diffWindow(part.got, want))
			}
			compared++
		}
		var refInput, refPlaceholder map[string]any
		if err := json.Unmarshal([]byte(records[name+".input"]), &refInput); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := json.Unmarshal([]byte(records[name+".placeholder"]), &refPlaceholder); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		compared += compareConfigTable(t, name, "input", tab.Input, refInput)
		compared += compareConfigTable(t, name, "placeholder", tab.Placeholder, refPlaceholder)
		options[chosen.Var] = true
	}
	if only == "" && len(options) < 500 {
		t.Fatalf("expected the whole option table, exercised %d", len(options))
	}
	t.Logf("config option differential: %d comparisons across %d cases covering %d options",
		compared, len(cases), len(options))
}

// configValueOf converts a decoded dump value into the tab's value type.
func configValueOf(v any) config.Value {
	switch t := v.(type) {
	case bool:
		return config.Bool(t)
	case float64:
		return config.Num(t)
	case string:
		return config.Str(t)
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return nil
		}
		return config.Num(f)
	}
	_ = fmt.Sprint(v)
	return nil
}
