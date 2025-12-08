package logger

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "runtime"
    "sync"
    "time"
)

type Level int

const (
    DEBUG Level = iota
    INFO
    WARN
    ERROR
    FATAL
)

func (l Level) String() string {
    switch l {
    case DEBUG:
        return "DEBUG"
    case INFO:
        return "INFO"
    case WARN:
        return "WARN"
    case ERROR:
        return "ERROR"
    case FATAL:
        return "FATAL"
    default:
        return "UNKNOWN"
    }
}

type Logger struct {
    mu       sync.Mutex
    level    Level
    outputs  []io.Writer
    file     *os.File
    filePath string
}

var (
    defaultLogger *Logger
    once          sync.Once
)

type Config struct {
    Level       string // DEBUG, INFO, WARN, ERROR
    FilePath    string // Optional: path to log file
    Console     bool   // Output to console (stdout)
    MaxSizeMB   int    // Max size before rotation (0 = no limit)
}

func Init(cfg Config) error {
    var err error
    once.Do(func() {
        defaultLogger, err = newLogger(cfg)
    })
    return err
}

func newLogger(cfg Config) (*Logger, error) {
    l := &Logger{
        level:   parseLevel(cfg.Level),
        outputs: make([]io.Writer, 0),
    }

    if cfg.Console {
        l.outputs = append(l.outputs, os.Stdout)
    }

    if cfg.FilePath != "" {
        dir := filepath.Dir(cfg.FilePath)
        if err := os.MkdirAll(dir, 0755); err != nil {
            return nil, fmt.Errorf("failed to create log directory: %w", err)
        }

        file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
        if err != nil {
            return nil, fmt.Errorf("failed to open log file: %w", err)
        }

        l.file = file
        l.filePath = cfg.FilePath
        l.outputs = append(l.outputs, file)
    }

    if len(l.outputs) == 0 {
        l.outputs = append(l.outputs, os.Stdout)
    }

    return l, nil
}

func parseLevel(s string) Level {
    switch s {
    case "DEBUG":
        return DEBUG
    case "INFO":
        return INFO
    case "WARN":
        return WARN
    case "ERROR":
        return ERROR
    case "FATAL":
        return FATAL
    default:
        return INFO
    }
}

func GetLogger() *Logger {
    if defaultLogger == nil {
        defaultLogger = &Logger{
            level:   INFO,
            outputs: []io.Writer{os.Stdout},
        }
    }
    return defaultLogger
}

func (l *Logger) Close() error {
    l.mu.Lock()
    defer l.mu.Unlock()

    if l.file != nil {
        return l.file.Close()
    }
    return nil
}

func (l *Logger) log(level Level, component, message string, fields map[string]interface{}) {
    if level < l.level {
        return
    }

    l.mu.Lock()
    defer l.mu.Unlock()

    now := time.Now().In(getJakartaLocation())

    _, file, line, ok := runtime.Caller(2)
    caller := "unknown"
    if ok {
        caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
    }

    logLine := fmt.Sprintf("[%s] [%-5s] [%-12s] [%s] %s",
        now.Format("2006-01-02 15:04:05.000"),
        level.String(),
        component,
        caller,
        message,
    )

    if len(fields) > 0 {
        logLine += " |"
        for k, v := range fields {
            logLine += fmt.Sprintf(" %s=%v", k, v)
        }
    }

    logLine += "\n"

    for _, w := range l.outputs {
        w.Write([]byte(logLine))
    }
}

func getJakartaLocation() *time.Location {
    loc, err := time.LoadLocation("Asia/Jakarta")
    if err != nil {
        return time.FixedZone("WIB", 7*60*60)
    }
    return loc
}

func (l *Logger) Debug(component, message string, fields ...map[string]interface{}) {
    f := mergeFields(fields)
    l.log(DEBUG, component, message, f)
}

func (l *Logger) Info(component, message string, fields ...map[string]interface{}) {
    f := mergeFields(fields)
    l.log(INFO, component, message, f)
}

func (l *Logger) Warn(component, message string, fields ...map[string]interface{}) {
    f := mergeFields(fields)
    l.log(WARN, component, message, f)
}

func (l *Logger) Error(component, message string, fields ...map[string]interface{}) {
    f := mergeFields(fields)
    l.log(ERROR, component, message, f)
}

func (l *Logger) Fatal(component, message string, fields ...map[string]interface{}) {
    f := mergeFields(fields)
    l.log(FATAL, component, message, f)
    os.Exit(1)
}

func (l *Logger) ErrorWithStack(component, message string, err error, fields ...map[string]interface{}) {
    f := mergeFields(fields)
    if f == nil {
        f = make(map[string]interface{})
    }
    f["error"] = err.Error()
    f["stack"] = getStackTrace()
    l.log(ERROR, component, message, f)
}

func mergeFields(fields []map[string]interface{}) map[string]interface{} {
    if len(fields) == 0 {
        return nil
    }
    result := make(map[string]interface{})
    for _, f := range fields {
        for k, v := range f {
            result[k] = v
        }
    }
    return result
}

func getStackTrace() string {
    buf := make([]byte, 4096)
    n := runtime.Stack(buf, false)
    return string(buf[:n])
}


func Debug(component, message string, fields ...map[string]interface{}) {
    GetLogger().Debug(component, message, fields...)
}

func Info(component, message string, fields ...map[string]interface{}) {
    GetLogger().Info(component, message, fields...)
}

func Warn(component, message string, fields ...map[string]interface{}) {
    GetLogger().Warn(component, message, fields...)
}

func Error(component, message string, fields ...map[string]interface{}) {
    GetLogger().Error(component, message, fields...)
}

func Fatal(component, message string, fields ...map[string]interface{}) {
    GetLogger().Fatal(component, message, fields...)
}

func ErrorWithStack(component, message string, err error, fields ...map[string]interface{}) {
    GetLogger().ErrorWithStack(component, message, err, fields...)
}

func Close() error {
    return GetLogger().Close()
}

func F(key string, value interface{}) map[string]interface{} {
    return map[string]interface{}{key: value}
}

func Fields(pairs ...interface{}) map[string]interface{} {
    result := make(map[string]interface{})
    for i := 0; i < len(pairs)-1; i += 2 {
        key, ok := pairs[i].(string)
        if ok {
            result[key] = pairs[i+1]
        }
    }
    return result
}