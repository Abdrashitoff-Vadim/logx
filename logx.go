package logx

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorWhite  = "\033[97m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	bold        = "\033[1m"

	lvlPanic = slog.Level(12)
	lvlFatal = slog.Level(16)
)

func customLevelToString(level slog.Level) string {
	switch level {
	case lvlPanic:
		return "PANIC"
	case lvlFatal:
		return "FATAL"
	default:
		return level.String()
	}
}

func colorByLevel(level slog.Level) string {
	switch level {
	case slog.LevelInfo:
		return colorWhite
	case slog.LevelDebug:
		return colorBlue
	case slog.LevelWarn:
		return colorYellow
	default:
		return colorRed
	}
}

func isInternalPackage(name string) bool {
	// Фильтруем вызовы из slog, runtime и логгера
	return strings.Contains(name, "log/slog") ||
		strings.Contains(name, "runtime.") ||
		strings.Contains(name, "CustomHandler") ||
		strings.Contains(name, "slog.New") ||
		strings.Contains(name, "logx.")
}

type Logger struct {
	*slog.Logger
}

type CustomHandler struct {
	Logger
	//slog.Handler
}

type FilePathMode int

const (
	Absent FilePathMode = iota
	Full
	Short
)

type Level slog.Level

const (
	LevelDebug Level = Level(slog.LevelDebug)
	LevelInfo  Level = Level(slog.LevelInfo)
	LevelWarn  Level = Level(slog.LevelWarn)
	LevelError Level = Level(slog.LevelError)
	LevelPanic Level = Level(lvlPanic)
	LevelFatal Level = Level(lvlFatal)
)

func (c *CustomHandler) Handle(_ context.Context, record slog.Record) error {
	formattedTime := record.Time.Format("15:04:05")

	var attrs string

	record.Attrs(func(a slog.Attr) bool {
		if a.Key == "!BADKEY" {
			attrs += fmt.Sprintf(" %v", a.Value) // извлекаем напрямую
		} else {
			attrs += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		}
		return true
	})

	if attrs == " []" {
		attrs = ""
	}

	var file string = "unknown"
	var line int
	for i := 2; i < 15; i++ { // Пробуем 10-15 уровней вверх
		pc, f, l, ok := runtime.Caller(i)
		if !ok {
			continue
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		funcName := fn.Name()
		if !isInternalPackage(funcName) {
			file = f
			line = l
			break
		}
	}

	if !isInternalPackage(file) && filePathMode == Short {
		file = filepath.Base(file) // <-- здесь берем только имя файла
	}

	msg := fmt.Sprintf("%s%s%s [%s:%d][%s]:%s %s%s%s\n", bold, colorWhite, formattedTime, file, line,
		customLevelToString(record.Level), colorByLevel(record.Level), record.Message, attrs, colorReset)

	if filePathMode == Absent {
		msg = fmt.Sprintf("%s%s%s [%s]:%s %s%s%s\n", bold, colorWhite, formattedTime,
			customLevelToString(record.Level), colorByLevel(record.Level), record.Message, attrs, colorReset)
	}

	_, err := fmt.Fprintf(os.Stderr, msg)
	return err
}

func (c *CustomHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= logLevel
}

func (c *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return c
}

func (c *CustomHandler) WithGroup(name string) slog.Handler {
	return c
}

func (l Logger) Panic(msg string, arg ...any) {
	if logLevel <= lvlPanic {
		l.Log(context.Background(), lvlPanic, msg, arg)
		panic("")
	}
}

func (l Logger) Fatal(msg string, arg ...any) {
	l.Log(context.Background(), lvlFatal, msg, arg)
	os.Exit(1)
}

var (
	logLevel     = slog.LevelDebug
	globalLogger *slog.Logger
	once         sync.Once
	filePathMode FilePathMode = Short
)

func initLogger() {
	globalLogger = slog.New(&CustomHandler{})
	slog.SetDefault(globalLogger) // опционально, если используешь slog.* напрямую
}

func getLogger() *slog.Logger {
	once.Do(initLogger)
	return globalLogger
}

// Debug Выводит сообщения уровня -4. По умолчанию использует синий цвет
func Debug(msg string, args ...any) {
	getLogger().Debug(msg, args)
}

// Info Выводит сообщения уровня 0. По умолчанию использует белый цвет
func Info(msg string, args ...any) {
	getLogger().Info(msg, args)
}

// Warn Выводит сообщения уровня 4. По умолчанию использует желтый цвет
func Warn(msg string, args ...any) {
	getLogger().Warn(msg, args)
}

// Error Выводит сообщения уровня 8. По умолчанию использует красный цвет
func Error(msg string, args ...any) {
	getLogger().Error(msg, args)
}

// Panic Выводит сообщения уровня 12. По умолчанию использует красный цвет, а также завершает программу с паникой
func Panic(msg string, args ...any) {
	Logger{getLogger()}.Panic(msg, args)
}

// Fatal Выводит сообщения уровня 16. По умолчанию использует красный цвет, а также экстренно завершает программу
func Fatal(msg string, args ...any) {
	Logger{getLogger()}.Fatal(msg, args)
}

// SetLevel Позволяет настраивать работу команд. Команды ниже установленного уровня просто не будут работать, отображаться. По умолчанию logx.LevelDebug
func SetLevel(level Level) {
	logLevel = slog.Level(level)
}

// SetPathMode Позволяет настраивать отображение вывода директории. По умолчанию logx.Short
func SetPathMode(mode FilePathMode) {
	filePathMode = mode
}
