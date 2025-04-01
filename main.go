package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	// Time allocated for init phase (connections + setup).
	initTO = 30 * time.Second
)

func run(prog string, filename string) {
	ctx := context.Background()

	// read config
	cfg := config{}
	if err := cfg.Populate(ctx, filename); err != nil {
		log.Error().Err(err).Msg("failed to read config")

		return
	}

	ems := EMS{
		ESS: ESS{
			P:         0,
			PmaxCh:    cfg.ESS.PmaxCh,
			PmaxDisch: cfg.ESS.PmaxDisch,
			E:         100,
			Capacity:  cfg.ESS.Capacity,
		},
		PV: PV{
			P:     100,
			Pprod: cfg.PV.Pprod,
			Peak:  cfg.PV.Peak,
		},
		POC: POC{
			P: 5000,
		},
		PMaxSite: 10000,
	}

	go func() {
		if err := ems.Serve(ctx, time.Second); err != nil {
			log.Error().Err(err).Msg("failed to serve ems")
		}
	}()

	log.Info().Msg("ems up")

	// listen for signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for sig := range c {
		switch sig {
		case syscall.SIGHUP:
			fallthrough
		case syscall.SIGINT:
			fallthrough
		case syscall.SIGTERM:
			fmt.Println("successfully closed ems")

			return
		}
	}
}

func main() {
	args := os.Args
	if len(args) != 2 { //nolint: gomnd
		fmt.Printf("Usage: ./%s configfile\n", args[0])

		return
	}

	run(args[0], args[1])
}
