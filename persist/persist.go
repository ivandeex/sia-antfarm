package persist

import (
	"fmt"

	"gitlab.com/NebulousLabs/Sia-Ant-Farm/build"
	"gitlab.com/NebulousLabs/log"
)

// Logger is a wrapper for log.Logger.
type Logger struct {
	logger *log.Logger
}

// LogCaller defines a type for callers of the log printing
type LogCaller string

// logLevel defines log level type
type logLevel string

// Log callers
var (
	// LogCallerAnt defines string id for ant logs
	LogCallerAnt LogCaller = "ant"

	// LogCallerAntBalanceMaintainer defines string id for ant's
	// balanceMaintainer logs
	LogCallerAntBalanceMaintainer LogCaller = "ant > balanceMaintainer"

	// LogCallerAntBigSpender defines string id for ant's bigSpender logs
	LogCallerAntBigSpender LogCaller = "ant > bigSpender"

	// LogCallerAntGateway defines string id for ant's gateway logs
	LogCallerAntGateway LogCaller = "ant > gateway"

	// LogCallerAntHost defines string id for ant's host logs
	LogCallerAntHost LogCaller = "ant > host"

	// LogCallerAntLittleSupplier defines string id for ant's littleSupplier
	// logs
	LogCallerAntLittleSupplier LogCaller = "ant > littleSupplier"

	// LogCallerAntMiner defines string id for ant's miner logs
	LogCallerAntMiner LogCaller = "ant > miner"

	// LogCallerAntRenter defines id for ant's renter job logs
	LogCallerAntRenter LogCaller = "ant > renter"

	// LogCallerAntfarm defines string id for antfarm logs
	LogCallerAntfarm LogCaller = "antfarm"

	// LogCallerBuildBinaries defines string id for build binaries logs
	LogCallerBuildBinaries LogCaller = "buildBinaries"

	// LogCallerTest defines string id for antfarm test logs
	LogCallerTest LogCaller = "test"

	// LogCallerUPnPRouter defines string id for UPnP router logs
	LogCallerUPnPRouter LogCaller = "upnpRouter"
)

// Log levels
var (
	// LogLevelDebug defines debug log level
	LogLevelDebug logLevel = "DEBUG"

	// LogLevelError defines error log level
	LogLevelError logLevel = "ERROR"

	// LogLevelInfo defines info log level
	LogLevelInfo logLevel = "INFO"
)

var (
	// options contains log options with Sia Antfarm- and build-specific
	// information.
	options = log.Options{
		BinaryName:   "Sia Antfarm",
		BugReportURL: build.IssuesURL,
		Debug:        build.DEBUG,
		Release:      buildReleaseType(),
		Version:      build.Version,
	}
)

// NewFileLogger returns a logger that logs to logFilename. The file is opened
// in append mode, and created if it does not exist.
func NewFileLogger(logFilename string) (*Logger, error) {
	logger, err := log.NewFileLogger(logFilename, options)
	return &Logger{logger}, err
}

// buildReleaseType returns the release type for this build, defaulting to
// Release.
func buildReleaseType() log.ReleaseType {
	switch build.Release {
	case "standard":
		return log.Release
	case "dev":
		return log.Dev
	case "testing":
		return log.Testing
	default:
		return log.Release
	}
}

// Println adds a log message to the logger
func (l *Logger) Println(logLevel logLevel, logCaller LogCaller, callerDataDir, msg string) {
	// Generate formated string from msg options
	formatedString := fmt.Sprintf("%v %v %v: %v", logLevel, logCaller, callerDataDir, msg)
	l.logger.Logger.Println(formatedString)
}
