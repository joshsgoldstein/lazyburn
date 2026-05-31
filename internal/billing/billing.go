package billing

import "math"

// Plan describes a subscription billing tier.
type Plan struct {
	Name        string
	MonthlyFee  float64
	// IncludedUSD is the API-equivalent value covered by the plan per month.
	// 0 means pure pay-per-use (API billing with no subscription credit).
	IncludedUSD float64
}

// KnownPlans lists Anthropic Claude subscription tiers.
// IncludedUSD figures are approximate; override with --plan-budget.
var KnownPlans = map[string]Plan{
	"api":     {Name: "API (pay-per-use)", MonthlyFee: 0, IncludedUSD: 0},
	"pro":     {Name: "Claude Pro ($20/mo)", MonthlyFee: 20, IncludedUSD: 20},
	"max-20":  {Name: "Claude Max ($20/mo)", MonthlyFee: 20, IncludedUSD: 20},
	"max-100": {Name: "Claude Max ($100/mo)", MonthlyFee: 100, IncludedUSD: 100},
}

// Summary is the result of applying a Plan to an observed API-equivalent cost.
type Summary struct {
	PlanName  string
	APIEquiv  float64 // total at pay-per-use rates
	PlanFee   float64 // monthly subscription fee
	Budget    float64 // included credit applied to this period
	Included  float64 // portion of APIEquiv covered by the budget
	Overage   float64 // amount beyond the budget (you may be billed this)
}

// Calculate applies budget to totalCost and returns a billing Summary.
// budget is the included USD for the period being displayed (caller may prorate a monthly plan).
func Calculate(totalCost float64, plan Plan, budget float64) Summary {
	var included, overage float64
	if budget > 0 {
		included = math.Min(totalCost, budget)
		overage = math.Max(0, totalCost-budget)
	} else {
		overage = totalCost
	}
	return Summary{
		PlanName: plan.Name,
		APIEquiv: totalCost,
		PlanFee:  plan.MonthlyFee,
		Budget:   budget,
		Included: included,
		Overage:  overage,
	}
}
