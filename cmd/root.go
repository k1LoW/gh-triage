/*
Copyright © 2025 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k1LoW/duration"
	"github.com/k1LoW/gh-triage/gh"
	"github.com/k1LoW/gh-triage/profile"
	"github.com/k1LoW/gh-triage/version"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
)

var (
	profileFlag  string
	watch        bool
	intervalFlag string
	verbose      bool
)

var rootCmd = &cobra.Command{
	Use:           "gh-triage",
	Short:         "gh-triage is a tool that helps you manage and triage GitHub issues and pull requests through unread notifications",
	Long:          `gh-triage is a tool that helps you manage and triage GitHub issues and pull requests through notifications.`,
	SilenceErrors: true,
	Args:          cobra.NoArgs,
	Version:       version.Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := profile.Load(profileFlag)
		if err != nil {
			return err
		}
		c, err := gh.New(cfg, colorable.NewColorableStdout(), verbose)
		if err != nil {
			return err
		}
		if watch {
			interval, err := duration.Parse(intervalFlag)
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}

			// Create context with cancellation for graceful shutdown
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			// Start signal handler goroutine
			go func() {
				sig := <-sigCh
				slog.Info("Received signal, shutting down gracefully", "signal", sig)
				cancel()
			}()

			slog.Info("Starting watch mode", "interval", interval)

			// Wait for initial interval before starting
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return nil
			}

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			if err := c.Triage(ctx); err != nil {
				slog.Error("Triage failed", "error", err)
			}

			for {
				select {
				case <-ticker.C:
					if err := c.Triage(ctx); err != nil {
						slog.Error("Triage failed", "error", err)
						// Continue watching even if triage fails (error continuation strategy)
					}
				case <-ctx.Done():
					slog.Info("Watch mode stopped")
					return nil
				}
			}

		} else {
			if err := c.Triage(cmd.Context()); err != nil {
				return err
			}
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "Profile name for configuration file")
	rootCmd.PersistentFlags().BoolVarP(&watch, "watch", "w", false, "Watch for notifications")
	rootCmd.PersistentFlags().StringVarP(&intervalFlag, "interval", "i", "5min", "Interval for watching notifications (e.g., 5min, 1hour)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Verbose output")
}
