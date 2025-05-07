package lang

import (
	"errors"
	"strconv"
	"strings"

	"github.com/roman-mazur/architecture-lab-3/painter" // Adjust import path
)

// Parse parses a single command line into a painter.Operation.
func Parse(commandLine string) (painter.Operation, error) {
	fields := strings.Fields(commandLine)
	if len(fields) == 0 {
		return nil, errors.New("empty command")
	}

	command := fields[0]
	args := fields[1:]

	switch command {
	case "white":
		if len(args) != 0 {
			return nil, errors.New("white command takes no arguments")
		}
		return painter.WhiteBg{}, nil
	case "green":
		if len(args) != 0 {
			return nil, errors.New("green command takes no arguments")
		}
		return painter.GreenBg{}, nil
	case "bgrect":
		if len(args) != 4 {
			return nil, errors.New("bgrect command requires 4 arguments (x1 y1 x2 y2)")
		}
		coords := make([]float64, 4)
		for i, arg := range args {
			val, err := strconv.ParseFloat(arg, 64)
			if err != nil {
				return nil, errors.New("invalid coordinate for bgrect: " + arg)
			}
			if val < 0 || val > 1 {
				return nil, errors.New("coordinate out of range (0.0-1.0) for bgrect: " + arg)
			}
			coords[i] = val
		}
		return painter.BgRect{X1: coords[0], Y1: coords[1], X2: coords[2], Y2: coords[3]}, nil
	case "figure":
		if len(args) != 2 {
			return nil, errors.New("figure command requires 2 arguments (x y)")
		}
		coords := make([]float64, 2)
		for i, arg := range args {
			val, err := strconv.ParseFloat(arg, 64)
			if err != nil {
				return nil, errors.New("invalid coordinate for figure: " + arg)
			}
			if val < 0 || val > 1 {
				return nil, errors.New("coordinate out of range (0.0-1.0) for figure: " + arg)
			}
			coords[i] = val
		}
		return painter.Figure{X: coords[0], Y: coords[1]}, nil
	case "move":
		if len(args) != 2 {
			return nil, errors.New("move command requires 2 arguments (x y)")
		}
		coords := make([]float64, 2)
		for i, arg := range args {
			val, err := strconv.ParseFloat(arg, 64)
			if err != nil {
				return nil, errors.New("invalid offset for move: " + arg)
			}
			// Note: move offsets can theoretically be outside 0-1 range
			coords[i] = val
		}
		return painter.Move{X: coords[0], Y: coords[1]}, nil
	case "reset":
		if len(args) != 0 {
			return nil, errors.New("reset command takes no arguments")
		}
		return painter.Reset{}, nil
	case "update":
		if len(args) != 0 {
			return nil, errors.New("update command takes no arguments")
		}
		return painter.UpdateOp{}, nil
	default:
		return nil, errors.New("unknown command: " + command)
	}
}

// ParseCommands parses multiple command lines from a reader.
// (This might be better placed in http.go, but keeping similar structure)
/*
func ParseCommands(r io.Reader) ([]painter.Operation, error) {
	scanner := bufio.NewScanner(r)
	var ops []painter.Operation
	var errors []string

	for scanner.Scan() {
		commandLine := scanner.Text()
		op, err := Parse(commandLine)
		if err != nil {
			errors = append(errors, "line \""+commandLine+"\": "+err.Error())
			continue // Skip invalid lines or decide on error handling
		}
		if op != nil { // Parse might return nil for empty/comment lines if modified
			ops = append(ops, op)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading commands: %w", err)
	}

	if len(errors) > 0 {
		// Return partial results and aggregated error, or just the error
		return ops, fmt.Errorf("errors parsing commands:\n%s", strings.Join(errors, "\n"))
	}

	return ops, nil
}
*/
