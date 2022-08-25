package log

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/aichy126/igo/config"
	"github.com/gin-gonic/gin"
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

	err := InitLogger(Filename, Level, MaxSizeInt, MaxBackupsInt, MaxAgeInt)
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

func InitAccessLogger(Filename, Level string, MaxSize, MaxBackups, MaxAge int) *zap.Logger {
	cfg := new(LogConfig)
	cfg.Filename = Filename
	cfg.Level = Level
	cfg.MaxSize = 1
	cfg.MaxBackups = 5
	cfg.MaxAge = 7
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
func InitLogger(Filename, Level string, MaxSize, MaxBackups, MaxAge int) (err error) {
	cfg := new(LogConfig)
	cfg.Filename = Filename
	cfg.Level = Level
	cfg.MaxSize = 1
	cfg.MaxBackups = 5
	cfg.MaxAge = 7
	writeSyncer := getLogWriter(cfg.Filename, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge)
	encoder := getEncoder()
	var l = new(zapcore.Level)
	err = l.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return
	}
	core := zapcore.NewCore(encoder, writeSyncer, l)

	lg = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	zap.ReplaceGlobals(lg) // 替换zap包中全局的logger实例，后续在其他包中只需使用zap.L()调用即可
	logger := &Logger{
		l:     lg,
		level: LevelToNum(Level),
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
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
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

// GinLogger 接收gin框架默认的日志
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		cost := time.Since(start)
		lg.Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("cost", cost),
		)
	}
}

// GinRecovery recover掉项目可能出现的panic，并使用zap记录相关日志
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					lg.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					lg.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					lg.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
