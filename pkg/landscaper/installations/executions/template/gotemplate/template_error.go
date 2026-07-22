// SPDX-FileCopyrightText: Copyright OpenControlPlane contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gotemplate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	lsv1alpha1 "github.com/openmcp-project/landscaper/apis/core/v1alpha1"
	"github.com/openmcp-project/landscaper/pkg/landscaper/installations/executions/template"
)

var (
	errorLineColumnRegexp = regexp.MustCompile("(?m):([0-9]+)(:([0-9]+))?:")
	// yamlLineRegexp matches the "line N" location marker used by
	// sigs.k8s.io/yaml / go-yaml when the generated YAML fails to parse
	// (e.g. "error converting YAML to JSON: yaml: line 19: could not find expected ':'").
	yamlLineRegexp = regexp.MustCompile(`line ([0-9]+)`)
)

// TemplateError wraps a go templating error and adds more human-readable information.
type TemplateError struct {
	err            error
	source         *string
	output         *string
	input          map[string]interface{}
	inputFormatter *template.TemplateInputFormatter
	message        string
}

// TemplateErrorBuilder creates a new TemplateError.
func TemplateErrorBuilder(err error) *TemplateError {
	return &TemplateError{
		err:     err,
		message: err.Error(),
	}
}

// WithSource adds the template source code to the error.
func (e *TemplateError) WithSource(source *string) *TemplateError {
	e.source = source
	return e
}

// WithFormattedOutput adds the rendered template output to the error. Used
// when the failure occurs after successful Go-template execution (e.g. the
// generated YAML is syntactically invalid or does not match the expected
// schema) - the user cannot otherwise see the generated document, so surface
// a snippet around the failing line.
func (e *TemplateError) WithFormattedOutput(output *string) *TemplateError {
	e.output = output
	return e
}

// WithInput adds the template input with a formatter to the error.
func (e *TemplateError) WithInput(input map[string]interface{}, inputFormatter *template.TemplateInputFormatter) *TemplateError {
	e.input = input
	e.inputFormatter = inputFormatter
	return e
}

// Build builds the error message.
func (e *TemplateError) Build() *TemplateError {
	builder := strings.Builder{}
	builder.WriteString(e.err.Error())

	if e.source != nil {
		builder.WriteString("\ntemplate source:\n")
		builder.WriteString(e.formatSource())
	}

	if e.output != nil {
		builder.WriteString("\ntemplated output:\n")
		builder.WriteString(e.formatOutput())
	}

	if e.input != nil && e.inputFormatter != nil {
		builder.WriteString("\ntemplate input:\n")
		builder.WriteString(e.inputFormatter.Format(e.input, "\t"))
	}

	e.message = builder.String()
	return e
}

// Error returns the error message.
func (e *TemplateError) Error() string {
	return e.message
}

// formatSource extracts the significant template source code that was the reason of the template error.
func (e *TemplateError) formatSource() string {
	line, column := extractLineColumn(e.err.Error())
	if line == 0 {
		return ""
	}
	return CreateSourceSnippet(line, column, strings.Split(*e.source, "\n"))
}

// formatOutput extracts the significant rendered output lines around the
// location reported by a downstream YAML parser.
func (e *TemplateError) formatOutput() string {
	line, column := extractLineColumn(e.err.Error())
	if line == 0 {
		return ""
	}
	return CreateSourceSnippet(line, column, strings.Split(*e.output, "\n"))
}

// extractLineColumn tries the Go-template ":N:M:" location marker first,
// then falls back to the YAML "line N" marker. Returns (0, 0) if neither
// matches.
func extractLineColumn(errStr string) (int, int) {
	if m := errorLineColumnRegexp.FindStringSubmatch(errStr); m != nil {
		line, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, 0
		}
		column := 0
		if len(m) >= 4 && m[3] != "" {
			if c, err := strconv.Atoi(m[3]); err == nil {
				column = c
			}
		}
		return line, column
	}
	if m := yamlLineRegexp.FindStringSubmatch(errStr); m != nil {
		line, err := strconv.Atoi(m[1])
		if err != nil {
			return 0, 0
		}
		return line, 0
	}
	return 0, 0
}

// wrapPostTemplateError builds a TemplateError for failures that occur after
// Go-template execution has already produced output (state persistence, YAML
// unmarshal into the expected schema). It attaches both the template source
// and the rendered output so the user can locate the failing line - which
// otherwise only exists in memory. The tmplExec name (and source file, when
// loaded from disk) are prefixed onto the message, together with a
// best-effort item name extracted from the generated output near the failing
// line, so the user can identify which of potentially many items failed.
func wrapPostTemplateError(msg string, err error, tmplExec lsv1alpha1.TemplateExecutor, source *string, output []byte) error {
	outStr := string(output)
	errLine, _ := extractLineColumn(err.Error())
	itemName := findEnclosingItemName(outStr, errLine)
	prefix := formatContextPrefix(tmplExec, itemName)
	return TemplateErrorBuilder(fmt.Errorf("%s%s: %w", prefix, msg, err)).
		WithSource(source).
		WithFormattedOutput(&outStr).
		Build()
}

var itemNameRegexp = regexp.MustCompile(`^\s*- name:\s*"?([^"\s]+)"?\s*$`)

// findEnclosingItemName scans the rendered output up to errorLine (1-based,
// inclusive) and returns the name of the closest preceding YAML list item
// with a "name:" field, if any. This is best-effort: with broken YAML we
// cannot use a proper parser, so the regex approach may miss deeply nested
// or unusually formatted entries. Returns an empty string when no match.
func findEnclosingItemName(output string, errorLine int) string {
	lines := strings.Split(output, "\n")
	if errorLine <= 0 || errorLine > len(lines) {
		errorLine = len(lines)
	}
	last := ""
	for i := 0; i < errorLine; i++ {
		if m := itemNameRegexp.FindStringSubmatch(lines[i]); m != nil {
			last = m[1]
		}
	}
	return last
}

// formatContextPrefix builds a "<item>, <template execution>: " prefix
// identifying the failing item and its enclosing template execution, or an
// empty string if no context is available.
func formatContextPrefix(tmplExec lsv1alpha1.TemplateExecutor, itemName string) string {
	parts := make([]string, 0, 2)
	if itemName != "" {
		parts = append(parts, fmt.Sprintf("item %q", itemName))
	}
	if tmplExec.Name != "" || tmplExec.File != "" {
		b := strings.Builder{}
		b.WriteString("template execution")
		if tmplExec.Name != "" {
			fmt.Fprintf(&b, " %q", tmplExec.Name)
		}
		if tmplExec.File != "" {
			fmt.Fprintf(&b, " (file %q)", tmplExec.File)
		}
		parts = append(parts, b.String())
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ") + ": "
}
