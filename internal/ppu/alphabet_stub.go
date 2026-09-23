package ppu

// Stub left so copied PPU compiles. sample1 intercepts stay inactive unless enabled.
type AlphabetPatternGenerator struct{}

func NewAlphabetPatternGenerator() *AlphabetPatternGenerator {
	return &AlphabetPatternGenerator{}
}

func (apg *AlphabetPatternGenerator) GetPatternData(char rune) ([16]uint8, bool) {
	return [16]uint8{}, false
}
