package ppu

import (
	"fmt"
	"github.com/misawa/go-nes-by-pstack/internal/cartridge"
	"os"
)

// PPU state enumeration for cycle-accurate timing
type PPUState int

const (
	StateVisible    PPUState = iota // Scanlines 0-239: visible rendering
	StatePostRender                 // Scanline 240: post-render idle
	StateVBlank                     // Scanlines 241-260: VBlank period
	StatePreRender                  // Scanline 261: pre-render setup
)

// PendingWrite represents a deferred nametable write for VBlank timing fix
type PendingWrite struct {
	addr uint16
	data uint8
}

// BackgroundFetcher manages the 8-cycle tile fetching pipeline
// Implements authentic NES PPU background tile fetching behavior
type BackgroundFetcher struct {
	cycle uint8 // Current cycle within 8-cycle fetch sequence (0-7)

	// Fetched tile data (loaded during 8-cycle sequence)
	nameTableByte uint8 // Tile index from nametable (cycle 0)
	attributeByte uint8 // Attribute data for palette selection (cycle 2)
	patternLow    uint8 // Pattern table low byte (cycle 4)
	patternHigh   uint8 // Pattern table high byte (cycle 6)
}

// NewBackgroundFetcher creates a new BackgroundFetcher
func NewBackgroundFetcher() BackgroundFetcher {
	return BackgroundFetcher{
		cycle:         0,
		nameTableByte: 0,
		attributeByte: 0,
		patternLow:    0,
		patternHigh:   0,
	}
}

// Clock advances the background fetcher by one cycle
func (bf *BackgroundFetcher) Clock(ppu *PPU) {
	switch bf.cycle {
	case 0:
		bf.fetchNameTableByte(ppu)
	case 2:
		bf.fetchAttributeByte(ppu)
	case 4:
		bf.fetchPatternLow(ppu)
	case 6:
		bf.fetchPatternHigh(ppu)
	case 7:
		bf.loadShiftRegisters(ppu)
	}

	bf.cycle = (bf.cycle + 1) % 8
}

// fetchNameTableByte fetches tile index from nametable (cycle 0)
func (bf *BackgroundFetcher) fetchNameTableByte(ppu *PPU) {
	addr := 0x2000 | (ppu.vramAddress.Get() & 0x0FFF)
	bf.nameTableByte = ppu.ppuRead(addr)

}

// fetchAttributeByte fetches attribute data for palette selection (cycle 2)
func (bf *BackgroundFetcher) fetchAttributeByte(ppu *PPU) {
	vramAddr := ppu.vramAddress.Get()
	// Calculate attribute table address
	addr := 0x23C0 | (vramAddr & 0x0C00) | ((vramAddr >> 4) & 0x38) | ((vramAddr >> 2) & 0x07)
	attrByte := ppu.ppuRead(addr)

	// Extract 2-bit palette index for current 2x2 tile area
	if (vramAddr & 0x0002) != 0 {
		attrByte >>= 2
	}
	if (vramAddr & 0x0040) != 0 {
		attrByte >>= 4
	}
	bf.attributeByte = attrByte & 0x03
}

// fetchPatternLow fetches pattern table low byte (cycle 4)
func (bf *BackgroundFetcher) fetchPatternLow(ppu *PPU) {
	// Determine pattern table base address from PPUCTRL
	var patternTableBase uint16 = 0x0000
	if ppu.ctrl&CTRL_PATTERN_BG != 0 {
		patternTableBase = 0x1000
	}

	// Calculate fine Y offset from VRAM address
	fineY := (ppu.vramAddress.Get() >> 12) & 0x07

	// Calculate pattern address: base + tile_index * 16 + fine_y
	addr := patternTableBase + uint16(bf.nameTableByte)*16 + fineY
	bf.patternLow = ppu.ppuRead(addr)
}

// fetchPatternHigh fetches pattern table high byte (cycle 6)
func (bf *BackgroundFetcher) fetchPatternHigh(ppu *PPU) {
	// Determine pattern table base address from PPUCTRL
	var patternTableBase uint16 = 0x0000
	if ppu.ctrl&CTRL_PATTERN_BG != 0 {
		patternTableBase = 0x1000
	}

	// Calculate fine Y offset from VRAM address
	fineY := (ppu.vramAddress.Get() >> 12) & 0x07

	// Calculate pattern address: base + tile_index * 16 + fine_y + 8
	addr := patternTableBase + uint16(bf.nameTableByte)*16 + fineY + 8
	bf.patternHigh = ppu.ppuRead(addr)
}

// loadShiftRegisters loads fetched tile data into shift registers (cycle 7)
func (bf *BackgroundFetcher) loadShiftRegisters(ppu *PPU) {
	// Use the new ShiftRegisters struct
	ppu.shiftRegisters.LoadTileData(bf.patternLow, bf.patternHigh, bf.attributeByte)
}

// VRAMAddress manages the PPU's internal address registers
// Implements the authentic NES PPU address system with v, t, x, w registers
type VRAMAddress struct {
	v uint16 // Current VRAM address (15 bits)
	t uint16 // Temporary VRAM address (15 bits) 
	x uint8  // Fine X scroll (3 bits)
	w bool   // Write toggle for $2005/$2006 (1 bit)
}

// NewVRAMAddress creates a new VRAMAddress with proper initialization
func NewVRAMAddress() VRAMAddress {
	return VRAMAddress{
		v: 0,
		t: 0,
		x: 0,
		w: false,
	}
}

// Get returns the current VRAM address (v register)
func (va *VRAMAddress) Get() uint16 {
	return va.v
}

// Set sets the current VRAM address (v register)
func (va *VRAMAddress) Set(addr uint16) {
	va.v = addr & 0x7FFF // Ensure 15-bit address
}

// GetTemp returns the temporary VRAM address (t register)
func (va *VRAMAddress) GetTemp() uint16 {
	return va.t
}

// SetTemp sets the temporary VRAM address (t register)
func (va *VRAMAddress) SetTemp(addr uint16) {
	va.t = addr & 0x7FFF // Ensure 15-bit address
}

// GetFineX returns the fine X scroll (x register)
func (va *VRAMAddress) GetFineX() uint8 {
	return va.x
}

// SetFineX sets the fine X scroll (x register)
func (va *VRAMAddress) SetFineX(x uint8) {
	va.x = x & 0x07 // Ensure 3-bit value
}

// GetWriteToggle returns the write toggle state (w register)
func (va *VRAMAddress) GetWriteToggle() bool {
	return va.w
}

// SetWriteToggle sets the write toggle state (w register)
func (va *VRAMAddress) SetWriteToggle(w bool) {
	va.w = w
}

// ResetWriteToggle resets the write toggle to false (used by PPUSTATUS reads)
func (va *VRAMAddress) ResetWriteToggle() {
	va.w = false
}

// Increment increments the current VRAM address by the specified amount
func (va *VRAMAddress) Increment(amount uint16) {
	va.v = (va.v + amount) & 0x7FFF
}

// Reset resets all registers to their power-on state
func (va *VRAMAddress) Reset() {
	va.v = 0
	va.t = 0
	va.x = 0
	va.w = false
}

// IncrementX increments horizontal scroll position with nametable switching
// Called every 8 cycles during rendering (after tile fetch completion)
func (va *VRAMAddress) IncrementX() {
	// Check if coarse X is at boundary (31)
	if (va.v & 0x001F) == 31 {
		va.v &= ^uint16(0x001F) // Reset coarse X to 0
		va.v ^= 0x0400          // Switch horizontal nametable
	} else {
		va.v++ // Increment coarse X
	}
}

// IncrementY increments vertical scroll position with fine Y overflow handling
// Called at cycle 256 of each visible scanline
func (va *VRAMAddress) IncrementY() {
	// Check if fine Y is at maximum (7)
	if (va.v & 0x7000) != 0x7000 {
		va.v += 0x1000 // Increment fine Y
	} else {
		va.v &= ^uint16(0x7000)   // Reset fine Y to 0
		y := (va.v & 0x03E0) >> 5 // Extract coarse Y
		if y == 29 {
			y = 0
			va.v ^= 0x0800 // Switch vertical nametable
		} else if y == 31 {
			y = 0 // Handle invalid coarse Y value
		} else {
			y++ // Increment coarse Y
		}
		va.v = (va.v & ^uint16(0x03E0)) | (y << 5) // Update coarse Y
	}
}

// ResetX resets horizontal scroll position from temporary address
// Called at cycle 257 of every scanline during rendering
func (va *VRAMAddress) ResetX() {
	// Copy horizontal bits from t to v: ....F.. ...EDCBA = t: ....F.. ...EDCBA
	va.v = (va.v & 0xFBE0) | (va.t & 0x041F)
}

// ResetY resets vertical scroll position from temporary address  
// Called during cycles 280-304 of pre-render scanline
func (va *VRAMAddress) ResetY() {
	// Copy vertical bits from t to v: IHGF.ED CBA..... = t: IHGF.ED CBA.....
	va.v = (va.v & 0x841F) | (va.t & 0x7BE0)
}

// ShiftRegisters manages pattern and attribute shift registers for pixel output
// Implements authentic NES PPU shift register behavior for background rendering
type ShiftRegisters struct {
	// Pattern shift registers (16-bit) - hold tile pattern data
	patternLow  uint16 // Pattern plane 0 (bit 0 of each pixel)
	patternHigh uint16 // Pattern plane 1 (bit 1 of each pixel)

	// Attribute shift registers (16-bit) - hold palette selection data
	attributeLow  uint16 // Attribute bit 0 for palette selection
	attributeHigh uint16 // Attribute bit 1 for palette selection

	// Attribute latches for next 8 pixels
	// These hold the attribute bits that will be loaded into the shift registers
	attributeLatch0 uint8 // Next attribute bit 0
	attributeLatch1 uint8 // Next attribute bit 1
}

// NewShiftRegisters creates a new ShiftRegisters with proper initialization
func NewShiftRegisters() ShiftRegisters {
	return ShiftRegisters{
		patternLow:      0,
		patternHigh:     0,
		attributeLow:    0,
		attributeHigh:   0,
		attributeLatch0: 0,
		attributeLatch1: 0,
	}
}

// LoadTileData loads fetched tile data into the shift registers
// Called at cycle 7 of each 8-cycle fetch sequence
func (sr *ShiftRegisters) LoadTileData(patternLow, patternHigh, attribute uint8) {
	// Debug output for LoadTileData calls (before) - removed for clarity

	// Load pattern data into the low 8 bits of shift registers
	sr.patternLow = (sr.patternLow & 0xFF00) | uint16(patternLow)
	sr.patternHigh = (sr.patternHigh & 0xFF00) | uint16(patternHigh)

	// Debug output for LoadTileData calls (after) - removed for clarity

	// Load attribute data for next 8 pixels
	// Each tile uses the same 2-bit attribute for all 8 pixels
	// Expand attribute bits to cover 8 pixels (low 8 bits of shift registers)
	if attribute&0x01 != 0 {
		sr.attributeLow = (sr.attributeLow & 0xFF00) | 0x00FF
	} else {
		sr.attributeLow = sr.attributeLow & 0xFF00
	}

	if attribute&0x02 != 0 {
		sr.attributeHigh = (sr.attributeHigh & 0xFF00) | 0x00FF
	} else {
		sr.attributeHigh = sr.attributeHigh & 0xFF00
	}

	// Store attribute bits in latches for debugging/inspection
	sr.attributeLatch0 = attribute & 0x01
	sr.attributeLatch1 = (attribute & 0x02) >> 1
}

// Shift shifts all registers left by 1 bit for next pixel
// Called every cycle during pixel rendering
func (sr *ShiftRegisters) Shift() {
	sr.patternLow <<= 1
	sr.patternHigh <<= 1
	sr.attributeLow <<= 1
	sr.attributeHigh <<= 1
}

// GetPixel extracts pixel data from shift registers with fine X scroll support
// Returns pattern bits (0-3) and attribute bits (0-3) for current pixel
func (sr *ShiftRegisters) GetPixel(fineX uint8) (pattern, attribute uint8) {
	// Calculate bit mask based on fine X scroll offset
	bitMask := uint16(0x8000) >> fineX

	// Extract pattern bits
	pattern = 0
	if sr.patternLow&bitMask != 0 {
		pattern |= 0x01
	}
	if sr.patternHigh&bitMask != 0 {
		pattern |= 0x02
	}

	// Extract attribute bits
	attribute = 0
	if sr.attributeLow&bitMask != 0 {
		attribute |= 0x01
	}
	if sr.attributeHigh&bitMask != 0 {
		attribute |= 0x02
	}

	return pattern, attribute
}

// Reset resets all shift registers to zero
func (sr *ShiftRegisters) Reset() {
	sr.patternLow = 0
	sr.patternHigh = 0
	sr.attributeLow = 0
	sr.attributeHigh = 0
	sr.attributeLatch0 = 0
	sr.attributeLatch1 = 0
}

// GetBackgroundPixel extracts background pixel data with proper bit mask generation
// Returns the final palette index for the current pixel
func (sr *ShiftRegisters) GetBackgroundPixel(fineX uint8) uint8 {
	// Calculate bit mask based on fine X scroll offset
	bitMask := uint16(0x8000) >> fineX

	// Extract pattern bits (0-3)
	pattern := uint8(0)
	if sr.patternLow&bitMask != 0 {
		pattern |= 0x01
	}
	if sr.patternHigh&bitMask != 0 {
		pattern |= 0x02
	}

	// Extract attribute bits (0-3) for palette selection
	attribute := uint8(0)
	if sr.attributeLow&bitMask != 0 {
		attribute |= 0x01
	}
	if sr.attributeHigh&bitMask != 0 {
		attribute |= 0x02
	}

	// Combine pattern and attribute bits to form final palette index
	// Pattern bits determine the color within the palette (0-3)
	// Attribute bits determine which palette to use (0-3)
	// Final index = attribute * 4 + pattern
	return attribute*4 + pattern
}

// SpriteEvaluator manages sprite evaluation for scanline sprite detection
// Implements authentic NES PPU sprite evaluation behavior
type SpriteEvaluator struct {
	// Secondary OAM (8 sprites for current scanline)
	secondaryOAM      [32]uint8 // 8 sprites * 4 bytes each
	secondaryOAMCount uint8     // Number of sprites in secondary OAM

	// Evaluation state
	spriteIndex        uint8 // Current sprite being evaluated (0-63)
	spriteCount        uint8 // Number of sprites found for current scanline
	sprite0InSecondary bool  // Whether sprite 0 is in secondary OAM

	// Evaluation cycle state
	evaluationCycle uint8 // Current cycle within sprite evaluation (0-63)
	evaluationPhase uint8 // Current phase of evaluation (0=Y check, 1=copy)
}

// NewSpriteEvaluator creates a new SpriteEvaluator
func NewSpriteEvaluator() SpriteEvaluator {
	return SpriteEvaluator{
		secondaryOAMCount:  0,
		spriteIndex:        0,
		spriteCount:        0,
		sprite0InSecondary: false,
		evaluationCycle:    0,
		evaluationPhase:    0,
	}
}

// EvaluateSprites performs sprite evaluation for the next scanline
// Called during cycles 257-320 of each visible scanline
func (se *SpriteEvaluator) EvaluateSprites(ppu *PPU, nextScanline int16) {
	// Clear secondary OAM at start of evaluation
	if se.evaluationCycle == 0 {
		se.secondaryOAMCount = 0
		se.spriteCount = 0
		se.sprite0InSecondary = false
		for i := 0; i < 32; i++ {
			se.secondaryOAM[i] = 0xFF
		}
	}

	// Sprite evaluation happens over 64 cycles (257-320)
	if se.evaluationCycle < 64 && se.spriteCount < 8 {
		spriteIndex := se.evaluationCycle
		if spriteIndex < 64 {
			// Get sprite Y position from OAM
			spriteY := ppu.OAM[spriteIndex*4]

			// Determine sprite height (8x8 or 8x16)
			spriteHeight := uint8(8)
			if ppu.ctrl&CTRL_SPRITE_SIZE != 0 {
				spriteHeight = 16
			}

			// Check if sprite is on next scanline
			if nextScanline >= int16(spriteY) && nextScanline < int16(spriteY)+int16(spriteHeight) {
				// Copy sprite to secondary OAM
				baseIndex := se.spriteCount * 4
				se.secondaryOAM[baseIndex] = ppu.OAM[spriteIndex*4]     // Y
				se.secondaryOAM[baseIndex+1] = ppu.OAM[spriteIndex*4+1] // Tile
				se.secondaryOAM[baseIndex+2] = ppu.OAM[spriteIndex*4+2] // Attributes
				se.secondaryOAM[baseIndex+3] = ppu.OAM[spriteIndex*4+3] // X

				// Track sprite 0
				if spriteIndex == 0 {
					se.sprite0InSecondary = true
				}

				se.spriteCount++
				se.secondaryOAMCount = se.spriteCount
			}
		}
	}

	// Set sprite overflow flag if more than 8 sprites on scanline
	if se.spriteCount >= 8 {
		ppu.status |= STATUS_SPRITE_OVERFLOW
	}

	se.evaluationCycle++
	if se.evaluationCycle >= 64 {
		se.evaluationCycle = 0
	}
}

// GetSecondaryOAM returns the secondary OAM data
func (se *SpriteEvaluator) GetSecondaryOAM() [32]uint8 {
	return se.secondaryOAM
}

// GetSpriteCount returns the number of sprites for current scanline
func (se *SpriteEvaluator) GetSpriteCount() uint8 {
	return se.spriteCount
}

// IsSprite0InSecondary returns whether sprite 0 is in secondary OAM
func (se *SpriteEvaluator) IsSprite0InSecondary() bool {
	return se.sprite0InSecondary
}

// SpriteRenderer manages sprite pattern fetching and rendering
// Implements authentic NES PPU sprite rendering behavior
type SpriteRenderer struct {
	// Sprite shift registers (8 sprites max per scanline)
	spritePatternLow  [8]uint8 // Pattern low bytes for 8 sprites
	spritePatternHigh [8]uint8 // Pattern high bytes for 8 sprites
	spriteAttributes  [8]uint8 // Attribute bytes for 8 sprites
	spriteXCounters   [8]uint8 // X position counters for 8 sprites

	// Active sprite count
	activeSpriteCount uint8

	// Sprite 0 hit detection
	sprite0Active bool  // Whether sprite 0 is active on current scanline
	sprite0Index  uint8 // Index of sprite 0 in shift registers (if active)
}

// NewSpriteRenderer creates a new SpriteRenderer
func NewSpriteRenderer() SpriteRenderer {
	return SpriteRenderer{
		activeSpriteCount: 0,
		sprite0Active:     false,
		sprite0Index:      0,
	}
}

// LoadSprites loads sprite data from secondary OAM into shift registers
// Called during cycles 257-320 after sprite evaluation
func (sr *SpriteRenderer) LoadSprites(ppu *PPU, evaluator *SpriteEvaluator, scanline int16) {
	sr.activeSpriteCount = evaluator.GetSpriteCount()
	sr.sprite0Active = evaluator.IsSprite0InSecondary()

	secondaryOAM := evaluator.GetSecondaryOAM()

	// Load up to 8 sprites into shift registers
	for i := uint8(0); i < sr.activeSpriteCount && i < 8; i++ {
		baseIndex := i * 4
		spriteY := secondaryOAM[baseIndex]
		tileIndex := secondaryOAM[baseIndex+1]
		attributes := secondaryOAM[baseIndex+2]
		spriteX := secondaryOAM[baseIndex+3]

		// Store sprite data
		sr.spriteAttributes[i] = attributes
		sr.spriteXCounters[i] = spriteX

		// Calculate pattern table address
		var patternAddr uint16
		spriteHeight := uint8(8)

		if ppu.ctrl&CTRL_SPRITE_SIZE != 0 {
			// 8x16 sprites
			spriteHeight = 16
			// Pattern table determined by bit 0 of tile index
			if tileIndex&0x01 != 0 {
				patternAddr = 0x1000
			} else {
				patternAddr = 0x0000
			}
			tileIndex &= 0xFE // Clear bit 0 for 8x16 sprites
		} else {
			// 8x8 sprites
			// Pattern table determined by PPUCTRL bit 3
			if ppu.ctrl&CTRL_PATTERN_SPRITE != 0 {
				patternAddr = 0x1000
			} else {
				patternAddr = 0x0000
			}
		}

		// Calculate fine Y within sprite
		fineY := uint8(scanline) - spriteY

		// Handle vertical flipping
		if attributes&0x80 != 0 {
			fineY = spriteHeight - 1 - fineY
		}

		// For 8x16 sprites, handle top/bottom tile selection
		if spriteHeight == 16 {
			if fineY >= 8 {
				tileIndex |= 0x01 // Bottom tile
				fineY -= 8
			}
		}

		// Fetch pattern data
		addr := patternAddr + uint16(tileIndex)*16 + uint16(fineY)
		patternLow := ppu.ppuRead(addr)
		patternHigh := ppu.ppuRead(addr + 8)

		// Handle horizontal flipping
		if attributes&0x40 != 0 {
			patternLow = sr.flipByte(patternLow)
			patternHigh = sr.flipByte(patternHigh)
		}

		sr.spritePatternLow[i] = patternLow
		sr.spritePatternHigh[i] = patternHigh

		// Track sprite 0 index
		if sr.sprite0Active && i == 0 {
			sr.sprite0Index = i
		}
	}
}

// flipByte flips the bits in a byte for horizontal sprite flipping
func (sr *SpriteRenderer) flipByte(b uint8) uint8 {
	result := uint8(0)
	for i := 0; i < 8; i++ {
		if b&(1<<i) != 0 {
			result |= 1 << (7 - i)
		}
	}
	return result
}

// RenderSprites renders sprites for the current pixel position
// Returns sprite pixel data and whether sprite 0 hit occurred
func (sr *SpriteRenderer) RenderSprites(x int, backgroundPixel uint8) (spritePixel uint8, sprite0Hit bool) {
	spritePixel = 0
	sprite0Hit = false

	// Check all active sprites from front to back (priority order)
	for i := int(sr.activeSpriteCount) - 1; i >= 0; i-- {
		// Check if sprite is at current X position
		if sr.spriteXCounters[i] == 0 {
			// Extract pixel from sprite pattern
			bitMask := uint8(0x80)
			pattern := uint8(0)

			if sr.spritePatternLow[i]&bitMask != 0 {
				pattern |= 0x01
			}
			if sr.spritePatternHigh[i]&bitMask != 0 {
				pattern |= 0x02
			}

			// Check for non-transparent pixel
			if pattern != 0 {
				// Calculate sprite palette index (16-31)
				attributes := sr.spriteAttributes[i]
				paletteIndex := 16 + (attributes&0x03)*4 + pattern

				// Check sprite priority (bit 5 of attributes)
				spritePriority := (attributes & 0x20) == 0

				// Render sprite pixel if priority allows
				if spritePriority || (backgroundPixel&0x03) == 0 {
					spritePixel = paletteIndex
				}

				// Check for sprite 0 hit
				if sr.sprite0Active && i == int(sr.sprite0Index) && (backgroundPixel&0x03) != 0 && x < 255 {
					sprite0Hit = true
				}
			}

			// Shift sprite pattern for next pixel
			sr.spritePatternLow[i] <<= 1
			sr.spritePatternHigh[i] <<= 1
		}

		// Decrement X counter if sprite is active
		if sr.spriteXCounters[i] > 0 {
			sr.spriteXCounters[i]--
		}
	}

	return spritePixel, sprite0Hit
}

// renderPixelFromShiftRegisters renders a single pixel using shift registers
// This implements authentic NES pixel rendering with fine X scroll support
func (ppu *PPU) renderPixelFromShiftRegisters(x, y int) {
	// Bounds checking
	if x < 0 || x >= 256 || y < 0 || y >= 240 {
		return
	}
	
	// Debug logging removed for clean execution

	var colorIndex uint8
	var backgroundPixel uint8

	// Background rendering
	if (ppu.mask & MASK_RENDER_BG) != 0 {
		// Get background pixel using shift registers with fine X scroll
		paletteIndex := ppu.shiftRegisters.GetBackgroundPixel(ppu.vramAddress.GetFineX())
		backgroundPixel = paletteIndex

		// Handle transparent pixels (pattern = 0)
		pattern := paletteIndex & 0x03
		if pattern == 0 {
			// Universal background color (palette index 0)
			colorIndex = ppu.tblPalette[0]
		} else {
			// Use calculated palette index
			if paletteIndex < 16 { // Background palettes are 0-15
				colorIndex = ppu.tblPalette[paletteIndex]
			} else {
				colorIndex = ppu.tblPalette[0] // Fallback to universal background
			}
		}
	} else {
		// Rendering disabled - show universal background color
		colorIndex = ppu.tblPalette[0]
		backgroundPixel = 0
	}

	// Sprite rendering
	if (ppu.mask & MASK_RENDER_SPR) != 0 {
		spritePixel, sprite0Hit := ppu.spriteRenderer.RenderSprites(x, backgroundPixel)

		// Use sprite pixel if non-transparent
		if spritePixel != 0 {
			if spritePixel < 32 { // Sprite palettes are 16-31
				colorIndex = ppu.tblPalette[spritePixel]
			}
		}

		// Set sprite 0 hit flag
		if sprite0Hit {
			ppu.status |= STATUS_SPRITE_ZERO_HIT
		}
	}

	// Write pixel to screen buffer
	ppu.sprScreen[y][x] = colorIndex
	


}

// Debug methods for shift register inspection

// GetShiftRegisterPatternLow returns the pattern low shift register value
func (ppu *PPU) GetShiftRegisterPatternLow() uint16 {
	return ppu.shiftRegisters.patternLow
}

// GetShiftRegisterPatternHigh returns the pattern high shift register value
func (ppu *PPU) GetShiftRegisterPatternHigh() uint16 {
	return ppu.shiftRegisters.patternHigh
}

// GetShiftRegisterAttributeLow returns the attribute low shift register value
func (ppu *PPU) GetShiftRegisterAttributeLow() uint16 {
	return ppu.shiftRegisters.attributeLow
}

// GetShiftRegisterAttributeHigh returns the attribute high shift register value
func (ppu *PPU) GetShiftRegisterAttributeHigh() uint16 {
	return ppu.shiftRegisters.attributeHigh
}

type PPU struct {
	// Palette memory (moved to beginning to avoid corruption)
	tblPalette [32]uint8
	// Name tables (4 * 1KB each, but only 2KB physical memory due to mirroring)
	tblName [2][1024]uint8
	// Pattern tables (stored in cartridge CHR-ROM)

	// Registers
	ctrl    uint8 // $2000 - PPUCTRL
	mask    uint8 // $2001 - PPUMASK
	status  uint8 // $2002 - PPUSTATUS
	oamAddr uint8 // $2003 - OAMADDR
	oamData uint8 // $2004 - OAMDATA
	scroll  uint8 // $2005 - PPUSCROLL
	addr    uint8 // $2006 - PPUADDR
	data    uint8 // $2007 - PPUDATA

	// VRAM address management system
	vramAddress VRAMAddress

	// Background tile fetching pipeline
	backgroundFetcher BackgroundFetcher

	// Shift registers for pixel output
	shiftRegisters ShiftRegisters

	// Sprite evaluation and rendering system
	spriteEvaluator SpriteEvaluator
	spriteRenderer  SpriteRenderer

	// Legacy fields (will be removed after migration)
	vramAddr uint16 // Current VRAM address (15 bits) - DEPRECATED
	tempAddr uint16 // Temporary VRAM address (15 bits) - DEPRECATED
	fineX    uint8  // Fine X scroll (3 bits) - DEPRECATED

	// Address latch for $2005 and $2006 - DEPRECATED
	addressLatch bool

	// Data buffer for $2007 reads
	dataBuffer uint8

	// Cycle-accurate timing state
	state    PPUState // Current PPU state
	scanline int16    // Current scanline (-1 to 261)
	dot      uint16   // Current cycle within scanline (0-340)

	// Rendering state
	renderingEnabled bool // Cache for rendering enabled check

	// Frame completion flag
	frameComplete bool
	
	// Frame counter for debugging timing issues (Task 9.1.4)
	frameCount uint64
	
	// VBlank timing fix for nametable updates (Task 9.1.4)
	pendingNametableWrites []PendingWrite

	// Object Attribute Memory (256 bytes, 64 sprites * 4 bytes each)
	OAM [256]uint8

	// Screen buffer (256x240 pixels)
	sprScreen [240][256]uint8

	// Cartridge connection
	cart *cartridge.Cartridge

	// NMI flag
	nmiOccurred bool

	// Internal shift registers (authentic NES PPU behavior)
	bgShiftPatternLo uint16
	bgShiftPatternHi uint16
	bgShiftAttribLo  uint16
	bgShiftAttribHi  uint16

	// Next tile data (fetched every 8 cycles)
	nextTileId     uint8
	nextTileAttrib uint8
	nextTileLsb    uint8
	nextTileMsb    uint8

	// Legacy fields for compatibility (will be removed later)
	cycle uint16 // Deprecated: use dot instead
}

// PPU register addresses
const (
	PPUCTRL   = 0x2000
	PPUMASK   = 0x2001
	PPUSTATUS = 0x2002
	OAMADDR   = 0x2003
	OAMDATA   = 0x2004
	PPUSCROLL = 0x2005
	PPUADDR   = 0x2006
	PPUDATA   = 0x2007
)

// PPUCTRL flags
const (
	CTRL_NAMETABLE_X    = 0x01
	CTRL_NAMETABLE_Y    = 0x02
	CTRL_INCREMENT_MODE = 0x04
	CTRL_PATTERN_SPRITE = 0x08
	CTRL_PATTERN_BG     = 0x10
	CTRL_SPRITE_SIZE    = 0x20
	CTRL_SLAVE_MODE     = 0x40
	CTRL_ENABLE_NMI     = 0x80
)

// PPUMASK flags
const (
	MASK_GRAYSCALE       = 0x01
	MASK_RENDER_BG_LEFT  = 0x02
	MASK_RENDER_SPR_LEFT = 0x04
	MASK_RENDER_BG       = 0x08
	MASK_RENDER_SPR      = 0x10
	MASK_ENHANCE_RED     = 0x20
	MASK_ENHANCE_GREEN   = 0x40
	MASK_ENHANCE_BLUE    = 0x80
)

// PPUSTATUS flags
const (
	STATUS_SPRITE_OVERFLOW = 0x20
	STATUS_SPRITE_ZERO_HIT = 0x40
	STATUS_VBLANK          = 0x80
)

func NewPPU() *PPU {
	ppu := &PPU{
		// Initialize cycle-accurate timing
		state:    StatePreRender, // Start in pre-render state
		scanline: 261,            // Pre-render scanline
		dot:      0,              // Start at cycle 0

		// Initialize VRAM address management
		vramAddress: NewVRAMAddress(),

		// Initialize background fetching pipeline
		backgroundFetcher: NewBackgroundFetcher(),

		// Initialize shift registers
		shiftRegisters: NewShiftRegisters(),

		// Initialize sprite system
		spriteEvaluator: NewSpriteEvaluator(),
		spriteRenderer:  NewSpriteRenderer(),

		// Legacy compatibility
		cycle: 0,

		// Initialize registers
		status:       0,
		vramAddr:     0,
		tempAddr:     0,
		fineX:        0,
		addressLatch: false,

		// Initialize rendering state
		renderingEnabled: false,
	}

	// パレット初期化 - 全て0x00でクリア（デバッグ用）
	for i := 0; i < 32; i++ {
		ppu.tblPalette[i] = 0x00
	}

	// レンダリング無効で開始
	ppu.mask = 0

	return ppu
}

func (ppu *PPU) ConnectCartridge(cart *cartridge.Cartridge) {
	ppu.cart = cart
}

func (ppu *PPU) CPURead(addr uint16) uint8 {
	data := uint8(0x00)

	switch addr {
	case PPUCTRL:
		// Cannot read from PPUCTRL
	case PPUMASK:
		// Cannot read from PPUMASK
	case PPUSTATUS:
		// Return status register value (bits 7-5) + open bus bits (4-0)
		data = (ppu.status & 0xE0) | (ppu.dataBuffer & 0x1F)

		// PPUSTATUS read side effects with cycle-accurate timing
		// Clear VBlank flag immediately after read
		ppu.status &= ^uint8(STATUS_VBLANK)

		// Reset write toggle for $2005/$2006 (w register)
		ppu.vramAddress.ResetWriteToggle()

		// Legacy compatibility
		ppu.addressLatch = false
	case OAMADDR:
		// Cannot read from OAMADDR
	case OAMDATA:
		data = ppu.OAM[ppu.oamAddr]
	case PPUSCROLL:
		// Cannot read from PPUSCROLL
	case PPUADDR:
		// Cannot read from PPUADDR
	case PPUDATA:
		data = ppu.dataBuffer
		currentAddr := ppu.vramAddress.Get()
		ppu.dataBuffer = ppu.ppuRead(currentAddr)

		// Palette memory reads are immediate
		if currentAddr >= 0x3F00 {
			data = ppu.dataBuffer
		}

		if ppu.ctrl&CTRL_INCREMENT_MODE != 0 {
			ppu.vramAddress.Increment(32)
		} else {
			ppu.vramAddress.Increment(1)
		}

		// Legacy compatibility
		ppu.vramAddr = ppu.vramAddress.Get()
	}

	return data
}

func (ppu *PPU) CPUWrite(addr uint16, data uint8) {

	switch addr {
	case PPUCTRL:
		ppu.ctrl = data

		// Update nametable bits in temporary address (t register)
		newTemp := (ppu.vramAddress.GetTemp() & 0xF3FF) | ((uint16(data) & 0x03) << 10)
		ppu.vramAddress.SetTemp(newTemp)

		// Handle NMI enable/disable with proper timing
		// If VBlank flag is set and NMI is being enabled, trigger NMI
		if (data&CTRL_ENABLE_NMI != 0) && (ppu.status&STATUS_VBLANK != 0) {
			ppu.nmiOccurred = true
		}

		// Legacy compatibility
		ppu.tempAddr = ppu.vramAddress.GetTemp()
	case PPUMASK:
		ppu.mask = data
	case PPUSTATUS:
		// Cannot write to PPUSTATUS
	case OAMADDR:
		ppu.oamAddr = data
	case OAMDATA:
		ppu.OAM[ppu.oamAddr] = data
	case PPUSCROLL:
		if !ppu.vramAddress.GetWriteToggle() {
			// First write - X scroll
			ppu.vramAddress.SetFineX(data & 0x07)                                 // Fine X (3 bits)
			newTemp := (ppu.vramAddress.GetTemp() & 0xFFE0) | (uint16(data) >> 3) // Coarse X (5 bits)
			ppu.vramAddress.SetTemp(newTemp)
			ppu.vramAddress.SetWriteToggle(true)
		} else {
			// Second write - Y scroll
			currentTemp := ppu.vramAddress.GetTemp()
			currentTemp = (currentTemp & 0x8FFF) | ((uint16(data) & 0x07) << 12) // Fine Y (3 bits)
			currentTemp = (currentTemp & 0xFC1F) | ((uint16(data) & 0xF8) << 2)  // Coarse Y (5 bits)
			ppu.vramAddress.SetTemp(currentTemp)
			ppu.vramAddress.SetWriteToggle(false)
		}

		// Legacy compatibility
		ppu.addressLatch = ppu.vramAddress.GetWriteToggle()
		ppu.fineX = ppu.vramAddress.GetFineX()
		ppu.tempAddr = ppu.vramAddress.GetTemp()
	case PPUADDR:
		if !ppu.vramAddress.GetWriteToggle() {
			// First write - high byte (only upper 6 bits used)
			newTemp := (ppu.vramAddress.GetTemp() & 0x00FF) | ((uint16(data) & 0x3F) << 8)
			ppu.vramAddress.SetTemp(newTemp)
			ppu.vramAddress.SetWriteToggle(true)
		} else {
			// Second write - low byte
			newTemp := (ppu.vramAddress.GetTemp() & 0xFF00) | uint16(data)
			ppu.vramAddress.SetTemp(newTemp)
			ppu.vramAddress.Set(newTemp) // t -> v
			ppu.vramAddress.SetWriteToggle(false)
		}

		// Legacy compatibility
		ppu.addressLatch = ppu.vramAddress.GetWriteToggle()
		ppu.tempAddr = ppu.vramAddress.GetTemp()
		ppu.vramAddr = ppu.vramAddress.Get()
	case PPUDATA:
		addrBefore := ppu.vramAddress.Get()
		
		ppu.ppuWrite(addrBefore, data)
		// Increment VRAM address using new system
		if ppu.ctrl&CTRL_INCREMENT_MODE != 0 {
			ppu.vramAddress.Increment(32)
		} else {
			ppu.vramAddress.Increment(1)
		}

		// Legacy compatibility
		ppu.vramAddr = ppu.vramAddress.Get()
	}
}

func (ppu *PPU) ppuRead(addr uint16) uint8 {
	addr &= 0x3FFF

	if addr >= 0x0000 && addr <= 0x1FFF {
		// Pattern table - read from cartridge
		return ppu.cart.PPURead(addr)
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		// Name table
		addr &= 0x0FFF

		if ppu.cart.Mirror == 0 { // 垂直ミラーリング
			if addr >= 0x0000 && addr <= 0x03FF {
				if addr < 1024 {
					return ppu.tblName[0][addr]
				}
			} else if addr >= 0x0400 && addr <= 0x07FF {
				index := addr - 0x0400
				if index < 1024 {
					return ppu.tblName[1][index]
				}
			} else if addr >= 0x0800 && addr <= 0x0BFF {
				index := addr - 0x0800
				if index < 1024 {
					return ppu.tblName[0][index]
				}
			} else if addr >= 0x0C00 && addr <= 0x0FFF {
				index := addr - 0x0C00
				if index < 1024 {
					return ppu.tblName[1][index]
				}
			}
		} else { // 水平ミラーリング
			if addr >= 0x0000 && addr <= 0x03FF {
				if addr < 1024 {
					return ppu.tblName[0][addr]
				}
			} else if addr >= 0x0400 && addr <= 0x07FF {
				index := addr - 0x0400
				if index < 1024 {
					return ppu.tblName[0][index]
				}
			} else if addr >= 0x0800 && addr <= 0x0BFF {
				index := addr - 0x0800
				if index < 1024 {
					return ppu.tblName[1][index]
				}
			} else if addr >= 0x0C00 && addr <= 0x0FFF {
				index := addr - 0x0C00
				if index < 1024 {
					return ppu.tblName[1][index]
				}
			}
		}
	} else if addr >= 0x3F00 && addr <= 0x3FFF {
		// Palette table
		addr &= 0x001F
		if addr == 0x0010 {
			addr = 0x0000
		}
		if addr == 0x0014 {
			addr = 0x0004
		}
		if addr == 0x0018 {
			addr = 0x0008
		}
		if addr == 0x001C {
			addr = 0x000C
		}
		return ppu.tblPalette[addr]
	}

	return 0x00
}

func (ppu *PPU) ppuWrite(addr uint16, data uint8) {
	addr &= 0x3FFF

	if addr >= 0x0000 && addr <= 0x1FFF {
		// Pattern table - write to cartridge
		ppu.cart.PPUWrite(addr, data)
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		// ネームテーブル - 標準的なNESエミュレーション
		addr &= 0x0FFF

		// Task 9.1.4: VBlank timing fix
		isRenderingPeriod := (ppu.scanline >= 0 && ppu.scanline <= 239)
		
		if isRenderingPeriod {
			// Buffer the write for VBlank period
			ppu.pendingNametableWrites = append(ppu.pendingNametableWrites, PendingWrite{
				addr: addr + 0x2000,
				data: data,
			})
			return // Don't write immediately, defer until VBlank
		}

		if ppu.cart.Mirror == 0 { // 垂直ミラーリング
			if addr >= 0x0000 && addr <= 0x03FF {
				if addr < 1024 {
					ppu.tblName[0][addr] = data
				}
			} else if addr >= 0x0400 && addr <= 0x07FF {
				index := addr - 0x0400
				if index < 1024 {
					ppu.tblName[1][index] = data
				}
			} else if addr >= 0x0800 && addr <= 0x0BFF {
				index := addr - 0x0800
				if index < 1024 {
					ppu.tblName[0][index] = data
				}
			} else if addr >= 0x0C00 && addr <= 0x0FFF {
				index := addr - 0x0C00
				if index < 1024 {
					ppu.tblName[1][index] = data
				}
			}
		} else { // 水平ミラーリング
			if addr >= 0x0000 && addr <= 0x03FF {
				if addr < 1024 {
					ppu.tblName[0][addr] = data
				}
			} else if addr >= 0x0400 && addr <= 0x07FF {
				index := addr - 0x0400
				if index < 1024 {
					ppu.tblName[0][index] = data
				}
			} else if addr >= 0x0800 && addr <= 0x0BFF {
				index := addr - 0x0800
				if index < 1024 {
					ppu.tblName[1][index] = data
				}
			} else if addr >= 0x0C00 && addr <= 0x0FFF {
				index := addr - 0x0C00
				if index < 1024 {
					ppu.tblName[1][index] = data
				}
			}
		}
	} else if addr >= 0x3F00 && addr <= 0x3FFF {
		// Palette table
		addr &= 0x001F
		if addr == 0x0010 {
			addr = 0x0000
		}
		if addr == 0x0014 {
			addr = 0x0004
		}
		if addr == 0x0018 {
			addr = 0x0008
		}
		if addr == 0x001C {
			addr = 0x000C
		}

		// Standard NES palette write
		if addr < 32 {
			ppu.tblPalette[addr] = data
		}
	}
}

// Clock advances the PPU by one cycle with cycle-accurate timing
func (ppu *PPU) Clock() {
	// Update rendering enabled state cache
	ppu.renderingEnabled = (ppu.mask & (MASK_RENDER_BG | MASK_RENDER_SPR)) != 0

	// State-based PPU processing
	switch ppu.state {
	case StateVisible:
		ppu.clockVisible()
	case StatePostRender:
		ppu.clockPostRender()
	case StateVBlank:
		ppu.clockVBlank()
	case StatePreRender:
		ppu.clockPreRender()
	}

	// Advance timing and update state
	ppu.advanceTiming()

	// Legacy compatibility - keep old cycle counter in sync
	ppu.cycle = ppu.dot
}

// clockVisible handles visible scanlines (0-239)
func (ppu *PPU) clockVisible() {
	if ppu.dot >= 1 && ppu.dot <= 256 {
		// Pixel rendering phase
		if ppu.renderingEnabled {
			// Render current pixel using shift registers
			x := int(ppu.dot - 1) // Convert dot to pixel X coordinate
			y := int(ppu.scanline)
			ppu.renderPixelFromShiftRegisters(x, y)

			// Shift registers every cycle for next pixel
			ppu.shiftRegisters.Shift()

			// Background tile fetching pipeline
			ppu.backgroundFetcher.Clock(ppu)

			// Horizontal scroll increment every 8 cycles after tile fetch completion
			// FIXED: Only increment during actual rendering, not during nestest.nes initialization
			if (ppu.dot-1)%8 == 7 && ppu.dot <= 256 && ppu.renderingEnabled && (ppu.mask & MASK_RENDER_BG) != 0 {
				ppu.vramAddress.IncrementX()
				// Legacy compatibility
				ppu.vramAddr = ppu.vramAddress.Get()
			}

			// Vertical scroll increment at end of visible area
			// FIXED: Only increment during actual rendering
			if ppu.dot == 256 && ppu.renderingEnabled && (ppu.mask & MASK_RENDER_BG) != 0 {
				ppu.vramAddress.IncrementY()
				// Legacy compatibility
				ppu.vramAddr = ppu.vramAddress.Get()
			}
		}
	} else if ppu.dot == 257 {
		// Horizontal scroll reset at start of HBlank
		if ppu.renderingEnabled {
			ppu.vramAddress.ResetX()
			// Legacy compatibility
			ppu.vramAddr = ppu.vramAddress.Get()
		}
	} else if ppu.dot >= 257 && ppu.dot <= 320 {
		// Sprite evaluation phase
		if ppu.renderingEnabled {
			nextScanline := ppu.scanline + 1
			if nextScanline > 239 {
				nextScanline = -1 // Wrap to pre-render scanline
			}
			ppu.spriteEvaluator.EvaluateSprites(ppu, nextScanline)

			// Load sprites into renderer at end of evaluation
			if ppu.dot == 320 {
				ppu.spriteRenderer.LoadSprites(ppu, &ppu.spriteEvaluator, nextScanline)
			}
		}
	} else if ppu.dot >= 321 && ppu.dot <= 336 {
		// Next scanline tile prefetch
		if ppu.renderingEnabled {
			// Background tile fetching for next scanline (but no shifting)
			ppu.backgroundFetcher.Clock(ppu)
		}
	}
}

// clockPostRender handles post-render scanline (240)
func (ppu *PPU) clockPostRender() {
	// Post-render scanline is idle - no processing needed
}

// clockVBlank handles VBlank scanlines (241-260)
func (ppu *PPU) clockVBlank() {
	// VBlank flag and NMI handling
	if ppu.scanline == 241 && ppu.dot == 1 {
		ppu.status |= STATUS_VBLANK
		if ppu.ctrl&CTRL_ENABLE_NMI != 0 {
			ppu.nmiOccurred = true
		}
		
		// Task 9.1.4: Apply pending nametable writes during VBlank
		if len(ppu.pendingNametableWrites) > 0 {
			for _, write := range ppu.pendingNametableWrites {
				// Apply the deferred write now
				ppu.applyNametableWrite(write.addr, write.data)
			}
			
			// Clear pending writes
			ppu.pendingNametableWrites = nil
		}
	}
}

// clockPreRender handles pre-render scanline (261)
func (ppu *PPU) clockPreRender() {
	// Clear PPU status flags at start of pre-render
	if ppu.dot == 1 {
		ppu.status &= ^uint8(STATUS_VBLANK)
		ppu.status &= ^uint8(STATUS_SPRITE_ZERO_HIT)
		ppu.status &= ^uint8(STATUS_SPRITE_OVERFLOW)
		ppu.nmiOccurred = false
	}

	// Vertical scroll reset during cycles 280-304
	if ppu.dot >= 280 && ppu.dot <= 304 {
		if ppu.renderingEnabled {
			ppu.vramAddress.ResetY()
			// Legacy compatibility
			ppu.vramAddr = ppu.vramAddress.Get()
		}
	}

	// Background fetching and scroll operations during pre-render (same as visible scanlines)
	if ppu.renderingEnabled {
		if ppu.dot >= 1 && ppu.dot <= 256 {
			// Background tile fetching pipeline (but no pixel rendering/shifting)
			ppu.backgroundFetcher.Clock(ppu)

			// Horizontal scroll increment every 8 cycles
			// FIXED: Only increment during actual rendering, not during initialization
			if (ppu.dot-1)%8 == 7 && (ppu.mask & MASK_RENDER_BG) != 0 {
				ppu.vramAddress.IncrementX()
				// Legacy compatibility
				ppu.vramAddr = ppu.vramAddress.Get()
			}

			// Vertical scroll increment at cycle 256
			// FIXED: Only increment during actual rendering
			if ppu.dot == 256 && (ppu.mask & MASK_RENDER_BG) != 0 {
				ppu.vramAddress.IncrementY()
				// Legacy compatibility
				ppu.vramAddr = ppu.vramAddress.Get()
			}
		} else if ppu.dot == 257 {
			// Horizontal scroll reset
			ppu.vramAddress.ResetX()
			// Legacy compatibility
			ppu.vramAddr = ppu.vramAddress.Get()
		} else if ppu.dot >= 321 && ppu.dot <= 336 {
			// Next scanline tile prefetch (but no shifting)
			ppu.backgroundFetcher.Clock(ppu)
		}
	}

	// Frame completion at end of pre-render
	if ppu.dot == 340 {
		ppu.frameComplete = true
		ppu.frameCount++
	}
}

// advanceTiming advances PPU timing and updates state
func (ppu *PPU) advanceTiming() {
	ppu.dot++

	// End of scanline - advance to next scanline
	if ppu.dot >= 341 {
		ppu.dot = 0
		ppu.scanline++

		// Update state based on scanline
		ppu.updateState()

		// Handle scanline wraparound
		if ppu.scanline > 261 {
			ppu.scanline = -1 // Pre-render scanline is -1
		}
	}
}

// updateState updates PPU state based on current scanline
func (ppu *PPU) updateState() {
	switch {
	case ppu.scanline >= 0 && ppu.scanline <= 239:
		ppu.state = StateVisible
	case ppu.scanline == 240:
		ppu.state = StatePostRender
	case ppu.scanline >= 241 && ppu.scanline <= 260:
		ppu.state = StateVBlank
	case ppu.scanline == 261 || ppu.scanline == -1:
		ppu.state = StatePreRender
	}
}

func (ppu *PPU) FrameComplete() bool {
	if ppu.frameComplete {
		ppu.frameComplete = false // Reset flag when checked
		return true
	}
	return false
}

// NMI returns true if NMI should be triggered and clears the flag
// Maintains proper CPU-PPU synchronization with 3:1 cycle ratio
func (ppu *PPU) NMI() bool {
	if ppu.nmiOccurred {
		ppu.nmiOccurred = false
		return true
	}
	return false
}

// GetScreen returns the current screen buffer for display
// Compatible with existing screen buffer format
func (ppu *PPU) GetScreen() *[240][256]uint8 {
	return &ppu.sprScreen
}

// Debug methods for investigating PPU state
// These methods maintain compatibility with existing debug interfaces
func (ppu *PPU) GetNametableByte(addr int) uint8 {
	if addr < 1024 {
		return ppu.tblName[0][addr]
	}
	return 0
}

func (ppu *PPU) GetPaletteByte(addr int) uint8 {
	if addr < 32 {
		return ppu.tblPalette[addr]
	}
	return 0
}

func (ppu *PPU) GetMask() uint8 {
	return ppu.mask
}

// DumpNametable dumps nametable contents to a file for debugging
func (ppu *PPU) DumpNametable(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Nametable 0 (0x2000-0x23FF):\n")
	for y := 0; y < 30; y++ {
		fmt.Fprintf(file, "Row %02d: ", y)
		for x := 0; x < 32; x++ {
			addr := y*32 + x
			if addr < 1024 {
				fmt.Fprintf(file, "%02X ", ppu.tblName[0][addr])
			}
		}
		fmt.Fprintf(file, "\n")
	}

	fmt.Fprintf(file, "\nNametable 1 (0x2400-0x27FF):\n")
	for y := 0; y < 30; y++ {
		fmt.Fprintf(file, "Row %02d: ", y)
		for x := 0; x < 32; x++ {
			addr := y*32 + x
			if addr < 1024 {
				fmt.Fprintf(file, "%02X ", ppu.tblName[1][addr])
			}
		}
		fmt.Fprintf(file, "\n")
	}

	return nil
}

func (ppu *PPU) GetCtrl() uint8 {
	return ppu.ctrl
}

func (ppu *PPU) SetTiming(scanline int16, dot uint16) {
	ppu.scanline = scanline
	ppu.dot = dot
	ppu.updateState()
}

func (ppu *PPU) GetScanline() int16 {
	return ppu.scanline
}

func (ppu *PPU) GetCycle() uint16 {
	return ppu.dot // Return current dot position
}

// GetTotalCycles returns the total PPU cycles since reset
func (ppu *PPU) GetTotalCycles() int {
	// Calculate total PPU cycles based on scanline and dot position
	// NES PPU: 341 cycles per scanline, 262 scanlines per frame
	totalScanlines := int(ppu.scanline)
	if ppu.scanline == -1 {
		totalScanlines = 261 // Pre-render scanline is 261
	}
	return totalScanlines*341 + int(ppu.dot)
}

// GetState returns current PPU state for debugging
// Updated to work with new PPU state machine
func (ppu *PPU) GetState() PPUState {
	return ppu.state
}

// GetDot returns current dot position within scanline
func (ppu *PPU) GetDot() uint16 {
	return ppu.dot
}

// IsRenderingEnabled returns whether background or sprite rendering is enabled
func (ppu *PPU) IsRenderingEnabled() bool {
	return ppu.renderingEnabled
}

// GetStateString returns human-readable state name for debugging
func (ppu *PPU) GetStateString() string {
	switch ppu.state {
	case StateVisible:
		return "Visible"
	case StatePostRender:
		return "PostRender"
	case StateVBlank:
		return "VBlank"
	case StatePreRender:
		return "PreRender"
	default:
		return "Unknown"
	}
}

func (ppu *PPU) GetStatus() uint8 {
	return ppu.status
}

func (ppu *PPU) GetVramAddr() uint16 {
	return ppu.vramAddress.Get()
}

func (ppu *PPU) GetPaletteRaw() *[32]uint8 {
	return &ppu.tblPalette
}

// OAMDMA - Sprite DMA transfer from CPU memory to OAM
func (ppu *PPU) OAMDMA(page uint8, cpuRam []uint8) {
	// Transfer 256 bytes from $XX00-$XXFF to OAM
	baseAddr := uint16(page) << 8

	for i := 0; i < 256; i++ {
		// Handle memory mirroring for RAM access
		srcAddr := baseAddr + uint16(i)
		var data uint8

		if srcAddr <= 0x1FFF {
			// CPU RAM (mirrored every 2KB)
			data = cpuRam[srcAddr&0x07FF]
		} else {
			// For DMA from other memory regions, we'd need full bus access
			// For now, assume 0x00 for non-RAM regions
			data = 0x00
		}

		ppu.OAM[i] = data
	}
}

// 正確なNES PPUタイル取得メソッド群

// fetchNameTableByte - ネームテーブルバイト取得 (サイクル0)
func (ppu *PPU) fetchNameTableByte() {
	// 現在のVRAMアドレス(v)からネームテーブルバイトを取得
	addr := 0x2000 | (ppu.vramAddr & 0x0FFF)
	ppu.nextTileId = ppu.ppuRead(addr)
}

// fetchAttributeByte - アトリビュートバイト取得 (サイクル2)
func (ppu *PPU) fetchAttributeByte() {
	// アトリビュートテーブルアドレス計算
	addr := 0x23C0 | (ppu.vramAddr & 0x0C00) | ((ppu.vramAddr >> 4) & 0x38) | ((ppu.vramAddr >> 2) & 0x07)
	attrByte := ppu.ppuRead(addr)

	// 2x2タイル領域内での位置を計算
	if (ppu.vramAddr & 0x0002) != 0 {
		attrByte >>= 2
	}
	if (ppu.vramAddr & 0x0040) != 0 {
		attrByte >>= 4
	}
	ppu.nextTileAttrib = attrByte & 0x03
}

// fetchPatternLow - パターンテーブル下位バイト取得 (サイクル4)
func (ppu *PPU) fetchPatternLow() {
	var patternTableBase uint16 = 0x0000
	if ppu.ctrl&CTRL_PATTERN_BG != 0 {
		patternTableBase = 0x1000
	}

	fineY := (ppu.vramAddr >> 12) & 0x07
	addr := patternTableBase + uint16(ppu.nextTileId)*16 + fineY
	ppu.nextTileLsb = ppu.ppuRead(addr)
}

// fetchPatternHigh - パターンテーブル上位バイト取得 (サイクル6)
func (ppu *PPU) fetchPatternHigh() {
	var patternTableBase uint16 = 0x0000
	if ppu.ctrl&CTRL_PATTERN_BG != 0 {
		patternTableBase = 0x1000
	}

	fineY := (ppu.vramAddr >> 12) & 0x07
	addr := patternTableBase + uint16(ppu.nextTileId)*16 + fineY + 8
	ppu.nextTileMsb = ppu.ppuRead(addr)
}

// loadShiftRegisters - シフトレジスタにタイルデータをロード (サイクル7)
func (ppu *PPU) loadShiftRegisters() {
	// パターンデータをシフトレジスタにロード
	ppu.bgShiftPatternLo = (ppu.bgShiftPatternLo & 0xFF00) | uint16(ppu.nextTileLsb)
	ppu.bgShiftPatternHi = (ppu.bgShiftPatternHi & 0xFF00) | uint16(ppu.nextTileMsb)

	// アトリビュートデータをシフトレジスタにロード
	if ppu.nextTileAttrib&0x01 != 0 {
		ppu.bgShiftAttribLo |= 0x00FF
	} else {
		ppu.bgShiftAttribLo &= 0xFF00
	}

	if ppu.nextTileAttrib&0x02 != 0 {
		ppu.bgShiftAttribHi |= 0x00FF
	} else {
		ppu.bgShiftAttribHi &= 0xFF00
	}
}

// renderPixel - シフトレジスタからピクセルをレンダリング
func (ppu *PPU) renderPixel() {
	x := int(ppu.cycle - 1)
	y := int(ppu.scanline)

	if x < 0 || x >= 256 || y < 0 || y >= 240 {
		return
	}

	bgPixel := uint8(0)
	bgPalette := uint8(0)

	if (ppu.mask & MASK_RENDER_BG) != 0 {
		// ファインXスクロールを考慮してシフトレジスタからピクセルを抽出
		bitMux := uint16(0x8000) >> ppu.fineX

		// パターンビット抽出
		bit0 := uint8(0)
		if (ppu.bgShiftPatternLo & bitMux) != 0 {
			bit0 = 1
		}
		bit1 := uint8(0)
		if (ppu.bgShiftPatternHi & bitMux) != 0 {
			bit1 = 1
		}
		bgPixel = (bit1 << 1) | bit0

		// アトリビュートビット抽出
		attr0 := uint8(0)
		if (ppu.bgShiftAttribLo & bitMux) != 0 {
			attr0 = 1
		}
		attr1 := uint8(0)
		if (ppu.bgShiftAttribHi & bitMux) != 0 {
			attr1 = 1
		}
		bgPalette = (attr1 << 1) | attr0
	}

	// 最終カラー決定
	var colorIndex uint8
	if bgPixel == 0 {
		colorIndex = ppu.tblPalette[0] // ユニバーサル背景色
	} else {
		colorIndex = ppu.tblPalette[bgPalette*4+bgPixel]
	}

	ppu.sprScreen[y][x] = colorIndex
}

// renderFrameImproved - 改良されたフレームレンダリング（nestest.nes互換性重視）
func (ppu *PPU) renderFrameImproved() {
	// 背景レンダリング
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			// スクロールを考慮したタイル位置計算
			// 修正: スクロール値は画面座標から減算するべき
			scrollX := x - int(ppu.fineX)
			scrollY := y
			
			// 負の値の処理
			if scrollX < 0 {
				scrollX += 256 // ラップアラウンド
			}

			// 基本的なスクロール処理（nestest.nesでは通常スクロールなし）
			tileX := scrollX / 8
			tileY := scrollY / 8

			// ネームテーブル境界チェック
			if tileX >= 32 {
				tileX = tileX % 32
			}
			if tileY >= 30 {
				tileY = tileY % 30
			}

			tileAddr := tileY*32 + tileX

			if tileAddr < 1024 {
				// ネームテーブルからパターンインデックス取得
				patternIndex := ppu.tblName[0][tileAddr]
				pixelX := scrollX % 8
				pixelY := scrollY % 8

				// PPUCTRLに基づくパターンテーブル選択
				var patternTableBase uint16 = 0x0000
				if ppu.ctrl&CTRL_PATTERN_BG != 0 {
					patternTableBase = 0x1000
				}

				// CHR-ROMからパターンデータ取得
				if ppu.cart != nil {
					patternAddr := patternTableBase + uint16(patternIndex)*16 + uint16(pixelY)
					patternLo := ppu.cart.PPURead(patternAddr)
					patternHi := ppu.cart.PPURead(patternAddr + 8)

					bit := 7 - pixelX
					pixel := ((patternHi >> bit) & 1) << 1
					pixel |= (patternLo >> bit) & 1

					// アトリビュートテーブルからパレット選択
					attrX := tileX / 4
					attrY := tileY / 4
					attrAddr := 0x03C0 + attrY*8 + attrX
					var palette uint8 = 0

					if attrAddr < 1024 {
						attrByte := ppu.tblName[0][attrAddr]
						quadrantX := (tileX % 4) / 2
						quadrantY := (tileY % 4) / 2
						shift := (quadrantY*2 + quadrantX) * 2
						palette = (attrByte >> shift) & 0x03
					}

					// 最終カラー決定
					var colorIndex uint8
					if pixel == 0 {
						colorIndex = ppu.tblPalette[0] // ユニバーサル背景色
					} else {
						// 正しいパレットインデックス計算
						paletteIndex := palette*4 + pixel
						if paletteIndex < 32 {
							colorIndex = ppu.tblPalette[paletteIndex]
						} else {
							colorIndex = ppu.tblPalette[0] // フォールバック
						}
					}
					ppu.sprScreen[y][x] = colorIndex
				} else {
					// カートリッジなし - 背景色
					ppu.sprScreen[y][x] = ppu.tblPalette[0]
				}
			} else {
				// 有効タイル範囲外 - 背景色
				ppu.sprScreen[y][x] = ppu.tblPalette[0]
			}
		}
	}
}

// renderSprites - Basic sprite rendering according to NES PPU specification
func (ppu *PPU) renderSprites(screenX, screenY int) {
	// Sprite rendering with 8 sprites per scanline limit and priority
	spritesOnLine := 0

	// Scan through all 64 sprites (OAM is 256 bytes = 64 sprites * 4 bytes)
	for sprite := 0; sprite < 64; sprite++ {
		// OAM format: Y, Tile Index, Attributes, X
		oamOffset := sprite * 4
		spriteY := int(ppu.OAM[oamOffset]) + 1 // Y position (Y-1 in OAM)
		tileIndex := ppu.OAM[oamOffset+1]      // Tile index
		attributes := ppu.OAM[oamOffset+2]     // Attributes
		spriteX := int(ppu.OAM[oamOffset+3])   // X position

		// Check if sprite is visible on current scanline
		spriteSize := 8 // TODO: Handle 8x16 sprites from PPUCTRL
		if screenY >= spriteY && screenY < spriteY+spriteSize {
			// Enforce 8 sprites per scanline limit
			if spritesOnLine >= 8 {
				ppu.status |= STATUS_SPRITE_OVERFLOW
				break
			}
			spritesOnLine++

			// Check if sprite pixel overlaps current screen position
			if screenX >= spriteX && screenX < spriteX+8 {
				pixelX := screenX - spriteX
				pixelY := screenY - spriteY

				// Handle horizontal flip
				if attributes&0x40 != 0 {
					pixelX = 7 - pixelX
				}
				// Handle vertical flip  
				if attributes&0x80 != 0 {
					pixelY = (spriteSize - 1) - pixelY
				}

				// Get sprite pattern table (from PPUCTRL bit 3)
				var patternTableBase uint16 = 0x0000
				if ppu.ctrl&CTRL_PATTERN_SPRITE != 0 {
					patternTableBase = 0x1000
				}

				// Read pattern data
				if ppu.cart != nil {
					patternAddr := patternTableBase + uint16(tileIndex)*16 + uint16(pixelY)
					patternLo := ppu.cart.PPURead(patternAddr)
					patternHi := ppu.cart.PPURead(patternAddr + 8)

					bit := 7 - pixelX
					pixel := ((patternHi >> bit) & 1) << 1
					pixel |= (patternLo >> bit) & 1

					// Render sprite pixel if not transparent
					if pixel != 0 {
						// Check sprite 0 hit
						if sprite == 0 && (ppu.mask&MASK_RENDER_BG) != 0 {
							// Sprite 0 hit occurs when sprite 0 overlaps with non-transparent background
							if ppu.sprScreen[screenY][screenX] != ppu.tblPalette[0] {
								ppu.status |= STATUS_SPRITE_ZERO_HIT
							}
						}

						// Background priority check (bit 5 of attributes)
						bgPriority := attributes&0x20 != 0
						if !bgPriority || ppu.sprScreen[screenY][screenX] == ppu.tblPalette[0] {
							// Get sprite palette (bits 0-1 of attributes) 
							spritePalette := attributes & 0x03
							// Sprite palettes start at $3F10
							colorIndex := ppu.tblPalette[0x10+spritePalette*4+pixel]
							ppu.sprScreen[screenY][screenX] = colorIndex
						}
					}
				}
			}
		}
	}
}

// 正確なNES PPUスクロール管理メソッド群

// incrementScrollX - 水平スクロール増分 (8サイクルごと)
func (ppu *PPU) incrementScrollX() {
	if (ppu.mask & MASK_RENDER_BG) != 0 {
		if (ppu.vramAddr & 0x001F) == 31 { // coarse X == 31の場合
			ppu.vramAddr &= ^uint16(0x001F) // coarse X = 0
			ppu.vramAddr ^= 0x0400          // ネームテーブル水平切り替え
		} else {
			ppu.vramAddr++ // coarse X増分
		}
	}
}

// incrementScrollY - 垂直スクロール増分 (サイクル256)
func (ppu *PPU) incrementScrollY() {
	if (ppu.mask & MASK_RENDER_BG) != 0 {
		if (ppu.vramAddr & 0x7000) != 0x7000 { // fine Y < 7の場合
			ppu.vramAddr += 0x1000 // fine Y増分
		} else {
			ppu.vramAddr &= ^uint16(0x7000)   // fine Y = 0
			y := (ppu.vramAddr & 0x03E0) >> 5 // coarse Y取得
			if y == 29 {
				y = 0
				ppu.vramAddr ^= 0x0800 // ネームテーブル垂直切り替え
			} else if y == 31 {
				y = 0 // 31は無効な値、0にリセット
			} else {
				y++
			}
			ppu.vramAddr = (ppu.vramAddr & ^uint16(0x03E0)) | (y << 5)
		}
	}
}

// resetScrollX - 水平スクロールリセット (サイクル257)
func (ppu *PPU) resetScrollX() {
	if (ppu.mask & MASK_RENDER_BG) != 0 {
		// v: ....F.. ...EDCBA = t: ....F.. ...EDCBA
		ppu.vramAddr = (ppu.vramAddr & 0xFBE0) | (ppu.tempAddr & 0x041F)
	}
}

// resetScrollY - 垂直スクロールリセット (サイクル280-304、プリレンダースキャンライン)
func (ppu *PPU) resetScrollY() {
	if (ppu.mask & MASK_RENDER_BG) != 0 {
		// v: IHGF.ED CBA..... = t: IHGF.ED CBA.....
		ppu.vramAddr = (ppu.vramAddr & 0x841F) | (ppu.tempAddr & 0x7BE0)
	}
}

// applyNametableWrite applies a nametable write immediately (used during VBlank)
func (ppu *PPU) applyNametableWrite(fullAddr uint16, data uint8) {
	// Handle palette writes (0x3F00-0x3FFF range)
	if fullAddr >= 0x3F00 && fullAddr <= 0x3FFF {
		addr := fullAddr & 0x001F
		if addr == 0x0010 {
			addr = 0x0000
		}
		if addr == 0x0014 {
			addr = 0x0004
		}
		if addr == 0x0018 {
			addr = 0x0008
		}
		if addr == 0x001C {
			addr = 0x000C
		}
		if addr < 32 {
			ppu.tblPalette[addr] = data
		}
		return
	}
	
	// Handle nametable writes (0x2000-0x3EFF range)
	addr := fullAddr & 0x0FFF
	
	if ppu.cart.Mirror == 0 { // 垂直ミラーリング
		if addr >= 0x0000 && addr <= 0x03FF {
			if addr < 1024 {
				ppu.tblName[0][addr] = data
			}
		} else if addr >= 0x0400 && addr <= 0x07FF {
			index := addr - 0x0400
			if index < 1024 {
				ppu.tblName[1][index] = data
			}
		} else if addr >= 0x0800 && addr <= 0x0BFF {
			index := addr - 0x0800
			if index < 1024 {
				ppu.tblName[0][index] = data
			}
		} else if addr >= 0x0C00 && addr <= 0x0FFF {
			index := addr - 0x0C00
			if index < 1024 {
				ppu.tblName[1][index] = data
			}
		}
	} else { // 水平ミラーリング
		if addr >= 0x0000 && addr <= 0x03FF {
			if addr < 1024 {
				ppu.tblName[0][addr] = data
			}
		} else if addr >= 0x0400 && addr <= 0x07FF {
			index := addr - 0x0400
			if index < 1024 {
				ppu.tblName[0][index] = data
			}
		} else if addr >= 0x0800 && addr <= 0x0BFF {
			index := addr - 0x0800
			if index < 1024 {
				ppu.tblName[1][index] = data
			}
		} else if addr >= 0x0C00 && addr <= 0x0FFF {
			index := addr - 0x0C00
			if index < 1024 {
				ppu.tblName[1][index] = data
			}
		}
	}
}
