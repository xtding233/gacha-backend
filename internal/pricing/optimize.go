package pricing

import "math"

// MinCostAtLeastTokens finds the minimum-cost combination to obtain at least targetTokens.
// It supports per-pack first-time x2 by expanding "effective packs": each pack can appear as
// an x2 variant (if first-time available) and a normal variant. Unbounded quantities allowed.
func MinCostAtLeastTokens(cat Catalog, targetTokens int, first FirstTimeState) Plan {
	if targetTokens <= 0 || len(cat.Packs) == 0 {
		return Plan{Currency: cat.Currency}
	}

	// Build effective pack variants.
	type eff struct {
		id   string
		name string
		tok  int
		price int
		baseID string // original pack id (to group in output if needed)
	}
	var effs []eff
	for _, p := range cat.Packs {
		base := p.Tokens + p.BonusTokens
		// x2 variant
		if p.FirstTimeX2 && first != nil && first[p.ID] {
			effs = append(effs, eff{
				id:    p.ID + "#x2",
				name:  p.Name + " (x2)",
				tok:   p.Tokens*2 + p.BonusTokens, // x2 applies to base Tokens only
				price: p.PriceCents,
				baseID: p.ID,
			})
		}
		// normal variant
		effs = append(effs, eff{
			id:    p.ID,
			name:  p.Name,
			tok:   base,
			price: p.PriceCents,
			baseID: p.ID,
		})
	}

	// DP over tokens up to target + maxPack to permit slight overshoot with minimal cost.
	maxTok := 0
	for _, e := range effs { if e.tok > maxTok { maxTok = e.tok } }
	if maxTok == 0 {
		return Plan{Currency: cat.Currency}
	}
	limit := targetTokens + maxTok

	const Inf = int(^uint(0) >> 1)
	dp := make([]int, limit+1)  // min cost to reach exactly t tokens
	pr := make([]int, limit+1)  // chosen eff index
	prev := make([]int, limit+1) // previous t
	for t := range dp { dp[t] = Inf; pr[t] = -1; prev[t] = -1 }
	dp[0] = 0

	for t := 0; t <= limit; t++ {
		if dp[t] == Inf { continue }
		for i, e := range effs {
			nt := t + e.tok
			if nt > limit { nt = limit }
			cost := dp[t] + e.price
			if cost < dp[nt] {
				dp[nt] = cost
				pr[nt] = i
				prev[nt] = t
			}
		}
	}

	// pick best t >= target
	bestT, bestCost := targetTokens, dp[targetTokens]
	for t := targetTokens; t <= limit; t++ {
		if dp[t] < bestCost {
			bestT, bestCost = t, dp[t]
		}
	}

	// reconstruct counts
	type key struct {
		id    string
		name  string
		price int
		tok   int
	}
	counts := map[key]int{}
	t := bestT
	for t > 0 && pr[t] != -1 {
		e := effs[pr[t]]
		k := key{id: e.id, name: e.name, price: e.price, tok: e.tok}
		counts[k]++
		t = prev[t]
	}

	// build plan
	var plan Plan
	plan.Currency = cat.Currency
	for k, qty := range counts {
		sub := k.price * qty
		plan.Purchases = append(plan.Purchases, Purchase{
			PackID:    k.id,
			Name:      k.name,
			Qty:       qty,
			UnitPrice: k.price,
			UnitTokens: k.tok,
			Subtotal:  sub,
		})
		plan.SubCents += sub
		plan.TotalTokens += k.tok * qty
	}
	plan.TaxCents, plan.TotalCents = applyTax(plan.SubCents, cat.TaxRate)
	return plan
}

// MaxTokensUnderBudget computes the maximum tokens purchasable with budgetCents.
// It ignores targetTokens and uses unbounded knapsack by value-density & DP hybrid.
//
// Simple approach: DP on budget (unbounded knapsack) using effective variants.
func MaxTokensUnderBudget(cat Catalog, budgetCents int, first FirstTimeState) Plan {
	if budgetCents <= 0 || len(cat.Packs) == 0 {
		return Plan{Currency: cat.Currency}
	}

	// Expand variants same as above.
	type eff struct {
		id, name string
		tok, price int
	}
	var effs []eff
	for _, p := range cat.Packs {
		base := p.Tokens + p.BonusTokens
		if p.FirstTimeX2 && first != nil && first[p.ID] {
			effs = append(effs, eff{p.ID + "#x2", p.Name + " (x2)", p.Tokens*2 + p.BonusTokens, p.PriceCents})
		}
		effs = append(effs, eff{p.ID, p.Name, base, p.PriceCents})
	}
	if len(effs) == 0 {
		return Plan{Currency: cat.Currency}
	}

	// If prices are pre-tax, reduce effective budget by tax to approximate pre-tax spend.
	// For exactness, you'd iterate quantity combos and then add tax; here we assume tax applies to subtotal.
	effBudget := budgetCents
	if cat.TaxRate > 0 {
		effBudget = int(math.Floor(float64(budgetCents) / (1 + cat.TaxRate)))
	}

	// dp[c] = max tokens with cost exactly c
	dp := make([]int, effBudget+1)
	choose := make([]int, effBudget+1)
	for c := 0; c <= effBudget; c++ {
		choose[c] = -1
	}
	for c := 0; c <= effBudget; c++ {
		for i, e := range effs {
			nc := c + e.price
			if nc <= effBudget {
				val := dp[c] + e.tok
				if val > dp[nc] {
					dp[nc] = val
					choose[nc] = i
				}
			}
		}
	}
	// best at any cost <= effBudget
	bestC := 0
	for c := 0; c <= effBudget; c++ {
		if dp[c] > dp[bestC] {
			bestC = c
		}
	}

	// reconstruct
	type key struct{ id, name string; price, tok int }
	counts := map[key]int{}
	c := bestC
	for c > 0 && choose[c] != -1 {
		e := effs[choose[c]]
		k := key{e.id, e.name, e.price, e.tok}
		counts[k]++
		c -= e.price
	}

	var plan Plan
	plan.Currency = cat.Currency
	for k, qty := range counts {
		sub := k.price * qty
		plan.Purchases = append(plan.Purchases, Purchase{
			PackID:    k.id,
			Name:      k.name,
			Qty:       qty,
			UnitPrice: k.price,
			UnitTokens: k.tok,
			Subtotal:  sub,
		})
		plan.SubCents += sub
		plan.TotalTokens += k.tok * qty
	}
	plan.TaxCents, plan.TotalCents = applyTax(plan.SubCents, cat.TaxRate)
	return plan
}
