package defaults

const (
	DefaultCPU       = 2
	DefaultRAM       = 4  // GB
	DefaultDisk      = 20 // GB
	DefaultTemplate  = "ubuntu-22.04-lts"
	DefaultDatastore = "datastore1"
	DefaultNetwork   = "VM Network"
)

func GetCPU() int {
	return DefaultCPU
}

func GetRAM() int {
	return DefaultRAM
}

func GetDisk() int {
	return DefaultDisk
}

func GetTemplate() string {
	return DefaultTemplate
}

func GetDatastore() string {
	return DefaultDatastore
}

func GetNetwork() string {
	return DefaultNetwork
}
