package lang

import (
	"bufio"
	"log"
	"net/http"

	"github.com/roman-mazur/architecture-lab-3/painter" // Adjust import path
)

// HttpHandler creates an HTTP handler that parses commands and posts them to the loop.
func HttpHandler(loop *painter.Loop) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Printf("HTTP Handler: Method not allowed %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		scanner := bufio.NewScanner(r.Body)
		defer r.Body.Close()

		var ops []painter.Operation // Collect operations from the request

		for scanner.Scan() {
			commandLine := scanner.Text()
			log.Printf("HTTP Handler: Received command: %s", commandLine) // Log received command

			op, err := Parse(commandLine)
			if err != nil {
				log.Printf("HTTP Handler: Error parsing command '%s': %v", commandLine, err)
				// Decide whether to stop processing or just skip the bad line
				// http.Error(w, "Error parsing command: "+err.Error(), http.StatusBadRequest)
				// return
				continue // Skip this line and process others
			}
			if op != nil {
				ops = append(ops, op)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("HTTP Handler: Error reading request body: %v", err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		// Post all parsed operations to the loop
		// Note: Posting as a single list might be better with OperationList
		// loop.Post(painter.OperationList(ops)) // If OperationList is implemented
		// Or post one by one:
		for _, op := range ops {
			loop.Post(op)
		}

		log.Printf("HTTP Handler: Successfully processed %d operations", len(ops))
		w.WriteHeader(http.StatusOK) // Send OK response
		w.Write([]byte("Commands processed\n"))

	}
}
