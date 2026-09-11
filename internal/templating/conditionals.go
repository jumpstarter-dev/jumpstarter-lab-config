package templating

import (
	"fmt"
	"strings"
)

// conditionalState tracks the state of a single $if/$elif/$else/$endif block.
type conditionalState struct {
	active   bool // whether we are currently emitting lines at this nesting level
	resolved bool // whether any branch at this level has already matched
	sawElse  bool // whether we have already seen $else for this block
}

// processConditionals evaluates $if/$elif/$else/$endif directives in the input
// and returns the filtered output with directive lines removed and excluded
// blocks stripped. This runs as a pre-pass before variable substitution.
//
// Supported condition expressions:
//
//	$if( params.key )          - true if key exists in replacements and is non-empty
//	$if( !params.key )         - true if key is missing or empty
//	$if( params.key == "val" ) - true if key exists and equals "val"
//	$if( params.key != "val" ) - true if key is missing or does not equal "val"
//
// Directive recognition rules:
//   - A line is only treated as a directive if the entire trimmed content matches
//     the directive pattern. Trailing # comments are allowed on directive lines.
//   - Lines that merely start with $if( but contain other content are treated as
//     regular content lines, not directives.
//
// Before evaluation, variable values referenced in conditions are resolved through
// the replacement map to handle recursive/chained variable references.
func processConditionals(data string, replacements map[string]string) (string, error) {
	// Resolve replacements so conditions evaluate against final values,
	// not raw $(vars.x) references.
	resolved := resolveReplacements(replacements)

	lines := strings.Split(data, "\n")
	var output []string
	var stack []conditionalState

	// Track whether the original input ended with a newline so we can
	// preserve it exactly in the output.
	endsWithNewline := len(data) > 0 && data[len(data)-1] == '\n'

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)

		directive, kind := classifyLine(trimmed)

		switch kind {
		case directiveIf:
			expr, err := extractExpression(directive, "$if(")
			if err != nil {
				return "", fmt.Errorf("line %d: %w", lineNum+1, err)
			}
			result, err := evaluateCondition(expr, resolved)
			if err != nil {
				return "", fmt.Errorf("line %d: %w", lineNum+1, err)
			}
			parentActive := isStackActive(stack)
			stack = append(stack, conditionalState{
				active:   parentActive && result,
				resolved: result,
			})

		case directiveElif:
			if len(stack) == 0 {
				return "", fmt.Errorf("line %d: $elif without matching $if", lineNum+1)
			}
			top := &stack[len(stack)-1]
			if top.sawElse {
				return "", fmt.Errorf("line %d: $elif after $else in the same $if block", lineNum+1)
			}
			expr, err := extractExpression(directive, "$elif(")
			if err != nil {
				return "", fmt.Errorf("line %d: %w", lineNum+1, err)
			}
			result, err := evaluateCondition(expr, resolved)
			if err != nil {
				return "", fmt.Errorf("line %d: %w", lineNum+1, err)
			}
			parentActive := isParentActive(stack)
			if top.resolved {
				top.active = false
			} else {
				top.active = parentActive && result
				if result {
					top.resolved = true
				}
			}

		case directiveElse:
			if len(stack) == 0 {
				return "", fmt.Errorf("line %d: $else without matching $if", lineNum+1)
			}
			top := &stack[len(stack)-1]
			if top.sawElse {
				return "", fmt.Errorf("line %d: duplicate $else in the same $if block", lineNum+1)
			}
			top.sawElse = true
			parentActive := isParentActive(stack)
			if top.resolved {
				top.active = false
			} else {
				top.active = parentActive
				top.resolved = true
			}

		case directiveEndif:
			if len(stack) == 0 {
				return "", fmt.Errorf("line %d: $endif without matching $if", lineNum+1)
			}
			stack = stack[:len(stack)-1]

		default: // directiveNone — regular content line
			if isStackActive(stack) {
				output = append(output, line)
			}
		}
	}

	if len(stack) > 0 {
		return "", fmt.Errorf("unclosed $if block: %d $if directive(s) without matching $endif", len(stack))
	}

	result := strings.Join(output, "\n")
	// Preserve trailing newline behavior: if the original ended with \n and
	// our output does not (e.g. last line was a directive), add it back.
	// Conversely, don't add one if the original didn't have it.
	if endsWithNewline && !strings.HasSuffix(result, "\n") && result != "" {
		result += "\n"
	}
	if !endsWithNewline && strings.HasSuffix(result, "\n") {
		result = strings.TrimRight(result, "\n")
	}

	return result, nil
}

// directiveKind represents the type of directive found on a line.
type directiveKind int

const (
	directiveNone  directiveKind = iota // not a directive
	directiveIf                         // $if(...)
	directiveElif                       // $elif(...)
	directiveElse                       // $else
	directiveEndif                      // $endif
)

// classifyLine determines whether a trimmed line is a directive.
// Returns the directive portion (without trailing comment) and its kind.
// A line is only a directive if the entire trimmed content matches the
// directive pattern; trailing # comments are allowed.
func classifyLine(trimmed string) (string, directiveKind) {
	// Check for $if( and $elif( — these require a closing ) and optionally
	// a trailing # comment.
	if strings.HasPrefix(trimmed, "$if(") {
		if dir, ok := extractDirectiveWithComment(trimmed); ok {
			return dir, directiveIf
		}
		return "", directiveNone // looks like $if( but doesn't parse — treat as content
	}

	if strings.HasPrefix(trimmed, "$elif(") {
		if dir, ok := extractDirectiveWithComment(trimmed); ok {
			return dir, directiveElif
		}
		return "", directiveNone
	}

	// $else and $endif — must be the entire content, or followed by whitespace + # comment
	if trimmed == "$else" || isKeywordWithComment(trimmed, "$else") {
		return "$else", directiveElse
	}

	if trimmed == "$endif" || isKeywordWithComment(trimmed, "$endif") {
		return "$endif", directiveEndif
	}

	return "", directiveNone
}

// extractDirectiveWithComment checks if a line like "$if( expr ) # comment" or
// "$if( expr )" is a valid directive. It finds the matching closing ) for the
// directive and verifies that anything after it is only whitespace or a # comment.
// Returns the directive portion (up to and including ')') and true if valid.
func extractDirectiveWithComment(trimmed string) (string, bool) {
	// Find the last ')' in the line
	closeIdx := strings.LastIndex(trimmed, ")")
	if closeIdx < 0 {
		return "", false
	}

	directive := trimmed[:closeIdx+1]
	rest := strings.TrimSpace(trimmed[closeIdx+1:])

	// After the closing ), only whitespace or a # comment is allowed
	if rest == "" || strings.HasPrefix(rest, "#") {
		return directive, true
	}

	return "", false
}

// isKeywordWithComment checks if trimmed is "$else # comment" or "$endif # comment".
func isKeywordWithComment(trimmed, keyword string) bool {
	if !strings.HasPrefix(trimmed, keyword) {
		return false
	}
	rest := trimmed[len(keyword):]
	rest = strings.TrimLeft(rest, " \t")
	return strings.HasPrefix(rest, "#")
}

// extractExpression extracts the condition expression from a directive line.
// For "$if( params.key )" with prefix "$if(", it returns "params.key".
func extractExpression(directive, prefix string) (string, error) {
	rest := strings.TrimPrefix(directive, prefix)
	if !strings.HasSuffix(rest, ")") {
		return "", fmt.Errorf("malformed directive: missing closing parenthesis in %q", directive)
	}
	expr := strings.TrimSuffix(rest, ")")
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", fmt.Errorf("empty condition in directive %q", directive)
	}
	return expr, nil
}

// evaluateCondition evaluates a condition expression against the replacement map.
//
// Supported forms:
//
//	"key"            - presence check: true if key exists and value is non-empty
//	"!key"           - negated presence: true if key is missing or empty
//	"key == \"val\"" - equality: true if key exists and equals val
//	"key != \"val\"" - inequality: true if key is missing or does not equal val
func evaluateCondition(expr string, replacements map[string]string) (bool, error) {
	// Check for equality/inequality operators.
	// We look for != first. To avoid ambiguity with keys containing != or ==,
	// we split on the first occurrence and require a quoted value on the right.
	if idx := strings.Index(expr, "!="); idx >= 0 {
		k := strings.TrimSpace(expr[:idx])
		val, err := extractQuotedValue(strings.TrimSpace(expr[idx+2:]))
		if err != nil {
			return false, fmt.Errorf("invalid condition %q: %w", expr, err)
		}
		actual, exists := replacements[k]
		return !exists || actual != val, nil
	}

	if idx := strings.Index(expr, "=="); idx >= 0 {
		k := strings.TrimSpace(expr[:idx])
		val, err := extractQuotedValue(strings.TrimSpace(expr[idx+2:]))
		if err != nil {
			return false, fmt.Errorf("invalid condition %q: %w", expr, err)
		}
		actual, exists := replacements[k]
		return exists && actual == val, nil
	}

	// Check for negation
	if strings.HasPrefix(expr, "!") {
		k := strings.TrimSpace(expr[1:])
		if k == "" {
			return false, fmt.Errorf("invalid condition: empty key after negation")
		}
		val, exists := replacements[k]
		return !exists || val == "", nil
	}

	// Simple presence check
	k := strings.TrimSpace(expr)
	val, exists := replacements[k]
	return exists && val != "", nil
}

// extractQuotedValue extracts a value from a quoted string like `"value"`.
// Supports escaped quotes within the value: `"say \"hello\""` -> `say "hello"`.
func extractQuotedValue(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' {
		return "", fmt.Errorf("expected quoted value, got %q", s)
	}

	// Walk the string looking for the closing unescaped quote
	var result strings.Builder
	i := 1 // skip opening quote
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			// Escaped character — emit the character after the backslash
			result.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == '"' {
			// Closing quote — ensure nothing meaningful follows
			rest := strings.TrimSpace(s[i+1:])
			if rest != "" {
				return "", fmt.Errorf("unexpected content after closing quote in %q", s)
			}
			return result.String(), nil
		}
		result.WriteByte(s[i])
		i++
	}

	return "", fmt.Errorf("unterminated quoted value: %q", s)
}

// resolveReplacements returns a copy of the replacement map with all values
// fully resolved. This handles chained references like:
//
//	params.x = "$(vars.y)" and vars.y = "hello" → params.x = "hello"
//
// so that conditions evaluate against final values rather than raw $(...)
// references. The resolution is best-effort with a recursion limit;
// unresolvable values are left as-is.
func resolveReplacements(replacements map[string]string) map[string]string {
	const maxIterations = 10

	resolved := make(map[string]string, len(replacements))
	for k, v := range replacements {
		resolved[k] = v
	}

	for i := 0; i < maxIterations; i++ {
		changed := false
		for k, v := range resolved {
			if !strings.Contains(v, "$(") {
				continue
			}
			newVal := v
			for rk, rv := range resolved {
				placeholder := "$(" + rk + ")"
				if strings.Contains(newVal, placeholder) {
					newVal = strings.ReplaceAll(newVal, placeholder, rv)
				}
				// Also handle whitespace variant: $( key )
				placeholderSpaced := "$( " + rk + " )"
				if strings.Contains(newVal, placeholderSpaced) {
					newVal = strings.ReplaceAll(newVal, placeholderSpaced, rv)
				}
			}
			if newVal != v {
				resolved[k] = newVal
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return resolved
}

// isStackActive returns true if all levels in the stack are active (or stack is empty).
func isStackActive(stack []conditionalState) bool {
	for _, s := range stack {
		if !s.active {
			return false
		}
	}
	return true
}

// isParentActive returns true if all levels except the top are active.
func isParentActive(stack []conditionalState) bool {
	if len(stack) <= 1 {
		return true
	}
	return isStackActive(stack[:len(stack)-1])
}
