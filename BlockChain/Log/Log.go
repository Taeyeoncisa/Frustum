package Log

import (
	"BlockChain/ID"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"math/rand"
	"time"
)

func LoggerInit(loglevel int, nodeId ID.NodeID) *zap.SugaredLogger {

	var level zap.AtomicLevel
	switch loglevel {
	case 0:
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case 1:
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	initialFields := make(map[string]interface{})
	initialFields["NodeID"] = nodeId.String()

	logConfig := zap.Config{
		Level:             level,
		Development:       true,
		DisableCaller:     true,
		DisableStacktrace: true,
		Sampling:          nil,
		Encoding:          "console",
		EncoderConfig:     zap.NewDevelopmentEncoderConfig(),
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
		InitialFields:     initialFields,
	}

	logger, _ := logConfig.Build()

	/*********************************************************/

	/*writeLog := true
	if nodeId.String() == "0-0" {
		writeLog = false
	}*/

	/*writeSyncer := getLogWriter(nodeId.String(), false)
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, level)
	logger := zap.New(core)*/

	return logger.Sugar()

}

func getEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
}

func getLogWriter(fileName string, intoFile bool) zapcore.WriteSyncer {

	if intoFile {
		file := "./" + fileName + ".log"
		//日志分割
		lumberJackLogger := &lumberjack.Logger{
			Filename:   file,
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   false,
		}

		return zapcore.AddSync(lumberJackLogger)
	} else {
		open, _, err := zap.Open([]string{"stderr"}...)
		if err != nil {
			return nil
		}
		return open
	}

}

// Random 根据区间产生随机数
func Random(min, max int) int {
	if min == max {
		return max
	} else {
		rand.Seed(time.Now().Unix())
		return rand.Intn(max-min) + min
	}
}
