package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTranslator(t *testing.T) {
	tr := NewTranslator()

	if tr == nil {
		t.Fatal("NewTranslator() returned nil")
	}

	if tr.GetLanguage() != English {
		t.Errorf("Default language = %s, expected en", tr.GetLanguage())
	}
}

func TestTranslator_LoadLanguage(t *testing.T) {
	tr := NewTranslator()

	translations := map[string]string{
		"hello": "Привет",
		"bye":   "Пока",
	}

	tr.LoadLanguage(Russian, translations)

	if !tr.HasLanguage(Russian) {
		t.Error("Russian language not loaded")
	}
}

func TestTranslator_SetLanguage(t *testing.T) {
	tr := NewTranslator()

	tr.SetLanguage(Russian)

	if tr.GetLanguage() != Russian {
		t.Errorf("Language = %s, expected ru", tr.GetLanguage())
	}
}

func TestTranslator_T(t *testing.T) {
	tr := NewTranslator()

	// Load English
	tr.LoadLanguage(English, map[string]string{
		"hello": "Hello",
	})

	// Load Russian
	tr.LoadLanguage(Russian, map[string]string{
		"hello": "Привет",
	})

	// Test English
	tr.SetLanguage(English)
	result := tr.T("hello")
	if result != "Hello" {
		t.Errorf("T(hello) = %s, expected Hello", result)
	}

	// Test Russian
	tr.SetLanguage(Russian)
	result = tr.T("hello")
	if result != "Привет" {
		t.Errorf("T(hello) = %s, expected Привет", result)
	}
}

func TestTranslator_T_Fallback(t *testing.T) {
	tr := NewTranslator()

	// Load only English
	tr.LoadLanguage(English, map[string]string{
		"hello": "Hello",
	})

	// Set to Russian (not loaded)
	tr.SetLanguage(Russian)

	// Should fallback to English
	result := tr.T("hello")
	if result != "Hello" {
		t.Errorf("T(hello) = %s, expected Hello (fallback)", result)
	}
}

func TestTranslator_T_Missing(t *testing.T) {
	tr := NewTranslator()

	// Translation not found - should return key
	result := tr.T("nonexistent.key")
	if result != "nonexistent.key" {
		t.Errorf("T(nonexistent) = %s, expected key itself", result)
	}
}

func TestTranslator_Tf(t *testing.T) {
	tr := NewTranslator()

	tr.LoadLanguage(English, map[string]string{
		"greeting": "Hello, %s!",
	})

	result := tr.Tf("greeting", "Alice")
	if result != "Hello, Alice!" {
		t.Errorf("Tf(greeting, Alice) = %s, expected 'Hello, Alice!'", result)
	}
}

func TestTranslator_LoadFromFile(t *testing.T) {
	tr := NewTranslator()

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "test.json")

	// Create test JSON file
	content := `{
		"hello": "Bonjour",
		"bye": "Au revoir"
	}`

	if err := os.WriteFile(jsonFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load from file
	err := tr.LoadFromFile(French, jsonFile)
	if err != nil {
		t.Fatalf("LoadFromFile() failed: %v", err)
	}

	tr.SetLanguage(French)
	result := tr.T("hello")
	if result != "Bonjour" {
		t.Errorf("T(hello) = %s, expected Bonjour", result)
	}
}

func TestTranslator_LoadAllFromDir(t *testing.T) {
	tr := NewTranslator()

	tmpDir := t.TempDir()

	// Create English file
	enFile := filepath.Join(tmpDir, "en.json")
	enContent := `{"hello": "Hello"}`
	os.WriteFile(enFile, []byte(enContent), 0644)

	// Create Russian file
	ruFile := filepath.Join(tmpDir, "ru.json")
	ruContent := `{"hello": "Привет"}`
	os.WriteFile(ruFile, []byte(ruContent), 0644)

	// Load all
	err := tr.LoadAllFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadAllFromDir() failed: %v", err)
	}

	if !tr.HasLanguage(English) {
		t.Error("English not loaded")
	}

	if !tr.HasLanguage(Russian) {
		t.Error("Russian not loaded")
	}
}

func TestTranslator_GetAvailableLanguages(t *testing.T) {
	tr := NewTranslator()

	tr.LoadLanguage(English, map[string]string{"key": "value"})
	tr.LoadLanguage(Russian, map[string]string{"key": "значение"})

	langs := tr.GetAvailableLanguages()

	if len(langs) != 2 {
		t.Errorf("Available languages = %d, expected 2", len(langs))
	}
}

func TestTranslator_GetTranslationCount(t *testing.T) {
	tr := NewTranslator()

	translations := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	tr.LoadLanguage(English, translations)
	tr.SetLanguage(English)

	count := tr.GetTranslationCount()
	if count != 3 {
		t.Errorf("Translation count = %d, expected 3", count)
	}
}

func TestGlobalTranslator(t *testing.T) {
	// Test global functions
	LoadLanguage(Spanish, map[string]string{
		"hello": "Hola",
	})

	SetLanguage(Spanish)

	if GetLanguage() != Spanish {
		t.Errorf("GetLanguage() = %s, expected es", GetLanguage())
	}

	result := T("hello")
	if result != "Hola" {
		t.Errorf("T(hello) = %s, expected Hola", result)
	}
}

func TestDefaultTranslations(t *testing.T) {
	defaults := DefaultTranslations()

	if len(defaults) == 0 {
		t.Error("DefaultTranslations() returned empty map")
	}

	// Check some keys exist
	requiredKeys := []string{
		"app.name",
		"action.call",
		"status.connected",
		"quality.excellent",
	}

	for _, key := range requiredKeys {
		if _, ok := defaults[key]; !ok {
			t.Errorf("DefaultTranslations() missing key: %s", key)
		}
	}
}

func TestInit(t *testing.T) {
	// After init, English should be loaded
	result := T("app.name")
	if result == "app.name" {
		t.Error("English translations not loaded by init()")
	}
}

func BenchmarkTranslator_T(b *testing.B) {
	tr := NewTranslator()
	tr.LoadLanguage(English, DefaultTranslations())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.T("app.name")
	}
}
