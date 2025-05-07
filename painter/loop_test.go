package painter_test // Або package painter, якщо тести в тому ж пакеті

import (
	"image"
	"image/color"
	"image/draw"
	"sync" // Додаємо імпорт sync
	"testing"
	"time"

	// --- Обов'язково замініть цей шлях на ваш! ---
	"github.com/roman-mazur/architecture-lab-3/painter"
	// ------------------------------------------

	"github.com/stretchr/testify/assert" // Популярна бібліотека для тестів
	"golang.org/x/exp/shiny/screen"
)

// --- Правильне визначення OperationFunc ---

// OperationFunc визначає тип функції, який може діяти як Operation.
// Сигнатура цієї функції має відповідати методу Do інтерфейсу Operation.
type OperationFunc func(s *painter.State, t screen.Texture) bool

// Do реалізує метод інтерфейсу painter.Operation для типу OperationFunc.
// Тепер будь-яка функція типу OperationFunc автоматично задовольняє інтерфейс Operation.
func (f OperationFunc) Do(s *painter.State, t screen.Texture) bool {
	return f(s, t) // Просто викликаємо саму функцію f
}

// --- Допоміжні типи-заглушки для тестування ---

// mockReceiver - це заглушка для painter.Receiver, щоб відстежувати виклики Update.
type mockReceiver struct {
	mu            sync.Mutex
	updateCalls   int
	lastTexture   screen.Texture // Записується в Update
	updateBlocked chan struct{}
	updateDone    chan struct{}
}

func newMockReceiver() *mockReceiver {
	return &mockReceiver{
		updateDone: make(chan struct{}, 1), // Буферизований для неблокуючої відправки сигналу
	}
}

// Update записує дані безпечно (під м'ютексом) і сигналізує про завершення.
func (m *mockReceiver) Update(t screen.Texture) {
	m.mu.Lock()
	m.updateCalls++
	m.lastTexture = t // Запис під м'ютексом
	m.mu.Unlock()

	// Якщо потрібно заблокувати для тестування тайм-аутів (необов'язково)
	if m.updateBlocked != nil {
		<-m.updateBlocked
	}

	// Сигналізуємо, що Update завершився
	select {
	case m.updateDone <- struct{}{}:
	default: // Не панікувати, якщо канал вже повний
	}
}

// GetLastTexture безпечно повертає останню отриману текстуру.
func (m *mockReceiver) GetLastTexture() screen.Texture {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastTexture
}

// BlockUpdate створює канал для блокування наступних викликів Update.
func (m *mockReceiver) BlockUpdate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateBlocked == nil {
		m.updateBlocked = make(chan struct{})
	}
}

// UnlockUpdate закриває блокуючий канал, розблоковуючи Update.
func (m *mockReceiver) UnlockUpdate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateBlocked != nil {
		close(m.updateBlocked)
		m.updateBlocked = nil
	}
}

// WaitForUpdate чекає на сигнал завершення Update протягом заданого часу.
func (m *mockReceiver) WaitForUpdate(timeout time.Duration) bool {
	select {
	case <-m.updateDone:
		return true
	case <-time.After(timeout):
		return false
	}
}

// UpdateCalls безпечно повертає кількість викликів Update.
func (m *mockReceiver) UpdateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateCalls
}

// --- Заглушки для Shiny Screen/Texture/Buffer ---

// mockTexture - заглушка для screen.Texture
type mockTexture struct {
	mu                sync.Mutex // М'ютекс для захисту mockReleaseCalled
	mockReleaseCalled bool
	size              image.Point
}

func newMockTexture(size image.Point) *mockTexture {
	if size.X == 0 || size.Y == 0 {
		size = image.Pt(800, 800) // Розмір за замовчуванням
	}
	return &mockTexture{size: size}
}

// Release безпечно встановлює прапорець mockReleaseCalled.
func (m *mockTexture) Release() {
	m.mu.Lock()
	m.mockReleaseCalled = true
	m.mu.Unlock()
}

func (m *mockTexture) Size() image.Point                                            { return m.size }
func (m *mockTexture) Bounds() image.Rectangle                                      { return image.Rectangle{Max: m.size} }
func (m *mockTexture) Upload(dp image.Point, src screen.Buffer, sr image.Rectangle) {}
func (m *mockTexture) Fill(dr image.Rectangle, src color.Color, op draw.Op)         {}

// IsReleased безпечно перевіряє, чи був викликаний Release.
func (m *mockTexture) IsReleased() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mockReleaseCalled
}

// mockBuffer - заглушка для screen.Buffer
type mockBuffer struct {
	size image.Point
	rgba *image.RGBA // Може бути nil, якщо не тестуємо вміст
}

func (m *mockBuffer) Release()                { /* нічого */ }
func (m *mockBuffer) Size() image.Point       { return m.size }
func (m *mockBuffer) Bounds() image.Rectangle { return image.Rectangle{Max: m.size} }
func (m *mockBuffer) RGBA() *image.RGBA       { return m.rgba }

// mockScreen - заглушка для screen.Screen
type mockScreen struct{}

func (m *mockScreen) NewBuffer(size image.Point) (screen.Buffer, error) {
	return &mockBuffer{size: size, rgba: nil}, nil // Повертаємо заглушку буфера
}

func (m *mockScreen) NewTexture(size image.Point) (screen.Texture, error) {
	return newMockTexture(size), nil // Повертаємо заглушку текстури
}

func (m *mockScreen) NewWindow(opts *screen.NewWindowOptions) (screen.Window, error) {
	return nil, nil // Вікно не потрібне для тестів Loop
}

// --- Тести для Loop ---

func TestLoop_PostSingleOperation(t *testing.T) {
	receiver := newMockReceiver()
	loop := painter.NewLoop(receiver, 800, 800)
	screenInstance := &mockScreen{}
	loop.Start(screenInstance) // Запускаємо цикл
	defer loop.Stop()          // Гарантуємо зупинку в кінці тесту

	// Чекаємо на початковий Update, який ініціюється в loop.Start()
	assert.True(t, receiver.WaitForUpdate(1*time.Second), "Initial update after Start was not received")
	initialCalls := receiver.UpdateCalls() // Запам'ятовуємо кількість початкових викликів

	// Створюємо тестову операцію, яка має викликати Update (повертає true)
	opExecuted := false
	testOp := OperationFunc(func(s *painter.State, tex screen.Texture) bool {
		opExecuted = true
		s.BgColor = color.Black // Змінюємо стан для прикладу
		return true             // Сигналізуємо, що оновлення вікна потрібне
	})

	loop.Post(testOp) // Відправляємо операцію в цикл

	// Чекаємо на Update, який має бути викликаний через testOp
	assert.True(t, receiver.WaitForUpdate(1*time.Second), "Receiver.Update should be called after testOp")

	// Перевіряємо, що операція дійсно виконалась
	assert.True(t, opExecuted, "Posted operation should be executed")
	// Перевіряємо, що кількість викликів Update збільшилась
	assert.Equal(t, initialCalls+1, receiver.UpdateCalls(), "Update call count should increment")
}

func TestLoop_PostMultipleOperations(t *testing.T) {
	receiver := newMockReceiver()
	loop := painter.NewLoop(receiver, 800, 800)
	screenInstance := &mockScreen{}
	loop.Start(screenInstance)
	defer loop.Stop()

	assert.True(t, receiver.WaitForUpdate(1*time.Second), "Initial update after Start was not received")
	initialCalls := receiver.UpdateCalls()

	op1Executed := false
	op2Executed := false

	// Перша операція НЕ викликає Update (повертає false)
	op1 := OperationFunc(func(s *painter.State, t screen.Texture) bool {
		op1Executed = true
		return false
	})
	// Друга операція ВИКЛИКАЄ Update (повертає true)
	op2 := OperationFunc(func(s *painter.State, t screen.Texture) bool {
		op2Executed = true
		return true
	})

	loop.Post(op1) // Відправляємо першу
	loop.Post(op2) // Відправляємо другу

	// Чекаємо на Update, який має бути викликаний через op2
	assert.True(t, receiver.WaitForUpdate(1*time.Second), "Receiver.Update should have been called for op2")

	assert.True(t, op1Executed, "Operation 1 should be executed")
	assert.True(t, op2Executed, "Operation 2 should be executed")
	// Кількість викликів має збільшитись на 1 (лише від op2)
	assert.Equal(t, initialCalls+1, receiver.UpdateCalls(), "Update call count should increment only once")
}

func TestLoop_Stop(t *testing.T) {
	receiver := newMockReceiver()
	loop := painter.NewLoop(receiver, 800, 800)
	screenInstance := &mockScreen{}
	loop.Start(screenInstance) // Запускаємо цикл (і його горутину)

	// Чекаємо на початковий Update, щоб текстура була створена
	if !receiver.WaitForUpdate(1 * time.Second) {
		t.Fatal("Initial update not received after Start")
	}

	// Отримуємо текстуру БЕЗПЕЧНО *після* першого Update
	initialTexture := receiver.GetLastTexture()
	if initialTexture == nil {
		t.Fatal("Initial texture is nil even after initial update")
	}
	// Перевіряємо тип і зберігаємо як mockTexture для доступу до IsReleased
	mockTex, ok := initialTexture.(*mockTexture)
	if !ok {
		t.Fatalf("Initial texture is not a *mockTexture (%T)", initialTexture)
	}

	// Зберігаємо кількість викликів Update *до* зупинки
	callsBeforeStop := receiver.UpdateCalls()

	// Зупиняємо цикл і чекаємо завершення його горутини
	loop.Stop()

	// --- Перевірки ПІСЛЯ завершення горутини циклу ---

	// 1. Перевіряємо, чи текстура була звільнена (безпечно через метод)
	assert.True(t, mockTex.IsReleased(), "Texture.Release should be called on stop")

	// 2. Чекаємо трохи, щоб переконатись, що нові Update не надходять
	time.Sleep(50 * time.Millisecond)
	callsAfterStop := receiver.UpdateCalls() // Безпечно отримуємо кількість викликів
	assert.Equal(t, callsBeforeStop, callsAfterStop, "Update should not be called after Stop")

	// 3. (Опціонально) Перевіряємо, чи спроба Post після Stop не викликає паніки
	//    Це не є гарантованою поведінкою, залежить від реалізації черги.
	// assert.NotPanics(t, func() {
	// 	loop.Post(OperationFunc(func(s *painter.State, tx screen.Texture) bool { return false }))
	// }, "Post after Stop should not panic")
}
