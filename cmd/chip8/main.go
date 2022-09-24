package main

import (
	"bytes"
	"fmt"
	"math/rand"
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
	V      [16]uint8    // Vx registers
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
			for i := 0; i < len(c.Vr); i++ {
				for j := 0; j < len(c.Vr[i]); j++ {
					c.Vr[i][j] = 0x0
				}
			}
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
		default:
			fmt.Printf("Invalid opcode %x", c.Opcode)
		}
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

		if c.V[x] == kk {
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

		if c.V[x] != kk {
			c.PC += 2
		}

		c.PC += 2

		break
	case 0x5000:
		// Skip next instruction if Vx = Vy
		x := (c.Opcode & 0x0F00) >> 8
		y := (c.Opcode & 0x00F0) >> 4

		c.PC += 2

		if c.V[x] == c.V[y] {
			c.PC += 2
		}

		break
	case 0x6000:
		// set Vx = kk
		// The interpreter puts the value kk into
		// register Vx
		x := (c.Opcode & 0x0F00) >> 8
		kk := byte(c.Opcode)

		c.V[x] = kk

		c.PC += 2

		break
	case 0x7000:
		// set Vx = Vx + kk

		// Adds the value kk to the value of register Vx
		// then stores the result in Vx
		x := (c.Opcode & 0x0F00) >> 8
		kk := byte(c.Opcode)

		c.V[x] = c.V[x] + kk

		c.PC += 2
		break
	case 0x8000:
		x := (c.Opcode & 0x0F00) >> 8
		y := (c.Opcode & 0x00F0) >> 4
		switch c.Opcode & 0x000F {
		case 0x0000:
			// set Vx = Vy
			c.V[x] = c.V[y]
			c.PC += 2
			break
		case 0x0001:
			// Set Vx = Vx or Vy
			c.V[x] = c.V[x] | c.V[y]
			c.PC += 2
			break
		case 0x0002:
			// Set Vx = Vx AND Vy
			c.V[x] = c.V[x] & c.V[y]
			c.PC += 2
			break
		case 0x0003:
			// Set Vx = Vx XOR Vy
			c.V[x] = c.V[x] ^ c.V[y]
			c.PC += 2
			break
		case 0x0004:
			// Set Vx = Vx + Vy, set VF = carry

			// The values of Vx and Vy are added together.
			result := uint16(c.V[x]) + uint16(c.V[y])

			// init to zero value
			var carry byte

			// If the resylt is greater than 8 bits VF is set to 1
			// otherwise 0.
			if result > 0xFF {
				carry = 1
			}
			c.V[0xF] = carry

			// only the lowest 8 bits of the result are kept,
			// and stored in Vx
			c.V[x] = byte(result)

			c.PC += 2
			break
		case 0x0005:
			// Set Vx = Vx - Vy, set VF = NOT borrow

			if c.V[x] > c.V[y] {
				c.V[0xF] = byte(1)
			}
			c.V[0xF] = byte(0)

			c.V[x] = c.V[x] - c.V[y]
			c.PC += 2
			break
		case 0x0006:
			// Set Vx = Vx SHR 1
			var carry byte

			if (c.V[x] & 0x01) == 0x01 {
				carry = 1
			}
			c.V[0xF] = carry
			c.V[x] = c.V[x] / 2
			c.PC += 2
			break
		case 0x0007:
			// Set Vx = Vy - Vx, set VF = NOT borrow
			var carry byte

			if c.V[y] > c.V[x] {
				carry = 1
			}
			c.V[0xF] = carry
			c.V[x] = c.V[y] - c.V[x]
			c.PC += 2
			break
		case 0x000E:
			// Set Vx = Vx SHL 1
			var carry byte

			if (c.V[x] & 0x01) == 0x01 {
				carry = 1
			}
			c.V[0xF] = carry
			c.V[x] = c.V[x] * 2
			c.PC += 2
			break
		}
	case 0x9000:
		x := (c.Opcode & 0x0F00) >> 8
		y := (c.Opcode & 0x00F0) >> 4
		switch c.Opcode & 0x000F {
		case 0x0000:
			if c.V[x] != c.V[y] {
				c.PC += 2
			}

			c.PC += 2
			break
		default:
			fmt.Printf("Invalid Opcode: %X", c.Opcode)
		}
	case 0xA000:
		// Set I = nnn
		// The value of register I is set to nnn
		c.I = c.Opcode & 0x0FFF
		c.PC += 2
		break
	case 0xB000:
		// Jump to location nnn + V0
		// The program counter is set to nnn plus the value of V0
		c.PC = c.Opcode&0x0FFF + uint16(c.V[0x0])
		break
	case 0xC000:
		// Set Vx = random byte AND kk
		x := (c.Opcode & 0x0F00) >> 8
		kk := byte(c.Opcode)

		c.V[x] = kk + randomByte()
		c.PC += 2
		break
	case 0xD000:
		// Display n-byte sprite starting at memory location
		// I at (Vx, Vy), set VF=collision.

	}
}

// GetOpcode is the main method to get the next opcode.
// An opcode is 2-bytes wide, this means that when reading
// from memory, we have to combine two bytes into one 16-bit DS.
func (c *CPU) GetOpcode() {
	fmt.Println("Fetching the next opcode.")

	c.Opcode = uint16(c.Memory[c.PC]<<8) | uint16(c.Memory[c.PC+1])
}

func (c *CPU) LoadRom(rom []byte) (int, error) {
	reader := bytes.NewReader(rom)
	return reader.Read(c.Memory[0x200:])
}

func randomByte() byte {
	return byte(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(255))
}
