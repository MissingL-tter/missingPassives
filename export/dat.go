// Port of .archive/src/Export/Classes/Dat64File.lua and the loading logic of
// GGPKData.lua/Main.lua: parses the game's .dat64 tables against the column
// schemas in spec_gen.go.
//
// Offsets here are 0-based; the Lua original is 1-based. Ref values are read
// as exact uint64 where the Lua reads them into float64 (precision above 2^53
// would differ; real files never get there).

package export

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var datMagic = bytes.Repeat([]byte{0xBB}, 8)

// Interval is the value of an Interval column: two signed 32-bit ints.
type Interval [2]int64

type colInfo struct {
	size   int
	offset int
}

// DatFile is one loaded .dat64 table.
type DatFile struct {
	Name       string // lowercase, no extension
	raw        []byte
	spec       []Col
	cols       []colInfo
	colMap     map[string]int
	RowCount   int
	rowSize    int
	dataOffset int // 0-based index of the 8x0xBB marker; len(raw) if absent
	rowCache   map[int]*Row
	set        *DatSet
}

// Row is one lazily-decoded row. Identity is stable: the same row index
// always yields the same *Row, so ref cells can be compared by pointer just
// like the Lua row tables.
type Row struct {
	File  *DatFile
	Index int // 1-based, matching the Lua rowIndex
	cells map[int]any
}

// DatSet is the registry of loaded tables; Enum/ShortKey/Key cells resolve
// through it (Main.lua's datFileByName).
type DatSet struct {
	ByName map[string]*DatFile
}

func typeSize(t string) int {
	switch t {
	case "Bool":
		return 1
	case "UInt16":
		return 2
	case "Int", "UInt", "Float", "Enum":
		return 4
	case "Interval", "String", "ShortKey":
		return 8
	case "Key":
		return 16
	}
	panic("unknown dat column type " + t)
}

func newDatFile(set *DatSet, name string, raw []byte) *DatFile {
	d := &DatFile{
		Name:     strings.ToLower(name),
		raw:      raw,
		set:      set,
		colMap:   map[string]int{},
		rowCache: map[int]*Row{},
	}
	d.spec = datSpecs[d.Name]

	d.RowCount = int(bytesToUInt(raw, 0))
	searchFrom := 4
	if len(raw) < 4 {
		searchFrom = len(raw)
	}
	idx := bytes.Index(raw[searchFrom:], datMagic)
	if idx >= 0 {
		d.dataOffset = idx + searchFrom
	} else {
		d.dataOffset = len(raw)
	}
	if d.RowCount > 0 {
		d.rowSize = (d.dataOffset - 4) / d.RowCount
	}

	offset := 0
	for i, col := range d.spec {
		size := typeSize(col.Type)
		if col.List {
			size = 16
		}
		d.cols = append(d.cols, colInfo{size: size, offset: offset})
		offset += size
		if col.Name != "" {
			d.colMap[col.Name] = i
		}
	}
	return d
}

// rowStart returns the 0-based byte offset of 1-based row i.
func (d *DatFile) rowStart(i int) int {
	return 4 + (i-1)*d.rowSize
}

// GetRowByIndex returns the row at 1-based index i, or nil out of range.
func (d *DatFile) GetRowByIndex(i int) *Row {
	if i < 1 || i > d.RowCount {
		return nil
	}
	if r := d.rowCache[i]; r != nil {
		return r
	}
	r := &Row{File: d, Index: i, cells: map[int]any{}}
	d.rowCache[i] = r
	return r
}

// Rows iterates all rows in order.
func (d *DatFile) Rows(yield func(*Row) bool) {
	for i := 1; i <= d.RowCount; i++ {
		if !yield(d.GetRowByIndex(i)) {
			return
		}
	}
}

// GetRow returns the first row whose cell in the named column equals value.
func (d *DatFile) GetRow(key string, value any) *Row {
	ki, ok := d.colMap[key]
	if !ok {
		panic(fmt.Sprintf("Unknown key %s for %s.datc64", key, d.Name))
	}
	for i := 1; i <= d.RowCount; i++ {
		if cellEquals(d.readCell(i, ki), value) {
			return d.GetRowByIndex(i)
		}
	}
	return nil
}

// GetRowList returns every row whose cell in the named column (or any element
// of it, for list columns) satisfies eq: exact equality here; the Lua
// pattern-match variant is GetRowListMatch.
func (d *DatFile) GetRowList(key string, value any) []*Row {
	return d.getRowList(key, func(v any) bool { return cellEquals(v, value) })
}

// GetRowListMatch is GetRowList with a predicate over string cells, standing
// in for the Lua original's pattern match.
func (d *DatFile) GetRowListMatch(key string, match func(string) bool) []*Row {
	return d.getRowList(key, func(v any) bool {
		s, ok := v.(string)
		return ok && match(s)
	})
}

func (d *DatFile) getRowList(key string, pred func(any) bool) []*Row {
	ki, ok := d.colMap[key]
	if !ok {
		panic(fmt.Sprintf("Unknown key %s for %s.datc64", key, d.Name))
	}
	var out []*Row
	for i := 1; i <= d.RowCount; i++ {
		cell := d.readCell(i, ki)
		if list, isList := cell.([]any); isList {
			for _, v := range list {
				if pred(v) {
					out = append(out, d.GetRowByIndex(i))
					break
				}
			}
		} else if pred(cell) {
			out = append(out, d.GetRowByIndex(i))
		}
	}
	return out
}

func cellEquals(cell, value any) bool {
	if n, ok := numeric(value); ok {
		if c, ok2 := numeric(cell); ok2 {
			return c == n
		}
		return false
	}
	return cell == value
}

func numeric(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// Get returns the lazily-decoded cell of the named column, caching it.
// Unknown names panic, matching the Lua error.
func (r *Row) Get(key string) any {
	ci, ok := r.File.colMap[key]
	if !ok {
		panic(fmt.Sprintf("Unknown key %s for %s.datc64", key, r.File.Name))
	}
	if v, done := r.cells[ci]; done {
		return v
	}
	v := r.File.readCell(r.Index, ci)
	r.cells[ci] = v
	return v
}

// readCell decodes row i (1-based), column ci (0-based).
func (d *DatFile) readCell(i, ci int) any {
	spec := d.spec[ci]
	base := d.rowStart(i) + d.cols[ci].offset
	if spec.List {
		count := bytesToULong(d.raw, base)
		offset := int(bytesToULong(d.raw, base+8)) + d.dataOffset
		size := typeSize(spec.Type)
		n := int(count)
		if n > 1000 {
			n = 1000
		}
		out := make([]any, 0, n)
		for j := 0; j < n; j++ {
			out = append(out, d.readValue(spec, offset))
			offset += size
		}
		return out
	}
	return d.readValue(spec, base)
}

// readValue decodes one value at the 0-based offset, resolving refs.
func (d *DatFile) readValue(spec Col, o int) any {
	b := d.raw
	var refVal uint64
	switch spec.Type {
	case "Bool":
		return o < len(b) && b[o] == 1
	case "Int":
		if o+4 > len(b) {
			return int64(-1337)
		}
		return int64(bytesToInt(b, o))
	case "UInt16":
		if o+2 > len(b) {
			return int64(1337)
		}
		return int64(bytesToUShort(b, o))
	case "UInt":
		if o+4 > len(b) {
			return int64(1337)
		}
		return int64(bytesToUInt(b, o))
	case "Interval":
		if o+8 > len(b) {
			return Interval{1337, 1337}
		}
		return Interval{int64(bytesToInt(b, o)), int64(bytesToInt(b, o+4))}
	case "Float":
		if o+4 > len(b) {
			return float64(-1337)
		}
		return bytesToFloat(b, o)
	case "String":
		if o+8 > len(b) {
			return "<no offset>"
		}
		stro := bytesToULong(b, o)
		if len(b) < 7 || stro > uint64(len(b)-7) {
			return "<bad offset>"
		}
		return convertUTF16to8(b, d.dataOffset+int(stro))
	case "Enum":
		if o+4 > len(b) {
			refVal = 1337
		} else {
			refVal = uint64(bytesToUInt(b, o))
		}
	case "ShortKey":
		if o+8 > len(b) {
			refVal = 1337
		} else {
			refVal = bytesToULong(b, o)
		}
	case "Key":
		if o+16 > len(b) {
			refVal = 1337
		} else {
			refVal = bytesToULong(b, o)
		}
	default:
		panic("unknown dat column type " + spec.Type)
	}
	// Ref resolution, matching Dat64File:ReadValue.
	if refVal == 0xFEFEFEFE || refVal == 0xFEFEFEFEFEFEFEFE {
		return nil
	}
	other := d.set.ByName[strings.ToLower(spec.RefTo)]
	if other == nil {
		return nil
	}
	if spec.Type == "Enum" && strings.ToLower(spec.RefTo) != d.Name {
		return int64(refVal)
	}
	r := other.GetRowByIndex(int(refVal) + 1)
	if r == nil {
		return nil
	}
	return r
}

// LoadDats loads every .datc64 file under dir (the extracted GGPK's Data
// directory) into a DatSet.
func LoadDats(dir string) (*DatSet, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	set := &DatSet{ByName: map[string]*DatFile{}}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".datc64") {
			names = append(names, e.Name())
		}
	}
	sort.Slice(names, func(a, b int) bool {
		return strings.ToLower(names[a]) < strings.ToLower(names[b])
	})
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		base := strings.TrimSuffix(strings.ToLower(name), ".datc64")
		set.ByName[base] = newDatFile(set, base, raw)
	}
	return set, nil
}

// Dat returns the named table, panicking like Main.lua's dat() when missing.
func (s *DatSet) Dat(name string) *DatFile {
	d := s.ByName[strings.ToLower(name)]
	if d == nil {
		panic(name + ".dat not found")
	}
	return d
}

// SetCell overwrites the cached cell of the named column, mirroring Lua
// scripts assigning into a row table (mods.lua fixes JewelExpansionPassiveNodes).
func (r *Row) SetCell(key string, v any) {
	ci, ok := r.File.colMap[key]
	if !ok {
		panic(fmt.Sprintf("Unknown key %s for %s.datc64", key, r.File.Name))
	}
	r.cells[ci] = v
}
