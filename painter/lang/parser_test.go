package lang_test // Use the _test package convention

import (
	"reflect" // Needed for DeepEqual comparison
	"testing"

	// Adjust these import paths to match your actual project structure/module path
	"github.com/roman-mazur/architecture-lab-3/painter"
	"github.com/roman-mazur/architecture-lab-3/painter/lang"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string            // Descriptive name for the test case
		commandLine string            // Input string to the Parse function
		expectedOp  painter.Operation // The expected operation struct (nil if error expected)
		expectError bool              // Whether an error is expected
	}{
		// --- Valid Cases ---
		{
			name:        "parse white command",
			commandLine: "white",
			expectedOp:  painter.WhiteBg{},
			expectError: false,
		},
		{
			name:        "parse green command",
			commandLine: "green",
			expectedOp:  painter.GreenBg{},
			expectError: false,
		},
		{
			name:        "parse update command",
			commandLine: "update",
			expectedOp:  painter.UpdateOp{},
			expectError: false,
		},
		{
			name:        "parse reset command",
			commandLine: "reset",
			expectedOp:  painter.Reset{},
			expectError: false,
		},
		{
			name:        "parse bgrect command valid coords",
			commandLine: "bgrect 0.1 0.2 0.8 0.9",
			expectedOp:  painter.BgRect{X1: 0.1, Y1: 0.2, X2: 0.8, Y2: 0.9},
			expectError: false,
		},
		{
			name:        "parse figure command valid coords",
			commandLine: "figure 0.55 0.45",
			expectedOp:  painter.Figure{X: 0.55, Y: 0.45},
			expectError: false,
		},
		{
			name:        "parse move command valid coords",
			commandLine: "move 0.1 -0.2", // Move coords can be anything
			expectedOp:  painter.Move{X: 0.1, Y: -0.2},
			expectError: false,
		},
		{
			name:        "parse command with extra spaces",
			commandLine: "  figure   0.3   0.7  ",
			expectedOp:  painter.Figure{X: 0.3, Y: 0.7},
			expectError: false,
		},

		// --- Error Cases ---
		{
			name:        "parse empty command line",
			commandLine: "",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse whitespace command line",
			commandLine: "   ",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse unknown command",
			commandLine: "unknowncmd 1 2 3",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse white with arguments",
			commandLine: "white 0.5",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse green with arguments",
			commandLine: "green arg",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse update with arguments",
			commandLine: "update extra",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse reset with arguments",
			commandLine: "reset 1",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse bgrect too few args",
			commandLine: "bgrect 0.1 0.2 0.8",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse bgrect too many args",
			commandLine: "bgrect 0.1 0.2 0.8 0.9 1.0",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse bgrect invalid coord type",
			commandLine: "bgrect 0.1 text 0.8 0.9",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse bgrect coord out of range (negative)",
			commandLine: "bgrect -0.1 0.2 0.8 0.9",
			expectedOp:  nil,
			expectError: true, // Assuming validation checks range 0-1
		},
		{
			name:        "parse bgrect coord out of range (greater than 1)",
			commandLine: "bgrect 0.1 0.2 1.8 0.9",
			expectedOp:  nil,
			expectError: true, // Assuming validation checks range 0-1
		},
		{
			name:        "parse figure too few args",
			commandLine: "figure 0.5",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse figure too many args",
			commandLine: "figure 0.5 0.6 0.7",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse figure invalid coord type",
			commandLine: "figure 0.5 abc",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse figure coord out of range (negative)",
			commandLine: "figure -0.1 0.5",
			expectedOp:  nil,
			expectError: true, // Assuming validation checks range 0-1
		},
		{
			name:        "parse figure coord out of range (greater than 1)",
			commandLine: "figure 0.5 1.1",
			expectedOp:  nil,
			expectError: true, // Assuming validation checks range 0-1
		},
		{
			name:        "parse move too few args",
			commandLine: "move 0.1",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse move too many args",
			commandLine: "move 0.1 0.2 0.3",
			expectedOp:  nil,
			expectError: true,
		},
		{
			name:        "parse move invalid coord type",
			commandLine: "move text 0.2",
			expectedOp:  nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { // Use t.Run for better test output
			op, err := lang.Parse(tt.commandLine)

			// Check if error expectation matches reality
			if tt.expectError {
				if err == nil {
					t.Errorf("Parse(%q) expected an error, but got nil", tt.commandLine)
				}
				// Optional: Check for specific error message if needed
				// if err != nil && !strings.Contains(err.Error(), "expected error text") {
				// 	t.Errorf("Parse(%q) expected error containing '...', got: %v", tt.commandLine, err)
				// }
			} else {
				if err != nil {
					t.Errorf("Parse(%q) expected no error, but got: %v", tt.commandLine, err)
				}
			}

			// Check if the returned operation matches the expected one
			// Use reflect.DeepEqual for reliable struct comparison
			if !reflect.DeepEqual(op, tt.expectedOp) {
				t.Errorf("Parse(%q) expected operation %+v, but got %+v", tt.commandLine, tt.expectedOp, op)
			}
		})
	}
}

// Optional: Add tests for ParseCommands if you implemented that function separately
/*
func TestParseCommands(t *testing.T) {
    // Similar table-driven structure, but input would be an io.Reader (e.g., strings.NewReader)
    // and expected output would be a slice of operations ([]painter.Operation)
}
*/
