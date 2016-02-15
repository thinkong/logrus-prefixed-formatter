package prefixed

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/mgutz/ansi"
	"github.com/thinkong/color"
)

const reset = ansi.Reset

var (
	baseTimestamp time.Time
	isTerminal    bool
)

func init() {
	baseTimestamp = time.Now()
	isTerminal = logrus.IsTerminal()
}

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

type TextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors.
	DisableColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging of just the time passed since beginning of execution.
	ShortTimestamp bool

	// Timestamp format to use for display when a full timestamp is printed.
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool
}

func (f *TextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		if k != "prefix" {
			keys = append(keys, k)
		}
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}

	b := &bytes.Buffer{}

	prefixFieldClashes(entry.Data)

	isColorTerminal := isTerminal && (runtime.GOOS != "windows")
	isColored := (f.ForceColors || isColorTerminal) && !f.DisableColors

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = time.Stamp
	}
	if isColored {
		f.printColored(b, entry, keys, timestampFormat)
	} else {
		if !f.DisableTimestamp {
			f.appendKeyValue(b, "time", entry.Time.Format(timestampFormat))
		}
		f.appendKeyValue(b, "level", entry.Level.String())
		if entry.Message != "" {
			f.appendKeyValue(b, "msg", entry.Message)
		}
		for _, key := range keys {
			f.appendKeyValue(b, key, entry.Data[key])
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *TextFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, timestampFormat string) {
	levelColor := color.White
	var levelText string
	var debugInf string
	switch entry.Level {
	case logrus.InfoLevel:
		//levelColor = ansi.Green
		levelColor = color.Green
	case logrus.WarnLevel:
		//levelColor = ansi.Yellow
		levelColor = color.Yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		//levelColor = ansi.Red
	levelColor = color.Red
	case logrus.DebugLevel:
		pc, file, line,_ := runtime.Caller(6)
       
		callername := runtime.FuncForPC(pc).Name()
		debugInf = fmt.Sprintf("[%s][%s][%d]", callername, file, line)
		fallthrough
	default:
		//levelColor = ansi.Blue
	levelColor = color.Blue
	}

	if entry.Level != logrus.WarnLevel {
		levelText = strings.ToUpper(entry.Level.String())
	} else {
		levelText = "WARN"
	}

	prefix := ""
	prefixValue, ok := entry.Data["prefix"]
	if ok {
		prefix = fmt.Sprint(" ", ansi.Cyan, prefixValue, ":", reset)
	}

	if f.ShortTimestamp {
		//fmt.Fprintf(b, "%s[%04d]%s %s%+5s%s%s %s", ansi.LightBlack, miniTS(), reset, levelColor, levelText, reset, prefix, entry.Message)
		s := fmt.Sprintf("%+5s [%04d] %s %s", levelText, ansi.LightBlack, miniTS(), prefix, entry.Message)
		levelColor(s)
	} else {
		s := fmt.Sprintf("%+5s [%s] %s %s %s", levelText, entry.Time.Format(timestampFormat), debugInf, prefix, entry.Message)
		levelColor(s)
	}
	for _, k := range keys {
		v := entry.Data[k]
		s := fmt.Sprintf(" %s=%+v",  k, v)
		levelColor(s)
	}
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func (f *TextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	b.WriteString(key)
	b.WriteByte('=')

	switch value := value.(type) {
	case string:
		if needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	default:
		fmt.Fprint(b, value)
	}

	b.WriteByte(' ')
}

func prefixFieldClashes(data logrus.Fields) {
	_, ok := data["time"]
	if ok {
		data["fields.time"] = data["time"]
	}
	_, ok = data["msg"]
	if ok {
		data["fields.msg"] = data["msg"]
	}
	_, ok = data["level"]
	if ok {
		data["fields.level"] = data["level"]
	}
}
