// Port of .archive/src/Export/Scripts/enums.lua: writes the two synthetic
// .datc64 enum tables into the extracted GGPK's data directory. The Lua runs
// it on every startup at the end of GGPKData:ExtractFiles, before the dat
// files are loaded; WriteEnumFiles is its Go counterpart and must run before
// LoadDats.

package export

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"unicode/utf16"
)

func writeEnumFile(path string, entries []string) error {
	var b []byte
	b = binary.LittleEndian.AppendUint32(b, uint32(len(entries)))

	stringIndex := uint64(8)
	var strs [][]byte
	for _, s := range entries {
		var enc []byte
		for _, u := range utf16.Encode([]rune(s)) {
			enc = binary.LittleEndian.AppendUint16(enc, u)
		}
		strs = append(strs, enc)
		b = binary.LittleEndian.AppendUint64(b, stringIndex)
		stringIndex += uint64(len(enc)) + 2
	}
	for i := 0; i < 8; i++ {
		b = append(b, 0xBB)
	}
	for _, enc := range strs {
		b = append(b, enc...)
		b = append(b, 0, 0)
	}
	return os.WriteFile(path, b, 0o644)
}

// WriteEnumFiles writes influencetypes.datc64 and passiveskilltypes.datc64
// into dataDir.
func WriteEnumFiles(dataDir string) error {
	influenceTypes := []string{
		"Shaper",
		"Elder",
		"Crusader",
		"Eyrie",
		"Basilisk",
		"Adjudicator",
		"None",
	}
	if err := writeEnumFile(filepath.Join(dataDir, "influenceTypes.datc64"), influenceTypes); err != nil {
		return err
	}
	passiveSkillTypes := []string{
		"Passive Tree",
		"Atlas Tree",
	}
	return writeEnumFile(filepath.Join(dataDir, "passiveSkillTypes.datc64"), passiveSkillTypes)
}
