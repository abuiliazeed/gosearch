package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/abuiliazeed/gosearch/internal/storage"
)

func TestNormalizeDomainInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "host only",
			input: "example.com",
			want:  "example.com",
		},
		{
			name:  "url with path and uppercase host",
			input: "https://Example.COM/products",
			want:  "example.com",
		},
		{
			name:  "host with trailing dot",
			input: "example.com.",
			want:  "example.com",
		},
		{
			name:  "host with port",
			input: "www.example.com:8443",
			want:  "www.example.com",
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "missing host",
			input:   "http:///path",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeDomainInput(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestHostMatchesDomain(t *testing.T) {
	target := "example.com"
	tests := []struct {
		host string
		want bool
	}{
		{host: "example.com", want: true},
		{host: "www.example.com", want: true},
		{host: "blog.api.example.com", want: true},
		{host: "example.com.", want: true},
		{host: "fakeexample.com", want: false},
		{host: "example.org", want: false},
	}

	for _, tc := range tests {
		got := hostMatchesDomain(tc.host, target)
		if got != tc.want {
			t.Fatalf("hostMatchesDomain(%q, %q) = %v, want %v", tc.host, target, got, tc.want)
		}
	}
}

func TestBuildExportFilenameDeterministic(t *testing.T) {
	docID := "abcdef1234567890"
	got := buildExportFilename("/Products/Lip Gloss/", docID)
	want := "products-lip-gloss-abcdef123456.md"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	gotAgain := buildExportFilename("/Products/Lip Gloss/", docID)
	if gotAgain != want {
		t.Fatalf("expected deterministic output %q, got %q", want, gotAgain)
	}

	rootPath := buildExportFilename("/", "abc")
	if rootPath != "index-abc.md" {
		t.Fatalf("expected root path filename %q, got %q", "index-abc.md", rootPath)
	}
}

func TestRunMDExport_ExportsOnlyIndexedAndMatching(t *testing.T) {
	dataDir := t.TempDir()
	setTestDataDir(t, dataDir)
	if err := writeSchemaVersion(dataDir); err != nil {
		t.Fatalf("failed to write schema marker: %v", err)
	}

	docStore, indexStore := newTestExportStores(t, dataDir)

	doc1 := &storage.Document{
		ID:              "111111111111aaaa",
		URL:             "https://example.com/page-one",
		Title:           "Page One",
		ContentMarkdown: "# one",
		CrawledAt:       time.Now(),
	}
	saveIndexedDocument(t, docStore, indexStore, doc1)

	doc2 := &storage.Document{
		ID:              "222222222222bbbb",
		URL:             "https://blog.example.com/two",
		Title:           "Page Two",
		ContentMarkdown: "# two",
		CrawledAt:       time.Now(),
	}
	saveIndexedDocument(t, docStore, indexStore, doc2)

	doc3 := &storage.Document{
		ID:              "333333333333cccc",
		URL:             "https://other.com/three",
		Title:           "Page Three",
		ContentMarkdown: "# three",
		CrawledAt:       time.Now(),
	}
	saveIndexedDocument(t, docStore, indexStore, doc3)

	unindexed := &storage.Document{
		ID:              "444444444444dddd",
		URL:             "https://example.com/unindexed",
		Title:           "Unindexed",
		ContentMarkdown: "# unindexed",
		CrawledAt:       time.Now(),
	}
	if err := docStore.Save(unindexed); err != nil {
		t.Fatalf("failed to save unindexed document: %v", err)
	}
	if err := indexStore.Close(); err != nil {
		t.Fatalf("failed to close index store before export: %v", err)
	}
	if err := docStore.Close(); err != nil {
		t.Fatalf("failed to close document store before export: %v", err)
	}

	outputRoot := filepath.Join(t.TempDir(), "exports")
	cmd, stdout := newTestMDExportCommand(t, outputRoot)
	if err := runMDExport(cmd, []string{"example.com"}); err != nil {
		t.Fatalf("runMDExport failed: %v", err)
	}

	outputDomainDir := filepath.Join(outputRoot, "example.com")
	entries, err := os.ReadDir(outputDomainDir)
	if err != nil {
		t.Fatalf("failed to read output directory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 exported files, got %d", len(entries))
	}

	file1 := filepath.Join(outputDomainDir, buildExportFilename("/page-one", doc1.ID))
	file2 := filepath.Join(outputDomainDir, buildExportFilename("/two", doc2.ID))
	assertFileContent(t, file1, doc1.ContentMarkdown)
	assertFileContent(t, file2, doc2.ContentMarkdown)

	unindexedPath := filepath.Join(outputDomainDir, buildExportFilename("/unindexed", unindexed.ID))
	if _, err := os.Stat(unindexedPath); !os.IsNotExist(err) {
		t.Fatalf("expected unindexed file to be absent, stat err=%v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Indexed docs scanned: 3") {
		t.Fatalf("expected scanned summary, got output:\n%s", out)
	}
	if !strings.Contains(out, "Matched domain docs: 2") {
		t.Fatalf("expected matched summary, got output:\n%s", out)
	}
	if !strings.Contains(out, "Exported docs: 2") {
		t.Fatalf("expected exported summary, got output:\n%s", out)
	}
	if !strings.Contains(out, "Skipped missing docs: 0") {
		t.Fatalf("expected skipped summary, got output:\n%s", out)
	}
}

func TestRunMDExport_OverwritesExistingFile(t *testing.T) {
	dataDir := t.TempDir()
	setTestDataDir(t, dataDir)
	if err := writeSchemaVersion(dataDir); err != nil {
		t.Fatalf("failed to write schema marker: %v", err)
	}

	docStore, indexStore := newTestExportStores(t, dataDir)
	doc := &storage.Document{
		ID:              "aaaaaaaaaaaabbbb",
		URL:             "https://example.com/page-one",
		Title:           "Page One",
		ContentMarkdown: "# fresh markdown",
		CrawledAt:       time.Now(),
	}
	saveIndexedDocument(t, docStore, indexStore, doc)

	outputRoot := filepath.Join(t.TempDir(), "exports")
	outputDomainDir := filepath.Join(outputRoot, "example.com")
	if err := os.MkdirAll(outputDomainDir, 0o755); err != nil {
		t.Fatalf("failed to create output directory: %v", err)
	}
	targetFile := filepath.Join(outputDomainDir, buildExportFilename("/page-one", doc.ID))
	if err := os.WriteFile(targetFile, []byte("old content"), 0o644); err != nil {
		t.Fatalf("failed to seed existing file: %v", err)
	}
	if err := indexStore.Close(); err != nil {
		t.Fatalf("failed to close index store before export: %v", err)
	}
	if err := docStore.Close(); err != nil {
		t.Fatalf("failed to close document store before export: %v", err)
	}

	cmd, _ := newTestMDExportCommand(t, outputRoot)
	if err := runMDExport(cmd, []string{"example.com"}); err != nil {
		t.Fatalf("runMDExport failed: %v", err)
	}

	assertFileContent(t, targetFile, doc.ContentMarkdown)
}

func TestRunMDExport_SkipsMissingIndexedDocumentPayload(t *testing.T) {
	dataDir := t.TempDir()
	setTestDataDir(t, dataDir)
	if err := writeSchemaVersion(dataDir); err != nil {
		t.Fatalf("failed to write schema marker: %v", err)
	}

	docStore, indexStore := newTestExportStores(t, dataDir)

	existing := &storage.Document{
		ID:              "eeeeeeeeeeeeffff",
		URL:             "https://example.com/existing",
		Title:           "Existing",
		ContentMarkdown: "# existing",
		CrawledAt:       time.Now(),
	}
	saveIndexedDocument(t, docStore, indexStore, existing)

	if err := indexStore.SaveDocInfo(&storage.PersistedDocInfo{
		DocID:      "missing-doc-1234567890",
		URL:        "https://example.com/missing",
		Title:      "Missing",
		IndexedAt:  time.Now(),
		TokenCount: 10,
		Length:     5,
	}); err != nil {
		t.Fatalf("failed to save missing doc info: %v", err)
	}
	if err := indexStore.Close(); err != nil {
		t.Fatalf("failed to close index store before export: %v", err)
	}
	if err := docStore.Close(); err != nil {
		t.Fatalf("failed to close document store before export: %v", err)
	}

	outputRoot := filepath.Join(t.TempDir(), "exports")
	cmd, stdout := newTestMDExportCommand(t, outputRoot)
	if err := runMDExport(cmd, []string{"example.com"}); err != nil {
		t.Fatalf("runMDExport failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Skipped missing docs: 1") {
		t.Fatalf("expected skipped missing docs summary, got output:\n%s", output)
	}
	if !strings.Contains(output, "Exported docs: 1") {
		t.Fatalf("expected exported docs summary, got output:\n%s", output)
	}

	expectedFile := filepath.Join(outputRoot, "example.com", buildExportFilename("/existing", existing.ID))
	assertFileContent(t, expectedFile, existing.ContentMarkdown)
}

func TestRunMDExport_ReturnsErrorWhenNoMatchingDomainDocs(t *testing.T) {
	dataDir := t.TempDir()
	setTestDataDir(t, dataDir)
	if err := writeSchemaVersion(dataDir); err != nil {
		t.Fatalf("failed to write schema marker: %v", err)
	}

	docStore, indexStore := newTestExportStores(t, dataDir)
	other := &storage.Document{
		ID:              "9999999999990000",
		URL:             "https://other.com/page",
		Title:           "Other",
		ContentMarkdown: "# other",
		CrawledAt:       time.Now(),
	}
	saveIndexedDocument(t, docStore, indexStore, other)
	if err := indexStore.Close(); err != nil {
		t.Fatalf("failed to close index store before export: %v", err)
	}
	if err := docStore.Close(); err != nil {
		t.Fatalf("failed to close document store before export: %v", err)
	}

	outputRoot := filepath.Join(t.TempDir(), "exports")
	cmd, _ := newTestMDExportCommand(t, outputRoot)
	err := runMDExport(cmd, []string{"example.com"})
	if err == nil {
		t.Fatal("expected error when no matching domain documents exist")
	}
	if !strings.Contains(err.Error(), `no indexed markdown documents found for domain "example.com"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTestMDExportCommand(t *testing.T, outputRoot string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.Flags().String("output-dir", outputRoot, "")
	var out bytes.Buffer
	cmd.SetOut(&out)

	return cmd, &out
}

func setTestDataDir(t *testing.T, dataDir string) {
	t.Helper()
	previous := viper.GetString("data-dir")
	viper.Set("data-dir", dataDir)
	t.Cleanup(func() {
		viper.Set("data-dir", previous)
	})
}

func newTestExportStores(t *testing.T, dataDir string) (*storage.DocumentStore, *storage.IndexStore) {
	t.Helper()

	docStore, err := storage.NewDocumentStore(filepath.Join(dataDir, "pages"))
	if err != nil {
		t.Fatalf("failed to create document store: %v", err)
	}
	t.Cleanup(func() {
		_ = docStore.Close()
	})

	indexStore, err := storage.NewIndexStore(filepath.Join(dataDir, "index", "index.db"))
	if err != nil {
		t.Fatalf("failed to create index store: %v", err)
	}
	t.Cleanup(func() {
		_ = indexStore.Close()
	})

	return docStore, indexStore
}

func saveIndexedDocument(t *testing.T, docStore *storage.DocumentStore, indexStore *storage.IndexStore, doc *storage.Document) {
	t.Helper()

	if err := docStore.Save(doc); err != nil {
		t.Fatalf("failed to save document %s: %v", doc.ID, err)
	}

	if err := indexStore.SaveDocInfo(&storage.PersistedDocInfo{
		DocID:      doc.ID,
		URL:        doc.URL,
		Title:      doc.Title,
		IndexedAt:  time.Now(),
		TokenCount: 10,
		Length:     5,
	}); err != nil {
		t.Fatalf("failed to save doc info %s: %v", doc.ID, err)
	}
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(data) != expected {
		t.Fatalf("unexpected file content for %s: got %q want %q", path, string(data), expected)
	}
}
