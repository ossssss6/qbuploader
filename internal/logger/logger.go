package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"qbuploader/internal/config"
)

// Log 是一个全局的、可供其他包使用的 logrus 日志实例。
var Log = logrus.New()

// textFormatter 是我们的纯文本格式化器，用于写入文件。
type textFormatter struct {
	TimestampFormat string
}

func (f *textFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	levelText := fmt.Sprintf("[%s]", strings.ToUpper(entry.Level.String()))

	// 为了对齐，处理多行
	messageLines := strings.Split(entry.Message, "\n")
	formattedMessage := fmt.Sprintf("[%s] %-7s %s", timestamp, levelText, messageLines[0])
	if len(messageLines) > 1 {
		indentation := len(timestamp) + 2 + 7 + 2 // 时间+括号+空格 + 级别+空格
		for _, line := range messageLines[1:] {
			formattedMessage += "\n" + strings.Repeat(" ", indentation) + line
		}
	}

	return []byte(formattedMessage + "\n"), nil
}

// colorFormatter 是我们的带颜色的格式化器，用于控制台输出。
type colorFormatter struct {
	TimestampFormat string
}

func (f *colorFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	levelText := fmt.Sprintf("[%s]", strings.ToUpper(entry.Level.String()))

	var levelColor *color.Color
	switch entry.Level {
	case logrus.InfoLevel:
		levelColor = color.New(color.FgHiBlue)
	case logrus.WarnLevel: // <<<--- 这里是修正的地方！
		levelColor = color.New(color.FgHiYellow)
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = color.New(color.FgHiRed)
	case logrus.DebugLevel:
		levelColor = color.New(color.FgHiWhite)
	default:
		levelColor = color.New(color.FgWhite)
	}

	coloredTimestamp := color.New(color.FgGreen).Sprint(fmt.Sprintf("[%s]", timestamp))
	coloredLevel := levelColor.Sprint(fmt.Sprintf("%-7s", levelText))

	messageLines := strings.Split(entry.Message, "\n")
	formattedMessage := coloredTimestamp + " " + coloredLevel + " " + messageLines[0]

	if len(messageLines) > 1 {
		indentation := len(timestamp) + 2 + 7 + 2
		for _, line := range messageLines[1:] {
			formattedMessage += "\n" + strings.Repeat(" ", indentation) + line
		}
	}

	return []byte(formattedMessage + "\n"), nil
}

// Init 函数是这个模块的唯一入口。
func Init() {
	// 1. 设置日志级别
	logLevelString := config.Cfg.General.LogLevel
	var level logrus.Level
	switch strings.ToLower(logLevelString) {
	case "debug":
		level = logrus.DebugLevel
	default:
		level = logrus.InfoLevel
	}
	Log.SetLevel(level)
	Log.SetReportCaller(false) // 关闭默认的调用者报告

	// 2. 配置日志文件轮转
	wd, err := os.Getwd()
	if err != nil {
		Log.SetFormatter(&colorFormatter{TimestampFormat: "2006-01-02 15:04:05"})
		Log.SetOutput(os.Stdout)
		Log.Errorf("无法获取当前工作目录，日志将只输出到控制台: %v", err)
		return
	}
	logFilePath := filepath.Join(wd, "qbuploader.log")

	fileLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    config.Cfg.Maintenance.LogMaxSizeMB,
		MaxBackups: config.Cfg.Maintenance.LogMaxBackups,
		LocalTime:  true,
		Compress:   false,
	}

	// 3. 设置主 Log 实例只输出到控制台，使用颜色格式化器
	Log.SetFormatter(&colorFormatter{TimestampFormat: "2006-01-02 15:04:05"})
	Log.SetOutput(os.Stdout)

	// 4. 使用 Hook 将纯文本日志写入文件
	Log.AddHook(&writerHook{
		Writer:    fileLogger,
		LogLevels: logrus.AllLevels,
		Formatter: &textFormatter{TimestampFormat: "2006-01-02 15:04:05"},
	})

	Log.Debug("日志模块初始化完成。")
}

// writerHook 是一个自定义的 hook，用于将纯文本日志写入文件
type writerHook struct {
	Writer    io.Writer
	LogLevels []logrus.Level
	Formatter logrus.Formatter
}

func (hook *writerHook) Fire(entry *logrus.Entry) error {
	line, err := hook.Formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = hook.Writer.Write(line)
	return err
}

func (hook *writerHook) Levels() []logrus.Level {
	return hook.LogLevels
}