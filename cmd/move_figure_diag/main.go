package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	url   = "http://localhost:17000"
	delay = 1 * time.Second // Затримка між кроками
	steps = 10              // Кількість кроків руху
	dx    = 0.02            // Відносне зміщення по X за крок
	dy    = 0.02            // Відносне зміщення по Y за крок
)

// sendCommands - допоміжна функція для відправки команд на сервер
func sendCommands(serverURL string, payload string) {
	bodyReader := strings.NewReader(payload)
	resp, err := http.Post(serverURL, "text/plain", bodyReader)
	if err != nil {
		log.Fatalf("Error making POST request to %s: %v", serverURL, err)
	}
	defer resp.Body.Close() // Важливо закрити тіло відповіді

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("Error reading response body after non-OK status: %v", readErr)
		}
		log.Fatalf("Server returned non-OK status: %s\nResponse body: %s", resp.Status, string(bodyBytes))
	}
	// Опціонально: можна прочитати та вивести тіло відповіді, якщо потрібно
	// io.Copy(io.Discard, resp.Body) // Просто читаємо, щоб з'єднання можна було перевикористати
}

func main() {
	fmt.Println("Setting initial state (White BG, Figure at 0.5, 0.5)...")
	// Важливо: Команда reset потрібна, щоб очистити попередній стан,
	// якщо сервер вже працював.
	// Ваш варіант 23 використовує білий фон, тому 'white'.
	// 'figure 0.5 0.5' розміщує фігуру в центрі.
	initialPayload := `reset
white
figure 0.5 0.5
update`
	sendCommands(url, initialPayload)

	time.Sleep(delay) // Чекаємо перед початком руху

	fmt.Println("Starting diagonal movement...")
	for i := 0; i < steps; i++ {
		// Формуємо рядок команди move. %.2f форматує float з 2 знаками після коми.
		// Важливо: Ми відправляємо відносні зміщення dx, dy згідно з логікою bash скрипта
		// і описом руху "по діагоналі". Сервер має обробляти 'move' як відносне зміщення.
		movePayload := fmt.Sprintf("move %.2f %.2f\nupdate", dx, dy)
		fmt.Printf("Step %d: Sending commands:\n%s", i+1, movePayload)

		sendCommands(url, movePayload) // Відправляємо команду руху та оновлення
		time.Sleep(delay)              // Чекаємо перед наступним кроком
	}

	fmt.Println("Movement finished.")
}
