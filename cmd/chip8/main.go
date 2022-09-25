package main

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

import (
	"bytes"
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"math/rand"
	"os"
	"time"
)

func main() {
	chip8 := NewCPU()
	file, err := os.OpenFile("./roms/invaders.c8", os.O_RDONLY, 0777)
	if err != nil {
		panic(err)
	}

	defer file.Close()

	fStat, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if int64(len(chip8.Memory)-0x200) < fStat.Size() {
		panic("Program size is bigger than memory")
	}

	buffer := make([]byte, fStat.Size())
	if _, err := file.Read(buffer); err != nil {
		fmt.Println("Buffer Error")
		panic(err)
	}

	for i := 0; i < len(buffer); i++ {
		chip8.Memory[i+0x200] = buffer[i]
	}

	// End Load Rom logic

	if sdlErr := sdl.Init(sdl.INIT_EVERYTHING); sdlErr != nil {
		panic(sdlErr)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Chip-8"+file.Name(),
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		64*10, 32*10, sdl.WINDOW_SHOWN)

	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// render
	canvas, err := sdl.CreateRenderer(window, -1, 0)
	if err != nil {
		panic(err)
	}

	defer canvas.Destroy()

	for {
		// instance of CPU
		chip8.Emulate()

		// Draw only if there are pixels to draw
		if chip8.Draw() {
			canvas.SetDrawColor(0, 0, 0, 255)
			canvas.Clear()

			// Display buffer and render
			v := chip8.Buffer()
			for j := 0; j < len(v); j++ {
				for i := 0; i < len(v[j]); i++ {
					if v[j][i] != 0 {
						canvas.SetDrawColor(255, 255, 0, 255)
					} else {
						canvas.SetDrawColor(0, 0, 0, 255)
					}
					canvas.FillRect(&sdl.Rect{
						Y: int32(j) * 10,
						X: int32(i) * 10,
						W: 10,
						H: 10,
					})
				}
			}
			canvas.Present()
		}

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch et := event.(type) {
			case *sdl.QuitEvent:
				os.Exit(0)
			case *sdl.KeyboardEvent:
				if et.Type == sdl.KEYUP {
					fmt.Println("User has unpressed a key")
					switch et.Keysym.Sym {
					case sdl.K_1:
						chip8.SetKey(0x1, false)
						break
					case sdl.K_2:
						chip8.SetKey(0x2, false)
						break
					case sdl.K_3:
						chip8.SetKey(0x3, false)
						break
					case sdl.K_4:
						chip8.SetKey(0xC, false)
					case sdl.K_q:
						chip8.SetKey(0x4, false)
						break
					case sdl.K_w:
						chip8.SetKey(0x5, false)
						break
					case sdl.K_e:
						chip8.SetKey(0x6, false)
						break
					case sdl.K_r:
						chip8.SetKey(0xD, false)
						break
					case sdl.K_a:
						chip8.SetKey(0x7, false)
						break
					case sdl.K_s:
						chip8.SetKey(0x8, false)
						break
					case sdl.K_d:
						chip8.SetKey(0x9, false)
						break
					case sdl.K_f:
						chip8.SetKey(0xE, false)
						break
					case sdl.K_z:
						chip8.SetKey(0xA, false)
						break
					case sdl.K_x:
						chip8.SetKey(0x0, false)
						break
					case sdl.K_c:
						chip8.SetKey(0xB, false)
						break
					case sdl.K_v:
						chip8.SetKey(0xF, false)
						break
					}
				} else if et.Type == sdl.KEYDOWN {
					switch et.Keysym.Sym {
					case sdl.K_1:
						chip8.SetKey(0x1, true)
						break
					case sdl.K_2:
						chip8.SetKey(0x2, true)
						break
					case sdl.K_3:
						chip8.SetKey(0x3, true)
						break
					case sdl.K_4:
						chip8.SetKey(0xC, true)
					case sdl.K_q:
						chip8.SetKey(0x4, true)
						break
					case sdl.K_w:
						chip8.SetKey(0x5, true)
						break
					case sdl.K_e:
						chip8.SetKey(0x6, true)
						break
					case sdl.K_r:
						chip8.SetKey(0xD, true)
						break
					case sdl.K_a:
						chip8.SetKey(0x7, true)
						break
					case sdl.K_s:
						chip8.SetKey(0x8, true)
						break
					case sdl.K_d:
						chip8.SetKey(0x9, true)
						break
					case sdl.K_f:
						chip8.SetKey(0xE, true)
						break
					case sdl.K_z:
						chip8.SetKey(0xA, true)
						break
					case sdl.K_x:
						chip8.SetKey(0x0, true)
						break
					case sdl.K_c:
						chip8.SetKey(0xB, true)
						break
					case sdl.K_v:
						chip8.SetKey(0xF, true)
						break
					}
				}
			}
			sdl.Delay(1000 / 60)
		}
	}
}

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
	Vr     [32][64]byte // V-ram display size
	Keys   [16]uint8    // Keys from keyboard
	Clock  <-chan time.Time
	ED     bool          // Execute Draw
	Stop   chan struct{} // Channel used to indicate shutdown
}

type UnknownOpcode struct {
	Opcode  uint16
	Address uint16
}

func NewCPU() *CPU {
	instance := CPU{
		PC:     0x200,
		CS:     60,
		I:      0x000,
		Opcode: 0,
		SP:     0,
		Clock:  time.Tick(time.Second / 60),
		ED:     true,
	}

	for i := 0; i < len(fontSet); i++ {
		instance.Memory[i] = fontSet[i]
	}

	return &instance
}

func (c *CPU) Emulate() {
	fmt.Println(".... Fetching Opcode......")
	// GetOpcode is the main method to get the next opcode.
	// An opcode is 2-bytes wide, this means that when reading
	// from memory, we have to combine two bytes into one 16-bit DS.

	opcode := c.getOpcode()

	fmt.Printf("Fetched opcode: 0x%04X\n", opcode)

	switch opcode & 0xF000 {
	case 0x0000:
		switch opcode & 0x000F {
		case 0x0000:
			fmt.Println(".....Clearing the screen.....")
			// clear screen
			for i := 0; i < len(c.Vr); i++ {
				for j := 0; j < len(c.Vr[i]); j++ {
					c.Vr[i][j] = 0x0
				}
			}
			// increment the PC by two
			c.PC += 2
			c.ED = true
			break
		case 0x000E:
			// Return from a subroutine

			// The interpreter sets the program counter
			// to the address at the top of the Stack
			c.PC = c.S[c.SP]
			// Then subtracts 1 from the stack pointer
			c.SP--

			c.PC += 2
			break
		default:
			fmt.Printf("Invalid opcode 0x%04X\n", opcode)
		}
	case 0x1000:
		// Jump to location nnn
		// The interpreter sets the program counter to nnn
		c.PC = opcode & 0x0FFF
		break
	case 0x2000:
		// Call subroutine at nnn
		// The interpreter increments the Stack Pointer
		c.SP++
		// Then, puts the current PC on the top of the stack.
		c.S[c.SP] = c.PC
		// The PC is then set to nnn
		c.PC = opcode & 0x0FFF

		break
	case 0x3000:
		// Skip the next instruction if Vx = kk

		// The interpreter compares register Vx to kk,
		// and if they are equal, increments the program
		// counter by 2.
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode)

		if c.V[x] == kk {
			c.PC += 2
		}

		c.PC += 2

		break
	case 0x4000:
		// Skip the next instruction if Vx != kk

		// The interpreter compares register Vx to kk
		// if they are not equal, increments the PC by 2.
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode)

		if c.V[x] != kk {
			c.PC += 2
		}

		c.PC += 2

		break
	case 0x5000:
		// Skip next instruction if Vx = Vy
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4

		if c.V[x] == c.V[y] {
			c.PC += 2
		}

		c.PC += 2
		break
	case 0x6000:
		// set Vx = kk
		// The interpreter puts the value kk into
		// register Vx
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode)

		c.V[x] = kk

		c.PC += 2

		break
	case 0x7000:
		// set Vx = Vx + kk

		// Adds the value kk to the value of register Vx
		// then stores the result in Vx
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode)

		c.V[x] = c.V[x] + kk

		c.PC += 2
		break
	case 0x8000:
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4
		switch opcode & 0x000F {
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
		x := (opcode & 0x0F00) >> 8
		y := (opcode & 0x00F0) >> 4
		switch opcode & 0x000F {
		case 0x0000:
			if c.V[x] != c.V[y] {
				c.PC += 2
			}

			c.PC += 2
			break
		default:
			fmt.Printf("Invalid Opcode: 0x%04X\n", opcode)
		}
	case 0xA000:
		// Set I = nnn
		// The value of register I is set to nnn
		c.I = opcode & 0x0FFF
		c.PC += 2
		break
	case 0xB000:
		// Jump to location nnn + V0
		// The program counter is set to nnn plus the value of V0
		c.PC = opcode&0x0FFF + uint16(c.V[0x0])
		break
	case 0xC000:
		// Set Vx = random byte AND kk
		x := (opcode & 0x0F00) >> 8
		kk := byte(opcode)

		c.V[x] = kk + randomByte()
		c.PC += 2
		break
	case 0xD000:
		// Display n-byte sprite starting at memory location
		// I at (Vx, Vy), set VF=collision.
		var carry byte

		x := c.V[(opcode&0x0F00)>>8]
		y := c.V[(opcode&0x00F0)>>4]
		h := opcode & 0x000F

		var i uint16 = 0
		var j uint16 = 0

		for j = 0; j < h; j++ {
			p := c.Memory[c.I+j]
			for i = 0; i < 8; i++ {
				if (p & (0x80 >> i)) != 0 {
					if c.Vr[(y + uint8(j))][x+uint8(i)] == 1 {
						carry = 1
					}
					c.Vr[(y + uint8(j))][(x + uint8(i))] ^= 1
				}
			}

		}
		c.V[0xF] = carry
		c.PC += 2
		c.ED = true
		break
	case 0xE000:
		x := (opcode & 0x0F00) >> 8
		switch opcode & 0x00FF {
		case 0x009E:
			// Skip the next instruction if key with the value of Vx is
			// pressed.
			fmt.Printf("Opcode %X\n", opcode)
			if c.Keys[c.V[x]] == 1 {
				c.PC += 4
			} else {
				c.PC += 2
			}
			break
		case 0x00A1:
			if c.Keys[c.V[x]] == 0 {
				c.PC += 4
			} else {
				c.PC += 2
			}
			break
		default:
			fmt.Printf(".... 0xE000, invalid Opcode %X ......", opcode)
		}
	case 0xF000:
		x := (opcode & 0x0F00) >> 8
		switch opcode & 0x00FF {
		case 0x0007:
			// set Vx = delay timer value.
			// The value of DT is placed into Vx
			c.V[x] = c.DT
			c.PC += 2
			break
		case 0x000A:
			// Wait for a Key press, store the value of the key in Vx
			// All execution stops until a key is pressed,
			// Then the value of that key is stored in Vx
			pressed := false
			for i := 0; i < len(c.Keys); i++ {
				if c.Keys[i] != 0 {
					c.V[x] = uint8(i)
					pressed = true
				}
			}

			if !pressed {
				return
			}
			c.PC += 2
			break
		case 0x0015:
			// Set DT = Vx
			// DT is set equal to the value of Vx
			c.DT = c.V[x]
			c.PC += 2
			break
		case 0x0018:
			// Set ST = Vx
			// ST is set equal to the value of Vx
			c.ST = c.V[x]
			c.PC += 2
			break
		case 0x001E:
			// Set I = I + Vx
			// The values of I and Vx are added,
			// and the results are stored in I
			c.I = c.I + uint16(c.V[x])
			c.PC += 2
			break
		case 0x0029:
			// Set I = location of sprite for digits Vx

			// The Sprint are 5 bytes longs
			c.I = uint16(c.V[x]) * uint16(0x05)
			c.PC += 2
			break
		case 0x0033:
			// Store BCD representation of Vx in memory location I, I+1 and I+2

			// The interpreter takes decimal value of Vx, and
			// places the hundreds digits in memory at location I
			c.Memory[c.I] = c.V[x] / 100
			// the tens digit at location I+1
			c.Memory[c.I+1] = (c.V[x] / 10) % 10
			// and the ones digit at location I+2
			c.Memory[c.I+2] = (c.V[x] / 100) % 10

			c.PC += 2
			break
		case 0x0055:
			// Stores registers V0 through Vx in memory starting at location I
			// The interpreter copies the values of register V0 through Vx
			// into memory, starting at the address in I.
			for i := 0; uint16(i) <= x; i++ {
				c.Memory[c.I+uint16(i)] = c.V[i]
			}
			c.PC += 2
			break
		case 0x0065:
			// Read registers V0 through Vx from memory starting at location I
			for i := 0; byte(i) <= byte(x); i++ {
				c.V[uint16(i)] = c.Memory[c.I+uint16(i)]
			}
			c.PC += 2
			break
		default:
			fmt.Printf("Unknown decoded Opcode 0x%04X\n", opcode)
		}
	default:
		fmt.Printf("Unknown opcode to decode 0x%04X\n", opcode)
	}

	if c.DT > 0 {
		c.DT--
	}

	if c.ST > 0 {
		c.ST--
	}
}

func (c *CPU) LoadRom(rom []byte) (int, error) {
	reader := bytes.NewReader(rom)
	return reader.Read(c.Memory[0x200:])
}

func (c *CPU) Draw() bool {
	ableToDraw := c.ED
	c.ED = false
	return ableToDraw
}

func (c *CPU) Buffer() [32][64]uint8 {
	return c.Vr
}

func (c *CPU) SetKey(num uint8, pressed bool) {
	if pressed {
		c.Keys[num] = 1
	} else {
		c.Keys[num] = 0
	}
}

func (c *CPU) getOpcode() uint16 {
	return uint16(c.Memory[c.PC])<<8 | uint16(c.Memory[c.PC+1])
}

func randomByte() byte {
	return byte(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(255))
}

func (c *CPU) Shutdown() {
	close(c.Stop)
}
