package cloudinit

import "fmt"

type CloudInitBuilder struct {
	UserData string
	MetaData string
}

func NewBuilder() *CloudInitBuilder {
	return &CloudInitBuilder{}
}

func (b *CloudInitBuilder) SetUserData(userData string) *CloudInitBuilder {
	b.UserData = userData
	return b
}

func (b *CloudInitBuilder) SetMetaData(metaData string) *CloudInitBuilder {
	b.MetaData = metaData
	return b
}

func (b *CloudInitBuilder) Build() (map[string]string, error) {
	// Stub implementation for Phase 1
	// Real cloud-init guestinfo injection will be implemented in Phase 2
	return nil, fmt.Errorf("cloud-init builder not implemented in Phase 1")
}
