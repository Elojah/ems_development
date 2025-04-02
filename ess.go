package main

import "fmt"

// ESS is an Energy Storage System (ESS, e.g. a battery) of capacity ess_capacity in kWh.
type ESS struct {
	P float64 `json:"p"` // active power output in kW (AC side) (< 0 means charge / > 0 means discharge)

	PmaxCh    float64 `json:"pmaxch"`    // maximal charge power capability in kW (AC side, necessarily <= 0 by convention)
	PmaxDisch float64 `json:"pmaxdisch"` // maximal discharge power capability in kW (AC side, necessarily >= 0 by convention)

	E        float64 `json:"e"`        // stored energy in kWh (necessarily >= 0 by convention)
	Capacity float64 `json:"capacity"` // capacity in kWh

	SetPointP float64 `json:"setpointp"` // active power setpoint computed by the EMS in kW (AC side, <0 for charge setpoint, >0 for discharge setpoint)
}

func (ess ESS) String() string {
	return fmt.Sprintf("P: %.2f, PmaxCh: %.2f, PmaxDisch: %.2f, E: %.2f, Capacity: %.2f, SetPointP: %.2f\n", ess.P, ess.PmaxCh, ess.PmaxDisch, ess.E, ess.Capacity, ess.SetPointP)
}

// GetMeasure() returns Pess, Pmaxch, Pmaxdisch, Eess
func (ess ESS) GetMeasure() (float64, float64, float64, float64) {
	return ess.P, ess.PmaxCh, ess.PmaxDisch, ess.E
}

// SetSetpoint(setpointPEss) sends setpointPEss to ESS
func (ess *ESS) SetSetpoint(setpointPEss float64) {
	fmt.Println("ESS SetSetpoint", setpointPEss)
	ess.SetPointP = setpointPEss
}

func (ess *ESS) AdjustDischarge(discharge float64) (float64, error) {
	if discharge < 0 {
		return ess.DecreaseDischarge(discharge)
	}

	return ess.IncreaseDischarge(discharge)
}

// IncreaseDischarge increases current discharge if possible and returns the remaining uncovered discharge.
// Same as DecreaseCharge.
func (ess *ESS) IncreaseDischarge(discharge float64) (float64, error) {
	// discharge must be always positive here

	// We use ESS to cover the remaining discharge
	p, _, maxDisch, e := ess.GetMeasure()

	// ESS needs to discharge but is empty
	if p+discharge > 0 && e <= 0 { // TODO: use margin for safety ?
		// We still discharge to 0
		ess.SetSetpoint(0)

		// return negative result to indicate that ESS is empty and add discharge to cover
		return -p, nil
	}

	// Compute if ESS discharge is enough to cover remaining discharge
	if p+discharge < maxDisch {
		ess.SetSetpoint(p + discharge)

		return 0, nil
	}

	// ESS discharge is not enough to cover remaining overconsumption
	// We still use max ESS discharge
	ess.SetSetpoint(maxDisch)

	// Adjust current overconsumption for clarity
	return discharge - (maxDisch - p), nil
}

// DecreaseDischarge decreases ESS discharge if possible and returns the remaining uncovered discharge.
// Same as IncreaseCharge.
func (ess *ESS) DecreaseDischarge(discharge float64) (float64, error) {
	// discharge must be always negative here

	// We use ESS to cover the remaining discharge
	p, maxCh, _, e := ess.GetMeasure()

	// EES needs to charge but is full
	if p+discharge < 0 && e >= ess.Capacity { // TODO: use margin for safety ?
		// We still cancel current discharge
		ess.SetSetpoint(0)

		// return negative result to indicate that ESS is full and add discharge to cover
		return -p, nil
	}

	// Compute if ESS discharge is enough to cover remaining discharge
	if p+discharge > maxCh {
		ess.SetSetpoint(p + discharge)

		return 0, nil
	}

	// ESS discharge is not enough to cover remaining discharge
	// We still use max ESS discharge
	ess.SetSetpoint(maxCh)

	// Adjust current discharge for clarity
	return discharge - (maxCh - p), nil
}

// BalanceEnergy balances the energy in the ESS by adjusting the charge/discharge and potentially modifying POC.
func (ess *ESS) BalanceEnergy(poc float64, pocMax float64) (float64, error) {
	pocPercentage := poc / pocMax
	ePercentage := ess.E / ess.Capacity

	// WARNING: RANDOM VALUES, need real life adjustments

	// If consumption is low and ESS is low energy, slowly modify charge/discharge
	// TODO: use some formula instead ?
	if pocPercentage < 0.3 && ePercentage < 0.7 && ess.P > ess.PmaxCh {
		// delta ensures this modification is not too big for global POC
		delta := max(-ess.PmaxCh/20, -pocMax/20)
		ess.AdjustDischarge(delta)

		return delta, nil
	}

	// If consumption is high and ESS is high energy, slowly modify charge/discharge
	// TODO: use some formula instead ?
	if pocPercentage > 0.7 && ePercentage > 0.7 && ess.P < ess.PmaxDisch {
		// delta ensures this modification is not too big for global POC
		delta := min(ess.PmaxDisch/20, pocMax/20)
		ess.AdjustDischarge(delta)

		return delta, nil
	}

	// If consumption is high and ESS is low energy, do nothing
	// If consumption is low and ESS is high energy, do nothing

	return 0, nil
}
