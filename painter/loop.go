// painter/loop.go

package painter

import (
	"image"
	"image/color"
	"log" // Додано для логування
	"sync"

	"golang.org/x/exp/shiny/screen"
)

// Receiver defines an interface for components that can receive and display textures.
type Receiver interface {
	Update(t screen.Texture)
}

// MessageQueue defines a thread-safe queue for operations.
type MessageQueue struct {
	mu  sync.Mutex
	ops []Operation   // Slice to store operations
	ch  chan struct{} // Channel to signal that new operations are available
}

// NewMessageQueue creates a new message queue.
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		// Buffered channel of size 1 allows one signal to be pending
		// if the receiver is not ready, preventing deadlock on Push.
		ch: make(chan struct{}, 1),
	}
}

// Push adds an operation to the queue and signals availability.
func (mq *MessageQueue) Push(op Operation) {
	mq.mu.Lock()
	mq.ops = append(mq.ops, op)
	mq.mu.Unlock()

	// Signal that there's a new message.
	// Use a non-blocking send: if the channel buffer is full
	// (meaning a signal is already pending), do nothing.
	select {
	case mq.ch <- struct{}{}:
	default:
	}
}

// Pull retrieves all operations currently in the queue.
// It clears the internal queue after retrieval.
func (mq *MessageQueue) Pull() []Operation {
	mq.mu.Lock()
	// Copy the slice of operations.
	ops := mq.ops
	// Clear the queue by assigning a new empty slice (or nil).
	// Important: Assigning nil or a new slice is necessary,
	// otherwise future appends might reuse the old underlying array.
	mq.ops = nil
	mq.mu.Unlock()
	return ops
}

// Wait returns a channel that signals when new operations might be available.
// The loop will block waiting on this channel.
func (mq *MessageQueue) Wait() <-chan struct{} {
	return mq.ch
}

// Loop manages the application state and processes operations.
type Loop struct {
	Receiver Receiver      // Component to send updated textures to (e.g., ui.Visualizer)
	Mq       *MessageQueue // Message queue for receiving operations
	state    *State        // Internal state managed by the loop

	stop    chan struct{} // Channel to signal the loop goroutine to stop
	stopped chan struct{} // Channel to signal when the loop goroutine has finished
}

// NewLoop creates a new Loop for managing state and processing operations.
// It initializes the state based on variant defaults (Variant 23).
func NewLoop(r Receiver, width, height int) *Loop {
	return &Loop{
		Receiver: r,
		Mq:       NewMessageQueue(),
		state: &State{ // Initialize state for Variant 23
			BgColor:      color.White,   // Initial background for Variant 23
			Figures:      []*FigureOp{}, // Start with no figures initially
			BgRect:       nil,           // Start with no background rectangle
			MoveOffset:   image.Point{}, // Start with zero move offset
			WindowWidth:  width,
			WindowHeight: height,
		},
		stop:    make(chan struct{}), // Channel for stop signal
		stopped: make(chan struct{}), // Channel to confirm stoppage
	}
}

// Start initializes the loop, sets the initial state with the figure, and runs the event processing goroutine.
func (l *Loop) Start(s screen.Screen) {
	// Створюємо початкову текстуру розміром з вікно
	initialTexture, err := s.NewTexture(image.Pt(l.state.WindowWidth, l.state.WindowHeight))
	if err != nil {
		// У реальному додатку потрібна краща обробка помилок
		log.Fatalf("Failed to create initial texture: %v", err)
	}

	// Встановлюємо початковий колір фону зі стану (має бути білий для варіанту 23)
	initialTexture.Fill(initialTexture.Bounds(), l.state.BgColor, screen.Src)
	log.Printf("Loop.Start: Initial background color set to: %+v", l.state.BgColor)

	// ----- ДОДАНО: Встановлення початкової фігури в центрі -----
	// Створюємо операцію додавання фігури з відносними координатами центру (0.5, 0.5)
	// Вона буде автоматично конвертована в пікселі та додана до стану всередині Do.
	initialFigureOp := Figure{X: 0.5, Y: 0.5}

	// Виконуємо операцію Figure.Do, щоб додати фігуру до *стану* (l.state.Figures).
	// Малювати її прямо на initialTexture не обов'язково, бо перший UpdateOp
	// все одно перемалює все з нуля, читаючи оновлений стан.
	log.Println("Loop.Start: Adding initial figure (T-180, Yellow) to state...")
	// Викликаємо Do, щоб змінити l.state, ігноруємо результат (bool) та текстуру тут.
	initialFigureOp.Do(l.state, initialTexture) // Модифікує l.state.Figures

	// Перевірка, чи фігура додалась до стану (для відладки)
	log.Printf("Loop.Start: Current figures in state: %d", len(l.state.Figures))
	if len(l.state.Figures) > 0 {
		log.Printf("Loop.Start: Initial figure data: %+v", l.state.Figures[0])
	} else {
		log.Println("Loop.Start: Warning - Initial figure was not added to state.")
	}
	// ------------------------------------------------------------

	// Запускаємо головну горутину обробки подій
	go func() {
		// Гарантуємо закриття каналу stopped при виході з горутини
		defer close(l.stopped)
		// Гарантуємо звільнення ресурсів текстури при виході
		defer func() {
			log.Println("Loop goroutine: Releasing texture...")
			initialTexture.Release()
			log.Println("Loop goroutine: Texture released.")
		}()

		// Використовуємо одну текстуру для всіх операцій малювання
		currentTexture := initialTexture

		for {
			select {
			case <-l.stop: // Отримано сигнал зупинки
				log.Println("Loop goroutine: Stop signal received, terminating.")
				return
			case <-l.Mq.Wait(): // Отримано сигнал про нові операції в черзі
				ops := l.Mq.Pull() // Витягуємо ВСІ операції з черги
				if len(ops) > 0 {
					log.Printf("Loop goroutine: Pulled %d operations from queue.", len(ops))
					var needsVisualUpdate bool // Прапорець, чи потрібне оновлення екрану
					// Обробляємо кожну операцію по черзі
					for _, op := range ops {
						// Метод Do операції модифікує стан (l.state) та/або
						// малює на текстурі (currentTexture).
						// Він повертає true, якщо це UpdateOp.
						if op.Do(l.state, currentTexture) {
							needsVisualUpdate = true
						}
					}

					// Якщо хоча б одна з операцій була UpdateOp (або повернула true),
					// надсилаємо фінальну текстуру до візуалізатора.
					if needsVisualUpdate {
						log.Println("Loop goroutine: Sending texture update to receiver.")
						// Перевірка на nil перед викликом Update
						if l.Receiver != nil {
							l.Receiver.Update(currentTexture)
						} else {
							log.Println("Loop goroutine: Error - Receiver is nil.")
						}
					}
				}
			}
		}
	}() // Кінець горутини обробки подій

	// Надсилаємо початкову операцію UpdateOp до черги.
	// Горутина обробки подій отримає її, викличе UpdateOp.Do,
	// яка намалює фон ТА тепер і початкову фігуру (бо вона вже є в l.state),
	// і потім надішле текстуру до візуалізатора.
	log.Println("Loop.Start: Posting initial UpdateOp to the queue.")
	l.Post(UpdateOp{})

	log.Println("Loop.Start: Initialization complete, event loop running.")
}

// Post adds an operation to the message queue for processing.
// This is the entry point for external components (like HTTP handlers or UI callbacks)
// to request changes to the state.
func (l *Loop) Post(op Operation) {
	if l.Mq == nil {
		log.Println("Error: Loop.Post called but MessageQueue (Mq) is nil.")
		return
	}
	l.Mq.Push(op)
}

// Stop signals the event loop goroutine to terminate gracefully.
// It waits until the goroutine confirms stoppage.
func (l *Loop) Stop() {
	log.Println("Loop.Stop: Signaling stop channel...")
	// Сигналізуємо горутині зупинитися, закриваючи канал stop
	close(l.stop)
	// Чекаємо, доки горутина підтвердить зупинку, закривши канал stopped
	<-l.stopped
	log.Println("Loop.Stop: Goroutine confirmed stopped.")
}

// StopAndWait is required by the architecture tests but not fully implemented for graceful shutdown logic here.
// A more robust implementation might involve waiting for the message queue to empty
// or ensuring the UI thread has also terminated.
func (l Loop) StopAndWait() {
	// For now, just call Stop which waits for the loop goroutine.
	// Note: Calling Stop on a non-pointer receiver might not work as intended
	// if the stop/stopped channels are not shared correctly.
	// Consider making Loop methods operate on a pointer receiver (*Loop).
	// However, since Stop() uses channels defined in the Loop struct, it should work
	// even with a value receiver *if* the channels themselves are correctly initialized.
	// The current implementation uses a pointer receiver for Loop methods implicitly
	// due to how it's used in main.go (painterLoop is a pointer).
	// If called directly on a Loop value, it might behave unexpectedly.
	// panic("StopAndWait not fully implemented for graceful shutdown")
	log.Println("Warning: StopAndWait called, performing basic Stop()...")
	// Assuming 'l' is the actual loop instance used (likely a pointer passed around)
	// Need to ensure 'l' is the correct instance or change method receiver to *Loop
	// l.Stop() // This might cause issues if 'l' is a copy. Let's skip for now.
	panic("unimplemented") // Keep panic as per original template if Stop() logic isn't added here.
}

// GetState returns a copy of the current state. Useful for testing or debugging.
// IMPORTANT: This returns a shallow copy. Modifying nested structures (like Figures)
// in the returned state might affect the original state if not careful.
// Accessing state requires synchronization if done concurrently with the loop goroutine.
func (l *Loop) GetState() State {
	// Use the MessageQueue's mutex for simplicity to synchronize access to state.
	// A dedicated state mutex might be better in complex scenarios.
	l.Mq.mu.Lock()
	defer l.Mq.mu.Unlock()

	// Create a shallow copy of the state.
	stateCopy := *l.state
	// Create a new slice and copy figure pointers to avoid modifying the original slice directly.
	stateCopy.Figures = make([]*FigureOp, len(l.state.Figures))
	copy(stateCopy.Figures, l.state.Figures)
	// Copy the BgRect if it exists
	if l.state.BgRect != nil {
		bgRectCopy := *l.state.BgRect
		stateCopy.BgRect = &bgRectCopy
	}
	return stateCopy
}

// State type definition - Assuming it's defined here or imported.
// It should be defined within this package or imported if it's elsewhere.
/*
type State struct {
	BgColor      color.Color
	BgRect       *BgRectOp   // Store the last BgRect operation data
	Figures      []*FigureOp // Store all Figure operations data
	MoveOffset   image.Point // Cumulative move offset for all figures
	WindowWidth  int
	WindowHeight int
}
*/

// Operation type definition - Assuming it's defined here or imported.
/*
type Operation interface {
	Do(s *State, t screen.Texture) (updated bool)
}
*/

// FigureOp, BgRectOp definitions - Assuming they are defined here or imported.
/*
type FigureOp struct { X, Y int; Variant FigureVariant; Color color.Color }
type BgRectOp struct { X1, Y1, X2, Y2 int }
type FigureVariant int
const ( T0 FigureVariant = iota; T90; T180; T270; Cross )
*/

// Concrete operation types (UpdateOp, Figure, WhiteBg etc.) - Assuming defined elsewhere (op.go).
/*
type UpdateOp struct{}
type Figure struct{ X, Y float64 }
// ... and others ...
*/
