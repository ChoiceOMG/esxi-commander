package config

type Config struct {
	ESXi     ESXiConfig     `yaml:"esxi"`
	Defaults DefaultsConfig `yaml:"defaults"`
	Security SecurityConfig `yaml:"security"`
}

type ESXiConfig struct {
	Host   string `yaml:"host"`
	User   string `yaml:"user"`
	SSHKey string `yaml:"ssh_key"`
}

type DefaultsConfig struct {
	Template  string `yaml:"template"`
	Datastore string `yaml:"datastore"`
	Network   string `yaml:"network"`
	CPU       int    `yaml:"cpu"`
	RAM       int    `yaml:"ram"`
	Disk      int    `yaml:"disk"`
}

type SecurityConfig struct {
	Mode string `yaml:"mode"` // restricted, standard, unrestricted
}
