package currency

type Code string

const (
	EUR Code = "EUR"
	USD Code = "USD"
	GBP Code = "GBP"
)

func SymbolFor(code Code) string {
	switch code {
	case EUR:
		return "€"
	case USD:
		return "$"
	case GBP:
		return "£"
	default:
		panic("unknown currency: " + string(code))
	}
}
