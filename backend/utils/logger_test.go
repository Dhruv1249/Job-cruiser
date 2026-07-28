// Package utils_test provides unit tests for utilities including color logging.
package utils_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/Dhruv1249/Job-cruiser/backend/utils"
)

// TestColorizeLogLineSeverityTags verifies that ERROR, WARN, SUCCESS, and INFO tags are colorized properly.
func TestColorizeLogLineSeverityTags(t *testing.T) {
	errorLog := "CRITICAL ERROR: Connection to Postgres database failed"
	colorizedError := utils.ColorizeLogLine(errorLog)
	if !strings.Contains(colorizedError, utils.AnsiBoldRed) {
		t.Errorf("expected red ANSI code in error log, got: %s", colorizedError)
	}

	warnLog := "WARNING: NVIDIA_API_KEY is missing"
	colorizedWarn := utils.ColorizeLogLine(warnLog)
	if !strings.Contains(colorizedWarn, utils.AnsiBoldYellow) {
		t.Errorf("expected yellow ANSI code in warn log, got: %s", colorizedWarn)
	}

	successLog := "[SUCCESS] Successfully registered new user"
	colorizedSuccess := utils.ColorizeLogLine(successLog)
	if !strings.Contains(colorizedSuccess, utils.AnsiBoldGreen) {
		t.Errorf("expected green ANSI code in success log, got: %s", colorizedSuccess)
	}

	infoLog := "[INFO] Server listening on port 8080"
	colorizedInfo := utils.ColorizeLogLine(infoLog)
	if !strings.Contains(colorizedInfo, utils.AnsiBoldBlue) {
		t.Errorf("expected blue ANSI code in info log, got: %s", colorizedInfo)
	}
}

// TestColorizeLogLineWorkerAndServiceTags verifies that worker goroutine and service tags receive magenta and cyan colors.
func TestColorizeLogLineWorkerAndServiceTags(t *testing.T) {
	workerLog := "[Worker-3] Processing batch of 20 jobs..."
	colorizedWorker := utils.ColorizeLogLine(workerLog)
	if !strings.Contains(colorizedWorker, utils.AnsiBoldMagenta) {
		t.Errorf("expected magenta ANSI code for worker tag, got: %s", colorizedWorker)
	}

	serviceLog := "[NvidiaNimService] Initialized with model nemotron"
	colorizedService := utils.ColorizeLogLine(serviceLog)
	if !strings.Contains(colorizedService, utils.AnsiBoldCyan) {
		t.Errorf("expected cyan ANSI code for service tag, got: %s", colorizedService)
	}
}

// TestColorizeLogLineStreamTag verifies that STREAM log lines receive a high-contrast purple background badge.
func TestColorizeLogLineStreamTag(t *testing.T) {
	streamLog := "[STREAM] [Worker-1] First reasoning token received after 200ms"
	colorizedStream := utils.ColorizeLogLine(streamLog)
	if !strings.Contains(colorizedStream, utils.AnsiBgPurple) {
		t.Errorf("expected purple background ANSI badge for streaming log, got: %s", colorizedStream)
	}
}

// TestColorLogWriterOutputStream verifies that ColorLogWriter intercepts standard Go log calls and writes colorized bytes.
func TestColorLogWriterOutputStream(t *testing.T) {
	var buffer bytes.Buffer
	writer := utils.NewColorLogWriter(&buffer)

	originalOutput := log.Writer()
	log.SetOutput(writer)
	defer log.SetOutput(originalOutput)

	log.Println("[NvidiaNimService] Test service log line")

	writtenOutput := buffer.String()
	if !strings.Contains(writtenOutput, utils.AnsiBoldCyan) {
		t.Errorf("expected color log writer output to contain ANSI cyan code, got: %s", writtenOutput)
	}
}
