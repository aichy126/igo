package log

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aichy126/igo/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Log
type Log struct {
}

// loggerConf 日志配置(从 local.logger 读取,带默认值)
type loggerConf struct {
	Dir        string
	Name       string
	Level      string
	MaxSize    int // 每个日志文件保存的最大尺寸 单位：MB
	MaxBackups int // 日志文件最多保存多少个备份
	MaxAge     int // 文件最多保存多少天
	Debug      bool
	Access     bool
}

// readLoggerConf 从配置中读取日志配置并填充默认值
func readLoggerConf(conf *config.Config) loggerConf {
	lc := loggerConf{
		Dir:        conf.GetString("local.logger.dir"),
		Name:       conf.GetString("local.logger.name"),
		Level:      conf.GetString("local.logger.level"),
		MaxSize:    conf.GetInt("local.logger.max_size"),
		MaxBackups: conf.GetInt("local.logger.max_backups"),
		MaxAge:     conf.GetInt("local.logger.max_age"),
		Debug:      conf.GetBool("local.debug"),
		Access:     conf.GetBool("local.logger.access"),
	}
	if lc.Dir == "" {
		lc.Dir = "./logs"
	}
	if lc.Name == "" {
		lc.Name = "log.log"
	}
	if lc.MaxSize <= 0 {
		lc.MaxSize = 100
	}
	if lc.MaxBackups <= 0 {
		lc.MaxBackups = 5
	}
	if lc.MaxAge <= 0 {
		lc.MaxAge = 7
	}
	return lc
}

// NewLog
func NewLog(conf *config.Config) (*Log, error) {
	log := new(Log)
	lc := readLoggerConf(conf)
	filename := fmt.Sprintf("%s/%s", lc.Dir, lc.Name)
	err := InitLogger(filename, lc.Level, lc.MaxSize, lc.MaxBackups, lc.MaxAge, lc.Debug)
	return log, err
}

// NewAccessLogger 从配置创建 access 日志 logger(独立实例,不影响全局 logger)
func NewAccessLogger(conf *config.Config) *zap.Logger {
	lc := readLoggerConf(conf)
	filename := fmt.Sprintf("%s/access.log", lc.Dir)
	return InitAccessLogger(filename, lc.Level, lc.MaxSize, lc.MaxBackups, lc.MaxAge)
}

var lg *zap.Logger

type LogConfig struct {
	Level      string `json:"level"`
	Filename   string `json:"filename"`
	MaxSize    int    `json:"maxsize"`
	MaxAge     int    `json:"max_age"`
	MaxBackups int    `json:"max_backups"`
}

var (
	std   *Logger
	stdMu sync.RWMutex
)

// active 返回当前生效的 Logger:未初始化时降级为控制台 logger,保证任何时候调用都不 panic
func active() *Logger {
	stdMu.RLock()
	l := std
	stdMu.RUnlock()
	if l != nil {
		return l
	}

	stdMu.Lock()
	defer stdMu.Unlock()
	if std == nil {
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), zapcore.InfoLevel)
		std = newLogger(zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)), InfoLevel)
	}
	return std
}

// setStd 替换全局 Logger
func setStd(l *Logger) {
	stdMu.Lock()
	std = l
	stdMu.Unlock()
}

// InitAccessLogger 创建 access 日志 logger(独立实例,不修改全局 logger)
func InitAccessLogger(filename, level string, maxSize, maxBackups, maxAge int) *zap.Logger {
	writeSyncer := getLogWriter(filename, maxSize, maxBackups, maxAge)
	encoder := getEncoder()
	var l = new(zapcore.Level)
	err := l.UnmarshalText([]byte(level))
	if err != nil {
		*l = zapcore.DebugLevel
	}
	core := zapcore.NewCore(encoder, writeSyncer, l)
	return zap.New(core)
}

// InitLogger 初始化全局 Logger
func InitLogger(filename, level string, maxSize, maxBackups, maxAge int, debug bool) (err error) {
	writeSyncer := getLogWriter(filename, maxSize, maxBackups, maxAge)
	encoder := getEncoder()
	var l = new(zapcore.Level)
	err = l.UnmarshalText([]byte(level))
	if err != nil {
		return fmt.Errorf("无效的日志级别 %q: %w", level, err)
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
	setStd(newLogger(lg, LevelToNum(level)))
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
		MaxSize:    maxSize,   // MB
		MaxBackups: maxBackup, // 最多保留的备份文件数
		MaxAge:     maxAge,    // 天
	}
	return zapcore.AddSync(lumberJackLogger)
}
