// Port of .archive/src/Export/Classes/Dat64File.lua and the loading logic of
// GGPKData.lua/Main.lua: parses the game's .dat64 tables against the column
// schemas in spec_gen.go.
//
// Offsets and row ids are 0-based (row ids are what ref cells carry); the
// Lua original is 1-based. Ref values are read as exact uint64 where the Lua
// reads them into float64 (precision above 2^53 would differ; real files
// never get there).

package export

import (
	"bytes"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var datMagic = bytes.Repeat([]byte{0xBB}, 8)

// Interval is the value of an Interval column: two signed 32-bit ints.
type Interval [2]int64

// ColType is a .dat64 column's value type.
type ColType uint8

const (
	ColBool ColType = iota
	ColInt
	ColUInt16
	ColUInt
	ColInterval
	ColFloat
	ColString
	ColEnum
	ColShortKey
	ColKey
)

// colSize is each type's fixed-data byte width (list cells are 16 bytes
// regardless of element type).
var colSize = [...]int{
	ColBool: 1, ColInt: 4, ColUInt16: 2, ColUInt: 4, ColInterval: 8,
	ColFloat: 4, ColString: 8, ColEnum: 4, ColShortKey: 8, ColKey: 16,
}

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

// Row is one lazily-decoded row. Identity is stable: the same row id always
// yields the same *Row, so ref cells can be compared by pointer just like
// the Lua row tables.
//
// The typed accessors decode by the column's spec type; a wrong accessor for
// a column is a spec/code mismatch and panics, as does an unknown column
// name. Nothing in the data itself can make them panic.
type Row struct {
	File  *DatFile
	ID    int // 0-based row id, what ref cells carry
	cells map[int]any
}

// DatSet is the registry of loaded tables; Enum/ShortKey/Key cells resolve
// through it (Main.lua's datFileByName).
type DatSet struct {
	ByName map[string]*DatFile
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
		size := colSize[col.Type]
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

// rowStart returns the byte offset of row id.
func (d *DatFile) rowStart(id int) int {
	return 4 + id*d.rowSize
}

// RowByID returns the row with the given id, or nil out of range.
func (d *DatFile) RowByID(id int) *Row {
	if id < 0 || id >= d.RowCount {
		return nil
	}
	if r := d.rowCache[id]; r != nil {
		return r
	}
	r := &Row{File: d, ID: id, cells: map[int]any{}}
	d.rowCache[id] = r
	return r
}

// Rows iterates all rows in id order.
func (d *DatFile) Rows() iter.Seq[*Row] {
	return func(yield func(*Row) bool) {
		for id := 0; id < d.RowCount; id++ {
			if !yield(d.RowByID(id)) {
				return
			}
		}
	}
}

// GetRow returns the first row whose cell in the named column equals value.
func (d *DatFile) GetRow(key string, value any) *Row {
	ki := d.col(key)
	for id := 0; id < d.RowCount; id++ {
		if cellEquals(d.readCell(id, ki), value) {
			return d.RowByID(id)
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
	ki := d.col(key)
	var out []*Row
	for id := 0; id < d.RowCount; id++ {
		cell := d.readCell(id, ki)
		if list, isList := cell.([]any); isList {
			for _, v := range list {
				if pred(v) {
					out = append(out, d.RowByID(id))
					break
				}
			}
		} else if pred(cell) {
			out = append(out, d.RowByID(id))
		}
	}
	return out
}

// col resolves a column name; an unknown name is a spec/code mismatch.
func (d *DatFile) col(key string) int {
	ci, ok := d.colMap[key]
	if !ok {
		panic(fmt.Sprintf("Unknown key %s for %s.datc64", key, d.Name))
	}
	return ci
}

func cellEquals(cell, value any) bool {
	if r, ok := value.(*Row); ok && r == nil {
		value = nil // a null ref matches null cells
	}
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

// cell returns the lazily-decoded cell at column index ci, caching it.
func (r *Row) cell(ci int) any {
	if v, done := r.cells[ci]; done {
		return v
	}
	v := r.File.readCell(r.ID, ci)
	r.cells[ci] = v
	return v
}

// Int reads an Int/UInt/UInt16 column, or an Enum column into another table
// (its raw index).
func (r *Row) Int(key string) int64 { return r.cell(r.File.col(key)).(int64) }

// IntAt reads an Int column by 0-based column index, for columns a name
// cannot reach (duplicate names resolve last-wins, matching the Lua colMap).
func (r *Row) IntAt(ci int) int64 { return r.cell(ci).(int64) }

// Str reads a String column.
func (r *Row) Str(key string) string { return r.cell(r.File.col(key)).(string) }

// Bool reads a Bool column.
func (r *Row) Bool(key string) bool { return r.cell(r.File.col(key)).(bool) }

// Float reads a Float column.
func (r *Row) Float(key string) float64 { return r.cell(r.File.col(key)).(float64) }

// Ivl reads an Interval column.
func (r *Row) Ivl(key string) Interval { return r.cell(r.File.col(key)).(Interval) }

// Ref reads a Key/ShortKey column: the referenced row, or nil for a null ref
// (or a ref into a table that is not loaded).
func (r *Row) Ref(key string) *Row {
	v := r.cell(r.File.col(key))
	if v == nil {
		return nil
	}
	return v.(*Row)
}

// Refs reads a Key-list column the way Lua ipairs walks it: the referenced
// rows up to the first null ref.
func (r *Row) Refs(key string) []*Row {
	var out []*Row
	for _, v := range r.cell(r.File.col(key)).([]any) {
		if v == nil {
			break
		}
		out = append(out, v.(*Row))
	}
	return out
}

// Ints reads an Int-list column.
func (r *Row) Ints(key string) []int64 {
	list := r.cell(r.File.col(key)).([]any)
	out := make([]int64, len(list))
	for i, v := range list {
		out[i] = v.(int64)
	}
	return out
}

// Floats reads a Float-list column.
func (r *Row) Floats(key string) []float64 {
	list := r.cell(r.File.col(key)).([]any)
	out := make([]float64, len(list))
	for i, v := range list {
		out[i] = v.(float64)
	}
	return out
}

// SetCell overwrites the cached cell of the named column, mirroring Lua
// scripts assigning into a row table (mods.lua fixes JewelExpansionPassiveNodes).
func (r *Row) SetCell(key string, v any) {
	r.cells[r.File.col(key)] = v
}

// readCell decodes row id, column ci (both 0-based).
func (d *DatFile) readCell(id, ci int) any {
	spec := d.spec[ci]
	base := d.rowStart(id) + d.cols[ci].offset
	if spec.List {
		count := bytesToULong(d.raw, base)
		offset := int(bytesToULong(d.raw, base+8)) + d.dataOffset
		size := colSize[spec.Type]
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

// readValue decodes one value at the 0-based offset, resolving refs. The
// out-of-range sentinels (-1337, 1337, "<no offset>", "<bad offset>") are
// the reference exporter's.
func (d *DatFile) readValue(spec Col, o int) any {
	b := d.raw
	var refVal uint64
	switch spec.Type {
	case ColBool:
		return o < len(b) && b[o] == 1
	case ColInt:
		if o+4 > len(b) {
			return int64(-1337)
		}
		return int64(bytesToInt(b, o))
	case ColUInt16:
		if o+2 > len(b) {
			return int64(1337)
		}
		return int64(bytesToUShort(b, o))
	case ColUInt:
		if o+4 > len(b) {
			return int64(1337)
		}
		return int64(bytesToUInt(b, o))
	case ColInterval:
		if o+8 > len(b) {
			return Interval{1337, 1337}
		}
		return Interval{int64(bytesToInt(b, o)), int64(bytesToInt(b, o+4))}
	case ColFloat:
		if o+4 > len(b) {
			return float64(-1337)
		}
		return bytesToFloat(b, o)
	case ColString:
		if o+8 > len(b) {
			return "<no offset>"
		}
		stro := bytesToULong(b, o)
		if len(b) < 7 || stro > uint64(len(b)-7) {
			return "<bad offset>"
		}
		return convertUTF16to8(b, d.dataOffset+int(stro))
	case ColEnum:
		if o+4 > len(b) {
			refVal = 1337
		} else {
			refVal = uint64(bytesToUInt(b, o))
		}
	case ColShortKey:
		if o+8 > len(b) {
			refVal = 1337
		} else {
			refVal = bytesToULong(b, o)
		}
	case ColKey:
		if o+16 > len(b) {
			refVal = 1337
		} else {
			refVal = bytesToULong(b, o)
		}
	}
	// Ref resolution, matching Dat64File:ReadValue: the value is the target's
	// 0-based row id.
	if refVal == 0xFEFEFEFE || refVal == 0xFEFEFEFEFEFEFEFE {
		return nil
	}
	other := d.set.ByName[strings.ToLower(spec.RefTo)]
	if other == nil {
		return nil
	}
	if spec.Type == ColEnum && strings.ToLower(spec.RefTo) != d.Name {
		return int64(refVal)
	}
	r := other.RowByID(int(refVal))
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

// Dat returns the named table (Main.lua's dat()); a table missing from the
// extract is an error.
func (s *DatSet) Dat(name string) (*DatFile, error) {
	d := s.ByName[strings.ToLower(name)]
	if d == nil {
		return nil, fmt.Errorf("%s.dat not found", name)
	}
	return d, nil
}
