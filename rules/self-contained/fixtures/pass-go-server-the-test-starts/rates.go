package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Table map[string]float64

func Fetch(ctx context.Context, baseURL string) (Table, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+ratesPath, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rates: service answered %d", response.StatusCode)
	}

	var table Table
	if err := json.NewDecoder(response.Body).Decode(&table); err != nil {
		return nil, err
	}
	return table, nil
}

const ratesPath = "/rates"
