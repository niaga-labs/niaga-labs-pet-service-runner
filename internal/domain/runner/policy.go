package runner

// BookingRequirements represents the requirements from a booking that must
// be matched against a runner's capabilities.
type BookingRequirements struct {
	PetType       PetType
	PetWeightKg   float64
	CrateSize     CrateSize
	NeedsAirCon   bool
	NeedsTempCtrl bool
}

// QualityPolicy defines the interface for evaluating runner eligibility.
type QualityPolicy interface {
	IsEligible(runner *Runner, crateSpecs []*CrateSpec, requirements BookingRequirements) (bool, []string)
}

// DefaultQualityPolicy implements the standard eligibility rules.
type DefaultQualityPolicy struct{}

// NewDefaultQualityPolicy creates a new DefaultQualityPolicy.
func NewDefaultQualityPolicy() *DefaultQualityPolicy {
	return &DefaultQualityPolicy{}
}

// IsEligible checks whether a runner meets the booking requirements.
// Returns true if eligible, along with a list of reasons if not.
func (p *DefaultQualityPolicy) IsEligible(runner *Runner, crateSpecs []*CrateSpec, requirements BookingRequirements) (bool, []string) {
	var reasons []string

	// Check if runner is active
	if !runner.IsActive() {
		reasons = append(reasons, "runner is not active")
	}

	// Check if runner has a matching crate size
	hasCrateSize := false
	var matchingCrate *CrateSpec
	for _, cs := range crateSpecs {
		if cs.Size == requirements.CrateSize {
			hasCrateSize = true
			matchingCrate = cs
			break
		}
	}
	if !hasCrateSize {
		reasons = append(reasons, "no crate of required size")
	}

	// Check if the matching crate supports the pet type
	if matchingCrate != nil {
		if !matchingCrate.SupportsPetType(requirements.PetType) {
			reasons = append(reasons, "crate does not support required pet type")
		}

		// Check weight capacity
		if !matchingCrate.SupportsWeight(requirements.PetWeightKg) {
			reasons = append(reasons, "crate weight capacity exceeded")
		}

		// Check temperature control if needed
		if requirements.NeedsTempCtrl && !matchingCrate.TemperatureControlled {
			reasons = append(reasons, "crate does not have temperature control")
		}
	}

	// Check vehicle air conditioning if needed
	if requirements.NeedsAirCon && !runner.AirConditioned() {
		reasons = append(reasons, "vehicle does not have air conditioning")
	}

	// Check minimum rating
	if runner.Rating() < 3.0 && runner.TotalTrips() > 0 {
		reasons = append(reasons, "runner rating below minimum threshold of 3.0")
	}

	eligible := len(reasons) == 0
	return eligible, reasons
}
