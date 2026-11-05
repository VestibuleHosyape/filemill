package cmd

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"filemill/internal/dedupe"
	"filemill/internal/walker"
	"github.com/spf13/cobra"
)

type deletion struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
type dedupeReport struct {
	dedupe.Result
	Mode    string     `json:"mode"`
	Actions []deletion `json:"actions"`
}

func newDedupeCommand() *cobra.Command {
	var filters filterFlags
	var remove, apply, yes bool
	var output string
	c := &cobra.Command{Use: "dedupe [path]", Short: "按大小和 SHA-256 查重，默认只报告", Args: cobra.MaximumNArgs(1)}
	filters.register(c)
	c.Flags().BoolVar(&remove, "delete", false, "计划删除重复项，按路径排序保留第一份")
	c.Flags().BoolVar(&apply, "apply", false, "结合 --delete 真正删除；请先停止目录中的并发写入")
	c.Flags().BoolVar(&yes, "yes", false, "同 --apply")
	c.Flags().StringVar(&output, "output", "table", "输出格式：table / csv / json")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if (apply || yes) && !remove {
			return fmt.Errorf("--apply/--yes requires --delete")
		}
		format := strings.ToLower(strings.TrimSpace(output))
		if format != "table" && format != "csv" && format != "json" {
			return fmt.Errorf("unsupported output format %q", output)
		}
		opts, err := filters.options()
		if err != nil {
			return err
		}
		root := "."
		if len(args) > 0 {
			root = args[0]
		}
		files, scanErr := walker.Collect(root, opts)
		r := dedupeReport{Result: dedupe.Find(files, Workers()), Mode: "report", Actions: []deletion{}}
		if scanErr != nil {
			r.Errors = append(r.Errors, dedupe.Failure{Path: root, Error: scanErr.Error()})
		}
		for _, failure := range r.Errors {
			slog.Error("dedupe scan/hash failed", "path", failure.Path, "err", failure.Error)
		}
		if remove {
			r.Mode = "dry-run"
		}
		execute := remove && (apply || yes) && !DryRun()
		// Incomplete discovery must never produce a destructive partial plan.
		if execute && len(r.Errors) > 0 {
			r.Mode = "blocked"
			execute = false
		} else if execute {
			r.Mode = "apply"
		}
		var actionErrors []error
		if remove {
			for _, group := range r.Groups {
				for _, file := range group.Duplicates {
					a := deletion{Path: file.Path, Status: "dry-run"}
					if r.Mode == "blocked" {
						a.Status = "blocked"
					}
					if execute {
						if err := dedupe.RemoveVerified(group, file); err != nil {
							slog.Error("dedupe delete failed", "path", file.Path, "err", err)
							a.Status, a.Error = "error", err.Error()
							actionErrors = append(actionErrors, fmt.Errorf("%s: %w", file.Path, err))
						} else {
							a.Status = "deleted"
						}
					}
					r.Actions = append(r.Actions, a)
				}
			}
		}
		outputErr := writeDedupeReport(cmd.OutOrStdout(), r, format)
		_, summaryErr := fmt.Fprintf(cmd.ErrOrStderr(), "mode=%s scanned=%d hashed=%d groups=%d reclaimable_bytes=%d errors=%d\n", r.Mode, r.Scanned, r.Hashed, len(r.Groups), r.ReclaimableBytes, len(r.Errors)+len(actionErrors))
		if len(r.Errors) > 0 {
			actionErrors = append(actionErrors, fmt.Errorf("%d scan/hash error(s)", len(r.Errors)))
		}
		return errors.Join(append(actionErrors, outputErr, summaryErr)...)
	}
	return c
}

func init() { rootCmd.AddCommand(newDedupeCommand()) }

func writeDedupeReport(w io.Writer, r dedupeReport, format string) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}
	actions := make(map[string]deletion)
	for _, a := range r.Actions {
		actions[a.Path] = a
	}
	var cw *csv.Writer
	if format == "csv" {
		cw = csv.NewWriter(w)
		if err := cw.Write([]string{"keep", "duplicate", "size", "sha256", "status", "error"}); err != nil {
			return err
		}
	}
	for _, g := range r.Groups {
		for _, f := range g.Duplicates {
			a, ok := actions[f.Path]
			if !ok {
				a.Status = "duplicate"
			}
			row := []string{g.Keep.Path, f.Path, strconv.FormatInt(g.Size, 10), g.Hash, a.Status, a.Error}
			if cw != nil {
				if err := cw.Write(row); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(w, "%s\tkeep=%s\tduplicate=%s\tsize=%d\t%s\n", a.Status, g.Keep.Path, f.Path, g.Size, a.Error); err != nil {
					return err
				}
			}
		}
	}
	for _, failure := range r.Errors {
		if cw != nil {
			if err := cw.Write([]string{"", failure.Path, "", "", "error", failure.Error}); err != nil {
				return err
			}
		} else if _, err := fmt.Fprintf(w, "error\t%s\t%s\n", failure.Path, failure.Error); err != nil {
			return err
		}
	}
	if cw != nil {
		cw.Flush()
		return cw.Error()
	}
	return nil
}
