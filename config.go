package main

import (
	"context"

	"github.com/ilyakaznacheev/cleanenv"
)

type config struct {
	ESS      ESS     `json:"ess" yaml:"ess" toml:"ess"`
	PV       PV      `json:"pv" yaml:"pv" toml:"pv"`
	POC      POC     `json:"poc" yaml:"poc" toml:"poc"`
	PMaxSite float64 `json:"pmaxsite" yaml:"pmaxsite" toml:"pmaxsite"`
}

// Populate populates config object reading file and env.
func (c *config) Populate(ctx context.Context, filename string) error {
	return cleanenv.ReadConfig(filename, c)
}
