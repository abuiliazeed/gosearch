## Description

Briefly describe the changes made in this pull request.

<!--
Example:
This PR adds support for PDF content extraction during crawling.
It adds a new `pdf` package that uses `unidoc/unipdf` to extract
text from PDF files encountered during crawl.
-->

## Type of Change

Mark the relevant option with an `x`:

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Refactoring / code quality improvement
- [ ] Performance improvement
- [ ] Test addition / improvement

## Related Issue

Closes #(issue number)

<!-- If this PR resolves an issue, reference it like "Closes #123" -->

## Changes Made

List the main changes:

-
-
-

## Testing

Describe how you tested these changes:

<!--
Example:
- Added unit tests in `crawler/pdf_test.go`
- Manually tested crawling a site with PDF links
- Ran `go test ./...` - all tests pass
- Ran `go run -race ./cmd/gosearch` - no race conditions detected
-->

## Checklist

- [ ] My code follows the [Go Code Standards](https://github.com/abuiliazeed/gosearch/blob/main/CLAUDE.md)
- [ ] I have performed a self-review of my code
- [ ] I have commented my code where necessary, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation (README, CLAUDE.md, etc.)
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing tests pass locally with `go test ./...`
- [ ] Any dependent changes have been merged and published
- [ ] I have run `go fmt ./...` and `go vet ./...`

## Screenshots (if applicable)

If your changes affect the UI (TUI) or add new CLI output, include screenshots:

<!-- Add screenshots here -->

## Additional Notes

Any additional information or context that reviewers should be aware of.

<!--
For example:
- This change required updating the storage schema - users may need to re-crawl
- The new flag is optional and defaults to the previous behavior
-->
