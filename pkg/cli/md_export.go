package cli

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/abuiliazeed/gosearch/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const exportDocIDPrefixLen = 12

var slugPartPattern = regexp.MustCompile(`[^a-z0-9]+`)

type mdExportSummary struct {
	Scanned        int
	Matched        int
	Exported       int
	SkippedMissing int
	OutputPath     string
}

var mdExportCmd = &cobra.Command{
	Use:   "md-export <domain>",
	Short: "Export indexed markdown documents for a domain",
	Long: `Export indexed markdown documents for a selected domain into a domain folder.

Domain matching includes subdomains. For example, exporting "example.com"
includes documents from "www.example.com" and "blog.example.com".

Behavior:
  - Exports only documents present in the index (DocInfo), not every stored page file.
  - Writes files to <output-dir>/<domain>/.
  - Uses deterministic filenames: <path-slug>-<docid-prefix>.md.
  - Overwrites existing files with the same generated name.
  - Returns an error if no indexed documents match the domain.`,
	Example: `  # Export sourcebeauty.com markdown into ./exports/sourcebeauty.com
  gosearch md-export sourcebeauty.com

  # Export into a custom base directory
  gosearch md-export https://sourcebeauty.com --output-dir ./domain-exports`,
	Args: cobra.ExactArgs(1),
	RunE: runMDExport,
}

func runMDExport(cmd *cobra.Command, args []string) error {
	targetDomain, err := normalizeDomainInput(args[0])
	if err != nil {
		return err
	}

	outputRoot, _ := cmd.Flags().GetString("output-dir")
	outputRoot = strings.TrimSpace(outputRoot)
	if outputRoot == "" {
		outputRoot = "./exports"
	}

	dataDir := viper.GetString("data-dir")
	if err := requireSchemaVersion(dataDir); err != nil {
		return err
	}

	indexPath := filepath.Join(dataDir, "index", "index.db")
	pagesPath := filepath.Join(dataDir, "pages")

	indexStore, err := storage.NewIndexStore(indexPath)
	if err != nil {
		return fmt.Errorf("failed to open index store: %w", err)
	}
	defer indexStore.Close()

	docStore, err := storage.NewDocumentStore(pagesPath)
	if err != nil {
		return fmt.Errorf("failed to open document store: %w", err)
	}
	defer docStore.Close()

	docIDs, err := indexStore.ListAllDocInfo()
	if err != nil {
		return fmt.Errorf("failed to list indexed documents: %w", err)
	}
	sort.Strings(docIDs)

	outputPath := filepath.Join(outputRoot, targetDomain)
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputPath, err)
	}

	summary := mdExportSummary{
		Scanned:    len(docIDs),
		OutputPath: outputPath,
	}

	for _, docID := range docIDs {
		doc, err := docStore.Get(docID)
		if err != nil {
			if isDocumentMissingError(err) {
				summary.SkippedMissing++
				continue
			}
			return fmt.Errorf("failed to load document %s: %w", docID, err)
		}

		parsedDocURL, err := url.Parse(doc.URL)
		if err != nil {
			return fmt.Errorf("failed to parse URL for document %s: %w", docID, err)
		}

		if !hostMatchesDomain(parsedDocURL.Hostname(), targetDomain) {
			continue
		}

		summary.Matched++

		fileName := buildExportFilename(parsedDocURL.Path, doc.ID)
		filePath := filepath.Join(outputPath, fileName)

		// Security: Validate that filePath is within outputPath to prevent path traversal
		absOutputPath, err := filepath.Abs(outputPath)
		if err != nil {
			return fmt.Errorf("failed to resolve output path: %w", err)
		}
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			return fmt.Errorf("failed to resolve file path: %w", err)
		}
		relPath, err := filepath.Rel(absOutputPath, absFilePath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("invalid file path %s: outside output directory", filePath)
		}

		if err := os.WriteFile(filePath, []byte(doc.ContentMarkdown), 0o644); err != nil {
			return fmt.Errorf("failed to write markdown for document %s: %w", docID, err)
		}
		summary.Exported++
	}

	if summary.Exported == 0 {
		return fmt.Errorf("no indexed markdown documents found for domain %q", targetDomain)
	}

	printMDExportSummary(cmd.OutOrStdout(), targetDomain, summary)
	return nil
}

func printMDExportSummary(w io.Writer, domain string, summary mdExportSummary) {
	fmt.Fprintln(w, "Markdown export complete:")
	fmt.Fprintf(w, "  Target domain: %s\n", domain)
	fmt.Fprintf(w, "  Indexed docs scanned: %d\n", summary.Scanned)
	fmt.Fprintf(w, "  Matched domain docs: %d\n", summary.Matched)
	fmt.Fprintf(w, "  Exported docs: %d\n", summary.Exported)
	fmt.Fprintf(w, "  Skipped missing docs: %d\n", summary.SkippedMissing)
	fmt.Fprintf(w, "  Output directory: %s\n", summary.OutputPath)
}

func normalizeDomainInput(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("domain cannot be empty")
	}

	candidate := trimmed
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return "", fmt.Errorf("invalid domain input %q: %w", input, err)
	}

	host := normalizeHost(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("invalid domain input %q: missing host", input)
	}

	return host, nil
}

func hostMatchesDomain(docURLHost string, targetDomain string) bool {
	host := normalizeHost(docURLHost)
	domain := normalizeHost(targetDomain)
	if host == "" || domain == "" {
		return false
	}
	return host == domain || strings.HasSuffix(host, "."+domain)
}

func slugFromURLPath(urlPath string) string {
	trimmed := strings.TrimSpace(urlPath)
	if trimmed == "" || trimmed == "/" {
		return "index"
	}

	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "index"
	}

	parts := strings.Split(trimmed, "/")
	slugs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		part = slugPartPattern.ReplaceAllString(part, "-")
		part = strings.Trim(part, "-")
		if part == "" {
			continue
		}
		slugs = append(slugs, part)
	}

	if len(slugs) == 0 {
		return "index"
	}

	return strings.Join(slugs, "-")
}

func buildExportFilename(urlPath string, docID string) string {
	slug := slugFromURLPath(urlPath)
	prefix := strings.TrimSpace(docID)
	if len(prefix) > exportDocIDPrefixLen {
		prefix = prefix[:exportDocIDPrefixLen]
	}
	if prefix == "" {
		prefix = "unknown"
	}
	return fmt.Sprintf("%s-%s.md", slug, prefix)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	return strings.TrimRight(host, ".")
}

func isDocumentMissingError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "document not found:")
}

func init() {
	rootCmd.AddCommand(mdExportCmd)
	mdExportCmd.Flags().String("output-dir", "./exports", "base output directory for exported markdown files")
}
