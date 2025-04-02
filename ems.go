package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

// There is not direct metering of industrial facility consumption Pload, but it can be deduced.
// At the point of connection (POC) with the grid, the meter measures Ppoc the power flowing from/to
// the grid. Ppoc is the resultant of smart grid productions and consumptions: Ppoc = Pess + Ppv +
// Pload, but remember that Pload is not directly available for the EMS).
//
// EMS objective is to ensure that the industrial site power consumption remains under a maximal
// value PmaxSite, and that no electricity is injected to the grid. This means PmaxSite < Ppoc <= 0
type EMS struct {
	ESS ESS
	PV  PV
	POC POC

	PMaxSite float64
}

func (ems EMS) String() string {
	return fmt.Sprintf("EMS:\n\tess:%v\n\tpv:%v\n\tpoc:%v\n\tpmaxsite:%v\n", ems.ESS, ems.PV, ems.POC, ems.PMaxSite)
}

// Next for debugging purposes simulates a next step in decision loop
func (ems *EMS) Next() {
	ems.ESS.E -= ems.ESS.P
	ems.ESS.P = ems.ESS.SetPointP

	ems.PV.P = ems.PV.SetPointP
	ems.PV.Pprod += float64(rand.Int63n(10)-5) / 100 * ems.PV.Peak
	if ems.PV.Pprod < 0 {
		ems.PV.Pprod = 0
	} else if ems.PV.Pprod > ems.PV.Peak {
		ems.PV.Pprod = ems.PV.Peak
	}

	ems.POC.P = -ems.PV.P - ems.ESS.P + (float64(rand.Int63n(10)+10) / 100 * ems.PMaxSite)
}

func (ems EMS) GetPLoad() float64 {
	// TODO: Ensure those 3 variables are returned at same timestamp to guarantee validity.
	return ems.POC.P - ems.PV.P - ems.ESS.P
}

func (ems EMS) Serve(ctx context.Context, delay time.Duration) error {
	// Adjust margins
	margin := 0.1 // 10% margin for safety triggers
	ems.PMaxSite = ems.PMaxSite - (ems.PMaxSite * margin)
	pMinSite := 0 + ems.PMaxSite*margin

	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	// execute decision loop every delay
	for range ticker.C {
		log.Info().Msg("ems decision in progress...")

		// force quit if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// DEBUG: simulate next iteration
		ems.Next()
		fmt.Println(ems)

		/*
		 DOMAIN LOGIC
		*/

		poc := ems.POC.GetMeterMeasure()
		// fmt.Printf("poc: %v\nems:%v\n", poc, ems)

		// WARNING: Consumption exceeds PmaxSite
		// CHANGE POC
		if poc > ems.PMaxSite {
			if err := ems.IncreaseSiteDischarge(poc - ems.PMaxSite); err != nil {
				log.Error().Err(err).Msg("failed to increase site discharge")

				continue
			}

			log.Info().Msg("increased site discharge")
			continue
		}

		// WARNING: Consumption is below pMinSite
		// CHANGE POC
		if poc < pMinSite {
			if err := ems.DecreaseSiteDischarge(poc - pMinSite); err != nil {
				log.Error().Err(err).Msg("failed to decrease site discharge")

				continue
			}

			log.Info().Msg("decreased site discharge")
			continue
		}

		// Balance PV and ESS productions
		// KEEP POC
		if err := ems.BalanceSiteDischarge(poc); err != nil {
			log.Error().Err(err).Msg("failed to balance site discharge")

			continue
		}
		log.Info().Msg("balanced site discharge")

		// Adjust PV charge (and POC) depending on current stored energy and poc %
		// CHANGE POC
		if delta, err := ems.PV.BalanceEnergy(poc, ems.PMaxSite); err != nil {
			log.Error().Err(err).Msg("failed to balance pv energy")

			continue
		} else {
			poc += delta
		}
		log.Info().Msg("balanced pv energy")

		// Adjust ESS charge (and POC) depending on current stored energy and poc %
		// CHANGE POC
		if delta, err := ems.ESS.BalanceEnergy(poc, ems.PMaxSite); err != nil {
			log.Error().Err(err).Msg("failed to balance ess energy")

			continue
		} else {
			poc += delta
		}
		log.Info().Msg("balanced ess energy")

		log.Info().Msg("ems decision done")
	}

	return nil
}

// BalanceSiteDischarge balances site discharge by adjusting ESS and PV productions.
// It keeps same POC value.
func (ems *EMS) BalanceSiteDischarge(poc float64) error {
	// We try to maximize PV discharge
	available := ems.PV.AvailableProd()

	// We try to minimize ESS discharge
	p, pMaxCh, _, _ := ems.ESS.GetMeasure()

	if available > 0 && p > 0 {
		// both PV and ESS are discharging
		if available > p {
			// We have more PV discharge than ESS discharge
			ems.PV.AdjustDischarge(p)
			ems.ESS.AdjustDischarge(-p)
		} else {
			ems.PV.AdjustDischarge(available)
			ems.ESS.AdjustDischarge(-available) // TODO: potentially check result is 0 ?
		}
	} else if available > 0 && p < 0 {
		// PV is discharging but ESS is charging
		// maxChAvailable is the positive diff between ESS max charge and current charge
		maxChAvailable := p - pMaxCh

		if available > maxChAvailable {
			ems.PV.AdjustDischarge(maxChAvailable)
			ems.ESS.AdjustDischarge(-maxChAvailable)
		} else {
			ems.PV.AdjustDischarge(available)
			ems.ESS.AdjustDischarge(-available) // TODO: potentially check result is 0 ?
		}
	}

	return nil
}

// IncreaseSiteDischarge handles external discharge by utilizing the grid.
func (ems *EMS) IncreaseSiteDischarge(discharge float64) error {
	// Prioritize PV discharge
	discharge = ems.PV.AdjustDischarge(discharge)
	if discharge == 0 {
		return nil
	}

	// Use ESS to cover remaining discharge
	var err error
	discharge, err = ems.ESS.AdjustDischarge(discharge)
	if err != nil {
		return err
	}

	if discharge != 0 {
		return ErrGridMissingCoverage{Required: discharge}
	}

	return nil
}

func (ems *EMS) DecreaseSiteDischarge(discharge float64) error {
	// Prioritize ESS discharge/charge
	discharge, err := ems.ESS.AdjustDischarge(discharge)
	if err != nil {
		return err
	}
	if discharge == 0 {
		return nil
	}

	// Use PV production to cover remaining discharge
	discharge = ems.PV.AdjustDischarge(discharge)

	if discharge != 0 {
		return ErrGridMissingCoverage{Required: discharge}
	}

	return nil
}
