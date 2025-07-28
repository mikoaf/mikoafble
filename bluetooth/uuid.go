package bluetooth

import "unsafe"

type UUID [4]uint32

func NewUUID(uuid [16]byte) UUID {
	u := UUID{}
	u[0] = uint32(uuid[15]) | uint32(uuid[14])<<8 | uint32(uuid[13])<<16 | uint32(uuid[12])<<24
	u[1] = uint32(uuid[11]) | uint32(uuid[10])<<8 | uint32(uuid[9])<<16 | uint32(uuid[8])<<24
	u[2] = uint32(uuid[7]) | uint32(uuid[6])<<8 | uint32(uuid[5])<<16 | uint32(uuid[4])<<24
	u[3] = uint32(uuid[3]) | uint32(uuid[2])<<8 | uint32(uuid[1])<<16 | uint32(uuid[0])<<24
	return u
}

func (u UUID) String() string {
	buf, _ := u.AppendText(make([]byte, 0, 36))

	return unsafe.String(unsafe.SliceData(buf), 36)
}

const hexDigitLower = "0123456789abcdef"

func (u UUID) AppendText(buf []byte) ([]byte, error) {
	for i := 3; i >= 0; i-- {
		// Insert a hyphen at the correct locations.
		// position 4 and 8
		if i != 3 && i != 0 {
			buf = append(buf, '-')
		}

		buf = append(buf, hexDigitLower[byte(u[i]>>24)>>4])
		buf = append(buf, hexDigitLower[byte(u[i]>>24)&0xF])

		buf = append(buf, hexDigitLower[byte(u[i]>>16)>>4])
		buf = append(buf, hexDigitLower[byte(u[i]>>16)&0xF])

		// Insert a hyphen at the correct locations.
		// position 6 and 10
		if i == 2 || i == 1 {
			buf = append(buf, '-')
		}

		buf = append(buf, hexDigitLower[byte(u[i]>>8)>>4])
		buf = append(buf, hexDigitLower[byte(u[i]>>8)&0xF])

		buf = append(buf, hexDigitLower[byte(u[i])>>4])
		buf = append(buf, hexDigitLower[byte(u[i])&0xF])
	}

	return buf, nil
}
