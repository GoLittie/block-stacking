//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"syscall/js"
	"time"
)

const GridWidth = 10
const GridHeight = 20

type ShapeId int8

const (
	ShapeNone ShapeId = iota
	LShape
	LShape2
	IShape
	OShape
	PShape
	SShape
	SShape2
)

const (
	ColorGreyDark   = "#565656"
	ColorGrey       = "#7f7f7f"
	ColorYellow     = "#ff0"
	ColorYellowDark = "#aa0"
	ColorBlue       = "#00f"
	ColorBlueDark   = "#00a"
	ColorRed        = "#f00"
	ColorRedDark    = "#a00"
	ColorGreen      = "#0f0"
	ColorGreenDark  = "#0a0"
	ColorOrange     = "#f70"
	ColorOrangeDark = "#d50"
	ColorPink       = "#f0f"
	ColorPinkDark   = "#a0a"
	ColorPurple     = "#f07"
	ColorPurpleDark = "#d05"
)

func (s ShapeId) ActiveColor() string {
	switch s {
	case LShape:
		return ColorOrangeDark
	case LShape2:
		return ColorPinkDark
	case IShape:
		return ColorYellowDark
	case OShape:
		return ColorBlueDark
	case PShape:
		return ColorPurpleDark
	case SShape:
		return ColorGreenDark
	case SShape2:
		return ColorRedDark
	default:
		return ColorGreyDark
	}
}

func (s ShapeId) Color() string {
	switch s {
	case LShape:
		return ColorOrange
	case LShape2:
		return ColorPink
	case IShape:
		return ColorYellow
	case OShape:
		return ColorBlue
	case PShape:
		return ColorPurple
	case SShape:
		return ColorGreen
	case SShape2:
		return ColorRed
	default:
		return ColorGrey
	}
}

type Mask [][]bool

var LShapeMask = [][]bool{
	{true, true},
	{true, false},
	{true, false},
}

var LShapeMask2 = [][]bool{
	{true, true},
	{false, true},
	{false, true},
}

var IShapeMask = [][]bool{
	{true},
	{true},
	{true},
	{true},
}

var OShapeMask = [][]bool{
	{true, true},
	{true, true},
}

var PShapeMask = [][]bool{
	{true, false},
	{true, true},
	{true, false},
}

var SShapeMask = [][]bool{
	{false, true},
	{true, true},
	{true, false},
}

var SShapeMask2 = [][]bool{
	{true, false},
	{true, true},
	{false, true},
}

func (s ShapeId) Mask() [][]bool {
	switch s {
	case LShape:
		return LShapeMask
	case LShape2:
		return LShapeMask2
	case OShape:
		return OShapeMask
	case IShape:
		return IShapeMask
	case PShape:
		return PShapeMask
	case SShape:
		return SShapeMask
	case SShape2:
		return SShapeMask2
	default:
		fmt.Println("Using default shape mask for shape id:", s)
		return LShapeMask
	}
}

type Game struct {
	Grid             [GridHeight][GridWidth]ShapeId
	CurrentPosition  [2]int
	CurrentShapeMask [][]bool
	CurrentShape     ShapeId
	NextShape        ShapeId
	Score            int
	Paused           bool
	ShowHint         bool
}

var game = Game{
	NextShape: RandomShape(),
	Paused:    true,
	ShowHint:  true,
}

func main() {
	fmt.Println("Hi from littie :0")
	shapePreviewGrid := QuerySelector("shape-preview")
	gameGrid := QuerySelector("game-grid")
	gameCtx, cancel := context.WithCancel(context.Background())

	js.Global().Set("destroy", js.FuncOf(func(value js.Value, args []js.Value) interface{} {
		cancel()
		return nil
	}))

	document := Document()
	// Create the shape preview grid
	for range 4 {
		gameRow := document.Call("createElement", "game-row")
		for range 2 {
			gameSquare := document.Call("createElement", "game-square")
			gameRow.Call("appendChild", gameSquare)
		}
		shapePreviewGrid.Call("appendChild", gameRow)
	}

	// Create the game grid
	for range GridHeight {
		gameRow := document.Call("createElement", "game-row")
		for range GridWidth {
			gameSquare := document.Call("createElement", "game-square")
			gameRow.Call("appendChild", gameSquare)
		}
		gameGrid.Call("appendChild", gameRow)
	}

	for _, buttonId := range []string{"play", "restart", "debug", "back", "settings", "github"} {
		QuerySelector("game-menu #"+buttonId).Call("addEventListener", "click", js.FuncOf(MenuButtonHandler))
	}

	NextShape()

	document.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		key := event.Get("code").String()
		keyAction(key)
		return nil
	}))

	go func() {
		for {
			RenderGrid()
			if game.ShowHint {
				RenderActiveShapeHint()
			}
			RenderActiveShape()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	for { // Move Tick
		select {
		case <-gameCtx.Done():
			return
		default:
			time.Sleep(1 * time.Second)
			MoveTick()
		}
	}
}

func Document() js.Value {
	return js.Global().Get("document")
}

func QuerySelector(elementQuery string) js.Value {
	return Document().Call("querySelector", elementQuery)
}

func MenuButtonHandler(this js.Value, args []js.Value) interface{} {
	id := this.Get("id").String()
	switch id {
	case "play":
		ToggleMenu()
	case "restart":
		Restart()
	case "debug":
		byt, err := json.Marshal(game)
		fmt.Println(string(byt))
		if err != nil {
			byt = []byte("Parse err!")
		}
		QuerySelector("#debug-info").Set("innerText", string(byt))
		ShowMenuPage(1)
	case "back":
		ShowMenuPage(0)
	case "github":
		js.Global().Get("window").Call("open", "https://github.com/golittie/block-stacking")
	}
	return nil
}

func ToggleMenu() {
	menu := QuerySelector("game-menu")
	class := menu.Get("className").String()
	if class == "hide" {
		ShowMenuPage(0)
		menu.Set("className", "")
		game.Paused = true
	} else {
		menu.Set("className", "hide")
		game.Paused = false
	}
}

func ShowMenuPage(page int) {
	mainMenu, debug := QuerySelector("game-menu #main-menu"), QuerySelector("game-menu #debug-menu")
	switch page {
	case 0: // Main menu page
		mainMenu.Set("className", "")
		debug.Set("className", "hide")
	case 1: // Debug page
		mainMenu.Set("className", "hide")
		debug.Set("className", "")
	}
}

func keyAction(key string) {
	if key == "Escape" {
		ToggleMenu()
	} else if game.Paused {
		return
	}

	switch key {
	case "KeyA", "ArrowLeft":
		if !IsValidPosition([2]int{game.CurrentPosition[0] - 1, game.CurrentPosition[1]}) {
			return
		}
		game.CurrentPosition[0]--
	case "KeyD", "ArrowRight":
		if !IsValidPosition([2]int{game.CurrentPosition[0] + 1, game.CurrentPosition[1]}) {
			return
		}
		game.CurrentPosition[0] = game.CurrentPosition[0] + 1
	case "KeyS", "ArrowDown":
		MoveTick()
	case "KeyW", "KeyR", "ArrowUp":
		game.CurrentShapeMask = RotateShape(game.CurrentShapeMask)
	case "KeyH":
		game.ShowHint = !game.ShowHint
		fmt.Println("Show hint:", game.ShowHint)
	case "KeyN":
		NextShape()
	case "Enter":
		PlaceShape()
	}
}

func MoveTick() {
	if game.Paused {
		return
	}

	if game.CurrentPosition[1] == YPlacePosition() {
		PlaceShape()
	} else {
		game.CurrentPosition[1]++
	}
}

func IsValidPosition(position [2]int) bool {
	shape := game.CurrentShapeMask
	x, y := position[0], position[1]

	if y == 0 {
		return true
	}

	if y+len(shape) > GridHeight {
		return false
	}

	shapeCols := len(shape[0])
	shapeRows := len(shape)

	for r := range shapeRows {
		for c := range shapeCols {
			if x+c > GridWidth-1 || x+c < 0 {
				return false
			}
			if shape[r][c] && game.Grid[y+r][x+c] != ShapeNone {
				return false
			}
		}
	}
	return true
}

func RotateShape(shape [][]bool) [][]bool {
	rows := len(shape)
	cols := len(shape[0])

	rotated := make([][]bool, cols)
	for i := range rotated {
		rotated[i] = make([]bool, rows)
	}

	for r := range rows {
		for c := range cols {
			rotated[c][rows-1-r] = shape[r][c]
		}
	}
	return rotated
}

func YPlacePosition() int {
	yPlace := game.CurrentPosition[1]
	for IsValidPosition([2]int{game.CurrentPosition[0], yPlace}) {
		yPlace++
	}
	yPlace--
	return yPlace
}

func Restart() {
	game.Paused = true
	game.Grid = [GridHeight][GridWidth]ShapeId{}
	game.Score = 0
	NextShape()
}

func PlaceShape() {
	x, y := game.CurrentPosition[0], YPlacePosition()
	shape := game.CurrentShapeMask
	shapeCols := len(shape[0])
	shapeRows := len(shape)
	for r := range shapeRows {
		for c := range shapeCols {
			if shape[r][c] {
				game.Grid[y+r][x+c] = game.CurrentShape
			}
		}
	}

	CheckForCompleteRows()
	NextShape()
}

func NextShape() {
	game.CurrentPosition = [2]int{4, 0}
	game.CurrentShapeMask = game.NextShape.Mask()
	game.CurrentShape = game.NextShape
	fmt.Println("Switched to next shape:", game.NextShape)
	game.NextShape = RandomShape()
	RenderNextShape()
}

func RandomShape() ShapeId {
	return ShapeId(rand.Intn(7)) + 1
}

func SetSquareColor(row, col int, hexColor string) {
	square := QuerySelector(fmt.Sprintf("game-grid :nth-child(%d) :nth-child(%d)", row+1, col+1))
	if square.IsNull() {
		return
	}
	//fmt.Println(fmt.Sprintf("Set %d, %d to %s", row, col, hexColor))
	style := square.Get("style")
	style.Set("background", hexColor)
}

func CheckForCompleteRows() {
	i := GridHeight - 1
	var newGrid [GridHeight][GridWidth]ShapeId
	for r := range GridHeight {
		r = GridHeight - 1 - r
		solved := true
		for c := range GridWidth {
			if game.Grid[r][c] == ShapeNone {
				solved = false
			} else if r == 1 {
				QuerySelector("#status").Set("innerHTML", fmt.Sprintf("You lost :(<br>Your score was %d", game.Score))
				Restart()
				ToggleMenu()
				return
			}
		}
		if solved {
			game.Score++
			for c := range GridWidth {
				SetSquareColor(r, c, "#fff")
			}
			fmt.Println("Line completed!")
		} else {
			newGrid[i] = game.Grid[r]
			i--
		}
	}
	game.Grid = newGrid
}

func SetPreviewSquareColor(row, col int, hexColor string) {
	square := QuerySelector(fmt.Sprintf("shape-preview :nth-child(%d) :nth-child(%d)", row+1, col+1))
	if square.IsNull() {
		return
	}
	//fmt.Println(fmt.Sprintf("Set %d, %d to %s", row, col, hexColor))
	style := square.Get("style")
	style.Set("background", hexColor)
}

func RenderNextShape() {
	shape := game.NextShape.Mask()
	shapeCols := len(shape[0])
	shapeRows := len(shape)

	fmt.Println("Rendered preview.")
	for r := range 4 {
		for c := range 2 {
			if shapeCols > c && shapeRows > r && shape[r][c] {
				SetPreviewSquareColor(r, c, game.NextShape.Color())
			} else {
				SetPreviewSquareColor(r, c, "none")
			}
		}
	}
}

func RenderGrid() {
	for r := range GridHeight {
		for c := range GridWidth {
			SetSquareColor(r, c, game.Grid[r][c].Color())
		}
	}
}

func RenderActiveShape() {
	position := &game.CurrentPosition
	shape := game.CurrentShapeMask
	x, y := position[0], position[1]
	shapeCols := len(shape[0])
	shapeRows := len(shape)

	if x > GridWidth-shapeCols {
		position[0] = 0
		x = 0
	}
	if x < 0 {
		position[0] = GridWidth - shapeCols
		x = GridWidth - shapeCols
	}

	for r := range shapeRows {
		for c := range shapeCols {
			if shape[r][c] {
				SetSquareColor(y+r, x+c, game.CurrentShape.ActiveColor())
			}
		}
	}
}

func RenderActiveShapeHint() {
	position := &game.CurrentPosition
	shape := game.CurrentShapeMask
	x, y := position[0], YPlacePosition()
	shapeCols := len(shape[0])
	shapeRows := len(shape)

	for r := range shapeRows {
		for c := range shapeCols {
			if shape[r][c] {
				SetSquareColor(y+r, x+c, ColorGreyDark)
			}
		}
	}
}
