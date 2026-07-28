package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"

	"github.com/matthijn/aritu/internal/domain/config"
)

func configPathFor(explicit string) (path string, isFound bool, err error) {
	if explicit != "" {
		return explicit, true, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("config search: %w", err)
	}
	return config.Find(dir)
}

func resolverFor(loaded config.Config) kong.ResolverFunc {
	return func(_ *kong.Context, _ *kong.Path, flag *kong.Flag) (any, error) {
		value, isSet := loaded.Lookup(flag.Name)
		if !isSet {
			return nil, nil
		}
		return mappable(value), nil
	}
}

// kong's duration mapper switches on concrete types and time.Duration is not
// among them.
func mappable(value any) any {
	if duration, isDuration := value.(time.Duration); isDuration {
		return int64(duration)
	}
	return value
}

// During kong's BeforeResolve the parsed values have not reached the grammar
// struct yet.
func flagValue(kctx *kong.Context, name string) string {
	for _, flag := range kctx.Flags() {
		if flag.Name == name {
			return fmt.Sprint(kctx.FlagValue(flag))
		}
	}
	return ""
}
