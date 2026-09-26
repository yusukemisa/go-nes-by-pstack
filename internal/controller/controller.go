package controller

// NES Controller implementation
type Controller struct {
	buttons    uint8 // Current button state
	strobe     bool  // Strobe state
	shiftReg   uint8 // Shift register for reading
}

// Button constants
const (
	BUTTON_A      = 0x01
	BUTTON_B      = 0x02
	BUTTON_SELECT = 0x04
	BUTTON_START  = 0x08
	BUTTON_UP     = 0x10
	BUTTON_DOWN   = 0x20
	BUTTON_LEFT   = 0x40
	BUTTON_RIGHT  = 0x80
)

func NewController() *Controller {
	return &Controller{
		buttons:  0,
		strobe:   false,
		shiftReg: 0,
	}
}

func (c *Controller) SetButton(button uint8, pressed bool) {
	if pressed {
		c.buttons |= button
	} else {
		c.buttons &= ^button
	}
}

func (c *Controller) Write(data uint8) {
	c.strobe = (data & 0x01) != 0
	if c.strobe {
		c.shiftReg = c.buttons
	}
}

func (c *Controller) Read() uint8 {
	result := c.shiftReg & 0x01
	if !c.strobe {
		c.shiftReg >>= 1
		c.shiftReg |= 0x80 // Set bit 7 for open bus behavior
	}
	return result
}