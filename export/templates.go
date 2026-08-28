// The hand-maintained export templates, in-repo (export/templates/):
// per-file JSON documents of directive lines (plus the Rares template's
// hand-written item blocks and the uniques item-text database). Formerly
// read from the archive's Export/ tree; the archive copies remain only as
// the render-test's source for the generated files' passthrough text.

package export

import (
	"embed"
	"encoding/json"
	"strings"
)

//go:embed templates
var templateFS embed.FS

type templateDoc struct {
	Directives []string   `json:"directives"`
	Items      [][]string `json:"items"`
}

// readTemplate loads one template document. inDir/name use the archive's
// naming ("Bases/", "act_str"); files are all-lowercase.
func readTemplate(inDir, name string) (templateDoc, error) {
	rel := "templates/" + strings.ToLower(strings.TrimSuffix(inDir, "/")+"/"+name+".json")
	raw, err := templateFS.ReadFile(rel)
	if err != nil {
		return templateDoc{}, err
	}
	var doc templateDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return templateDoc{}, err
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
