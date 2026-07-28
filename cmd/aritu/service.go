package main

import (
	"fmt"
	"os"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/lib/service"
)

const attempts = 3

func askFor(cli *CLI) (service.Ask, error) {
	endpoint := valueOr(cli.Loaded.Service.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("no service.endpoint configured: set one in %s so model calls have somewhere to go", config.FileName)
	}
	token, err := service.Token(valueOr(cli.Loaded.Service.AuthTokenVar), os.LookupEnv)
	if err != nil {
		return nil, err
	}
	return service.Throttle(service.Retry(service.New(endpoint, token), attempts), cli.Jobs), nil
}

func valueOr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
