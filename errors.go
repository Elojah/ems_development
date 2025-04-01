package main

import "fmt"

type ErrESSEmpty struct {
	Required float64
}

func (err ErrESSEmpty) Error() string {
	return fmt.Sprintf("ESS is empty, required %f kWh", err.Required)
}

type ErrGridMissingCoverage struct {
	Required float64
}

func (err ErrGridMissingCoverage) Error() string {
	return fmt.Sprintf("Grid missing coverage, required %f kWh", err.Required)
}
