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

func (r RiskLevel) GetString() string {
	switch r {
	case GREEN:
		return "GREEN"
	case YELLOW:
		return "YELLOW"
	case RED:
		return "RED"
	}
	return ""
}

func RiskLevels() []int {
	return []int{int(GREEN), int(YELLOW), int(RED)}
}
