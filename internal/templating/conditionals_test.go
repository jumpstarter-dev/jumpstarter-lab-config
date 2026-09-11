package templating

import (
	"strings"
	"testing"
)

// --- Basic presence tests ---

func TestConditionals_IfPresent_Included(t *testing.T) {
	input := "before\n$if( params.key )\nincluded\n$endif\nafter"
	replacements := map[string]string{"params.key": "value"}
	expected := "before\nincluded\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_IfAbsent_Excluded(t *testing.T) {
	input := "before\n$if( params.key )\nexcluded\n$endif\nafter"
	replacements := map[string]string{}
	expected := "before\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_IfEmpty_Excluded(t *testing.T) {
	input := "before\n$if( params.key )\nexcluded\n$endif\nafter"
	replacements := map[string]string{"params.key": ""}
	expected := "before\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- If/Else tests ---

func TestConditionals_IfElse_TrueBranch(t *testing.T) {
	input := "before\n$if( params.key )\ntrue-branch\n$else\nfalse-branch\n$endif\nafter"
	replacements := map[string]string{"params.key": "yes"}
	expected := "before\ntrue-branch\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_IfElse_FalseBranch(t *testing.T) {
	input := "before\n$if( params.key )\ntrue-branch\n$else\nfalse-branch\n$endif\nafter"
	replacements := map[string]string{}
	expected := "before\nfalse-branch\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- If/Elif/Else tests ---

func TestConditionals_IfElifElse_FirstMatch(t *testing.T) {
	input := "$if( params.a )\nbranch-a\n$elif( params.b )\nbranch-b\n$else\nbranch-else\n$endif"
	replacements := map[string]string{"params.a": "yes", "params.b": "yes"}
	expected := "branch-a"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_IfElifElse_SecondMatch(t *testing.T) {
	input := "$if( params.a )\nbranch-a\n$elif( params.b )\nbranch-b\n$else\nbranch-else\n$endif"
	replacements := map[string]string{"params.b": "yes"}
	expected := "branch-b"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_IfElifElse_ElseBranch(t *testing.T) {
	input := "$if( params.a )\nbranch-a\n$elif( params.b )\nbranch-b\n$else\nbranch-else\n$endif"
	replacements := map[string]string{}
	expected := "branch-else"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_IfElif_NoMatch_NoElse(t *testing.T) {
	input := "before\n$if( params.a )\nbranch-a\n$elif( params.b )\nbranch-b\n$endif\nafter"
	replacements := map[string]string{}
	expected := "before\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Negation tests ---

func TestConditionals_NegationAbsent(t *testing.T) {
	input := "$if( !params.key )\nincluded\n$endif"
	replacements := map[string]string{}
	expected := "included"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_NegationPresent(t *testing.T) {
	input := "$if( !params.key )\nexcluded\n$endif"
	replacements := map[string]string{"params.key": "value"}
	expected := ""
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_NegationEmpty(t *testing.T) {
	input := "$if( !params.key )\nincluded\n$endif"
	replacements := map[string]string{"params.key": ""}
	expected := "included"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Equality tests ---

func TestConditionals_EqualityMatch(t *testing.T) {
	input := "$if( params.type == \"snmp\" )\nsnmp-block\n$endif"
	replacements := map[string]string{"params.type": "snmp"}
	expected := "snmp-block"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_EqualityNoMatch(t *testing.T) {
	input := "$if( params.type == \"snmp\" )\nsnmp-block\n$endif"
	replacements := map[string]string{"params.type": "ipmi"}
	expected := ""
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_EqualityKeyAbsent(t *testing.T) {
	input := "$if( params.type == \"snmp\" )\nsnmp-block\n$endif"
	replacements := map[string]string{}
	expected := ""
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_InequalityMatch(t *testing.T) {
	input := "$if( params.type != \"snmp\" )\nnon-snmp-block\n$endif"
	replacements := map[string]string{"params.type": "ipmi"}
	expected := "non-snmp-block"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_InequalityNoMatch(t *testing.T) {
	input := "$if( params.type != \"snmp\" )\nnon-snmp-block\n$endif"
	replacements := map[string]string{"params.type": "snmp"}
	expected := ""
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_InequalityKeyAbsent(t *testing.T) {
	input := "$if( params.type != \"snmp\" )\nnon-snmp-block\n$endif"
	replacements := map[string]string{}
	expected := "non-snmp-block"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Nesting tests ---

func TestConditionals_Nested_BothActive(t *testing.T) {
	input := "$if( params.a )\nouter\n$if( params.b )\ninner\n$endif\nouter-after\n$endif"
	replacements := map[string]string{"params.a": "yes", "params.b": "yes"}
	expected := "outer\ninner\nouter-after"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_Nested_OuterActive_InnerInactive(t *testing.T) {
	input := "$if( params.a )\nouter\n$if( params.b )\ninner\n$endif\nouter-after\n$endif"
	replacements := map[string]string{"params.a": "yes"}
	expected := "outer\nouter-after"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_Nested_OuterInactive(t *testing.T) {
	input := "$if( params.a )\nouter\n$if( params.b )\ninner\n$endif\nouter-after\n$endif"
	replacements := map[string]string{"params.b": "yes"}
	expected := ""
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_TripleNested(t *testing.T) {
	input := "$if( params.a )\nlevel1\n$if( params.b )\nlevel2\n$if( params.c )\nlevel3\n$endif\n$endif\n$endif"
	replacements := map[string]string{"params.a": "1", "params.b": "2", "params.c": "3"}
	expected := "level1\nlevel2\nlevel3"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Edge cases ---

func TestConditionals_NoDirectives_Passthrough(t *testing.T) {
	input := "line1\nline2\nline3"
	replacements := map[string]string{"params.key": "value"}
	expected := input
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_EmptyInput(t *testing.T) {
	input := ""
	replacements := map[string]string{}
	expected := ""
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_VarsNamespace(t *testing.T) {
	input := "$if( vars.debug )\ndebug-mode\n$endif"
	replacements := map[string]string{"vars.debug": "true"}
	expected := "debug-mode"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_DirectiveWithLeadingWhitespace(t *testing.T) {
	input := "before\n  $if( params.key )\n  included\n  $endif\nafter"
	replacements := map[string]string{"params.key": "value"}
	expected := "before\n  included\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_MultipleIfBlocks(t *testing.T) {
	input := "$if( params.a )\nblock-a\n$endif\n$if( params.b )\nblock-b\n$endif"
	replacements := map[string]string{"params.a": "yes"}
	expected := "block-a"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_MultipleIfBlocks_BothPresent(t *testing.T) {
	input := "$if( params.a )\nblock-a\n$endif\nmiddle\n$if( params.b )\nblock-b\n$endif"
	replacements := map[string]string{"params.a": "yes", "params.b": "yes"}
	expected := "block-a\nmiddle\nblock-b"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Multiline block with YAML content ---

func TestConditionals_YAMLBlock(t *testing.T) {
	input := `export:
  serial:
    type: "PySerial"
$if( params.pdu_host )
  power:
    type: "SNMPServer"
    config:
      host: "$( params.pdu_host )"
$endif
  ssh:
    type: "TcpNetwork"`
	replacements := map[string]string{"params.pdu_host": "10.0.0.1"}
	expected := `export:
  serial:
    type: "PySerial"
  power:
    type: "SNMPServer"
    config:
      host: "$( params.pdu_host )"
  ssh:
    type: "TcpNetwork"`
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

func TestConditionals_YAMLBlock_Excluded(t *testing.T) {
	input := `export:
  serial:
    type: "PySerial"
$if( params.pdu_host )
  power:
    type: "SNMPServer"
    config:
      host: "$( params.pdu_host )"
$endif
  ssh:
    type: "TcpNetwork"`
	replacements := map[string]string{}
	expected := `export:
  serial:
    type: "PySerial"
  ssh:
    type: "TcpNetwork"`
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
	}
}

// --- Value-based switching with YAML ---

func TestConditionals_ValueSwitch_YAML(t *testing.T) {
	input := `export:
$if( params.power_type == "snmp" )
  power:
    type: "SNMPServer"
$elif( params.power_type == "ipmi" )
  power:
    type: "IPMIServer"
$else
  # no power driver
$endif`

	t.Run("snmp", func(t *testing.T) {
		replacements := map[string]string{"params.power_type": "snmp"}
		expected := "export:\n  power:\n    type: \"SNMPServer\""
		result, err := processConditionals(input, replacements)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
		}
	})

	t.Run("ipmi", func(t *testing.T) {
		replacements := map[string]string{"params.power_type": "ipmi"}
		expected := "export:\n  power:\n    type: \"IPMIServer\""
		result, err := processConditionals(input, replacements)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
		}
	})

	t.Run("none", func(t *testing.T) {
		replacements := map[string]string{"params.power_type": "other"}
		expected := "export:\n  # no power driver"
		result, err := processConditionals(input, replacements)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected:\n%s\n\ngot:\n%s", expected, result)
		}
	})
}

// --- Error tests ---

func TestConditionals_Error_ElseWithoutIf(t *testing.T) {
	input := "before\n$else\nafter"
	_, err := processConditionals(input, map[string]string{})
	if err == nil {
		t.Fatal("expected error for $else without $if")
	}
	if !strings.Contains(err.Error(), "$else without matching $if") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConditionals_Error_EndifWithoutIf(t *testing.T) {
	input := "before\n$endif\nafter"
	_, err := processConditionals(input, map[string]string{})
	if err == nil {
		t.Fatal("expected error for $endif without $if")
	}
	if !strings.Contains(err.Error(), "$endif without matching $if") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConditionals_Error_ElifWithoutIf(t *testing.T) {
	input := "before\n$elif( params.key )\nafter"
	_, err := processConditionals(input, map[string]string{})
	if err == nil {
		t.Fatal("expected error for $elif without $if")
	}
	if !strings.Contains(err.Error(), "$elif without matching $if") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConditionals_Error_UnclosedIf(t *testing.T) {
	input := "$if( params.key )\ncontent"
	_, err := processConditionals(input, map[string]string{"params.key": "val"})
	if err == nil {
		t.Fatal("expected error for unclosed $if")
	}
	if !strings.Contains(err.Error(), "unclosed $if block") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConditionals_Error_EmptyCondition(t *testing.T) {
	input := "$if()\ncontent\n$endif"
	_, err := processConditionals(input, map[string]string{})
	if err == nil {
		t.Fatal("expected error for empty condition")
	}
	if !strings.Contains(err.Error(), "empty condition") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestConditionals_Error_InvalidQuotedValue(t *testing.T) {
	input := "$if( params.key == unquoted )\ncontent\n$endif"
	_, err := processConditionals(input, map[string]string{})
	if err == nil {
		t.Fatal("expected error for unquoted value in comparison")
	}
	if !strings.Contains(err.Error(), "expected quoted value") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- Multiple $elif branches ---

func TestConditionals_MultipleElif(t *testing.T) {
	input := "$if( params.x == \"a\" )\nA\n$elif( params.x == \"b\" )\nB\n$elif( params.x == \"c\" )\nC\n$else\nD\n$endif"

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"match-a", "a", "A"},
		{"match-b", "b", "B"},
		{"match-c", "c", "C"},
		{"match-else", "d", "D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacements := map[string]string{"params.x": tt.value}
			result, err := processConditionals(input, replacements)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// --- Meta namespace test ---

func TestConditionals_MetaNamespace(t *testing.T) {
	input := "$if( name )\nhas-name\n$endif"
	replacements := map[string]string{"name": "my-exporter"}
	expected := "has-name"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Nested if with elif in outer ---

func TestConditionals_NestedIfWithElifOuter(t *testing.T) {
	input := "$if( params.a )\nouter-a\n$if( params.b )\ninner-b\n$endif\n$elif( params.c )\nouter-c\n$endif"

	t.Run("a-and-b", func(t *testing.T) {
		replacements := map[string]string{"params.a": "1", "params.b": "2"}
		expected := "outer-a\ninner-b"
		result, err := processConditionals(input, replacements)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("a-no-b", func(t *testing.T) {
		replacements := map[string]string{"params.a": "1"}
		expected := "outer-a"
		result, err := processConditionals(input, replacements)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("c-only", func(t *testing.T) {
		replacements := map[string]string{"params.c": "3"}
		expected := "outer-c"
		result, err := processConditionals(input, replacements)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

// ==========================================================================
// Fix #2: Content lines starting with $if( treated as content, not directives
// ==========================================================================

func TestConditionals_ContentLineStartingWithDollarIf(t *testing.T) {
	// A line that starts with $if( but has non-comment content after the )
	// should be treated as regular content, not as a directive.
	input := `before
$if(something) do stuff
after`
	replacements := map[string]string{}
	expected := "before\n$if(something) do stuff\nafter"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_ContentLineMalformedIfNoParen(t *testing.T) {
	// A line that starts with $if( but has no closing ) — treat as content.
	input := "$if( params.key\ncontent\nmore content"
	replacements := map[string]string{}
	expected := "$if( params.key\ncontent\nmore content"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_ContentLineEchoWithDollarIf(t *testing.T) {
	// A shell command in a config template that happens to contain $if(
	input := `script: |
  echo "$if(condition) is true"
  echo "done"`
	replacements := map[string]string{}
	expected := input
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_CommentLineLookingLikeDirective(t *testing.T) {
	// A YAML comment that mentions $if syntax — should be treated as content
	// because trimmed starts with # not $if(
	input := "# $if( params.key ) is used for conditionals\ncontent"
	replacements := map[string]string{}
	expected := input
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// ==========================================================================
// Fix #2: Trailing # comments on directives
// ==========================================================================

func TestConditionals_DirectiveWithTrailingComment(t *testing.T) {
	input := "$if( params.key ) # enable power section\npower: on\n$endif # end power"
	replacements := map[string]string{"params.key": "yes"}
	expected := "power: on"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_ElseWithTrailingComment(t *testing.T) {
	input := "$if( params.key )\ntrue\n$else # fallback\nfalse\n$endif"
	replacements := map[string]string{}
	expected := "false"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_ElifWithTrailingComment(t *testing.T) {
	input := "$if( params.a )\nA\n$elif( params.b ) # try b\nB\n$endif"
	replacements := map[string]string{"params.b": "yes"}
	expected := "B"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_TrailingCommentWithParenthesis(t *testing.T) {
	// A trailing comment containing ')' should not break directive parsing.
	// Previously, LastIndex(")") would find the ')' inside the comment.
	input := "$if( params.key ) # enable power (optional)\npower: on\n$endif"
	replacements := map[string]string{"params.key": "yes"}
	expected := "power: on"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_TrailingCommentWithParenthesisAbsent(t *testing.T) {
	// Same as above but param is absent — block should be excluded.
	input := "$if( params.key ) # enable power (optional)\npower: on\n$endif"
	replacements := map[string]string{}
	expected := ""
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_TrailingCommentWithEqualityAndParenthesis(t *testing.T) {
	// Equality check with a comment containing parentheses
	input := "$if( params.type == \"snmp\" ) # use SNMP (default)\nsnmp\n$endif"
	replacements := map[string]string{"params.type": "snmp"}
	expected := "snmp"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// ==========================================================================
// Fix #3: Duplicate $else detection
// ==========================================================================

func TestConditionals_Error_DuplicateElse(t *testing.T) {
	input := "$if( params.a )\nA\n$else\nB\n$else\nC\n$endif"
	_, err := processConditionals(input, map[string]string{"params.a": "yes"})
	if err == nil {
		t.Fatal("expected error for duplicate $else")
	}
	if !strings.Contains(err.Error(), "duplicate $else") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ==========================================================================
// Fix #8: $elif after $else detection
// ==========================================================================

func TestConditionals_Error_ElifAfterElse(t *testing.T) {
	input := "$if( params.a )\nA\n$else\nB\n$elif( params.c )\nC\n$endif"
	_, err := processConditionals(input, map[string]string{})
	if err == nil {
		t.Fatal("expected error for $elif after $else")
	}
	if !strings.Contains(err.Error(), "$elif after $else") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ==========================================================================
// Fix #1: Conditions evaluate resolved (substituted) values
// ==========================================================================

func TestConditionals_ResolvedRecursiveParam(t *testing.T) {
	// params.power_type references $(vars.default_power) which is "snmp".
	// The condition should evaluate against the resolved value "snmp".
	replacements := map[string]string{
		"params.power_type":  "$(vars.default_power)",
		"vars.default_power": "snmp",
	}
	input := "$if( params.power_type == \"snmp\" )\nsnmp-block\n$endif"
	expected := "snmp-block"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_ResolvedRecursiveParamPresence(t *testing.T) {
	// params.pdu_host references $(vars.pdu_addr) which has a value.
	// Presence check should see the resolved non-empty value.
	replacements := map[string]string{
		"params.pdu_host": "$(vars.pdu_addr)",
		"vars.pdu_addr":   "10.0.0.1",
	}
	input := "$if( params.pdu_host )\npower-block\n$endif"
	expected := "power-block"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_ResolvedChainedReferences(t *testing.T) {
	// Three levels deep: params.x -> $(vars.y) -> $(vars.z) -> "final"
	replacements := map[string]string{
		"params.x": "$(vars.y)",
		"vars.y":   "$(vars.z)",
		"vars.z":   "final",
	}
	input := "$if( params.x == \"final\" )\nresolved\n$endif"
	expected := "resolved"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_ResolvedWithSpacedRef(t *testing.T) {
	// Handle the $( key ) spaced variant used in real templates
	replacements := map[string]string{
		"params.pdu_user": "$( vars.pdu_user )",
		"vars.pdu_user":   "admin",
	}
	input := "$if( params.pdu_user == \"admin\" )\nadmin-block\n$endif"
	expected := "admin-block"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// ==========================================================================
// Fix #5: Escaped quotes in comparison values
// ==========================================================================

func TestConditionals_EscapedQuotesInValue(t *testing.T) {
	input := "$if( params.msg == \"say \\\"hello\\\"\" )\nmatched\n$endif"
	replacements := map[string]string{"params.msg": `say "hello"`}
	expected := "matched"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_EscapedBackslashInValue(t *testing.T) {
	input := "$if( params.path == \"C:\\\\Users\" )\nmatched\n$endif"
	replacements := map[string]string{"params.path": `C:\Users`}
	expected := "matched"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// ==========================================================================
// Fix #7: Trailing newline preservation
// ==========================================================================

func TestConditionals_TrailingNewlinePreserved(t *testing.T) {
	// Input ends with \n — output should also end with \n
	input := "line1\nline2\n"
	replacements := map[string]string{}
	expected := "line1\nline2\n"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_NoTrailingNewlinePreserved(t *testing.T) {
	// Input does NOT end with \n — output should not either
	input := "line1\nline2"
	replacements := map[string]string{}
	expected := "line1\nline2"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_TrailingNewlineWhenLastLineIsDirective(t *testing.T) {
	// Input ends with $endif\n — output should preserve the trailing newline
	input := "content\n$if( params.key )\nextra\n$endif\n"
	replacements := map[string]string{}
	expected := "content\n"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConditionals_NoTrailingNewlineWhenLastLineIsDirective(t *testing.T) {
	// Input ends with $endif (no trailing \n) — output should not have one
	input := "content\n$if( params.key )\nextra\n$endif"
	replacements := map[string]string{}
	expected := "content"
	result, err := processConditionals(input, replacements)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
