package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const url = "http://localhost:17000"

func main() {
	// Дані, які відправляються в тілі POST-запиту.
	// Використовуємо backticks для багаторядкового рядка.
	payload := `green
bgrect 0.1 0.1 0.9 0.9
update`

	fmt.Println("Sending green frame commands...")

	// Створюємо новий рідер з нашого рядка payload
	bodyReader := strings.NewReader(payload)

	// Виконуємо POST-запит
	// Перший аргумент - URL
	// Другий - Content-Type (важливо для сервера, щоб знати, як обробляти тіло запиту)
	// Третій - Тіло запиту (має бути io.Reader)
	resp, err := http.Post(url, "text/plain", bodyReader)
	if err != nil {
		log.Fatalf("Error making POST request: %v", err)
	}
	// Важливо закрити тіло відповіді, щоб уникнути витоку ресурсів
	defer resp.Body.Close()

	// Перевіряємо статус-код відповіді
	if resp.StatusCode != http.StatusOK {
		// Читаємо тіло відповіді для детальнішої інформації про помилку (опціонально)
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("Error reading response body after non-OK status: %v", readErr)
		}
		log.Fatalf("Server returned non-OK status: %s\nResponse body: %s", resp.Status, string(bodyBytes))
	}

	fmt.Println("Done.")
	// Опціонально: можна прочитати і вивести тіло відповіді, якщо сервер щось повертає
	// responseBody, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Printf("Error reading response body: %v", err)
	// } else {
	//  fmt.Printf("Server response: %s\n", string(responseBody))
	// }
}
