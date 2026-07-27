package scenario

type Order struct {
	SubtotalCents int
	Coupon        string
}

type Decision struct {
	Applied    bool
	TotalCents int
	Reason     string
}

func Apply(order Order) Decision {
	if order.Coupon != springCoupon {
		return Decision{TotalCents: order.SubtotalCents, Reason: "unknown coupon"}
	}
	if order.SubtotalCents < springMinimumCents {
		return Decision{TotalCents: order.SubtotalCents, Reason: "below minimum"}
	}
	return Decision{Applied: true, TotalCents: order.SubtotalCents * (100 - springPercentOff) / 100}
}

const (
	springCoupon       = "SPRING"
	springMinimumCents = 5000
	springPercentOff   = 10
)
