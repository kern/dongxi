package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kern/dongxi/dongxi"
	"github.com/spf13/cobra"
)

var (
	flagExportFormat string
	flagExportType   string
	flagExportFilter string
	flagExportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export items from Things Cloud to JSON or CSV",
	Long: `Export items from the locally cached Things Cloud state.

Examples:
  dongxi export                          # JSON export of open tasks (default)
  dongxi export --format csv             # CSV export of open tasks
  dongxi export --type projects          # Export only projects
  dongxi export --type areas             # Export only areas
  dongxi export --type tags              # Export only tags
  dongxi export --type checklist         # Export only checklist items
  dongxi export --type all               # Export everything
  dongxi export --filter all             # Include completed/trashed
  dongxi export --filter completed       # Only completed items
  dongxi export --filter trash           # Only trashed items
  dongxi export -o output.json           # Write to file`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&flagExportFormat, "format", "json", "Output format: json or csv")
	exportCmd.Flags().StringVar(&flagExportType, "type", "tasks", "Item type: tasks, projects, areas, tags, checklist, all")
	exportCmd.Flags().StringVar(&flagExportFilter, "filter", "open", "Filter: open, completed, trash, all")
	exportCmd.Flags().StringVarP(&flagExportOutput, "output", "o", "", "Output file (default: stdout)")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Validate format.
	format := strings.ToLower(flagExportFormat)
	if format != "json" && format != "csv" {
		return fmt.Errorf("unknown format %q: must be json or csv", flagExportFormat)
	}

	// Validate type.
	exportType := strings.ToLower(flagExportType)
	switch exportType {
	case "tasks", "projects", "areas", "tags", "checklist", "all":
	default:
		return fmt.Errorf("unknown type %q: must be tasks, projects, areas, tags, checklist, or all", flagExportType)
	}

	// Validate filter.
	filter := strings.ToLower(flagExportFilter)
	switch filter {
	case "open", "completed", "trash", "all":
	default:
		return fmt.Errorf("unknown filter %q: must be open, completed, trash, or all", flagExportFilter)
	}

	s, _, _, err := loadState()
	if err != nil {
		return err
	}

	// Collect matching items.
	var outputs []ItemOutput
	for i := range s.items {
		item := &s.items[i]

		if !matchesExportType(item, exportType) {
			continue
		}

		if !matchesExportFilter(item, filter) {
			continue
		}

		outputs = append(outputs, s.itemToOutput(item))
	}

	// Determine writer.
	var w io.Writer = os.Stdout
	if flagExportOutput != "" {
		f, err := os.Create(flagExportOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	switch format {
	case "json":
		return writeJSON(w, outputs)
	case "csv":
		return writeCSV(w, outputs)
	}
	return nil
}

// matchesExportType returns true if the item matches the requested export type.
func matchesExportType(item *replayedItem, exportType string) bool {
	switch exportType {
	case "tasks":
		return item.entity == string(dongxi.EntityTask) &&
			toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeTask)
	case "projects":
		return item.entity == string(dongxi.EntityTask) &&
			toInt(item.fields[dongxi.FieldType]) == int(dongxi.TaskTypeProject)
	case "areas":
		return item.entity == string(dongxi.EntityArea)
	case "tags":
		return item.entity == string(dongxi.EntityTag)
	case "checklist":
		return item.entity == string(dongxi.EntityChecklistItem)
	case "all":
		return true
	}
	return false
}

// matchesExportFilter returns true if the item matches the requested filter.
func matchesExportFilter(item *replayedItem, filter string) bool {
	if filter == "all" {
		return true
	}

	// Tags have no status/trashed fields; always include them.
	if item.entity == string(dongxi.EntityTag) {
		return true
	}

	trashed := toBool(item.fields[dongxi.FieldTrashed])
	status := toInt(item.fields[dongxi.FieldStatus])

	switch filter {
	case "open":
		return !trashed && status == int(dongxi.TaskStatusOpen)
	case "completed":
		return !trashed && status == int(dongxi.TaskStatusCompleted)
	case "trash":
		return trashed
	}
	return false
}

// writeJSON writes items as a pretty-printed JSON array.
func writeJSON(w io.Writer, items []ItemOutput) error {
	if items == nil {
		items = []ItemOutput{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(items)
}

// writeCSV writes items as CSV with a header row.
func writeCSV(w io.Writer, items []ItemOutput) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{
		"uuid", "entity", "type", "title", "status", "destination",
		"area", "project", "created", "modified", "scheduled", "deadline",
		"notes", "tags", "evening", "trashed",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, item := range items {
		evening := ""
		if item.Evening != nil {
			if *item.Evening {
				evening = "true"
			} else {
				evening = "false"
			}
		}

		trashed := ""
		if item.Trashed != nil {
			if *item.Trashed {
				trashed = "true"
			} else {
				trashed = "false"
			}
		}

		row := []string{
			item.UUID,
			item.Entity,
			item.Type,
			item.Title,
			item.Status,
			item.Destination,
			item.Area,
			item.Project,
			item.Created,
			item.Modified,
			item.Scheduled,
			item.Deadline,
			item.Notes,
			strings.Join(item.Tags, ";"),
			evening,
			trashed,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}
