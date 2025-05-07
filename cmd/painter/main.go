// cmd/painter/main.go

package main

import (
	"log"
	"net/http"

	"github.com/roman-mazur/architecture-lab-3/painter"
	"github.com/roman-mazur/architecture-lab-3/painter/lang"
	"github.com/roman-mazur/architecture-lab-3/ui"

	"golang.org/x/exp/shiny/screen"
)

const (
	WindowWidth  = 800
	WindowHeight = 800
	HttpPort     = ":17000"
)

func main() {
	log.Println("Starting Painter Application...")

	// 1. Ініціалізуємо Visualizer БЕЗ Loop на цьому етапі
	visualizer := &ui.Visualizer{
		Title:  "Painter Lab 3 - Variant 23",
		Width:  WindowWidth,
		Height: WindowHeight,
		// Loop тут поки що nil
	}

	// 2. Ініціалізуємо Painter Loop, передаючи visualizer як Receiver
	// NewLoop повертає *painter.Loop
	painterLoop := painter.NewLoop(visualizer, WindowWidth, WindowHeight)

	// 3. Встановлюємо ВКАЗІВНИК на painterLoop у visualizer
	visualizer.Loop = painterLoop

	// 4. Ініціалізуємо HTTP обробник, передаючи ВКАЗІВНИК на painterLoop
	httpHandler := lang.HttpHandler(painterLoop) // Припускаємо, що HttpHandler приймає *painter.Loop
	go func() {
		log.Printf("Starting HTTP server on port %s", HttpPort)
		err := http.ListenAndServe(HttpPort, httpHandler)
		if err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// 5. Визначаємо функцію для відкладеного запуску Loop
	// Замикання захопить ВКАЗІВНИК painterLoop
	visualizer.StartLoopAndRunUI = func(s screen.Screen) {
		log.Println("StartLoopAndRunUI: Starting painter loop...")
		painterLoop.Start(s) // Start захопить правильний painterLoop
	}

	// 6. Запускаємо головний цикл UI
	visualizer.Main() // Цей виклик тепер запустить і painterLoop всередині

	log.Println("Painter Application Closed.")
}
