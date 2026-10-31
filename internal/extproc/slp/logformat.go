package slp

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// logFormat は slow log の形式で、値は slp のサブコマンド名
type logFormat string

const (
	logFormatMySQL      logFormat = "my"
	logFormatPostgreSQL logFormat = "pg"
)

var (
	// log_min_duration_statement で出る PostgreSQL のログの行
	pgDurationRe = regexp.MustCompile(`LOG:\s+duration:\s+[0-9.]+\s+ms\s+`)
	// 拡張問い合わせプロトコルの実行の行。slp は statement: の行しか読まないので書き換える
	pgExecuteRe = regexp.MustCompile(`(LOG:\s+duration:\s+[0-9.]+\s+ms\s+)execute [^:]*:`)
)

// detectLogFormat は最初に見つかった判別できる行から形式を決める。
// 判別できる行が無ければ、従来どおり MySQL として扱う
func detectLogFormat(r io.Reader) (logFormat, error) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if strings.HasPrefix(line, "# Query_time:") {
			return logFormatMySQL, nil
		}
		if pgDurationRe.MatchString(line) {
			return logFormatPostgreSQL, nil
		}
		if err == io.EOF {
			return logFormatMySQL, nil
		}
		if err != nil {
			return "", err
		}
	}
}

// normalizePostgreSQLLog は execute の行を statement の行に書き換えて w に書き出す。
// parse と bind の行は slp が読まずに捨てるので、実行時間は execute の分だけが集計される
func normalizePostgreSQLLog(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(w)
	for {
		line, err := br.ReadString('\n')
		if _, werr := bw.WriteString(pgExecuteRe.ReplaceAllString(line, "${1}statement:")); werr != nil {
			return werr
		}
		if err == io.EOF {
			return bw.Flush()
		}
		if err != nil {
			return err
		}
	}
}
