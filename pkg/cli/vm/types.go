package vm

type VM struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	IP     string `json:"ip"`
	CPU    int    `json:"cpu"`
	RAM    int    `json:"ram"`
}
