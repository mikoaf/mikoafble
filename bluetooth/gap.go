package nbable

type MACAddress struct {
	MAC
	isRandom bool
}

type Connection uint16

type AdvertisingType int

const (
	AdvertisingTypeInd AdvertisingType = iota

	AdvertisingTypeDirectInd

	AdvertisingTypeScanInd

	AdvertisingTypeNonConnInd
)

type AdvertisementOptions struct {
	AdvertisementType AdvertisingType

	LocalName string

	ServiceUUIDs []UUID

	Interval Duration

	ManufacturerData []ManufacturerDataElement

	ServiceData []ServiceDataElement
}

type Duration uint16

type ManufacturerDataElement struct {
	CompanyID uint16
	Data      []byte
}

type ServiceDataElement struct {
	UUID UUID
	Data []byte
}
