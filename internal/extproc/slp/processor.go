package slp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/kaz/pprotein/internal/collect"
)

type (
	processor struct {
		confPath string
	}
)

func (p *processor) Cacheable() bool {
	return true
}

func (p *processor) Process(snapshot *collect.Snapshot) (io.ReadCloser, error) {
	bodyPath, err := snapshot.BodyPath()
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshot body: %w", err)
	}

	logPath, format, err := prepareLog(bodyPath)
	if err != nil {
		return nil, err
	}
	if logPath != bodyPath {
		defer os.Remove(logPath)
	}

	cmd := exec.Command("slp", string(format), "--config", p.confPath, "--format", "tsv", "--file", logPath)

	res, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("external process aborted: %w", err)
	}

	return io.NopCloser(bytes.NewBuffer(res)), nil
}

// prepareLog は slow log の形式を判定し、slp に渡すファイルのパスを返す。
// PostgreSQL のログは slp が読める形に書き換えた一時ファイルに書き出すので、呼び出し側で削除する
func prepareLog(bodyPath string) (string, logFormat, error) {
	body, err := os.Open(bodyPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open snapshot body: %w", err)
	}
	defer body.Close()

	format, err := detectLogFormat(body)
	if err != nil {
		return "", "", fmt.Errorf("failed to detect slow log format: %w", err)
	}
	if format == logFormatMySQL {
		return bodyPath, format, nil
	}

	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return "", "", fmt.Errorf("failed to rewind snapshot body: %w", err)
	}

	normalized, err := os.CreateTemp("", "pprotein-pg-slowlog-*.log")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer normalized.Close()

	if err := normalizePostgreSQLLog(body, normalized); err != nil {
		os.Remove(normalized.Name())
		return "", "", fmt.Errorf("failed to normalize PostgreSQL log: %w", err)
	}
	return normalized.Name(), format, nil
}
