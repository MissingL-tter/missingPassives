// The hand-maintained export templates, in-repo (export/templates/):
// per-file JSON documents of typed directives (plus the Rares template's
// hand-written item blocks and the uniques item-text database). Formerly
// read from the archive's Export/ tree; the archive copies remain only as
// the render-test's source for the generated files' passthrough text.

package export

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed templates
var templateFS embed.FS

// templateDoc is one template document: the ordered directive stream of a
// family D plus any hand-written item blocks.
type templateDoc[D any] struct {
	Directives []D
	Items      [][]string
}

// readTemplate loads one template document, decoding each directive into
// the family type its "directive" key names. inDir/name use the archive's
// naming ("Bases/", "act_str"); files are all-lowercase.
func readTemplate[D any](inDir, name string, kinds map[string]func() D) (templateDoc[D], error) {
	rel := "templates/" + strings.ToLower(strings.TrimSuffix(inDir, "/")+"/"+name+".json")
	raw, err := templateFS.ReadFile(rel)
	if err != nil {
		return templateDoc[D]{}, err
	}
	var rawDoc struct {
		Directives []json.RawMessage `json:"directives"`
		Items      [][]string        `json:"items"`
	}
	if err := json.Unmarshal(raw, &rawDoc); err != nil {
		return templateDoc[D]{}, fmt.Errorf("%s: %w", rel, err)
	}
	doc := templateDoc[D]{Items: rawDoc.Items}
	for i, e := range rawDoc.Directives {
		var head kind
		if err := json.Unmarshal(e, &head); err != nil {
			return templateDoc[D]{}, fmt.Errorf("%s: directive %d: %w", rel, i, err)
		}
		mk := kinds[head.Directive]
		if mk == nil {
			return templateDoc[D]{}, fmt.Errorf("%s: directive %d: unknown directive %q", rel, i, head.Directive)
		}
		d := mk()
		dec := json.NewDecoder(bytes.NewReader(e))
		dec.DisallowUnknownFields()
		if err := dec.Decode(d); err != nil {
			return templateDoc[D]{}, fmt.Errorf("%s: directive %d (%s): %w", rel, i, head.Directive, err)
		}
		doc.Directives = append(doc.Directives, d)
	}
	return doc, nil
}

type uniqueTemplateDoc struct {
	Sections []struct {
		Items [][]string `json:"items"`
	} `json:"sections"`
}

// readUniqueTemplate loads one uniques item-text template.
func readUniqueTemplate(name string) (uniqueTemplateDoc, error) {
	raw, err := templateFS.ReadFile("templates/uniques/" + strings.ToLower(name) + ".json")
	if err != nil {
		return uniqueTemplateDoc{}, err
	}
	var doc uniqueTemplateDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return uniqueTemplateDoc{}, err
	}
	return doc, nil
}

// FoulbornMap returns the hand-maintained foulborn map document.
func FoulbornMap() ([]byte, error) {
	return templateFS.ReadFile("templates/modfoulbornmap.jsonc")
}
