package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Language represents a supported language
type Language string

const (
	English    Language = "en"
	Russian    Language = "ru"
	Spanish    Language = "es"
	French     Language = "fr"
	German     Language = "de"
	Chinese    Language = "zh"
	Japanese   Language = "ja"
)

// Translator handles translations
type Translator struct {
	mu              sync.RWMutex
	currentLanguage Language
	translations    map[Language]map[string]string
	fallback        Language
}

// NewTranslator creates a new translator
func NewTranslator() *Translator {
	return &Translator{
		currentLanguage: English,
		translations:    make(map[Language]map[string]string),
		fallback:        English,
	}
}

// LoadLanguage loads translations for a specific language
func (t *Translator) LoadLanguage(lang Language, translations map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.translations[lang] = translations
}

// LoadFromFile loads translations from a JSON file
func (t *Translator) LoadFromFile(lang Language, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read translation file: %w", err)
	}

	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to parse translations: %w", err)
	}

	t.LoadLanguage(lang, translations)
	return nil
}

// LoadAllFromDir loads all translation files from a directory
func (t *Translator) LoadAllFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		// Extract language from filename (e.g., "en.json" -> "en")
		base := filepath.Base(file)
		langCode := base[:len(base)-5] // Remove ".json"
		lang := Language(langCode)

		if err := t.LoadFromFile(lang, file); err != nil {
			return err
		}
	}

	return nil
}

// SetLanguage sets the current language
func (t *Translator) SetLanguage(lang Language) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.currentLanguage = lang
}

// GetLanguage returns the current language
func (t *Translator) GetLanguage() Language {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.currentLanguage
}

// T translates a key to the current language
func (t *Translator) T(key string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Try current language
	if langMap, ok := t.translations[t.currentLanguage]; ok {
		if translation, ok := langMap[key]; ok {
			return translation
		}
	}

	// Fallback to English
	if t.currentLanguage != t.fallback {
		if langMap, ok := t.translations[t.fallback]; ok {
			if translation, ok := langMap[key]; ok {
				return translation
			}
		}
	}

	// Return key if no translation found
	return key
}

// Tf translates with format arguments (like Printf)
func (t *Translator) Tf(key string, args ...interface{}) string {
	translation := t.T(key)
	return fmt.Sprintf(translation, args...)
}

// GetAvailableLanguages returns list of loaded languages
func (t *Translator) GetAvailableLanguages() []Language {
	t.mu.RLock()
	defer t.mu.RUnlock()

	langs := make([]Language, 0, len(t.translations))
	for lang := range t.translations {
		langs = append(langs, lang)
	}
	return langs
}

// HasLanguage checks if a language is loaded
func (t *Translator) HasLanguage(lang Language) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	_, ok := t.translations[lang]
	return ok
}

// GetTranslationCount returns number of translations for current language
func (t *Translator) GetTranslationCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if langMap, ok := t.translations[t.currentLanguage]; ok {
		return len(langMap)
	}
	return 0
}

// Global translator instance
var globalTranslator = NewTranslator()

// SetLanguage sets the global language
func SetLanguage(lang Language) {
	globalTranslator.SetLanguage(lang)
}

// GetLanguage gets the global language
func GetLanguage() Language {
	return globalTranslator.GetLanguage()
}

// T translates using global translator
func T(key string) string {
	return globalTranslator.T(key)
}

// Tf translates with format using global translator
func Tf(key string, args ...interface{}) string {
	return globalTranslator.Tf(key, args...)
}

// LoadLanguage loads to global translator
func LoadLanguage(lang Language, translations map[string]string) {
	globalTranslator.LoadLanguage(lang, translations)
}

// LoadFromFile loads from file to global translator
func LoadFromFile(lang Language, filePath string) error {
	return globalTranslator.LoadFromFile(lang, filePath)
}

// LoadAllFromDir loads all from directory to global translator
func LoadAllFromDir(dir string) error {
	return globalTranslator.LoadAllFromDir(dir)
}

// GetAvailableLanguages gets available languages from global translator
func GetAvailableLanguages() []Language {
	return globalTranslator.GetAvailableLanguages()
}

// DefaultTranslations returns default English translations
func DefaultTranslations() map[string]string {
	return map[string]string{
		// General
		"app.name":    "TVCP",
		"app.version": "Version %s",

		// Actions
		"action.call":      "Call",
		"action.hangup":    "Hang Up",
		"action.answer":    "Answer",
		"action.decline":   "Decline",
		"action.mute":      "Mute",
		"action.unmute":    "Unmute",
		"action.settings":  "Settings",
		"action.exit":      "Exit",

		// Status
		"status.connecting":  "Connecting...",
		"status.connected":   "Connected",
		"status.disconnected": "Disconnected",
		"status.calling":     "Calling...",
		"status.ringing":     "Ringing",

		// Quality
		"quality.excellent": "Excellent",
		"quality.good":      "Good",
		"quality.fair":      "Fair",
		"quality.poor":      "Poor",

		// Settings
		"settings.audio":    "Audio Settings",
		"settings.video":    "Video Settings",
		"settings.network":  "Network Settings",
		"settings.language": "Language",

		// Errors
		"error.connection_failed": "Connection failed",
		"error.call_failed":       "Call failed",
		"error.no_camera":         "No camera found",
		"error.no_microphone":     "No microphone found",
	}
}

// init loads default translations
func init() {
	LoadLanguage(English, DefaultTranslations())
}
