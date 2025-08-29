// resolve.go
package game

// Resolve merges default → game → pool → overrides into engine params.
// 'overrides' carries query overrides like cushion/p_base/etc.
type Overrides struct {
	PBase     *float64
	StartAt   *int
	StartPct  *float64
	Target    *float64
	Increment *float64
	Easing    *string
	OffProbs  *[]float64
	MaxOff    *int
	Cushion   *int
}

type Resolver interface {
	// Returns merged RawConfig and normalized EngineParams
	Resolve(game, pool string, o Overrides) (RawConfig, EngineParams, error)
}
