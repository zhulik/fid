package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/urfave/cli/v3"
)

var ErrHealthCheckFailed = errors.New("healthcheck failed")

var healthcheckCMD = &cli.Command{
	Name:     "healthcheck",
	Aliases:  []string{"hc"},
	Usage:    "Run healthcheck.",
	Category: "Utility",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8081/health", nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		client := http.Client{
			Timeout: time.Second,
			Transport: &http.Transport{
				ResponseHeaderTimeout: time.Second,
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrHealthCheckFailed, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%w: non-200 status: %d", ErrHealthCheckFailed, resp.StatusCode)
		}

		return nil
	},
}
