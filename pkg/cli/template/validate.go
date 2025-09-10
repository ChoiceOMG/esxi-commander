package template

type TemplateValidator struct {
	Name string
	Path string
}

type ValidationResult struct {
	Valid                bool     `json:"valid"`
	CloudInitInstalled   bool     `json:"cloud_init_installed"`
	VMwareToolsInstalled bool     `json:"vmware_tools_installed"`
	GuestinfoEnabled     bool     `json:"guestinfo_enabled"`
	Errors               []string `json:"errors,omitempty"`
}

func (tv *TemplateValidator) Validate() (*ValidationResult, error) {
	// Stub implementation for Phase 1
	// Real validation will be added in Phase 2
	return &ValidationResult{
		Valid:                true,
		CloudInitInstalled:   true,
		VMwareToolsInstalled: true,
		GuestinfoEnabled:     true,
	}, nil
}
