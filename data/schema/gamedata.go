// Package gamedata defines the structured output of the export pipeline:
// typed, JSON-serialisable game data, one document per export script.
//
// This is the exporter's real product. The Lua Data/*.lua text the reference
// exporter produced is reproduced only inside the differential test, by
// internal/luarender, which turns these documents back into byte-identical
// Lua files for comparison against the archive.
package schema
