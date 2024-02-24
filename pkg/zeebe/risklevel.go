package zeebe

type RiskLevel int

const (
	GREEN RiskLevel = iota
	YELLOW
	RED
)

func (r RiskLevel) GetLevel() int {
	return int(r)
}
