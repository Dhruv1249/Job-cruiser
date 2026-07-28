// Package utils provides helper utilities for string parsing, security tokens, salary extraction, and ANSI color logging.
package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	AnsiReset          = "\033[0m"
	AnsiBoldRed        = "\033[1;31m"
	AnsiBoldGreen      = "\033[1;32m"
	AnsiBoldYellow     = "\033[1;33m"
	AnsiBoldBlue       = "\033[1;34m"
	AnsiBoldMagenta    = "\033[1;35m"
	AnsiBoldCyan       = "\033[1;36m"
	AnsiBoldWhite      = "\033[1;37m"
	AnsiBgPurple       = "\033[45;1;37m"
	AnsiBgRed          = "\033[41;1;37m"
	AnsiBrightPurple   = "\033[1;95m"
)

// ColorLogWriter wraps an underlying io.Writer to inject ANSI color escape sequences into Go log outputs.
type ColorLogWriter struct {
	TargetWriter io.Writer
	writeMutex   sync.Mutex
}

// NewColorLogWriter initializes a ColorLogWriter wrapping the provided target stream.
func NewColorLogWriter(targetWriter io.Writer) *ColorLogWriter {
	return &ColorLogWriter{
		TargetWriter: targetWriter,
	}
}

// Write processes incoming log message bytes, injects ANSI color codes based on log level or service tags, and writes to target stream.
func (c *ColorLogWriter) Write(p []byte) (n int, err error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	rawLine := string(p)
	colorizedLine := ColorizeLogLine(rawLine)

	_, writeErr := c.TargetWriter.Write([]byte(colorizedLine))
	if writeErr != nil {
		return 0, writeErr
	}
	return len(p), nil
}

// ColorizeLogLine parses log line content and wraps tags, log levels, and worker prefixes with matching ANSI colors.
func ColorizeLogLine(line string) string {
	result := line

	if strings.Contains(line, "[STREAM]") || strings.Contains(line, "Streaming") || strings.Contains(line, "Live Progress") {
		result = replaceSeverityTag(result, "[STREAM]", AnsiBgPurple+" STREAM "+AnsiReset)
		result = strings.Replace(result, "Streaming", AnsiBrightPurple+"Streaming"+AnsiReset, 1)
		result = strings.Replace(result, "Live Progress:", AnsiBrightPurple+"Live Progress:"+AnsiReset, 1)
	}

	if strings.Contains(line, "JSON parse error") || strings.Contains(line, "Unmarshal error") || strings.Contains(line, "API call failed") || strings.Contains(line, "[ERROR]") || strings.Contains(line, "CRITICAL ERROR") {
		result = replaceSeverityTag(result, "[ERROR]", AnsiBgRed+" ERROR "+AnsiReset)
		result = strings.Replace(result, "JSON parse error", AnsiBgRed+" ERROR "+AnsiReset+" "+AnsiBoldRed+"JSON parse error"+AnsiReset, 1)
		result = strings.Replace(result, "Unmarshal error", AnsiBgRed+" ERROR "+AnsiReset+" "+AnsiBoldRed+"Unmarshal error"+AnsiReset, 1)
		result = strings.Replace(result, "API call failed", AnsiBgRed+" ERROR "+AnsiReset+" "+AnsiBoldRed+"API call failed"+AnsiReset, 1)
		result = replaceSeverityTag(result, "CRITICAL ERROR:", AnsiBgRed+" ERROR "+AnsiReset+" "+AnsiBoldRed+"CRITICAL ERROR:"+AnsiReset)
	}

	if strings.Contains(line, "WARNING") || strings.Contains(line, "[WARN]") {
		result = replaceSeverityTag(result, "[WARN]", AnsiBoldYellow)
		result = replaceSeverityTag(result, "WARNING:", AnsiBoldYellow+"WARNING:"+AnsiReset)
	}

	if strings.Contains(line, "[SUCCESS]") || strings.Contains(line, "Successfully") || strings.Contains(line, "Pass complete") {
		result = replaceSeverityTag(result, "[SUCCESS]", AnsiBoldGreen)
		result = strings.Replace(result, "Successfully", AnsiBoldGreen+"Successfully"+AnsiReset, 1)
		result = strings.Replace(result, "Pass complete", AnsiBoldGreen+"Pass complete"+AnsiReset, 1)
	}

	if strings.Contains(line, "[INFO]") {
		result = replaceSeverityTag(result, "[INFO]", AnsiBoldBlue)
	}

	workerRegex := regexp.MustCompile(`\[(Worker-\d+|NvidiaNimWorker-\d+)\]`)
	result = workerRegex.ReplaceAllStringFunc(result, func(match string) string {
		return AnsiBoldMagenta + match + AnsiReset
	})

	serviceRegex := regexp.MustCompile(`\[(NvidiaNimService|HybridMatcher|IngestHandler|AdminHandler|BackgroundMatcher|IngestRaw|NvidiaNimHTTP)\]`)
	result = serviceRegex.ReplaceAllStringFunc(result, func(match string) string {
		return AnsiBoldCyan + match + AnsiReset
	})

	return result
}

func replaceSeverityTag(source string, tag string, colorCode string) string {
	if !strings.Contains(source, tag) {
		return source
	}
	if strings.HasPrefix(colorCode, "\033") && strings.HasSuffix(colorCode, "\033[0m") {
		return strings.Replace(source, tag, colorCode, 1)
	}
	return strings.Replace(source, tag, colorCode+tag+AnsiReset, 1)
}

// InitColorLogger configures the global Go logger to write ANSI colorized output to stderr.
func InitColorLogger() {
	log.SetOutput(NewColorLogWriter(os.Stderr))
	log.SetFlags(log.Ldate | log.Ltime)
}

// LogInfo writes a formatted informational log message with blue ANSI status indicator.
func LogInfo(format string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(format, args...)
	log.Printf("%s[INFO]%s %s", AnsiBoldBlue, AnsiReset, formattedMsg)
}

// LogSuccess writes a formatted success log message with green ANSI status indicator.
func LogSuccess(format string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(format, args...)
	log.Printf("%s[SUCCESS]%s %s", AnsiBoldGreen, AnsiReset, formattedMsg)
}

// LogWarn writes a formatted warning log message with yellow ANSI status indicator.
func LogWarn(format string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(format, args...)
	log.Printf("%s[WARN]%s %s", AnsiBoldYellow, AnsiReset, formattedMsg)
}

// LogError writes a formatted error log message with red ANSI status indicator.
func LogError(format string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(format, args...)
	log.Printf("%s[ERROR]%s %s", AnsiBoldRed, AnsiReset, formattedMsg)
}

// LogWorker writes a worker thread log message with magenta ANSI status indicator.
func LogWorker(workerID int, format string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(format, args...)
	log.Printf("%s[WORKER-%d]%s %s", AnsiBoldMagenta, workerID, AnsiReset, formattedMsg)
}

var logFileMutex sync.Mutex

// ClearRawAILogFile truncates logs/nim_generation.log so each evaluation run starts with a clean log file.
func ClearRawAILogFile() {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	logDirPath := "logs"
	_ = os.MkdirAll(logDirPath, 0755)
	logFilePath := logDirPath + "/nim_generation.log"
	_ = os.Truncate(logFilePath, 0)
}

// LogRawAIResponse appends raw AI generation outputs to a dedicated debug log file (logs/nim_generation.log).
func LogRawAIResponse(callerTag string, modelName string, promptContent string, rawGeneratedOutput string, duration time.Duration, isError bool) {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	logDirPath := "logs"
	_ = os.MkdirAll(logDirPath, 0755)

	logFilePath := logDirPath + "/nim_generation.log"
	fileHandle, openErr := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if openErr != nil {
		return
	}
	defer fileHandle.Close()

	timestamp := time.Now().Format("2006/01/02 15:04:05")
	status := "SUCCESS"
	if isError {
		status = "ERROR"
	}

	separator := fmt.Sprintf(
		"================================================================================\n"+
			"[%s] TIMESTAMP: %s | TAG: %s | MODEL: %s | DURATION: %v\n"+
			"--------------------------------------------------------------------------------\n"+
			"%s\n"+
			"================================================================================\n\n",
		status, timestamp, callerTag, modelName, duration.Truncate(time.Millisecond), rawGeneratedOutput,
	)

	_, _ = fileHandle.WriteString(separator)
}
