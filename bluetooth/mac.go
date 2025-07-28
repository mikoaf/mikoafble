package bluetooth

import "errors"

type MAC [6]byte

var ErrInvalidMAC = errors.New("bluetooth: failed to parse MAC address")

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
