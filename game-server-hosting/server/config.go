package server

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

type (
	// Config represents the game server configuration variables provided from the Unity Game Server Hosting platform.
	Config struct {
		// AllocatedUUID is the allocation ID provided to an event.
		AllocatedUUID string `json:"allocatedUUID"`

		// EnableBackfillStr is the string representation of whether the backfill feature in Unity Matchmaker is enabled.
		EnableBackfillStr string `json:"enableBackfill"`

		// IP is the IPv4 address of this server.
		IP string `json:"ip"`

		// IPV6 is the IPv6 address of this server. This can be empty.
		IPv6 string `json:"ipv6"`

		// LocalProxyURL is the URL to the local proxy service, which can handle interactions with the allocations payload store.
		LocalProxyURL string `json:"localProxyUrl"`

		// MatchmakerURL is the URL to the matchmaker service this server is using.
		MatchmakerURL string `json:"matchmakerUrl"`

		// Port is the port number this server uses for game interactions. It is up to the implemented to bind their game
		// server to this port.
		Port json.Number `json:"port"`

		// QueryPort is the port number this server uses for query interactions. These interactions are handled over UDP.
		QueryPort json.Number `json:"queryPort"`

		// QueryType represents the query protocol used by this server.
		QueryType QueryProtocol `json:"queryType"`

		// ServerID is the ID of the running server in the Unity Game Server Hosting platform.
		ServerID json.Number `json:"serverID"`

		// ServerLogDir is the directory where the server should place its log files. These will be detected by Unity Game Server
		// Hosting and made available in the dashboard.
		ServerLogDir string `json:"serverLogDir"`

		// Extra represents any other arguments passed to this server, for example, those specified in a build configuration.
		Extra map[string]string `json:"-"`
	}
)

// newConfigFromFile loads configuration from the specified file.
func newConfigFromFile(configFile string) (*Config, error) {
	var cfg *Config

	f, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	defer f.Close()

	// Decode known fields into struct.
	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	if _, err = f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("unable to seek: %w", err)
	}

	// Decode all other fields into Extra - this can include custom Build Configuration data.
	if err = json.NewDecoder(f).Decode(&cfg.Extra); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	// Remove already decoded fields.
	v := reflect.TypeOf(*cfg)
	for i := 0; i < v.NumField(); i++ {
		delete(cfg.Extra, v.Field(i).Tag.Get("json"))
	}

	// Set query type to 'sqp' if one is not defined.
	if cfg.QueryType == "" {
		cfg.QueryType = QueryProtocolSQP
	}

	// Set backfill to default value if one is not defined.
	if cfg.EnableBackfillStr == "" {
		cfg.EnableBackfillStr = "false"
	}

	if cfg.MatchmakerURL == "" {
		cfg.MatchmakerURL = "https://matchmaker.services.api.unity.com"
	}

	if cfg.LocalProxyURL == "" {
		cfg.LocalProxyURL = "http://localhost:8086"
	}

	return cfg, nil
}

// BackfillEnabled returns a boolean representation of the `enableBackfill` configuration item.
func (c Config) BackfillEnabled() bool {
	b, _ := strconv.ParseBool(c.EnableBackfillStr)
	return b
}
