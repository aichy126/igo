package log

import (
	"fmt"
	"os"
	"time"

	"github.com/aichy126/igo/config"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log
type Log struct {
}

// NewLog
func NewLog(conf *config.Config) (*Log, error) {
	log := new(Log)
	//log
	Filename := fmt.Sprintf("%s/%s", conf.GetString("local.logger.dir"), conf.GetString("local.logger.name"))
	Level := conf.GetString("local.logger.level")
	MaxSize := conf.GetInt("local.logger.max_size")
	MaxSizeInt := 1 //每个日志文件保存的最大尺寸 单位：M
	if MaxSize > 0 {
		MaxSizeInt = MaxSize
	}

	MaxBackups := conf.GetInt("local.logger.max_backups")
	MaxBackupsInt := 5 //文件最多保存多少天
	if MaxBackups > 0 {
		MaxBackupsInt = MaxBackups
	}

	MaxAge := conf.GetInt("local.logger.max_age")
	MaxAgeInt := 7 //日志文件最多保存多少个备份
	if MaxAge > 0 {
		MaxAgeInt = MaxAge
	}
	debug := conf.GetBool("local.debug")
	err := InitLogger(Filename, Level, MaxSizeInt, MaxBackupsInt, MaxAgeInt, debug)
	if err != nil {
		return log, err
	}

	return log, err
}

var lg *zap.Logger

type LogConfig struct {
	Level      string `json:"level"`
	Filename   string `json:"filename"`
	MaxSize    int    `json:"maxsize"`
	MaxAge     int    `json:"max_age"`
	MaxBackups int    `json:"max_backups"`
}

var std *Logger

func InitAccessLogger(filename, level string, maxSize, maxBackups, maxAge int) *zap.Logger {
	cfg := new(LogConfig)
	cfg.Filename = filename
	cfg.Level = level
	cfg.MaxSize = maxSize
	cfg.MaxBackups = maxBackups
	cfg.MaxAge = maxAge
	writeSyncer := getLogWriter(cfg.Filename, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge)
	encoder := getEncoder()
	var l = new(zapcore.Level)
	err := l.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		*l = zapcore.DebugLevel
	}
	core := zapcore.NewCore(encoder, writeSyncer, l)

	lg = zap.New(core)
	return lg
}

// InitLogger 初始化Logger
func InitLogger(filename, level string, maxSize, maxBackups, maxAge int, debug bool) (err error) {
	cfg := new(LogConfig)
	cfg.Filename = filename
	cfg.Level = level
	cfg.MaxSize = maxSize
	cfg.MaxBackups = maxBackups
	cfg.MaxAge = maxAge
	writeSyncer := getLogWriter(cfg.Filename, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge)
	encoder := getEncoder()
	var l = new(zapcore.Level)
	err = l.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return
	}
	var baseCore zapcore.Core
	if debug {
		//输出到日志和控制台
		baseCore = zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writeSyncer, zapcore.AddSync(os.Stdout)), l)
	} else {
		//只输出到日志
		baseCore = zapcore.NewCore(encoder, writeSyncer, l)
	}

	// 包装为hookCore，支持日志钩子
	hookCoreInstance = newHookCore(baseCore)

	lg = zap.New(hookCoreInstance, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(lg) // 替换zap包中全局的logger实例，后续在其他包中只需使用zap.L()调用即可
	logger := &Logger{
		l:     lg,
		level: LevelToNum(level),
	}
	std = logger
	Info = std.Info
	Warn = std.Warn
	Error = std.Error
	DPanic = std.DPanic
	Panic = std.Panic
	Fatal = std.Fatal
	Debug = std.Debug
	return
}

func getEncoder() zapcore.Encoder {

	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.EncodeTime = customTimeEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter(filename string, maxSize, maxBackup, maxAge int) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackup,
		MaxAge:     maxAge,
	}
	return zapcore.AddSync(lumberJackLogger)
}
