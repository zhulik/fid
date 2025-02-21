package main

import (
	"os"

	"github.com/samber/do"
	"github.com/sirupsen/logrus"
	"github.com/zhulik/fid/internal/di"
)

const (
	DefaultFileName = "functions.yaml"
)

func main() {
	injector := di.New()
	logger := do.MustInvoke[logrus.FieldLogger](injector).WithField("component", "main")

	fileName := DefaultFileName
	if len(os.Args) > 1 {
		fileName = os.Args[1]
	}

	logger.Info("Starting...")
	logger.Infof("Loading %s...", fileName)

	functions, err := ParseFile(fileName)
	if err != nil {
		logger.Fatalf("error: %v", err)
	}

	logger.Printf("Parsed YAML: %+v\n", *(functions["demo_function"]))
	// Reads functions.yaml
	// Create or update all necessary JetStream resources
	// Start gateway
	// Start scaler per function
	// Wait until they are healhy
	// Exit
}
