//go:build experimental

package whiteboard

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// Point represents a 2D point on the canvas
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Color represents an RGBA color
type Color struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

// Common colors
var (
	ColorBlack   = Color{0, 0, 0, 255}
	ColorWhite   = Color{255, 255, 255, 255}
	ColorRed     = Color{255, 0, 0, 255}
	ColorGreen   = Color{0, 255, 0, 255}
	ColorBlue    = Color{0, 0, 255, 255}
	ColorYellow  = Color{255, 255, 0, 255}
	ColorOrange  = Color{255, 165, 0, 255}
	ColorPurple  = Color{128, 0, 128, 255}
	ColorGray    = Color{128, 128, 128, 255}
)

// DrawingTool represents drawing tools
type DrawingTool int

const (
	ToolPen DrawingTool = iota
	ToolMarker
	ToolHighlighter
	ToolEraser
	ToolLine
	ToolRectangle
	ToolCircle
	ToolText
	ToolSelect
)

func (t DrawingTool) String() string {
	switch t {
	case ToolPen:
		return "Pen"
	case ToolMarker:
		return "Marker"
	case ToolHighlighter:
		return "Highlighter"
	case ToolEraser:
		return "Eraser"
	case ToolLine:
		return "Line"
	case ToolRectangle:
		return "Rectangle"
	case ToolCircle:
		return "Circle"
	case ToolText:
		return "Text"
	case ToolSelect:
		return "Select"
	default:
		return "Unknown"
	}
}

// DrawingElement represents a drawing element
type DrawingElement struct {
	ID        string      `json:"id"`
	Type      DrawingTool `json:"type"`
	UserID    string      `json:"user_id"`
	Points    []Point     `json:"points"`
	Color     Color       `json:"color"`
	Width     float64     `json:"width"`
	Text      string      `json:"text,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	ZIndex    int         `json:"z_index"`
}

// Action represents a whiteboard action
type Action struct {
	Type      ActionType       `json:"type"`
	Element   *DrawingElement  `json:"element,omitempty"`
	ElementID string           `json:"element_id,omitempty"`
	UserID    string           `json:"user_id"`
	Timestamp time.Time        `json:"timestamp"`
}

// ActionType represents types of actions
type ActionType int

const (
	ActionAdd ActionType = iota
	ActionUpdate
	ActionDelete
	ActionClear
	ActionUndo
	ActionRedo
)

func (a ActionType) String() string {
	switch a {
	case ActionAdd:
		return "Add"
	case ActionUpdate:
		return "Update"
	case ActionDelete:
		return "Delete"
	case ActionClear:
		return "Clear"
	case ActionUndo:
		return "Undo"
	case ActionRedo:
		return "Redo"
	default:
		return "Unknown"
	}
}

// Cursor represents a user's cursor position
type Cursor struct {
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Position  Point     `json:"position"`
	Color     Color     `json:"color"`
	Timestamp time.Time `json:"timestamp"`
}

// Whiteboard represents a collaborative whiteboard
type Whiteboard struct {
	mu sync.RWMutex

	ID            string
	Name          string
	Width         int
	Height        int
	Background    Color
	Elements      map[string]*DrawingElement
	ElementOrder  []string // Z-order
	History       []Action
	HistoryIndex  int
	Cursors       map[string]*Cursor
	ActiveUsers   map[string]bool
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// Callbacks
	OnElementAdded   func(element *DrawingElement)
	OnElementUpdated func(element *DrawingElement)
	OnElementDeleted func(elementID string)
	OnCleared        func()
	OnCursorMoved    func(cursor *Cursor)
}

// NewWhiteboard creates a new whiteboard
func NewWhiteboard(id, name string, width, height int) *Whiteboard {
	return &Whiteboard{
		ID:           id,
		Name:         name,
		Width:        width,
		Height:       height,
		Background:   ColorWhite,
		Elements:     make(map[string]*DrawingElement),
		ElementOrder: make([]string, 0),
		History:      make([]Action, 0),
		HistoryIndex: 0,
		Cursors:      make(map[string]*Cursor),
		ActiveUsers:  make(map[string]bool),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// AddElement adds a new drawing element
func (wb *Whiteboard) AddElement(element *DrawingElement) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if element.ID == "" {
		return errors.New("element ID is required")
	}

	if _, exists := wb.Elements[element.ID]; exists {
		return errors.New("element already exists")
	}

	element.Timestamp = time.Now()
	element.ZIndex = len(wb.ElementOrder)

	wb.Elements[element.ID] = element
	wb.ElementOrder = append(wb.ElementOrder, element.ID)
	wb.UpdatedAt = time.Now()

	// Add to history
	wb.addToHistory(Action{
		Type:      ActionAdd,
		Element:   element,
		UserID:    element.UserID,
		Timestamp: time.Now(),
	})

	if wb.OnElementAdded != nil {
		go wb.OnElementAdded(element)
	}

	return nil
}

// UpdateElement updates an existing element
func (wb *Whiteboard) UpdateElement(elementID string, updater func(*DrawingElement)) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	element, exists := wb.Elements[elementID]
	if !exists {
		return errors.New("element not found")
	}

	// Apply update
	updater(element)
	element.Timestamp = time.Now()
	wb.UpdatedAt = time.Now()

	// Add to history
	wb.addToHistory(Action{
		Type:      ActionUpdate,
		Element:   element,
		ElementID: elementID,
		Timestamp: time.Now(),
	})

	if wb.OnElementUpdated != nil {
		go wb.OnElementUpdated(element)
	}

	return nil
}

// DeleteElement deletes an element
func (wb *Whiteboard) DeleteElement(elementID string, userID string) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	element, exists := wb.Elements[elementID]
	if !exists {
		return errors.New("element not found")
	}

	delete(wb.Elements, elementID)

	// Remove from order
	newOrder := make([]string, 0)
	for _, id := range wb.ElementOrder {
		if id != elementID {
			newOrder = append(newOrder, id)
		}
	}
	wb.ElementOrder = newOrder
	wb.UpdatedAt = time.Now()

	// Add to history
	wb.addToHistory(Action{
		Type:      ActionDelete,
		Element:   element,
		ElementID: elementID,
		UserID:    userID,
		Timestamp: time.Now(),
	})

	if wb.OnElementDeleted != nil {
		go wb.OnElementDeleted(elementID)
	}

	return nil
}

// Clear clears all elements
func (wb *Whiteboard) Clear(userID string) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.Elements = make(map[string]*DrawingElement)
	wb.ElementOrder = make([]string, 0)
	wb.UpdatedAt = time.Now()

	// Add to history
	wb.addToHistory(Action{
		Type:      ActionClear,
		UserID:    userID,
		Timestamp: time.Now(),
	})

	if wb.OnCleared != nil {
		go wb.OnCleared()
	}
}

// Undo undoes the last action
func (wb *Whiteboard) Undo() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if wb.HistoryIndex <= 0 {
		return errors.New("nothing to undo")
	}

	wb.HistoryIndex--
	wb.replayHistory()
	wb.UpdatedAt = time.Now()

	return nil
}

// Redo redoes the last undone action
func (wb *Whiteboard) Redo() error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if wb.HistoryIndex >= len(wb.History) {
		return errors.New("nothing to redo")
	}

	wb.HistoryIndex++
	wb.replayHistory()
	wb.UpdatedAt = time.Now()

	return nil
}

// UpdateCursor updates a user's cursor position
func (wb *Whiteboard) UpdateCursor(cursor *Cursor) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	cursor.Timestamp = time.Now()
	wb.Cursors[cursor.UserID] = cursor

	if wb.OnCursorMoved != nil {
		go wb.OnCursorMoved(cursor)
	}
}

// GetCursors gets all active cursors
func (wb *Whiteboard) GetCursors() []*Cursor {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	cursors := make([]*Cursor, 0, len(wb.Cursors))
	for _, cursor := range wb.Cursors {
		cursors = append(cursors, cursor)
	}

	return cursors
}

// JoinUser marks a user as active
func (wb *Whiteboard) JoinUser(userID string) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.ActiveUsers[userID] = true
}

// LeaveUser removes a user
func (wb *Whiteboard) LeaveUser(userID string) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	delete(wb.ActiveUsers, userID)
	delete(wb.Cursors, userID)
}

// GetActiveUserCount returns the number of active users
func (wb *Whiteboard) GetActiveUserCount() int {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	return len(wb.ActiveUsers)
}

// GetElements returns all elements in order
func (wb *Whiteboard) GetElements() []*DrawingElement {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	elements := make([]*DrawingElement, 0, len(wb.ElementOrder))
	for _, id := range wb.ElementOrder {
		if element, exists := wb.Elements[id]; exists {
			elements = append(elements, element)
		}
	}

	return elements
}

// GetElementByID gets an element by ID
func (wb *Whiteboard) GetElementByID(id string) (*DrawingElement, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	element, exists := wb.Elements[id]
	if !exists {
		return nil, errors.New("element not found")
	}

	return element, nil
}

// GetElementCount returns the number of elements
func (wb *Whiteboard) GetElementCount() int {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	return len(wb.Elements)
}

// ExportJSON exports the whiteboard to JSON
func (wb *Whiteboard) ExportJSON() ([]byte, error) {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	data := map[string]interface{}{
		"id":          wb.ID,
		"name":        wb.Name,
		"width":       wb.Width,
		"height":      wb.Height,
		"background":  wb.Background,
		"elements":    wb.GetElements(),
		"created_at":  wb.CreatedAt,
		"updated_at":  wb.UpdatedAt,
	}

	return json.Marshal(data)
}

// ImportJSON imports whiteboard from JSON
func (wb *Whiteboard) ImportJSON(data []byte) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	var imported map[string]interface{}
	if err := json.Unmarshal(data, &imported); err != nil {
		return err
	}

	// Import elements
	if elementsData, ok := imported["elements"].([]interface{}); ok {
		for _, elemData := range elementsData {
			elemJSON, _ := json.Marshal(elemData)
			var element DrawingElement
			if err := json.Unmarshal(elemJSON, &element); err == nil {
				wb.Elements[element.ID] = &element
				wb.ElementOrder = append(wb.ElementOrder, element.ID)
			}
		}
	}

	wb.UpdatedAt = time.Now()
	return nil
}

// GetStats returns whiteboard statistics
func (wb *Whiteboard) GetStats() map[string]interface{} {
	wb.mu.RLock()
	defer wb.mu.RUnlock()

	return map[string]interface{}{
		"id":            wb.ID,
		"name":          wb.Name,
		"width":         wb.Width,
		"height":        wb.Height,
		"element_count": len(wb.Elements),
		"active_users":  len(wb.ActiveUsers),
		"history_size":  len(wb.History),
		"history_index": wb.HistoryIndex,
		"created_at":    wb.CreatedAt,
		"updated_at":    wb.UpdatedAt,
	}
}

// addToHistory adds an action to history (must hold lock)
func (wb *Whiteboard) addToHistory(action Action) {
	// Remove any actions after current index (for redo)
	if wb.HistoryIndex < len(wb.History) {
		wb.History = wb.History[:wb.HistoryIndex]
	}

	wb.History = append(wb.History, action)
	wb.HistoryIndex = len(wb.History)

	// Limit history size
	maxHistory := 1000
	if len(wb.History) > maxHistory {
		wb.History = wb.History[len(wb.History)-maxHistory:]
		wb.HistoryIndex = len(wb.History)
	}
}

// replayHistory replays history up to current index (must hold lock)
func (wb *Whiteboard) replayHistory() {
	// Clear current state
	wb.Elements = make(map[string]*DrawingElement)
	wb.ElementOrder = make([]string, 0)

	// Replay actions up to current index
	for i := 0; i < wb.HistoryIndex; i++ {
		action := wb.History[i]

		switch action.Type {
		case ActionAdd:
			if action.Element != nil {
				wb.Elements[action.Element.ID] = action.Element
				wb.ElementOrder = append(wb.ElementOrder, action.Element.ID)
			}

		case ActionUpdate:
			if action.Element != nil {
				wb.Elements[action.Element.ID] = action.Element
			}

		case ActionDelete:
			delete(wb.Elements, action.ElementID)
			newOrder := make([]string, 0)
			for _, id := range wb.ElementOrder {
				if id != action.ElementID {
					newOrder = append(newOrder, id)
				}
			}
			wb.ElementOrder = newOrder

		case ActionClear:
			wb.Elements = make(map[string]*DrawingElement)
			wb.ElementOrder = make([]string, 0)
		}
	}
}

// WhiteboardManager manages multiple whiteboards
type WhiteboardManager struct {
	mu sync.RWMutex

	whiteboards map[string]*Whiteboard
}

// NewWhiteboardManager creates a new whiteboard manager
func NewWhiteboardManager() *WhiteboardManager {
	return &WhiteboardManager{
		whiteboards: make(map[string]*Whiteboard),
	}
}

// CreateWhiteboard creates a new whiteboard
func (wm *WhiteboardManager) CreateWhiteboard(id, name string, width, height int) (*Whiteboard, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.whiteboards[id]; exists {
		return nil, errors.New("whiteboard already exists")
	}

	wb := NewWhiteboard(id, name, width, height)
	wm.whiteboards[id] = wb

	return wb, nil
}

// GetWhiteboard gets a whiteboard by ID
func (wm *WhiteboardManager) GetWhiteboard(id string) (*Whiteboard, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	wb, exists := wm.whiteboards[id]
	if !exists {
		return nil, errors.New("whiteboard not found")
	}

	return wb, nil
}

// DeleteWhiteboard deletes a whiteboard
func (wm *WhiteboardManager) DeleteWhiteboard(id string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.whiteboards[id]; !exists {
		return errors.New("whiteboard not found")
	}

	delete(wm.whiteboards, id)
	return nil
}

// GetAllWhiteboards returns all whiteboards
func (wm *WhiteboardManager) GetAllWhiteboards() []*Whiteboard {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	whiteboards := make([]*Whiteboard, 0, len(wm.whiteboards))
	for _, wb := range wm.whiteboards {
		whiteboards = append(whiteboards, wb)
	}

	return whiteboards
}

// GetWhiteboardCount returns the number of whiteboards
func (wm *WhiteboardManager) GetWhiteboardCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return len(wm.whiteboards)
}
