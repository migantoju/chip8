package main

import (
	"fmt"
	"time"
)

/*
Memory Map

---------------+= 0xFFF (4095) End of Chip-8 RAM
|               |
|               |
|               |
|               |
|               |
| 0x200 to 0xFFF|
|     Chip-8    |
| Program / Data|
|     Space     |
|               |
|               |
|               |
+- - - - - - - -+= 0x600 (1536) Start of ETI 660 Chip-8 programs
|               |
|               |
|               |
+---------------+= 0x200 (512) Start of most Chip-8 programs
| 0x000 to 0x1FF|
| Reserved for  |
|  interpreter  |
+---------------+= 0x000 (0) Start of Chip-8 RAM

Basic CPU loop
1. Fetch Opcode
2. Decode Opcode
3. Execute Opcode
4. Update timers

Program Counter (PC) starts at 0x200

- Clear display
- Clear stack
- Clear registers V0-VF
- Clear memory
- Load fontset
- reset timers

We need to low the clock speed to 60 Opcodes per second (60Hz)

Keypad       Keyboard
+-+-+-+-+    +-+-+-+-+
|1|2|3|C|    |1|2|3|4|
+-+-+-+-+    +-+-+-+-+
|4|5|6|D|    |Q|W|E|R|
+-+-+-+-+ => +-+-+-+-+
|7|8|9|E|    |A|S|D|F|
+-+-+-+-+    +-+-+-+-+
|A|0|B|F|    |Z|X|C|V|
+-+-+-+-+    +-+-+-+-+

*/

// fonts to display
var fontSet = [80]uint8{
	0xF0, 0x90, 0x90, 0x90, 0xF0, //0
	0x20, 0x60, 0x20, 0x20, 0x70, //1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, //2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, //3
	0x90, 0x90, 0xF0, 0x10, 0x10, //4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, //5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, //6
	0xF0, 0x10, 0x20, 0x40, 0x40, //7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, //8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, //9
	0xF0, 0x90, 0xF0, 0x90, 0x90, //A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, //B
	0xF0, 0x80, 0x80, 0x80, 0xF0, //C
	0xE0, 0x90, 0x90, 0x90, 0xE0, //D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, //E
	0xF0, 0x80, 0xF0, 0x80, 0x80, //F
}

type CPU struct {
	Memory [4096]byte   // memory ram
	Opcode uint16       // current opcode
	I      uint16       // Index stack - (rightmost) 12 bits are usually used
	Vx     [16]uint8    // Vx registers
	PC     uint16       // Program Counter
	SP     uint16       // Stack Pointer
	S      [16]uint16   // Stack
	DT     uint8        // Delay Timer
	ST     uint8        // Sound Timer
	CS     int          // Clock Speed
	Vr     [64][32]byte // V-ram display size
	Keys   [16]uint8    // Keys from keyboard
	Clock  <-chan time.Time
}

type UnknownOpcode struct {
	Opcode  uint16
	Address uint16
}

func NewCPU() *CPU {
	return &CPU{
		PC:     0x200,
		CS:     60,
		I:      0x000,
		Opcode: 0,
		SP:     0,
		Clock:  time.Tick(time.Second / 60),
	}
}

func (c *CPU) Emulate() {
	fmt.Println(".... Fetching Opcode......")

	switch c.Opcode & 0xF000 {
	case 0x0000:
		switch c.Opcode & 0x000F {
		case 0x00E0:
			fmt.Println(".....Clearing the screen.....")
			// clear screen

			// increment the PC by two
			c.PC += 2
			break
		case 0x00EE:
			// Return from a subroutine

			// The interpreter sets the program counter
			// to the address at the top of the Stack
			c.PC = c.S[c.SP]
			// Then subtracts 1 from the stack pointer
			c.SP--

			c.PC += 2
			break
		case 0x1000:
			// Jump to location nnn
			// The interpreter sets the program counter to nnn
			c.PC = c.Opcode & 0x0FFF
			break
		case 0x2000:
			// Call subroutine at nnn
			// The interpreter increments the Stack Pointer
			c.SP++
			// Then, puts the current PC on the top of the stack.
			c.S[c.SP] = c.PC
			// The PC is then set to nnn
			c.PC = c.Opcode & 0x0FFF

			break
		case 0x3000:
			// Skip the next instruction if Vx = kk

			// The interpreter compares register Vx to kk,
			// and if they are equal, increments the program
			// counter by 2.
			x := (c.Opcode & 0x0F00) >> 8
			kk := byte(c.Opcode)

			if c.Vx[x] == kk {
				c.PC += 2
			}

			c.PC += 2

			break
		case 0x4000:
			// Skip the next instruction if Vx != kk

			// The interpreter compares register Vx to kk
			// if they are not equal, increments the PC by 2.
			x := (c.Opcode & 0x0F00) >> 8
			kk := byte(c.Opcode)

			if c.Vx[x] != kk {
				c.PC += 2
			}

			c.PC += 2

			break
		case 0x5000:
			// Skip next instruction if Vx = Vy
			x := (c.Opcode & 0x0F00) >> 8
			y := (c.Opcode & 0x00F0) >> 4

			c.PC += 2

			if c.Vx[x] == c.Vx[y] {
				c.PC += 2
			}

			break
		case 0x6000:
			// set Vx = kk
			// The interpreter puts the value kk into
			// register Vx
			x := (c.Opcode & 0x0F00) >> 8
			kk := byte(c.Opcode)

			c.Vx[x] = kk

			c.PC += 2

			break
		case 0x7000:
			// set Vx = Vx + kk

			// Adds the value kk to the value of register Vx
			// then stores the result in Vx
			x := (c.Opcode & 0x0F00) >> 8
			kk := byte(c.Opcode)

			c.Vx[x] = c.Vx[x] + kk

			c.PC += 2
			break
		}
	}
}

// GetOpcode is the main method to get the next opcode.
// An opcode is 2-bytes wide, this means that when reading
// from memory, we have to combine two bytes into one 16-bit DS.
func (c *CPU) GetOpcode() {
	fmt.Println("Fetching the next opcode.")

	c.Opcode = uint16(c.Memory[c.PC]<<8) | uint16(c.Memory[c.PC+1])
}
