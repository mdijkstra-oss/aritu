package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    func(dir string) Config
		wantErr string
	}{
		{
			name: "every key the file may carry survives the round trip",
			file: `votes: 4
jobs: 3
timeout: 1h30s
output: json
service:
  endpoint: https://gateway.internal/v1
  auth_token_var: ARITU_TOKEN
  model: opus
  effort: high
rules:
  dir: ./rules
  enabled: [named-for-behavior, one-reason-to-fail]
targets:
  tests:
    - 'internal/**/*_test.go'
    - 'cmd/**/*_test.go'
  migrations:
    - 'db/migrate/**/*.sql'
`,
			want: func(dir string) Config {
				return Config{
					Dir:    dir,
					Output: given("json"),
					Service: Service{
						Endpoint:     given("https://gateway.internal/v1"),
						AuthTokenVar: given("ARITU_TOKEN"),
						Model:        given("opus"),
						Effort:       given("high"),
					},
					Votes:   given(4),
					Jobs:    given(3),
					Timeout: given(Duration(time.Hour + 30*time.Second)),
					Rules: Rules{
						Dir:     given(filepath.Join(dir, "rules")),
						Enabled: []string{"named-for-behavior", "one-reason-to-fail"},
					},
					Targets: map[string][]string{
						"tests": {
							filepath.Join(dir, "internal/**/*_test.go"),
							filepath.Join(dir, "cmd/**/*_test.go"),
						},
						"migrations": {filepath.Join(dir, "db/migrate/**/*.sql")},
					},
				}
			},
		},
		{
			name: "keys the file omits stay absent",
			file: "votes: 2\n",
			want: func(dir string) Config {
				return Config{Dir: dir, Votes: given(2)}
			},
		},
		{
			name: "a zero the file writes down is kept apart from a key it never carried",
			file: "votes: 0\n",
			want: func(dir string) Config {
				return Config{Dir: dir, Votes: given(0)}
			},
		},
		{
			name: "an empty file loads as a config that sets nothing",
			file: "",
			want: func(dir string) Config {
				return Config{Dir: dir}
			},
		},
		{
			name: "a file holding only comments loads as a config that sets nothing",
			file: "# nothing to say yet\n",
			want: func(dir string) Config {
				return Config{Dir: dir}
			},
		},
		{
			name:    "a misspelled key fails the load naming the key",
			file:    "vote: 4\n",
			wantErr: "vote",
		},
		{
			name: "a misspelled key inside the rules block fails the load naming the key",
			file: `rules:
  folder: ./rules
`,
			wantErr: "folder",
		},
		{
			name:    "a model written outside the service block fails the load rather than being ignored",
			file:    "model: opus\n",
			wantErr: "model",
		},
		{
			name:    "a misspelled service block fails the load rather than leaving the endpoint unset",
			file:    "servce:\n  endpoint: https://gateway.internal/v1\n",
			wantErr: "servce",
		},
		{
			name: "a misspelled key inside the service block fails the load naming the key",
			file: `service:
  endpoint: https://gateway.internal/v1
  auth_token: ARITU_TOKEN
`,
			wantErr: "auth_token",
		},
		{
			name: "an endpoint on its own is a whole service block",
			file: `service:
  endpoint: http://localhost:8080/v1
`,
			want: func(dir string) Config {
				return Config{
					Dir:     dir,
					Service: Service{Endpoint: given("http://localhost:8080/v1")},
				}
			},
		},
		{
			name:    "a timeout that is not a duration fails the load naming the value",
			file:    "timeout: soon\n",
			wantErr: `"soon"`,
		},
		{
			name:    "a timeout given as a bare number fails the load rather than meaning nanoseconds",
			file:    "timeout: 600\n",
			wantErr: "600",
		},
		{
			name:    "a timeout given as a list rather than a scalar fails the load",
			file:    "timeout: [10m]\n",
			wantErr: "unmarshal",
		},
		{
			name:    "malformed yaml fails the load",
			file:    "votes: [1\n",
			wantErr: "yaml",
		},
		{
			name: "a relative rules dir resolves against the config file rather than the working directory",
			file: `rules:
  dir: ../shared/rules
`,
			want: func(dir string) Config {
				return Config{Dir: dir, Rules: Rules{Dir: given(filepath.Join(filepath.Dir(dir), "shared/rules"))}}
			},
		},
		{
			name: "an absolute rules dir is left as written",
			file: `rules:
  dir: /opt/aritu/rules
`,
			want: func(dir string) Config {
				return Config{Dir: dir, Rules: Rules{Dir: given("/opt/aritu/rules")}}
			},
		},
		{
			name: "a relative target pattern resolves against the config file",
			file: `targets:
  tests:
    - '../sibling/**/*_test.go'
`,
			want: func(dir string) Config {
				return Config{Dir: dir, Targets: map[string][]string{
					"tests": {filepath.Join(filepath.Dir(dir), "sibling/**/*_test.go")},
				}}
			},
		},
		{
			name: "an absolute target pattern is left as written",
			file: `targets:
  tests:
    - '/srv/checkout/**/*_test.go'
`,
			want: func(dir string) Config {
				return Config{Dir: dir, Targets: map[string][]string{
					"tests": {"/srv/checkout/**/*_test.go"},
				}}
			},
		},
		{
			name:    "the include list this repository used to carry is now a key nobody defined",
			file:    "include:\n  - 'internal/**/*_test.go'\n",
			wantErr: "include",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeConfig(t, dir, tt.file)

			got, err := Load(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Load(%q) = %+v, want an error mentioning %q", tt.file, got, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Load(%q) error = %v, want it to mention %q", tt.file, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load(%q) error = %v, want none", tt.file, err)
			}
			if want := tt.want(dir); !reflect.DeepEqual(got, want) {
				t.Errorf("Load(%q) = %+v, want %+v", tt.file, got, want)
			}
		})
	}
}

func TestLoadOfAPathThatHoldsNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "elsewhere", FileName)

	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load(%q) succeeded, want an error naming the path", path)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("Load(%q) error = %v, want it to name the path", path, err)
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name      string
		configs   []string
		start     string
		want      string
		wantFound bool
	}{
		{
			name:      "a config beside the start directory is the one found",
			configs:   []string{"."},
			start:     ".",
			want:      ".",
			wantFound: true,
		},
		{
			name:      "a config two levels up is found from a subdirectory",
			configs:   []string{"."},
			start:     "internal/lib/vote",
			want:      ".",
			wantFound: true,
		},
		{
			name:      "the nearest config wins over one further up",
			configs:   []string{".", "internal"},
			start:     "internal/lib/vote",
			want:      "internal",
			wantFound: true,
		},
		{
			name:    "no config above the start directory stops at the filesystem root without erroring",
			configs: nil,
			start:   "internal/lib/vote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, dir := range tt.configs {
				writeConfig(t, mkdir(t, root, dir), "")
			}

			got, found, err := Find(mkdir(t, root, tt.start))
			if err != nil {
				t.Fatalf("Find error = %v, want none", err)
			}
			if found != tt.wantFound {
				t.Fatalf("Find found = %v, want %v", found, tt.wantFound)
			}
			want := ""
			if tt.wantFound {
				want = filepath.Join(root, tt.want, FileName)
			}
			if got != want {
				t.Errorf("Find = %q, want %q", got, want)
			}
		})
	}
}

func TestLookupOfAConfiguredValue(t *testing.T) {
	config := Config{
		Output:  given("json"),
		Votes:   given(4),
		Jobs:    given(3),
		Timeout: given(Duration(90 * time.Second)),
		Service: Service{Model: given("opus"), Effort: given("high")},
		Rules:   Rules{Dir: given("/repo/rules")},
	}
	tests := []struct {
		name string
		flag string
		want any
	}{
		{name: "the model the file names reaches the resolver", flag: "model", want: "opus"},
		{name: "the effort the file names reaches the resolver", flag: "effort", want: "high"},
		{name: "the output the file names reaches the resolver", flag: "output", want: "json"},
		{name: "the vote count the file sets reaches the resolver", flag: "votes", want: 4},
		{name: "the job limit the file sets reaches the resolver", flag: "jobs", want: 3},
		{name: "the timeout arrives as a duration rather than the yaml text", flag: "timeout", want: 90 * time.Second},
		{name: "the rules block supplies the rules directory", flag: "rules", want: "/repo/rules"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isSet := config.Lookup(tt.flag)
			if !isSet {
				t.Fatalf("Lookup(%q) reported unset, want set", tt.flag)
			}
			if got != tt.want {
				t.Errorf("Lookup(%q) = %#v, want %#v", tt.flag, got, tt.want)
			}
		})
	}
}

// TestLookupOfAZeroTheFileWroteDown pins the difference an omitted key and a key
// carrying zero have to keep: the omission leaves the built-in default standing,
// while the zero has to reach the one validator and be rejected there.
func TestLookupOfAZeroTheFileWroteDown(t *testing.T) {
	config := Config{Votes: given(0), Jobs: given(0), Service: Service{Model: given("")}, Timeout: given(Duration(0))}
	tests := []struct {
		name string
		flag string
		want any
	}{
		{name: "a votes of zero resolves rather than leaving the default standing", flag: "votes", want: 0},
		{name: "a jobs of zero resolves rather than leaving the default standing", flag: "jobs", want: 0},
		{name: "an empty model resolves rather than leaving the default standing", flag: "model", want: ""},
		{name: "a zero timeout resolves rather than leaving the default standing", flag: "timeout", want: time.Duration(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isSet := config.Lookup(tt.flag)
			if !isSet {
				t.Fatalf("Lookup(%q) reported unset, want set", tt.flag)
			}
			if got != tt.want {
				t.Errorf("Lookup(%q) = %#v, want %#v", tt.flag, got, tt.want)
			}
		})
	}
}

func TestLookupOfAValueTheFileNeverSet(t *testing.T) {
	tests := []struct {
		name string
		flag string
	}{
		{name: "an omitted model leaves the built-in default standing", flag: "model"},
		{name: "an omitted effort leaves the built-in default standing", flag: "effort"},
		{name: "an omitted output leaves the built-in default standing", flag: "output"},
		{name: "an omitted votes does not resolve to zero votes", flag: "votes"},
		{name: "an omitted jobs does not resolve to no jobs", flag: "jobs"},
		{name: "an omitted timeout does not resolve to an instant deadline", flag: "timeout"},
		{name: "an omitted rules dir leaves the built-in default standing", flag: "rules"},
		{name: "a flag the file has no key for is simply unset", flag: "config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isSet := Config{}.Lookup(tt.flag)
			if isSet {
				t.Fatalf("Lookup(%q) = %#v, want unset", tt.flag, got)
			}
			if got != nil {
				t.Errorf("Lookup(%q) = %#v, want nil alongside unset", tt.flag, got)
			}
		})
	}
}

// given is a key the file carried, as opposed to nil, which is one it did not.
func given[T any](value T) *T {
	return &value
}

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

func mkdir(t *testing.T, root, relative string) string {
	t.Helper()
	dir := filepath.Join(root, relative)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
	return dir
}
