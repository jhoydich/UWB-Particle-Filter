package readings

type Reading struct {
	Anchor  string  `json:"Device"`
	Dist    float64 `json:"Range"`
	RXPower float64 `json:"RX power"`
}
