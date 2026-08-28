package test

// The shipped-mod-cache differential. data/raw/modcache.jsonl carries
// Data/ModCache.lua's pre-parsed entries, and modparser serves them for
// those lines instead of parsing. Two proofs per entry:
//   1. decode-echo: the served result re-encodes to the file's exact
//      bytes (the decoder is lossless);
//   2. fresh-parse equivalence: a from-scratch parse with every number
//      rounded through %.14g text (the precision PoB wrote the file at)
//      produces those same bytes — the shipped file holds no entry our
//      parser disagrees with.

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/MissingL-tter/missingPassives/data"
	"github.com/MissingL-tter/missingPassives/modparser"
)

type modCacheRec struct {
	K string          `json:"k"`
	M json.RawMessage `json:"m"`
	E *string         `json:"e"`
}

func readModCacheRecs(t *testing.T) []modCacheRec {
	t.Helper()
	blob := data.LoadedModCache
	if len(blob) == 0 {
		t.Fatal("data.Load left no mod cache blob")
	}
	var recs []modCacheRec
	dec := json.NewDecoder(bytes.NewReader(blob))
	for dec.More() {
		var rec modCacheRec
		if err := dec.Decode(&rec); err != nil {
			t.Fatal(err)
		}
		recs = append(recs, rec)
	}
	return recs
}

func canonOrNull(mods []any) string {
	if mods == nil {
		return "null"
	}
	return string(modparser.EncodeMods(mods))
}

func extraOrEmpty(e *string) string {
	if e == nil {
		return ""
	}
	return *e
}

func TestModCacheAgainstShippedFile(t *testing.T) {
	loadData(t)
	recs := readModCacheRecs(t)

	// Pass 1: served entries re-encode to the file's bytes.
	modparser.SetModCache(data.LoadedModCache)
	echoBad := 0
	for _, rec := range recs {
		mods, extra := modparser.Parse(rec.K)
		if got, want := canonOrNull(mods), string(rec.M); got != want {
			echoBad++
			if echoBad <= 3 {
				t.Errorf("decode-echo %q:\n%s", rec.K, diffWindow(got, want))
			}
		}
		if extra != extraOrEmpty(rec.E) {
			t.Errorf("decode-echo %q: extra %q vs %q", rec.K, extra, extraOrEmpty(rec.E))
		}
	}

	// Pass 2: fresh parses, rounded the way the file was written, agree
	// with the file.
	modparser.SetModCache(nil)
	freshBad := 0
	for _, rec := range recs {
		mods, extra := modparser.Parse(rec.K)
		if got, want := canonOrNull(modparser.Quantize14(mods)), string(rec.M); got != want {
			freshBad++
			if freshBad <= 3 {
				t.Errorf("fresh-parse %q:\n%s", rec.K, diffWindow(got, want))
			}
		}
		if extra != extraOrEmpty(rec.E) {
			t.Errorf("fresh-parse %q: extra %q vs %q", rec.K, extra, extraOrEmpty(rec.E))
		}
	}
	modparser.SetModCache(data.LoadedModCache)
	if echoBad > 3 || freshBad > 3 {
		t.Errorf("suppressed diffs: echo %d, fresh %d", echoBad, freshBad)
	}
	t.Logf("mod cache: %d entries; decode-echo and quantized fresh parse both byte-identical", len(recs))
}
