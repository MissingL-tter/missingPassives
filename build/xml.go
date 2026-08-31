package build

import (
	"encoding/xml"

	"github.com/MissingL-tter/missingPassives/skills"
)

// Doc is a saved build file: the elements the calc reads, decoded but not
// yet interpreted. Numeric attributes stay strings because the reference
// reads them through tonumber, where a missing or malformed attribute
// falls back to a documented default rather than failing the load.
type Doc struct {
	Build  buildElem        `xml:"Build"`
	Items  itemsElem        `xml:"Items"`
	Tree   treeElem         `xml:"Tree"`
	Skills skills.XMLSkills `xml:"Skills"`
	Config configElem       `xml:"Config"`
}

type buildElem struct {
	Level            string `xml:"level,attr"`
	ClassName        string `xml:"className,attr"`
	AscendClassName  string `xml:"ascendClassName,attr"`
	MainSocketGroup  string `xml:"mainSocketGroup,attr"`
	MainSkillIndex   string `xml:"mainSkillIndex,attr"`
	Bandit           string `xml:"bandit,attr"`
	PantheonMajorGod string `xml:"pantheonMajorGod,attr"`
	PantheonMinorGod string `xml:"pantheonMinorGod,attr"`
	Spectres         []struct {
		ID string `xml:"id,attr"`
	} `xml:"Spectre"`
}

type itemsElem struct {
	ActiveItemSet      string        `xml:"activeItemSet,attr"`
	UseSecondWeaponSet string        `xml:"useSecondWeaponSet,attr"`
	Items              []savedItem   `xml:"Item"`
	ItemSets           []itemSetElem `xml:"ItemSet"`
	// Slots are the pre-item-set legacy form: slots saved outside any set.
	Slots []slotElem `xml:"Slot"`
}

type savedItem struct {
	ID          int             `xml:"id,attr"`
	Variant     string          `xml:"variant,attr"`
	VariantAlt  string          `xml:"variantAlt,attr"`
	VariantAlt2 string          `xml:"variantAlt2,attr"`
	VariantAlt3 string          `xml:"variantAlt3,attr"`
	VariantAlt4 string          `xml:"variantAlt4,attr"`
	VariantAlt5 string          `xml:"variantAlt5,attr"`
	Raw         string          `xml:",chardata"`
	ModRanges   []savedModRange `xml:"ModRange"`
}

type savedModRange struct {
	ID    int     `xml:"id,attr"`
	Range float64 `xml:"range,attr"`
}

type itemSetElem struct {
	ID                 string     `xml:"id,attr"`
	Title              string     `xml:"title,attr"`
	UseSecondWeaponSet string     `xml:"useSecondWeaponSet,attr"`
	Slots              []slotElem `xml:"Slot"`
}

type slotElem struct {
	Name   string `xml:"name,attr"`
	ItemID string `xml:"itemId,attr"`
	Active string `xml:"active,attr"`
}

type treeElem struct {
	ActiveSpec int        `xml:"activeSpec,attr"`
	Specs      []specElem `xml:"Spec"`
}

type specElem struct {
	ClassID                  string `xml:"classId,attr"`
	AscendClassID            string `xml:"ascendClassId,attr"`
	SecondaryAscendClassID   string `xml:"secondaryAscendClassId,attr"`
	TreeVersion              string `xml:"treeVersion,attr"`
	Nodes                    string `xml:"nodes,attr"`
	MasteryEffects           string `xml:"masteryEffects,attr"`
	ClusterHashFormatVersion string `xml:"clusterHashFormatVersion,attr"`
	Sockets                  struct {
		Sockets []struct {
			NodeID int64 `xml:"nodeId,attr"`
			ItemID int   `xml:"itemId,attr"`
		} `xml:"Socket"`
	} `xml:"Sockets"`
	Overrides struct {
		Overrides []struct {
			NodeID            int64  `xml:"nodeId,attr"`
			Dn                string `xml:"dn,attr"`
			Icon              string `xml:"icon,attr"`
			ActiveEffectImage string `xml:"activeEffectImage,attr"`
		} `xml:"Override"`
	} `xml:"Overrides"`
}

type configElem struct {
	ActiveConfigSet string `xml:"activeConfigSet,attr"`
	Sets            []struct {
		ID     string      `xml:"id,attr"`
		Title  string      `xml:"title,attr"`
		Inputs []configVal `xml:"Input"`
	} `xml:"ConfigSet"`
	Inputs []configVal `xml:"Input"`
}

// configVal is one <Input>: exactly one of the value attributes is set,
// by the option's type.
type configVal struct {
	Name    string `xml:"name,attr"`
	String  string `xml:"string,attr"`
	Number  string `xml:"number,attr"`
	Boolean string `xml:"boolean,attr"`
}

// Decode parses a saved build file.
func Decode(blob []byte) (*Doc, error) {
	var d Doc
	if err := xml.Unmarshal(blob, &d); err != nil {
		return nil, err
	}
	return &d, nil
}
