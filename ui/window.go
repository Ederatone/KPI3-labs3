package ui

import (
	"image/color"
	"log"

	"github.com/roman-mazur/architecture-lab-3/painter" // Перевірте правильність шляху імпорту

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

// Visualizer manages the application window and displays textures.
type Visualizer struct {
	Title         string
	Width, Height int
	// ЗМІНЕНО ТИП ПОЛЯ НА ВКАЗІВНИК
	Loop *painter.Loop // Reference to the painter loop for posting events

	// Shiny specific fields
	pw screen.Window  // The window handle
	tx screen.Texture // Current texture to display
	sz size.Event     // Current window size

	// Function to be called inside driver.Main to start the painter loop
	StartLoopAndRunUI func(s screen.Screen)

	// Optional: To signal when Update has finished processing (не використовується зараз)
	// updateDone chan struct{}
}

// Update receives a texture from the painter loop and schedules a repaint.
// Цей метод реалізує інтерфейс painter.Receiver.
func (v *Visualizer) Update(t screen.Texture) {
	if t == nil {
		log.Println("Visualizer.Update: Received nil texture, ignoring.")
		return
	}
	v.tx = t // Store the reference to the current texture
	if v.pw != nil {
		// Надіслати подію paint.Event до черги подій вікна.
		// Це неблокуюча операція.
		v.pw.Send(paint.Event{})
	} else {
		log.Println("Visualizer.Update: Window handle (pw) is nil, cannot send paint event.")
	}
}

// Main starts the UI driver loop.
func (v *Visualizer) Main() {
	// v.updateDone = make(chan struct{}) // Ініціалізація, якщо канал потрібен

	driver.Main(func(s screen.Screen) {
		// Створюємо нове вікно
		w, err := s.NewWindow(&screen.NewWindowOptions{
			Title:  v.Title,
			Width:  v.Width,
			Height: v.Height,
		})
		if err != nil {
			log.Fatalf("Failed to create window: %v", err)
			return
		}
		// Гарантуємо звільнення ресурсів вікна при виході з driver.Main
		defer func() {
			log.Println("Releasing window resources...")
			w.Release()
			log.Println("Window resources released.")
		}()

		v.pw = w // Зберігаємо хендл вікна

		// Запускаємо painter loop, якщо функція StartLoopAndRunUI надана.
		// Це відбувається ПІСЛЯ створення вікна та отримання screen.Screen.
		if v.StartLoopAndRunUI != nil {
			log.Println("Visualizer.Main: Calling StartLoopAndRunUI...")
			v.StartLoopAndRunUI(s) // Викликаємо функцію, передану з main.go
		} else {
			log.Println("Warning: Visualizer.StartLoopAndRunUI is nil, painter loop might not start correctly.")
		}

		// Головний цикл обробки подій вікна
		for {
			// Отримуємо наступну подію з черги вікна
			evt := w.NextEvent()

			// Обробляємо різні типи подій
			switch e := evt.(type) {
			case lifecycle.Event:
				// Обробка подій життєвого циклу вікна (закриття, видимість)
				if e.To == lifecycle.StageDead {
					log.Println("Lifecycle: StageDead - Exiting UI loop")
					// За бажанням, тут можна надіслати сигнал зупинки painter loop
					// if v.Loop != nil {
					//     v.Loop.Stop() // Або спеціальну StopOp
					// }
					return // Вихід з циклу подій та driver.Main
				}
				// Якщо вікно стає видимим, запитуємо перемальовку
				if e.Crosses(lifecycle.StageVisible) == lifecycle.CrossOn {
					v.pw.Send(paint.Event{})
				}

			case size.Event:
				// Обробка зміни розміру вікна
				log.Printf("Size Event: New size %+v", e)
				v.sz = e                 // Оновлюємо збережену інформацію про розмір
				v.pw.Send(paint.Event{}) // Запитуємо перемальовку після зміни розміру

			case paint.Event:
				// Обробка запитів на перемальовку
				if v.pw == nil {
					log.Println("Paint Event: Window handle (pw) is nil.")
					continue
				}
				if v.tx != nil {
					// Малюємо поточну текстуру (v.tx) на вікно (v.pw)
					v.pw.Scale(v.sz.Bounds(), v.tx, v.tx.Bounds(), screen.Src, nil)
				} else {
					// Якщо текстури немає, заповнюємо вікно чорним кольором
					v.pw.Fill(v.sz.Bounds(), color.Black, screen.Src)
				}
				// Публікуємо зміни, щоб вони стали видимими
				v.pw.Publish()

			case mouse.Event:
				// Обробка подій миші (Варіант 23: права кнопка)
				if e.Button == mouse.ButtonRight && e.Direction == mouse.DirPress {
					// Перевіряємо, чи доступні розміри вікна
					if v.sz.WidthPx == 0 || v.sz.HeightPx == 0 {
						log.Println("Mouse Event: Window size not yet available, ignoring click.")
						continue
					}

					// Конвертуємо піксельні координати миші у відносні (0.0 до 1.0)
					relX := float64(e.X) / float64(v.sz.WidthPx)
					relY := float64(e.Y) / float64(v.sz.HeightPx)

					// Обмежуємо значення діапазоном [0, 1]
					if relX < 0 {
						relX = 0
					} else if relX > 1 {
						relX = 1
					}
					if relY < 0 {
						relY = 0
					} else if relY > 1 {
						relY = 1
					}

					// Перевіряємо, чи ініціалізовано painter loop
					if v.Loop != nil {
						log.Printf("Mouse Event: Right Button Press at pixel (%.0f, %.0f), relative (%.2f, %.2f)", e.X, e.Y, relX, relY)

						// Надсилаємо операції до painter loop
						v.Loop.Post(painter.Figure{X: relX, Y: relY}) // Додати фігуру в точці кліку
						v.Loop.Post(painter.UpdateOp{})               // Запросити оновлення вікна
					} else {
						log.Println("Error: Visualizer.Loop is nil, cannot post mouse event.")
					}
				}

			case key.Event:
				// Обробка подій клавіатури (вихід по Escape)
				if e.Code == key.CodeEscape {
					log.Println("Key Event: Escape pressed - Exiting")
					return // Вихід з циклу подій та driver.Main
				}

			case error:
				// Обробка системних помилок
				log.Printf("System Error Event: %v", e)
				// Розгляньте можливість виходу з програми при серйозних помилках
				// return

			default:
				// Інші типи подій ігноруються
				// log.Printf("Ignored Event: %T", e)
			} // end switch
		} // end for: event loop
	}) // end driver.Main

	log.Println("UI Main loop finished.")
} // end func Main
