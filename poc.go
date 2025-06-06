package main

import "fmt"

type POC struct {
	// current active power measure at POC in kW (< 0 means smart grid draws power from the grid, > 0 means smart grid injects power to grid), expected to be PmaxSite < Ppoc <= 0
	P float64 `json:"p" yaml:"p" toml:"p"`
}

func (poc POC) String() string {
	return fmt.Sprintf("P: %.2f\n", poc.P)
}

// • GetMeterMeasure() returns Ppoc
func (poc POC) GetMeterMeasure() float64 {
	return poc.P
}
