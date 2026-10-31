package slp

import (
	"bytes"
	"strings"
	"testing"
)

func TestDetectLogFormat(t *testing.T) {
	tests := []struct {
		name string
		log  string
		want logFormat
	}{
		{
			name: "MySQL",
			log:  "# Time: 2026-10-31T00:00:00.000000Z\n# Query_time: 0.000123  Lock_time: 0.000000 Rows_sent: 1  Rows_examined: 1\nSELECT 1;\n",
			want: logFormatMySQL,
		},
		{
			name: "PostgreSQL",
			log:  "2026-10-31 00:00:00.000 UTC [123] LOG:  checkpoint starting: time\n2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.123 ms  statement: SELECT 1\n",
			want: logFormatPostgreSQL,
		},
		{
			name: "空のログは MySQL として扱う",
			log:  "",
			want: logFormatMySQL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectLogFormat(strings.NewReader(tt.log))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizePostgreSQLLog(t *testing.T) {
	log := strings.Join([]string{
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.050 ms  parse stmtcache_1: SELECT * FROM users WHERE id = $1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.020 ms  bind stmtcache_1: SELECT * FROM users WHERE id = $1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.300 ms  execute stmtcache_1: SELECT * FROM users WHERE id = $1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.400 ms  execute <unnamed>: SELECT 1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.500 ms  statement: SELECT 2",
	}, "\n")
	want := strings.Join([]string{
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.050 ms  parse stmtcache_1: SELECT * FROM users WHERE id = $1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.020 ms  bind stmtcache_1: SELECT * FROM users WHERE id = $1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.300 ms  statement: SELECT * FROM users WHERE id = $1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.400 ms  statement: SELECT 1",
		"2026-10-31 00:00:00.000 UTC [456] LOG:  duration: 0.500 ms  statement: SELECT 2",
	}, "\n")

	var got bytes.Buffer
	if err := normalizePostgreSQLLog(strings.NewReader(log), &got); err != nil {
		t.Fatal(err)
	}
	if got.String() != want {
		t.Errorf("got\n%s\nwant\n%s", got.String(), want)
	}
}
