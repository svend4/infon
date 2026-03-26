//go:build experimental
// +build experimental

package features

import (
	"fmt"
	"sync"
)

// Stability уровень стабильности функции
type Stability int

const (
	// Alpha - нестабильно, может сломаться, API может измениться
	Alpha Stability = iota
	// Beta - относительно стабильно, API может минимально измениться
	Beta
	// Candidate - готово к переходу в core, стабильное API
	Candidate
)

func (s Stability) String() string {
	switch s {
	case Alpha:
		return "Alpha"
	case Beta:
		return "Beta"
	case Candidate:
		return "Candidate"
	default:
		return "Unknown"
	}
}

// Feature представляет экспериментальную функцию
type Feature struct {
	ID          string
	Name        string
	Description string
	Stability   Stability
	Version     string
	Enabled     bool

	// Метрики
	TestCoverage float64 // Процент покрытия тестами
	TestsPassed  int
	TestsTotal   int

	// Производительность
	PerformanceNote string

	// Зависимости
	Dependencies []string

	// Контакт разработчика
	Maintainer string

	// Риски
	Risks []string
}

// Registry глобальный реестр экспериментальных функций
var Registry = &FeatureRegistry{
	features: make(map[string]*Feature),
}

// FeatureRegistry реестр функций
type FeatureRegistry struct {
	mu       sync.RWMutex
	features map[string]*Feature
}

func init() {
	// Регистрация всех экспериментальных функций

	// WebRTC Audio Processing
	Registry.Register(&Feature{
		ID:          "webrtc-processing",
		Name:        "WebRTC Audio Processing",
		Description: "AEC3, Noise Suppression, AGC, VAD для высококачественной обработки аудио",
		Stability:   Beta,
		Version:     "1.0.0",
		Enabled:     true,
		TestCoverage: 100.0,
		TestsPassed:  31,
		TestsTotal:   31,
		PerformanceNote: "~60μs на фрейм (600x запас реального времени)",
		Dependencies: []string{},
		Maintainer:  "TVCP Team",
		Risks: []string{
			"Может увеличить задержку на 10-20ms",
			"Требует дополнительно ~5% CPU",
		},
	})

	// FFT Filters
	Registry.Register(&Feature{
		ID:          "fft-filters",
		Name:        "FFT Bandpass Filters",
		Description: "Частотная фильтрация на основе FFT для точного контроля спектра",
		Stability:   Beta,
		Version:     "1.0.0",
		Enabled:     false, // Отключено по умолчанию
		TestCoverage: 100.0,
		TestsPassed:  24,
		TestsTotal:   24,
		PerformanceNote: "~1.8ms на 100ms аудио (55x запас)",
		Dependencies: []string{},
		Maintainer:  "TVCP Team",
		Risks: []string{
			"Высокая нагрузка на CPU",
			"Может вызвать артефакты при неправильной настройке",
		},
	})

	// Advanced AEC
	Registry.Register(&Feature{
		ID:          "advanced-aec",
		Name:        "Advanced AEC (NLMS/RLS)",
		Description: "Продвинутые алгоритмы эхоподавления: NLMS и RLS",
		Stability:   Alpha,
		Version:     "0.9.0",
		Enabled:     false, // Экспериментально
		TestCoverage: 100.0,
		TestsPassed:  23,
		TestsTotal:   23,
		PerformanceNote: "NLMS: ~609μs, RLS: ~41ms на фрейм",
		Dependencies: []string{},
		Maintainer:  "TVCP Team",
		Risks: []string{
			"RLS слишком медленный для real-time (41ms > 10ms)",
			"NLMS требует тонкой настройки",
			"Может вызвать артефакты эха при быстрых изменениях",
		},
	})

	// Hardware Acceleration
	Registry.Register(&Feature{
		ID:          "hardware-accel",
		Name:        "Hardware Codec Acceleration",
		Description: "GPU-ускорение видео кодеков (NVIDIA, Intel, AMD, Apple)",
		Stability:   Alpha,
		Version:     "0.8.0",
		Enabled:     true, // С автофолбэком на CPU
		TestCoverage: 100.0,
		TestsPassed:  23,
		TestsTotal:   23,
		PerformanceNote: "10-100x ускорение vs CPU",
		Dependencies: []string{
			"NVIDIA: CUDA drivers",
			"Intel: Media SDK",
			"AMD: AMF",
		},
		Maintainer:  "TVCP Team",
		Risks: []string{
			"Требует наличие GPU",
			"Драйверы могут быть несовместимы",
			"Ограничение на количество одновременных сессий",
			"Качество может отличаться от CPU кодирования",
		},
	})
}

// Register регистрирует новую функцию
func (r *FeatureRegistry) Register(feature *Feature) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.features[feature.ID] = feature
}

// Get возвращает функцию по ID
func (r *FeatureRegistry) Get(id string) (*Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	feature, ok := r.features[id]
	if !ok {
		return nil, fmt.Errorf("feature not found: %s", id)
	}
	return feature, nil
}

// List возвращает список всех функций
func (r *FeatureRegistry) List() []*Feature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features := make([]*Feature, 0, len(r.features))
	for _, f := range r.features {
		features = append(features, f)
	}
	return features
}

// ListEnabled возвращает только включенные функции
func (r *FeatureRegistry) ListEnabled() []*Feature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features := make([]*Feature, 0)
	for _, f := range r.features {
		if f.Enabled {
			features = append(features, f)
		}
	}
	return features
}

// ListByStability возвращает функции с определенным уровнем стабильности
func (r *FeatureRegistry) ListByStability(stability Stability) []*Feature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features := make([]*Feature, 0)
	for _, f := range r.features {
		if f.Stability == stability {
			features = append(features, f)
		}
	}
	return features
}

// Enable включает функцию
func (r *FeatureRegistry) Enable(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	feature, ok := r.features[id]
	if !ok {
		return fmt.Errorf("feature not found: %s", id)
	}

	feature.Enabled = true
	return nil
}

// Disable выключает функцию
func (r *FeatureRegistry) Disable(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	feature, ok := r.features[id]
	if !ok {
		return fmt.Errorf("feature not found: %s", id)
	}

	feature.Enabled = false
	return nil
}

// PrintSummary выводит сводку по всем функциям
func (r *FeatureRegistry) PrintSummary() {
	features := r.List()

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║       Experimental Features Registry                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	for _, f := range features {
		status := "🔴 Disabled"
		if f.Enabled {
			status = "🟢 Enabled"
		}

		stabilityIcon := "⚠️"
		switch f.Stability {
		case Alpha:
			stabilityIcon = "⚠️  Alpha"
		case Beta:
			stabilityIcon = "🔶 Beta"
		case Candidate:
			stabilityIcon = "✅ Candidate"
		}

		fmt.Printf("┌─ %s\n", f.Name)
		fmt.Printf("│  ID:          %s\n", f.ID)
		fmt.Printf("│  Status:      %s\n", status)
		fmt.Printf("│  Stability:   %s\n", stabilityIcon)
		fmt.Printf("│  Version:     %s\n", f.Version)
		fmt.Printf("│  Tests:       %d/%d (%.1f%% coverage)\n",
			f.TestsPassed, f.TestsTotal, f.TestCoverage)
		fmt.Printf("│  Performance: %s\n", f.PerformanceNote)
		fmt.Printf("│  Description: %s\n", f.Description)

		if len(f.Risks) > 0 {
			fmt.Println("│  Risks:")
			for _, risk := range f.Risks {
				fmt.Printf("│    - %s\n", risk)
			}
		}

		fmt.Println("└─")
		fmt.Println()
	}
}
