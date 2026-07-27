package scenario

type Band int

const (
	BandLow Band = iota + 1
	BandNormal
	BandHigh
)

func ClassifyPressure(psi int) Band {
	switch {
	case psi < lowThresholdPSI:
		return BandLow
	case psi > highThresholdPSI:
		return BandHigh
	default:
		return BandNormal
	}
}

const (
	lowThresholdPSI  = 30
	highThresholdPSI = 35
)
