package main

import "fmt"

// PV is a photovoltaic (PV) power plant of peak power pv_peak in kW.
// ALL VALUES ARE IN KW FOR CONSISTENCY AND CONVENIENCE
type PV struct {
	P     float64 `json:"p"`     // inverter active power output in kW (AC side, necessarily >= 0 by convention)
	Pprod float64 `json:"pprod"` // production estimation from pyranometer in kW (DC side)

	Peak float64 `json:"peak"` // peak power in kW

	SetPointP float64 `json:"setpointp"` // inverter active power setpoint computed by the EMS in kW (AC side, necessarily >= 0 by convention)
}

func (pv PV) String() string {
	return fmt.Sprintf("P: %.2f, Pprod: %.2f, Peak: %.2f, SetPointP: %.2f\n", pv.P, pv.Pprod, pv.Peak, pv.SetPointP)
}

// GetMeasure() returns Ppv, Pprod
func (pv PV) GetMeasure() (float64, float64) {
	return pv.P, pv.Pprod
}

// SetSetpoint(setpointPPv) sends setpointPPv to PV inverter
func (pv *PV) SetSetpoint(setpointPPv float64) {
	fmt.Println("PV SetSetpoint", setpointPPv)
	pv.SetPointP = setpointPPv
}

func (pv *PV) AvailableProd() float64 {
	return pv.Pprod - pv.P
}

// AdjustDischarge(discharge) adjusts PV production to cover discharge if possible and returns the remaining uncovered discharge.
func (pv *PV) AdjustDischarge(discharge float64) float64 {
	if discharge < 0 {
		return pv.DecreaseDischarge(discharge)
	}

	return pv.IncreaseDischarge(discharge)
}

// IncreaseDischarge increases PV discharge if possible and returns the remaining uncovered discharge.
func (pv *PV) IncreaseDischarge(discharge float64) float64 {
	// discharge must be always positive here

	// Prioritize live PV production
	p, pProd := pv.GetMeasure()

	// Compute if PV production is enough to cover discharge
	if p+discharge < pProd { // TODO: use margin for safety ?
		pv.SetSetpoint(p + discharge)

		return 0
	}

	// PV production is not enough to cover discharge
	// We still use max PV production
	pv.SetSetpoint(pProd)

	// Adjust current discharge for clarity
	return discharge - (pProd - p)
}

// DecreaseDischarge decreases PV discharge if possible and returns the remaining uncovered discharge.
func (pv *PV) DecreaseDischarge(discharge float64) float64 {
	// discharge must be always negative here

	// We use PV to cover the remaining discharge
	p, _ := pv.GetMeasure()

	// Compute if PV production is enough to cover discharge
	if p+discharge > 0 {
		pv.SetSetpoint(p + discharge)

		return 0
	}

	// PV production is not enough to cover discharge
	// We still use max PV production
	pv.SetSetpoint(0)

	// Adjust current discharge for clarity
	return discharge - p
}

// BalanceEnergy balances the energy in the PV by adjusting the discharge and potentially modifying POC.
func (pv *PV) BalanceEnergy(poc float64, pocMax float64) (float64, error) {
	// We try to maximize PV discharge
	pocPercentage := poc / pocMax
	available := pv.AvailableProd()

	if available < 0 {
		pv.AdjustDischarge(available)

		return available, nil
	}

	if available > 0 && pocPercentage < 0.8 {
		// delta ensures this modification is not too big for global POC
		delta := min(available, pocMax/20)
		pv.AdjustDischarge(delta)

		return delta, nil
	}

	return 0, nil
}
