package main

import "fmt"

// PV is a photovoltaic (PV) power plant of peak power pv_peak in kW.
// ALL VALUES ARE IN KW FOR CONSISTENCY AND CONVENIENCE
type PV struct {
	P     float64 // inverter active power output in kW (AC side, necessarily >= 0 by convention)
	Pprod float64 // production estimation from pyranometer in kW (DC side)

	Peak float64 // peak power in kW

	SetPointP float64 // inverter active power setpoint computed by the EMS in kW (AC side, necessarily>= 0 by convention)
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
	// Prioritize live PV production
	p, pProd := pv.GetMeasure()

	// Compute if PV production is enough to cover discharge
	if (p + discharge) < pProd { // TODO: use margin for safety ?
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
	// discharge must be always positive here

	// We use PV to cover the remaining discharge
	p, _ := pv.GetMeasure()

	// PV needs to decrease production but is already at 0
	if p+discharge < 0 {
		// We still decrease to 0
		pv.SetSetpoint(0)

		// return negative result to indicate that PV is already at 0 and add discharge to cover
		return -p
	}

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
