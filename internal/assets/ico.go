package assets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
)

// EncodeICO packs PNG images into a Windows .ico container.
//
// Icons8's download host rejects format=ico outright, so favicons are assembled
// here from the PNG renderings it does serve. ICO has allowed PNG-compressed
// entries since Vista, and every current browser reads them, so the entries are
// stored as-is rather than re-encoded to BMP.
func EncodeICO(pngs map[int][]byte) ([]byte, error) {
	if len(pngs) == 0 {
		return nil, fmt.Errorf("no images to pack")
	}
	sizes := make([]int, 0, len(pngs))
	for s := range pngs {
		if s <= 0 || s > 256 {
			return nil, fmt.Errorf("ICO entries must be 1-256px, got %d", s)
		}
		sizes = append(sizes, s)
	}
	sort.Ints(sizes)

	var buf bytes.Buffer
	// ICONDIR: reserved, type 1 (icon), image count.
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(len(sizes)))

	offset := 6 + 16*len(sizes)
	for _, s := range sizes {
		data := pngs[s]
		// 256 is encoded as 0 in the single-byte dimension fields.
		dim := byte(s)
		if s == 256 {
			dim = 0
		}
		buf.WriteByte(dim)                                      // width
		buf.WriteByte(dim)                                      // height
		buf.WriteByte(0)                                        // palette size (0 = truecolour)
		buf.WriteByte(0)                                        // reserved
		_ = binary.Write(&buf, binary.LittleEndian, uint16(1))  // colour planes
		_ = binary.Write(&buf, binary.LittleEndian, uint16(32)) // bits per pixel
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(data)))
		_ = binary.Write(&buf, binary.LittleEndian, uint32(offset))
		offset += len(data)
	}
	for _, s := range sizes {
		buf.Write(pngs[s])
	}
	return buf.Bytes(), nil
}

// Icon size presets per platform. These are the sets the Icons8 UI offers under
// its iOS / Android / favicon download options.
var SizePresets = map[string][]int{
	"favicon": {16, 32, 48, 64, 128, 180, 192, 256, 512},
	"ios":     {20, 29, 40, 58, 60, 76, 80, 87, 120, 152, 167, 180, 1024},
	"android": {36, 48, 72, 96, 144, 192, 512},
	"macos":   {16, 32, 64, 128, 256, 512, 1024},
	"windows": {16, 24, 32, 48, 64, 128, 256},
	"web":     {16, 32, 48, 64, 128, 256, 512},
}

// icoMaxSize is the largest dimension the ICO format can address.
const icoMaxSize = 256

// ICOSizes filters a preset down to what ICO can actually hold.
func ICOSizes(sizes []int) []int {
	out := make([]int, 0, len(sizes))
	for _, s := range sizes {
		if s > 0 && s <= icoMaxSize {
			out = append(out, s)
		}
	}
	return out
}
