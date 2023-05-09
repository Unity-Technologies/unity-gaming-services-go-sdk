package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewConfigFromFile_defaults(t *testing.T) {
	f := filepath.Join(t.TempDir(), "new-config-from-file")
	require.NoError(t,
		os.WriteFile(f, []byte(
			`{
				"allocatedUUID": "a-uuid",
				"ip": "127.0.0.1",
				"ipv6": "::1",
				"port": "9000",
				"queryPort": "9010",
				"serverID": "1234",
				"serverLogDir": "1234/logs",
				"a": "b"
			}`,
		),
			0600,
		),
	)

	cfg, err := newConfigFromFile(f)
	require.NoError(t, err)
	require.Equal(t, &Config{
		AllocatedUUID: "a-uuid",
		IP:            "127.0.0.1",
		IPv6:          "::1",
		LocalProxyURL: "http://localhost:8086",
		Port:          "9000",
		QueryPort:     "9010",
		QueryType:     QueryProtocolSQP,
		ServerID:      "1234",
		ServerLogDir:  "1234/logs",
		Extra: map[string]string{
			"a": "b",
		},
	}, cfg)
}

func Test_NewConfigFromFile_supported_values(t *testing.T) {
	f := filepath.Join(t.TempDir(), "new-config-from-file")
	require.NoError(t,
		os.WriteFile(f, []byte(
			`{
				"allocatedUUID": "a-uuid",
				"ip": "127.0.0.1",
				"ipv6": "::1",
				"localProxyUrl": "http://my-localproxy",
				"port": "9000",
				"queryPort": "9010",
				"serverID": "1234",
				"serverLogDir": "1234/logs"
			}`,
		),
			0600,
		),
	)

	cfg, err := newConfigFromFile(f)
	require.NoError(t, err)
	require.Equal(t, &Config{
		AllocatedUUID: "a-uuid",
		IP:            "127.0.0.1",
		IPv6:          "::1",
		LocalProxyURL: "http://my-localproxy",
		Port:          "9000",
		QueryPort:     "9010",
		QueryType:     QueryProtocolSQP,
		ServerID:      "1234",
		ServerLogDir:  "1234/logs",
		Extra:         map[string]string{},
	}, cfg)
}
