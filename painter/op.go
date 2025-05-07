package painter

import (
	"image"
	"image/color"
	"log"

	"golang.org/x/exp/shiny/screen"
)

// Operation defines an interface for commands that modify the texture state.
// It returns true if the texture was updated and requires a screen update.
type Operation interface {
	Do(s *State, t screen.Texture) (updated bool)
}

// State holds the current drawing state managed by the loop.
type State struct {
	BgColor      color.Color // Поточний колір фону
	BgRect       *BgRectOp   // Дані для останнього фонового прямокутника (nil якщо немає)
	Figures      []*FigureOp // Слайс усіх фігур на екрані
	MoveOffset   image.Point // Кумулятивне зміщення для команди 'move' (застосовується в UpdateOp)
	WindowWidth  int         // Ширина вікна в пікселях
	WindowHeight int         // Висота вікна в пікселях
}

// FigureOp represents the state for drawing a single figure instance.
type FigureOp struct {
	X, Y    int           // Абсолютні піксельні координати центру фігури
	Variant FigureVariant // Тип фігури (T0, T90, T180, T270, Cross)
	Color   color.Color   // Колір фігури
}

// BgRectOp represents the state for the background rectangle.
type BgRectOp struct {
	X1, Y1, X2, Y2 int // Абсолютні піксельні координати прямокутника
}

// OperationList groups multiple operations. Useful for batch processing.
type OperationList []Operation

func (ol OperationList) Do(s *State, t screen.Texture) (updated bool) {
	for _, o := range ol {
		// Якщо будь-яка операція в списку сигналізує про оновлення,
		// то весь список вважається таким, що потребує оновлення.
		if o.Do(s, t) {
			updated = true
		}
	}
	return updated
}

// UpdateOp signals that the texture should be redrawn based on the current state
// and sent to the screen. This is where all actual drawing happens.
type UpdateOp struct{}

func (op UpdateOp) Do(s *State, t screen.Texture) bool {
	log.Println("UpdateOp.Do: Starting update...")
	// 1. Очищуємо текстуру поточним кольором фону зі стану (має бути білий після команди 'white')
	log.Printf("UpdateOp.Do: Filling background with: %+v", s.BgColor)
	t.Fill(t.Bounds(), s.BgColor, screen.Src)

	// 2. Малюємо фоновий прямокутник (чорний із зеленою рамкою згідно з завданням)
	if s.BgRect != nil {
		// Створюємо image.Rectangle з піксельних координат
		rect := image.Rect(s.BgRect.X1, s.BgRect.Y1, s.BgRect.X2, s.BgRect.Y2)
		log.Printf("UpdateOp.Do: Drawing BgRect %+v", rect)

		// Малюємо ЧОРНИЙ фон прямокутника
		blackColor := color.Black
		t.Fill(rect, blackColor, screen.Src)

		// Малюємо ЗЕЛЕНУ рамку (товщиною 1 піксель)
		greenColor := color.NRGBA{G: 0xff, A: 0xff} // Зелений
		// Важливо: Перевіряємо, чи рамка не виходить за межі прямокутника, якщо він дуже малий
		if rect.Dx() > 0 && rect.Dy() > 0 { // Перевірка, що прямокутник має ширину і висоту > 0
			// Верхня лінія
			t.Fill(image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+1).Intersect(rect), greenColor, screen.Src)
			// Нижня лінія (якщо висота > 1)
			if rect.Dy() > 1 {
				t.Fill(image.Rect(rect.Min.X, rect.Max.Y-1, rect.Max.X, rect.Max.Y).Intersect(rect), greenColor, screen.Src)
			}
			// Ліва лінія (враховуючи вже намальовані кути)
			if rect.Dx() > 1 {
				t.Fill(image.Rect(rect.Min.X, rect.Min.Y+1, rect.Min.X+1, rect.Max.Y-1).Intersect(rect), greenColor, screen.Src)
				// Права лінія (враховуючи вже намальовані кути)
				t.Fill(image.Rect(rect.Max.X-1, rect.Min.Y+1, rect.Max.X, rect.Max.Y-1).Intersect(rect), greenColor, screen.Src)
			}
		} else if rect.Dx() > 0 { // Якщо це горизонтальна лінія
			t.Fill(rect, greenColor, screen.Src)
		} else if rect.Dy() > 0 { // Якщо це вертикальна лінія
			t.Fill(rect, greenColor, screen.Src)
		}

		log.Println("UpdateOp.Do: BgRect drawn with black fill and green border.")
	} else {
		log.Println("UpdateOp.Do: No BgRect defined in state.")
	}

	// 3. Малюємо всі фігури зі стану (мають бути жовті T180)
	log.Printf("UpdateOp.Do: Drawing %d figures from state.", len(s.Figures))
	for i, fig := range s.Figures {
		// Застосовуємо кумулятивне зміщення від команди 'move' (якщо є)
		centerX := fig.X + s.MoveOffset.X
		centerY := fig.Y + s.MoveOffset.Y
		log.Printf("UpdateOp.Do: Drawing figure %d: Type=%v, Color=%+v at final coords (%d, %d)", i, fig.Variant, fig.Color, centerX, centerY)
		// Викликаємо допоміжну функцію для малювання конкретної фігури
		drawFigure(t, centerX, centerY, fig.Variant, fig.Color, s.WindowWidth, s.WindowHeight)
	}

	// 4. Сигналізуємо, що екран потрібно оновити (текстура готова)
	log.Println("UpdateOp.Do: Update complete, returning true.")
	return true
}

// WhiteBg defines the operation for setting the background to white.
type WhiteBg struct{}

func (op WhiteBg) Do(s *State, t screen.Texture) bool {
	log.Println("WhiteBg.Do: Setting background color to White")
	s.BgColor = color.White // Змінюємо колір фону в стані
	// Колір фігур не змінюємо, вони визначаються в Figure.Do
	return false // Сама зміна кольору не вимагає негайного Update
}

// GreenBg defines the operation for setting the background to green.
type GreenBg struct{}

func (op GreenBg) Do(s *State, t screen.Texture) bool {
	log.Println("GreenBg.Do: Setting background color to Green")
	s.BgColor = color.NRGBA{G: 0xff, A: 0xff} // Змінюємо колір фону в стані
	// Колір фігур не змінюємо, вони визначаються в Figure.Do
	return false // Не вимагає негайного Update
}

// Figure defines the operation for ADDING a new figure.
type Figure struct {
	X, Y float64
}

// Do для Figure: ЗАВЖДИ додає нову фігуру (ЖОВТА T180).
func (op Figure) Do(s *State, t screen.Texture) bool {
	// Конвертуємо відносні координати в абсолютні піксельні координати
	pixelX := int(op.X * float64(s.WindowWidth))
	pixelY := int(op.Y * float64(s.WindowHeight))

	log.Printf("Figure.Do: Received request to add figure at relative (%.2f, %.2f) -> pixel (%d, %d)", op.X, op.Y, pixelX, pixelY)

	// --- ЛОГІКА: ЗАВЖДИ ДОДАЄМО ЖОВТУ T180 ---
	figureColor := color.NRGBA{R: 0xff, G: 0xff, A: 0xff} // ЖОВТИЙ колір (R=255, G=255, B=0)
	figureVariant := T180                                 // Тип T180

	newFig := &FigureOp{
		X:       pixelX,
		Y:       pixelY,
		Variant: figureVariant,
		Color:   figureColor, // Використовуємо жовтий колір
	}
	s.Figures = append(s.Figures, newFig) // Додаємо вказівник на нову фігуру до слайсу
	log.Printf("Figure.Do: Successfully added YELLOW T180 figure. State now has %d figures.", len(s.Figures))
	// -----------------------------------

	// Сама операція Figure не вимагає негайного оновлення екрану.
	// Оновлення відбудеться при отриманні команди UpdateOp.
	return false
}

// BgRect defines the operation for setting/updating the background rectangle.
// Coordinates are relative (0.0 to 1.0).
type BgRect struct {
	X1, Y1, X2, Y2 float64
}

func (op BgRect) Do(s *State, t screen.Texture) bool {
	// Конвертуємо відносні координати в абсолютні піксельні координати
	px1 := int(op.X1 * float64(s.WindowWidth))
	py1 := int(op.Y1 * float64(s.WindowHeight))
	px2 := int(op.X2 * float64(s.WindowWidth))
	py2 := int(op.Y2 * float64(s.WindowHeight))

	log.Printf("BgRect.Do: Setting BgRect state to pixel coords: p1=(%d,%d), p2=(%d,%d)", px1, py1, px2, py2)
	// Зберігаємо дані в стані (перезаписуємо попереднє значення, якщо було)
	// Переконуємось, що X1 <= X2 та Y1 <= Y2 для image.Rect
	if px1 > px2 {
		px1, px2 = px2, px1
	}
	if py1 > py2 {
		py1, py2 = py2, py1
	}
	s.BgRect = &BgRectOp{X1: px1, Y1: py1, X2: px2, Y2: py2}
	return false // Не вимагає негайного Update
}

// Move defines the operation for applying a relative offset to ALL existing figures.
// Offset coordinates are relative (0.0 to 1.0).
type Move struct {
	X, Y float64
}

func (op Move) Do(s *State, t screen.Texture) bool {
	offsetX := int(op.X * float64(s.WindowWidth))
	offsetY := int(op.Y * float64(s.WindowHeight))
	log.Printf("Move.Do: Adding relative offset (%.2f, %.2f) -> pixel (%d, %d) to MoveOffset %+v", op.X, op.Y, offsetX, offsetY, s.MoveOffset)
	s.MoveOffset = s.MoveOffset.Add(image.Pt(offsetX, offsetY))
	return false // Не вимагає негайного Update
}

// Reset defines the operation for clearing the state to default values.
type Reset struct{}

func (op Reset) Do(s *State, t screen.Texture) bool {
	log.Println("Reset.Do: Resetting state...")
	s.BgColor = color.Black      // Скидаємо фон на чорний
	s.BgRect = nil               // Видаляємо фоновий прямокутник
	s.Figures = []*FigureOp{}    // Очищуємо список фігур
	s.MoveOffset = image.Point{} // Скидаємо зміщення
	log.Println("Reset.Do: State reset complete. Requesting screen update.")
	return true // Повертаємо true, щоб екран очистився
}

// FigureVariant defines the type of figure to draw.
type FigureVariant int

const (
	T0    FigureVariant = iota // Стандартна T
	T90                        // T повернута на 90° за годинниковою
	T180                       // T повернута на 180° (догори дригом)
	T270                       // T повернута на 270° за годинниковою
	Cross                      // Хрест
)

// drawFigure - допоміжна функція для малювання фігури на текстурі.
// cx, cy - піксельні координати центру фігури.
func drawFigure(t screen.Texture, cx, cy int, variant FigureVariant, figureColor color.Color, winWidth, winHeight int) {
	// Визначаємо базові розміри фігури (можна зробити їх динамічними або константами)
	// За умовою, не більше половини вікна. Візьмемо фіксований розмір, наприклад 30% меншої сторони вікна.
	baseSize := int(float64(min(winWidth, winHeight)) * 0.3) // Розмір фігури відносно вікна
	if baseSize < 20 {
		baseSize = 20
	} // Мінімальний розмір

	// Параметри для T-фігури
	barWidth := baseSize               // Ширина горизонтальної/вертикальної перекладини T або хреста
	barHeight := baseSize / 3          // Висота/товщина перекладини T
	stemWidth := baseSize / 3          // Ширина ніжки T
	stemHeight := baseSize - barHeight // Висота ніжки T (щоб загальна висота була baseSize)
	totalHeight := baseSize            // Повна висота/ширина Т

	// Параметри для Хреста
	crossSize := baseSize
	armThickness := baseSize / 3

	var rects []image.Rectangle // Слайс для зберігання прямокутників, що складають фігуру
	log.Printf("drawFigure: Drawing variant %v at (%d, %d) with baseSize %d", variant, cx, cy, baseSize)

	switch variant {
	case T0: // Стандартна T
		hbX1 := cx - barWidth/2
		hbY1 := cy - totalHeight/2
		hbX2 := cx + barWidth/2
		hbY2 := cy - totalHeight/2 + barHeight
		rects = append(rects, image.Rect(hbX1, hbY1, hbX2, hbY2)) // Горизонтальна шапка
		vsX1 := cx - stemWidth/2
		vsY1 := hbY2 // Починається під шапкою
		vsX2 := cx + stemWidth/2
		vsY2 := vsY1 + stemHeight
		rects = append(rects, image.Rect(vsX1, vsY1, vsX2, vsY2)) // Вертикальна ніжка
	case T90: // T повернута на 90°
		vbX1 := cx + totalHeight/2 - barHeight
		vbY1 := cy - barWidth/2
		vbX2 := cx + totalHeight/2
		vbY2 := cy + barWidth/2
		rects = append(rects, image.Rect(vbX1, vbY1, vbX2, vbY2)) // Вертикальна шапка (справа)
		hsX1 := vbX1 - stemHeight
		hsY1 := cy - stemWidth/2
		hsX2 := vbX1 // Закінчується де починається шапка
		hsY2 := cy + stemWidth/2
		rects = append(rects, image.Rect(hsX1, hsY1, hsX2, hsY2)) // Горизонтальна ніжка (зліва)
	case T180: // T повернута на 180° (догори дригом)
		hbX1 := cx - barWidth/2
		hbY1 := cy + totalHeight/2 - barHeight
		hbX2 := cx + barWidth/2
		hbY2 := cy + totalHeight/2
		rects = append(rects, image.Rect(hbX1, hbY1, hbX2, hbY2)) // Горизонтальна шапка (знизу)
		vsX1 := cx - stemWidth/2
		vsY1 := hbY1 - stemHeight // Починається над шапкою
		vsX2 := cx + stemWidth/2
		vsY2 := hbY1                                              // Закінчується де починається шапка
		rects = append(rects, image.Rect(vsX1, vsY1, vsX2, vsY2)) // Вертикальна ніжка (зверху)
	case T270: // T повернута на 270°
		vbX1 := cx - totalHeight/2
		vbY1 := cy - barWidth/2
		vbX2 := cx - totalHeight/2 + barHeight
		vbY2 := cy + barWidth/2
		rects = append(rects, image.Rect(vbX1, vbY1, vbX2, vbY2)) // Вертикальна шапка (зліва)
		hsX1 := vbX2                                              // Починається де закінчується шапка
		hsY1 := cy - stemWidth/2
		hsX2 := hsX1 + stemHeight
		hsY2 := cy + stemWidth/2
		rects = append(rects, image.Rect(hsX1, hsY1, hsX2, hsY2)) // Горизонтальна ніжка (справа)
	case Cross: // Хрест
		// Горизонтальна частина
		hbX1 := cx - crossSize/2
		hbY1 := cy - armThickness/2
		hbX2 := cx + crossSize/2
		hbY2 := cy + armThickness/2
		rects = append(rects, image.Rect(hbX1, hbY1, hbX2, hbY2))
		// Вертикальна частина
		vbX1 := cx - armThickness/2
		vbY1 := cy - crossSize/2
		vbX2 := cx + armThickness/2
		vbY2 := cy + crossSize/2
		// Уникаємо подвійного заповнення центру
		rects = append(rects, image.Rect(vbX1, vbY1, vbX2, hbY1)) // Верхня частина вертикалі
		rects = append(rects, image.Rect(vbX1, hbY2, vbX2, vbY2)) // Нижня частина вертикалі
	default:
		log.Printf("drawFigure: Unknown figure variant: %d", variant)
		return // Не малюємо нічого для невідомого варіанту
	}

	// Малюємо всі прямокутники, що складають фігуру
	log.Printf("drawFigure: Filling %d rectangles with color %+v", len(rects), figureColor)
	textureBounds := t.Bounds()
	for i, r := range rects {
		// Обрізаємо прямокутник межами текстури
		clippedRect := r.Intersect(textureBounds)
		if !clippedRect.Empty() {
			t.Fill(clippedRect, figureColor, screen.Src)
		} else {
			log.Printf("drawFigure: Rectangle %d (%+v) is outside texture bounds %+v", i, r, textureBounds)
		}
	}
	log.Printf("drawFigure: Finished drawing variant %v", variant)
}

// Допоміжна функція min для цілих чисел (якщо не використовується math.Min)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
