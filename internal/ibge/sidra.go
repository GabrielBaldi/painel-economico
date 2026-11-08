// Package ibge reads tables from SIDRA, the IBGE aggregate data API
// (https://apisidra.ibge.gov.br).
package ibge

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"painel-economico/internal/indicator"
)

// DefaultBaseURL is the public SIDRA endpoint.
const DefaultBaseURL = "https://apisidra.ibge.gov.br/values"

// sidraPeriod is the layout of SIDRA monthly period codes (e.g. 202609).
const sidraPeriod = "200601"

// missingValues are the symbols SIDRA uses instead of a number:
// "-" zero by rounding, ".." not applicable, "..." not available, "X" withheld.
var missingValues = map[string]bool{"-": true, "..": true, "...": true, "X": true}

// Getter fetches a URL and returns the response body.
type Getter interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

// Client builds sources for SIDRA variables.
type Client struct {
	baseURL string
	http    Getter
}

// New returns a Client that queries baseURL through http.
func New(baseURL string, http Getter) *Client {
	return &Client{baseURL: baseURL, http: http}
}

// Variable returns a source for one variable of a monthly SIDRA table, for
// Brazil as a whole.
func (c *Client) Variable(table, variable int) Variable {
	return Variable{client: c, table: table, variable: variable}
}

// Variable is one variable of a SIDRA table. It implements indicator.Source.
type Variable struct {
	client          *Client
	table, variable int
}

// Fetch returns the observations whose period falls between from and to.
func (v Variable) Fetch(ctx context.Context, from, to time.Time) ([]indicator.Point, error) {
	// n1/all: Brazil; p: period range; f/c: codes only; h/n: no header row.
	url := fmt.Sprintf("%s/t/%d/n1/all/v/%d/p/%s-%s/f/c/h/n",
		v.client.baseURL, v.table, v.variable, from.Format(sidraPeriod), to.Format(sidraPeriod))

	body, err := v.client.http.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("sidra table %d: %w", v.table, err)
	}
	points, err := parse(body)
	if err != nil {
		return nil, fmt.Errorf("sidra table %d: %w", v.table, err)
	}
	return points, nil
}

// parse decodes a SIDRA response. With a territorial level and a variable in
// the query, the period is dimension 3: D3C holds its code, V the value.
func parse(body []byte) ([]indicator.Point, error) {
	var rows []struct {
		Period string `json:"D3C"`
		Value  string `json:"V"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	points := make([]indicator.Point, 0, len(rows))
	for _, row := range rows {
		if missingValues[row.Value] {
			continue
		}
		period, err := time.Parse(sidraPeriod, row.Period)
		if err != nil {
			return nil, fmt.Errorf("parse period %q: %w", row.Period, err)
		}
		value, err := strconv.ParseFloat(row.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("parse value %q: %w", row.Value, err)
		}
		points = append(points, indicator.Point{Date: period.Format(time.DateOnly), Value: value})
	}
	return points, nil
}
