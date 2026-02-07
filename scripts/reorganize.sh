#!/bin/bash
# Скрипт реорганизации проекта в монорепозиторий

set -e

echo "🔧 Реорганизация проекта TVCP в монорепозиторий..."

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Функция для создания директорий
create_dirs() {
    echo -e "${YELLOW}📁 Создание новой структуры директорий...${NC}"

    mkdir -p core/audio
    mkdir -p core/video
    mkdir -p core/network
    mkdir -p core/codec

    mkdir -p experimental/audio
    mkdir -p experimental/codec
    mkdir -p experimental/features

    mkdir -p server/sfu
    mkdir -p server/signaling
    mkdir -p server/turn

    mkdir -p pkg/protocol
    mkdir -p pkg/utils
    mkdir -p pkg/logger

    mkdir -p cmd/client
    mkdir -p cmd/server
    mkdir -p cmd/tools

    mkdir -p docs/api

    echo -e "${GREEN}✅ Директории созданы${NC}"
}

# Функция для копирования файлов (сохраняет оригиналы)
migrate_files() {
    echo -e "${YELLOW}📦 Миграция файлов...${NC}"

    # Core audio (стабильные файлы)
    if [ -f "internal/audio/audio_linux.go" ]; then
        cp internal/audio/audio_linux.go core/audio/
        echo "  ✓ audio_linux.go → core/audio/"
    fi

    if [ -f "internal/audio/audio_darwin.go" ]; then
        cp internal/audio/audio_darwin.go core/audio/
        echo "  ✓ audio_darwin.go → core/audio/"
    fi

    if [ -f "internal/audio/audio_windows.go" ]; then
        cp internal/audio/audio_windows.go core/audio/
        cp internal/audio/audio_windows_test.go core/audio/
        echo "  ✓ audio_windows.go → core/audio/"
    fi

    # Experimental audio
    if [ -f "internal/audio/webrtc_processor.go" ]; then
        cp internal/audio/webrtc_processor.go experimental/audio/
        cp internal/audio/webrtc_processor_test.go experimental/audio/
        echo "  ✓ webrtc_processor.go → experimental/audio/"
    fi

    if [ -f "internal/audio/fft_filter.go" ]; then
        cp internal/audio/fft_filter.go experimental/audio/
        cp internal/audio/fft_filter_test.go experimental/audio/
        echo "  ✓ fft_filter.go → experimental/audio/"
    fi

    if [ -f "internal/audio/advanced_aec.go" ]; then
        cp internal/audio/advanced_aec.go experimental/audio/
        cp internal/audio/advanced_aec_test.go experimental/audio/
        echo "  ✓ advanced_aec.go → experimental/audio/"
    fi

    # Core codec
    if [ -f "internal/codec/codec.go" ]; then
        cp internal/codec/codec.go core/codec/
        cp internal/codec/codec_test.go core/codec/ 2>/dev/null || true
        echo "  ✓ codec.go → core/codec/"
    fi

    # Experimental codec
    if [ -f "internal/codec/hardware_accel.go" ]; then
        cp internal/codec/hardware_accel.go experimental/codec/
        cp internal/codec/hardware_accel_test.go experimental/codec/
        echo "  ✓ hardware_accel.go → experimental/codec/"
    fi

    # Server
    if [ -d "internal/sfu" ]; then
        cp -r internal/sfu/* server/sfu/
        echo "  ✓ sfu/* → server/sfu/"
    fi

    echo -e "${GREEN}✅ Миграция файлов завершена${NC}"
}

# Функция для добавления build tags
add_build_tags() {
    echo -e "${YELLOW}🏷️  Добавление build tags...${NC}"

    for file in experimental/audio/*.go experimental/codec/*.go; do
        if [ -f "$file" ] && ! grep -q "//go:build experimental" "$file"; then
            # Добавляем build tag в начало файла
            echo "//go:build experimental" > "$file.tmp"
            echo "// +build experimental" >> "$file.tmp"
            echo "" >> "$file.tmp"
            cat "$file" >> "$file.tmp"
            mv "$file.tmp" "$file"
            echo "  ✓ Добавлен build tag: $file"
        fi
    done

    echo -e "${GREEN}✅ Build tags добавлены${NC}"
}

# Функция для создания go.work
create_workspace() {
    echo -e "${YELLOW}🔨 Создание Go workspace...${NC}"

    cat > go.work <<EOF
go 1.24

use (
    .
    ./core
    ./experimental
    ./server
    ./pkg
)
EOF

    echo -e "${GREEN}✅ go.work создан${NC}"
}

# Функция для создания go.mod в поддиректориях
create_modules() {
    echo -e "${YELLOW}📦 Создание go.mod файлов...${NC}"

    # Core module
    if [ ! -f "core/go.mod" ]; then
        cat > core/go.mod <<EOF
module github.com/svend4/infon/core

go 1.24

require (
    github.com/svend4/infon/pkg v0.0.0
)

replace github.com/svend4/infon/pkg => ../pkg
EOF
        echo "  ✓ core/go.mod"
    fi

    # Experimental module
    if [ ! -f "experimental/go.mod" ]; then
        cat > experimental/go.mod <<EOF
module github.com/svend4/infon/experimental

go 1.24

require (
    github.com/svend4/infon/core v0.0.0
    github.com/svend4/infon/pkg v0.0.0
)

replace (
    github.com/svend4/infon/core => ../core
    github.com/svend4/infon/pkg => ../pkg
)
EOF
        echo "  ✓ experimental/go.mod"
    fi

    # Server module
    if [ ! -f "server/go.mod" ]; then
        cat > server/go.mod <<EOF
module github.com/svend4/infon/server

go 1.24

require (
    github.com/svend4/infon/core v0.0.0
    github.com/svend4/infon/pkg v0.0.0
)

replace (
    github.com/svend4/infon/core => ../core
    github.com/svend4/infon/pkg => ../pkg
)
EOF
        echo "  ✓ server/go.mod"
    fi

    # Pkg module
    if [ ! -f "pkg/go.mod" ]; then
        cat > pkg/go.mod <<EOF
module github.com/svend4/infon/pkg

go 1.24
EOF
        echo "  ✓ pkg/go.mod"
    fi

    echo -e "${GREEN}✅ Модули созданы${NC}"
}

# Функция для создания README в каждой директории
create_readmes() {
    echo -e "${YELLOW}📝 Создание README файлов...${NC}"

    # Core README
    cat > core/README.md <<EOF
# Core - Основное ядро TVCP

Стабильные, проверенные компоненты системы.

## Компоненты

- **audio**: Базовая аудио-подсистема (ALSA, WASAPI, CoreAudio)
- **video**: Базовая видео-подсистема
- **network**: Сетевая подсистема
- **codec**: Базовые кодеки (H.264, Opus)

## Требования к добавлению

- ✅ 100% покрытие тестами
- ✅ Документация API
- ✅ Протестировано на всех платформах
- ✅ Обратная совместимость
- ✅ Без экспериментальных зависимостей
EOF

    # Experimental README
    cat > experimental/README.md <<EOF
# Experimental - Экспериментальные функции

Функции в разработке, нестабильное API.

## Как использовать

\`\`\`bash
# Сборка с экспериментальными функциями
go build -tags experimental
\`\`\`

## Функции

- **audio/webrtc_processor**: WebRTC AEC/NS/AGC [Beta]
- **audio/fft_filter**: FFT фильтры [Beta]
- **audio/advanced_aec**: NLMS/RLS алгоритмы [Alpha]
- **codec/hardware_accel**: GPU ускорение [Alpha]

## Уровни стабильности

- **Alpha**: Нестабильно, может сломаться
- **Beta**: Относительно стабильно
- **Candidate**: Готово к переходу в Core
EOF

    # Server README
    cat > server/README.md <<EOF
# Server - Серверные компоненты

Микросервисы для масштабирования системы.

## Компоненты

- **sfu**: SFU сервер для групповых звонков
- **signaling**: Сигнальный сервер
- **turn**: TURN/STUN сервер

## Запуск

\`\`\`bash
# SFU сервер
go run ./server/sfu/cmd/main.go
\`\`\`
EOF

    echo -e "${GREEN}✅ README файлы созданы${NC}"
}

# Функция для создания Makefile
create_makefile() {
    echo -e "${YELLOW}🔨 Создание Makefile...${NC}"

    cat > Makefile <<'EOF'
.PHONY: all build test clean help

# Переменные
BINARY_NAME=tvcp
BUILD_DIR=build
TAGS=

# Цели по умолчанию
all: test build

# Сборка только core
build-core:
	@echo "🔨 Сборка core версии..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME)-core ./cmd/client

# Сборка с experimental
build-experimental:
	@echo "🔨 Сборка с experimental функциями..."
	go build -tags experimental -o $(BUILD_DIR)/$(BINARY_NAME)-exp ./cmd/client

# Сборка всех вариантов
build: build-core build-experimental
	@echo "✅ Сборка завершена"

# Тестирование
test:
	@echo "🧪 Запуск тестов..."
	go test ./core/...
	go test ./pkg/...

test-experimental:
	@echo "🧪 Запуск experimental тестов..."
	go test -tags experimental ./experimental/...

test-all: test test-experimental
	@echo "✅ Все тесты пройдены"

# Бенчмарки
bench:
	go test -bench=. -benchmem ./core/...

bench-experimental:
	go test -tags experimental -bench=. -benchmem ./experimental/...

# Очистка
clean:
	@echo "🧹 Очистка..."
	rm -rf $(BUILD_DIR)
	go clean

# Форматирование
fmt:
	go fmt ./...
	gofmt -s -w .

# Линтинг
lint:
	golangci-lint run

# Установка зависимостей
deps:
	go mod download
	go mod tidy

# Помощь
help:
	@echo "Доступные команды:"
	@echo "  make build-core         - Сборка только core"
	@echo "  make build-experimental - Сборка с experimental"
	@echo "  make build              - Сборка всех вариантов"
	@echo "  make test               - Тесты core"
	@echo "  make test-experimental  - Тесты experimental"
	@echo "  make test-all           - Все тесты"
	@echo "  make bench              - Бенчмарки"
	@echo "  make clean              - Очистка"
	@echo "  make fmt                - Форматирование"
	@echo "  make lint               - Линтинг"
EOF

    echo -e "${GREEN}✅ Makefile создан${NC}"
}

# Главная функция
main() {
    echo ""
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║  Реорганизация TVCP в монорепозиторий                 ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo ""

    # Создаём бэкап
    echo -e "${YELLOW}💾 Создание резервной копии...${NC}"
    if [ -d "../infon-backup" ]; then
        rm -rf ../infon-backup
    fi
    cp -r . ../infon-backup
    echo -e "${GREEN}✅ Бэкап создан: ../infon-backup${NC}"
    echo ""

    # Выполняем миграцию
    create_dirs
    echo ""

    migrate_files
    echo ""

    add_build_tags
    echo ""

    create_workspace
    echo ""

    create_modules
    echo ""

    create_readmes
    echo ""

    create_makefile
    echo ""

    echo "╔════════════════════════════════════════════════════════╗"
    echo "║  ✅ Реорганизация завершена!                          ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo ""
    echo "📋 Следующие шаги:"
    echo "  1. Проверьте новую структуру: ls -la core/ experimental/ server/"
    echo "  2. Запустите тесты: make test-all"
    echo "  3. Соберите проект: make build"
    echo "  4. См. ARCHITECTURE.md для деталей"
    echo ""
    echo "⚠️  Оригинальные файлы сохранены в internal/"
    echo "⚠️  Бэкап создан в ../infon-backup/"
    echo ""
}

# Запуск
main
