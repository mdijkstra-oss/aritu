package main

import (
	"fmt"
	"os"

	"github.com/matthijn/aritu/internal/domain/config"
	"github.com/matthijn/aritu/internal/lib/service"
)

const attempts = 3

func askFor(resolved settings) (service.Ask, error) {
	endpoint := stringOr(resolved.Config.Service.Endpoint, "")
	if endpoint == "" {
		return nil, fmt.Errorf("no service.endpoint configured: set one in %s so model calls have somewhere to go", config.FileName)
	}
	token, err := service.Token(stringOr(resolved.Config.Service.AuthTokenVar, ""), os.LookupEnv)
	if err != nil {
		return nil, err
	}
	return service.Throttle(service.Retry(service.New(endpoint, token), attempts), resolved.Parallel), nil
}
