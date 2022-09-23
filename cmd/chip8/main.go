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

// fonts to display
var fontSet = []uint8{
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
	Memory [4096]byte // memory ram
	Opcode uint16     // current opcode
	I      uint16     // Index stack - (rightmost) 12 bits are usually used
	Vx     [16]uint8  // Vx registers
	PC     uint16     // Program Counter
	SP     uint16     // Stack Pointer
	S      [16]uint16 // Stack
	DT     uint8      // Delay Timer
	ST     uint8      // Sound Timer
	CS     int        // Clock Speed
}

func (c *CPU) LoadRom(rom []byte) {
	for m := 0; m < 4096; m++ {
		c.Memory[m] = 0x00
	}
	for index, b := range rom {
		c.Memory[index+0x200] = b
	}
}

type Chip8 struct {
	cpu     CPU
	display [64 * 32]uint8
	Keys    [16]uint8 // keyboard
}

type UnknownOpcode struct {
	Opcode  uint16
	Address uint16
}

func NewCPU() *CPU {
	return &CPU{
		PC: 0x200,
		CS: 60,
		I:  0x000,
	}
}
