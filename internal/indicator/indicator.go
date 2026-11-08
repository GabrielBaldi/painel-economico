// Package indicator holds the domain types (indicators and their data points)
// and the service that serves them from a cache in front of the upstream APIs.
package indicator

import (
	"context"
	"time"
)

// Frequency is how often an indicator gets a new value.
type Frequency string

const (
	Daily   Frequency = "daily"
	Monthly Frequency = "monthly"
)

// Indicator describes one economic series as exposed by the API.
type Indicator struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ShortName   string    `json:"shortName"`
	Unit        string    `json:"unit"`
	Frequency   Frequency `json:"frequency"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"sourceUrl"`
}

// Point is one observation. Date is an ISO date (YYYY-MM-DD); monthly series
// use the first day of the month. ISO dates sort correctly as plain strings.
type Point struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// Source fetches the observations of one series between two dates (inclusive).
type Source interface {
	Fetch(ctx context.Context, from, to time.Time) ([]Point, error)
}

// Definition pairs an indicator with the upstream that provides its data.
type Definition struct {
	Indicator
	Upstream Source
}
