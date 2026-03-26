//go:build experimental

package whiteboard

import (
	"fmt"
	"testing"
	"time"
)

// TestNewWhiteboard tests whiteboard creation
func TestNewWhiteboard(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test Board", 1920, 1080)

	if wb.ID != "wb1" {
		t.Errorf("Expected ID 'wb1', got '%s'", wb.ID)
	}

	if wb.Name != "Test Board" {
		t.Errorf("Expected name 'Test Board', got '%s'", wb.Name)
	}

	if wb.Width != 1920 {
		t.Errorf("Expected width 1920, got %d", wb.Width)
	}

	if wb.Height != 1080 {
		t.Errorf("Expected height 1080, got %d", wb.Height)
	}
}

// TestAddElement tests adding elements
func TestAddElement(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	element := &DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}, {X: 30, Y: 40}},
		Color:  ColorBlack,
		Width:  2.0,
	}

	err := wb.AddElement(element)
	if err != nil {
		t.Fatalf("Failed to add element: %v", err)
	}

	if wb.GetElementCount() != 1 {
		t.Errorf("Expected 1 element, got %d", wb.GetElementCount())
	}

	retrieved, err := wb.GetElementByID("elem1")
	if err != nil {
		t.Fatalf("Failed to get element: %v", err)
	}

	if retrieved.ID != "elem1" {
		t.Error("Retrieved element doesn't match")
	}
}

// TestAddDuplicateElement tests preventing duplicate elements
func TestAddDuplicateElement(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	element := &DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	}

	wb.AddElement(element)

	// Try to add duplicate
	err := wb.AddElement(element)
	if err == nil {
		t.Error("Should not allow duplicate element")
	}
}

// TestUpdateElement tests updating elements
func TestUpdateElement(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	element := &DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	}

	wb.AddElement(element)

	// Update element
	err := wb.UpdateElement("elem1", func(e *DrawingElement) {
		e.Color = ColorRed
		e.Width = 5.0
	})

	if err != nil {
		t.Fatalf("Failed to update element: %v", err)
	}

	updated, _ := wb.GetElementByID("elem1")
	if updated.Color != ColorRed {
		t.Error("Element color should be updated")
	}

	if updated.Width != 5.0 {
		t.Error("Element width should be updated")
	}
}

// TestDeleteElement tests deleting elements
func TestDeleteElement(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	element := &DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	}

	wb.AddElement(element)

	err := wb.DeleteElement("elem1", "user1")
	if err != nil {
		t.Fatalf("Failed to delete element: %v", err)
	}

	if wb.GetElementCount() != 0 {
		t.Errorf("Expected 0 elements, got %d", wb.GetElementCount())
	}

	_, err = wb.GetElementByID("elem1")
	if err == nil {
		t.Error("Element should not exist after deletion")
	}
}

// TestClearWhiteboard tests clearing the whiteboard
func TestClearWhiteboard(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	// Add multiple elements
	for i := 0; i < 5; i++ {
		wb.AddElement(&DrawingElement{
			ID:     fmt.Sprintf("elem%d", i),
			Type:   ToolPen,
			UserID: "user1",
			Points: []Point{{X: float64(i), Y: float64(i)}},
			Color:  ColorBlack,
			Width:  2.0,
		})
	}

	wb.Clear("user1")

	if wb.GetElementCount() != 0 {
		t.Errorf("Expected 0 elements after clear, got %d", wb.GetElementCount())
	}
}

// TestUndoRedo tests undo/redo functionality
func TestUndoRedo(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	// Add element
	element := &DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	}
	wb.AddElement(element)

	if wb.GetElementCount() != 1 {
		t.Error("Should have 1 element")
	}

	// Undo
	err := wb.Undo()
	if err != nil {
		t.Fatalf("Failed to undo: %v", err)
	}

	if wb.GetElementCount() != 0 {
		t.Errorf("Expected 0 elements after undo, got %d", wb.GetElementCount())
	}

	// Redo
	err = wb.Redo()
	if err != nil {
		t.Fatalf("Failed to redo: %v", err)
	}

	if wb.GetElementCount() != 1 {
		t.Errorf("Expected 1 element after redo, got %d", wb.GetElementCount())
	}
}

// TestUndoRedoLimits tests undo/redo limits
func TestUndoRedoLimits(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	// Try to undo with no history
	err := wb.Undo()
	if err == nil {
		t.Error("Should not be able to undo with no history")
	}

	// Add element
	wb.AddElement(&DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	})

	// Try to redo with nothing to redo
	err = wb.Redo()
	if err == nil {
		t.Error("Should not be able to redo when at end of history")
	}
}

// TestCursorManagement tests cursor tracking
func TestCursorManagement(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	cursor := &Cursor{
		UserID:   "user1",
		UserName: "Alice",
		Position: Point{X: 100, Y: 200},
		Color:    ColorBlue,
	}

	wb.UpdateCursor(cursor)

	cursors := wb.GetCursors()
	if len(cursors) != 1 {
		t.Errorf("Expected 1 cursor, got %d", len(cursors))
	}

	if cursors[0].UserID != "user1" {
		t.Error("Cursor should match")
	}
}

// TestUserJoinLeave tests user management
func TestUserJoinLeave(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	wb.JoinUser("user1")
	wb.JoinUser("user2")

	if wb.GetActiveUserCount() != 2 {
		t.Errorf("Expected 2 active users, got %d", wb.GetActiveUserCount())
	}

	wb.LeaveUser("user1")

	if wb.GetActiveUserCount() != 1 {
		t.Errorf("Expected 1 active user after leave, got %d", wb.GetActiveUserCount())
	}
}

// TestGetElements tests retrieving all elements
func TestGetElements(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	// Add elements
	for i := 0; i < 3; i++ {
		wb.AddElement(&DrawingElement{
			ID:     fmt.Sprintf("elem%d", i),
			Type:   ToolPen,
			UserID: "user1",
			Points: []Point{{X: float64(i), Y: float64(i)}},
			Color:  ColorBlack,
			Width:  2.0,
		})
	}

	elements := wb.GetElements()

	if len(elements) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(elements))
	}
}

// TestExportImportJSON tests JSON export/import
func TestExportImportJSON(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	// Add element
	wb.AddElement(&DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	})

	// Export
	data, err := wb.ExportJSON()
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	if len(data) == 0 {
		t.Error("Exported data should not be empty")
	}

	// Import into new whiteboard
	wb2 := NewWhiteboard("wb2", "Test2", 1920, 1080)
	err = wb2.ImportJSON(data)
	if err != nil {
		t.Fatalf("Failed to import: %v", err)
	}

	if wb2.GetElementCount() != wb.GetElementCount() {
		t.Error("Imported whiteboard should have same element count")
	}
}

// TestGetStats tests statistics retrieval
func TestGetStats(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	wb.AddElement(&DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	})

	wb.JoinUser("user1")

	stats := wb.GetStats()

	if stats["id"] != "wb1" {
		t.Error("Stats should include ID")
	}

	if stats["element_count"] != 1 {
		t.Error("Stats should include element count")
	}

	if stats["active_users"] != 1 {
		t.Error("Stats should include active users")
	}
}

// TestCallbacks tests callback functions
func TestCallbacks(t *testing.T) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	addedCalled := false
	updatedCalled := false
	deletedCalled := false
	clearedCalled := false

	wb.OnElementAdded = func(e *DrawingElement) { addedCalled = true }
	wb.OnElementUpdated = func(e *DrawingElement) { updatedCalled = true }
	wb.OnElementDeleted = func(id string) { deletedCalled = true }
	wb.OnCleared = func() { clearedCalled = true }

	element := &DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	}

	wb.AddElement(element)
	time.Sleep(50 * time.Millisecond)

	if !addedCalled {
		t.Error("OnElementAdded should be called")
	}

	wb.UpdateElement("elem1", func(e *DrawingElement) { e.Color = ColorRed })
	time.Sleep(50 * time.Millisecond)

	if !updatedCalled {
		t.Error("OnElementUpdated should be called")
	}

	wb.DeleteElement("elem1", "user1")
	time.Sleep(50 * time.Millisecond)

	if !deletedCalled {
		t.Error("OnElementDeleted should be called")
	}

	wb.AddElement(&DrawingElement{
		ID:     "elem2",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	})

	wb.Clear("user1")
	time.Sleep(50 * time.Millisecond)

	if !clearedCalled {
		t.Error("OnCleared should be called")
	}
}

// TestDrawingTools tests drawing tool types
func TestDrawingTools(t *testing.T) {
	tools := []DrawingTool{
		ToolPen,
		ToolMarker,
		ToolHighlighter,
		ToolEraser,
		ToolLine,
		ToolRectangle,
		ToolCircle,
		ToolText,
		ToolSelect,
	}

	for _, tool := range tools {
		str := tool.String()
		if str == "Unknown" {
			t.Errorf("Tool %d should have a name", tool)
		}
	}
}

// TestWhiteboardManager tests whiteboard manager
func TestWhiteboardManager(t *testing.T) {
	manager := NewWhiteboardManager()

	if manager.GetWhiteboardCount() != 0 {
		t.Error("Manager should start with 0 whiteboards")
	}
}

// TestWhiteboardManagerCreateGet tests creating and getting whiteboards
func TestWhiteboardManagerCreateGet(t *testing.T) {
	manager := NewWhiteboardManager()

	wb, err := manager.CreateWhiteboard("wb1", "Test", 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to create whiteboard: %v", err)
	}

	if wb == nil {
		t.Fatal("Whiteboard should not be nil")
	}

	if manager.GetWhiteboardCount() != 1 {
		t.Errorf("Expected 1 whiteboard, got %d", manager.GetWhiteboardCount())
	}

	retrieved, err := manager.GetWhiteboard("wb1")
	if err != nil {
		t.Fatalf("Failed to get whiteboard: %v", err)
	}

	if retrieved.ID != "wb1" {
		t.Error("Retrieved whiteboard should match")
	}
}

// TestWhiteboardManagerDuplicate tests preventing duplicate whiteboards
func TestWhiteboardManagerDuplicate(t *testing.T) {
	manager := NewWhiteboardManager()

	manager.CreateWhiteboard("wb1", "Test", 1920, 1080)

	// Try to create duplicate
	_, err := manager.CreateWhiteboard("wb1", "Test", 1920, 1080)
	if err == nil {
		t.Error("Should not allow duplicate whiteboard")
	}
}

// TestWhiteboardManagerDelete tests deleting whiteboards
func TestWhiteboardManagerDelete(t *testing.T) {
	manager := NewWhiteboardManager()

	manager.CreateWhiteboard("wb1", "Test", 1920, 1080)

	err := manager.DeleteWhiteboard("wb1")
	if err != nil {
		t.Fatalf("Failed to delete whiteboard: %v", err)
	}

	if manager.GetWhiteboardCount() != 0 {
		t.Errorf("Expected 0 whiteboards after delete, got %d", manager.GetWhiteboardCount())
	}
}

// TestWhiteboardManagerGetAll tests getting all whiteboards
func TestWhiteboardManagerGetAll(t *testing.T) {
	manager := NewWhiteboardManager()

	manager.CreateWhiteboard("wb1", "Test1", 1920, 1080)
	manager.CreateWhiteboard("wb2", "Test2", 1920, 1080)

	whiteboards := manager.GetAllWhiteboards()

	if len(whiteboards) != 2 {
		t.Errorf("Expected 2 whiteboards, got %d", len(whiteboards))
	}
}

// BenchmarkAddElement benchmarks adding elements
func BenchmarkAddElement(b *testing.B) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb.AddElement(&DrawingElement{
			ID:     fmt.Sprintf("elem%d", i),
			Type:   ToolPen,
			UserID: "user1",
			Points: []Point{{X: float64(i), Y: float64(i)}},
			Color:  ColorBlack,
			Width:  2.0,
		})
	}
}

// BenchmarkUpdateElement benchmarks updating elements
func BenchmarkUpdateElement(b *testing.B) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	wb.AddElement(&DrawingElement{
		ID:     "elem1",
		Type:   ToolPen,
		UserID: "user1",
		Points: []Point{{X: 10, Y: 20}},
		Color:  ColorBlack,
		Width:  2.0,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb.UpdateElement("elem1", func(e *DrawingElement) {
			e.Width = float64(i % 10)
		})
	}
}

// BenchmarkUndoRedo benchmarks undo/redo operations
func BenchmarkUndoRedo(b *testing.B) {
	wb := NewWhiteboard("wb1", "Test", 1920, 1080)

	// Add some elements
	for i := 0; i < 10; i++ {
		wb.AddElement(&DrawingElement{
			ID:     fmt.Sprintf("elem%d", i),
			Type:   ToolPen,
			UserID: "user1",
			Points: []Point{{X: float64(i), Y: float64(i)}},
			Color:  ColorBlack,
			Width:  2.0,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			wb.Undo()
		} else {
			wb.Redo()
		}
	}
}
