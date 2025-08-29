package pricing

import "math"

// Pack models a purchasable SKU in the store.
type Pack struct {
	ID          string // SKU id, e.g., "6480"
	Name        string // display name, e.g., "6480 Pack"
	Tokens      int    // base tokens granted
	BonusTokens int    // permanent extra tokens (non-first-time)
	FirstTimeX2 bool   // if true, first-time purchase doubles base Tokens (not BonusTokens)
	PriceCents  int    // price in minor units (e.g., cents)
}

// Catalog is a regional product catalog and tax info.
type Catalog struct {
	TokenName string  // e.g., "Stellar Jade"
	Currency  string  // ISO code, e.g., "CAD"
	// If prices are pre-tax, TaxRate is applied on subtotal to compute total.
	// If your prices are tax-inclusive, set TaxRate=0 and pass the inclusive price as PriceCents.
	TaxRate float64 // e.g., 0.13 for 13%
	Packs   []Pack
}

// FirstTimeState describes per-pack first-time eligibility.
type FirstTimeState map[string]bool // packID -> true if first-time x2 is still available

// Plan summarizes a purchase plan.
type Plan struct {
	Purchases []Purchase
	SubCents  int // subtotal before tax
	TaxCents  int
	TotalCents int
	TotalTokens int
	Currency   string
}

// Purchase is one line item in the plan.
type Purchase struct {
	PackID    string
	Name      string
	Qty       int
	UnitPrice int // cents
	UnitTokens int // tokens received per unit in this plan (x2/bonus applied)
	Subtotal  int // cents
}

// applyTax computes tax and total given a subtotal and a tax rate.
func applyTax(sub int, taxRate float64) (tax int, total int) {
	if taxRate <= 0 {
		return 0, sub
	}
	t := int(math.Round(float64(sub) * taxRate))
	return t, sub + t
}
