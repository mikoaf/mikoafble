package nbable

type CharacteristicPermissions uint8

type Service struct {
	handle uint16
	UUID
	Characteristic []CharacteristicConfig
}

type WriteEvent = func(client Connection, offset int, value []byte)

type CharacteristicConfig struct {
	Handle *Characteristic
	UUID
	Value      []byte
	Flags      CharacteristicPermissions
	WriteEvent WriteEvent
}

const (
	CharacteristicBroadcastPermission CharacteristicPermissions = 1 << iota
	CharacteristicReadPermission
	CharacteristicWriteWithoutResponsePermission
	CharacteristicWritePermission
	CharacteristicNotifyPermission
	CharacteristicIndicatePermission
)

func (p CharacteristicPermissions) Broadcast() bool {
	return p&CharacteristicBroadcastPermission != 0
}

func (p CharacteristicPermissions) Read() bool {
	return p&CharacteristicReadPermission != 0
}

func (p CharacteristicPermissions) Write() bool {
	return p&CharacteristicWritePermission != 0
}

func (p CharacteristicPermissions) WriteWithoutResponse() bool {
	return p&CharacteristicWriteWithoutResponsePermission != 0
}

func (p CharacteristicPermissions) Notify() bool {
	return p&CharacteristicNotifyPermission != 0
}

func (p CharacteristicPermissions) Indicate() bool {
	return p&CharacteristicIndicatePermission != 0
}
