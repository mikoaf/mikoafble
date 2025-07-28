package bluetooth

import (
	"errors"
	"unsafe"
)

type MAC [6]byte

var ErrInvalidMAC = errors.New("bluetooth: failed to parse MAC address")

const hexDigit = "0123456789ABCDEF"

func ParseMAC(s string) (mac MAC, err error) {
	err = (&mac).UnmarshalText([]byte(s))
	return
}

func (mac *MAC) UnmarshalText(s []byte) error {
	macIndex := 11
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ':' {
			continue
		}
		var nibble byte
		if c >= '0' && c <= '9' {
			nibble = c - '0' + 0x0
		} else if c >= 'A' && c <= 'F' {
			nibble = c - 'A' + 0xA
		} else {
			return ErrInvalidMAC
		}
		if macIndex < 0 {
			return ErrInvalidMAC
		}
		if macIndex%2 == 0 {
			mac[macIndex/2] |= nibble
		} else {
			mac[macIndex/2] |= nibble << 4
		}
		macIndex--
	}
	if macIndex != -1 {
		return ErrInvalidMAC
	}
	return nil
}

func (mac MAC) String() string {
	buf, _ := mac.MarshalText()
	return unsafe.String(unsafe.SliceData(buf), 17)
}

func (mac MAC) MarshalText() (text []byte, err error) {
	return mac.AppendText(make([]byte, 0, 17))
}

func (mac MAC) AppendText(buf []byte) ([]byte, error) {
	for i := 5; i >= 0; i-- {
		if i != 5 {
			buf = append(buf, ':')
		}
		buf = append(buf, hexDigit[mac[i]>>4])
		buf = append(buf, hexDigit[mac[i]&0xF])
	}
	return buf, nil
}
