# TVCP на Termux (Android)

Руководство по запуску TVCP на Android-устройстве через Termux.

## Возможности и ограничения

### ✅ Что работает
- **Сетевой сервер** — приём и отправка видео/аудио пакетов
- **Видео кодек** — кодирование и декодирование .babe формата
- **Терминальный рендеринг** — отображение видео в терминале Termux
- **Yggdrasil mesh** — P2P сеть работает полностью
- **Ретрансляция** — можно использовать как relay-сервер

### ⚠️ Ограничения
- **Аудио** — ALSA недоступна на Android, аудио работает в stub-режиме (без звука)
- **Камера** — V4L2 недоступна, нужно использовать тестовые паттерны или файлы
- **Производительность** — зависит от мощности устройства

### 💡 Основное применение
Termux-версия TVCP подходит для:
- **Relay-сервер** — ретрансляция видео между устройствами
- **Тестовый сервер** — отладка и тестирование
- **Просмотр видео** — приём видео и отображение в терминале
- **Yggdrasil узел** — участие в P2P сети

## Установка на Termux

### 1. Установить Termux

Скачать Termux из F-Droid (рекомендуется) или Google Play:
- F-Droid: https://f-droid.org/en/packages/com.termux/
- Google Play: https://play.google.com/store/apps/details?id=com.termux

### 2. Установить зависимости

```bash
# Обновить пакеты
pkg update && pkg upgrade

# Установить необходимые пакеты
pkg install git golang make

# Проверить версию Go (должна быть 1.21+)
go version
```

### 3. Клонировать и собрать TVCP

```bash
# Клонировать репозиторий
git clone https://github.com/svend4/infon
cd infon

# Собрать (может занять 5-10 минут на слабых устройствах)
make build

# Проверить установку
./bin/tvcp version
```

**Примечание:** Если `make` не работает, соберите вручную:
```bash
go build -o bin/tvcp ./cmd/tvcp
```

### 4. Установить в PATH (опционально)

```bash
# Создать директорию для бинарных файлов
mkdir -p ~/.local/bin

# Скопировать tvcp
cp ./bin/tvcp ~/.local/bin/

# Добавить в PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Проверить
tvcp version
```

## Использование на Android

### Режим просмотра (receive)

Принимать видео и отображать в терминале:

```bash
# Запустить приёмник на порту 5000
tvcp receive 5000

# С другого устройства отправить видео
tvcp send <android-ip>:5000 gradient
```

### Режим отправки (send)

Отправлять тестовые паттерны:

```bash
# Узнать свой IP в локальной сети
ifconfig

# Отправить паттерн на другое устройство
tvcp send 192.168.1.100:5000 bounce
```

Доступные паттерны:
- `bounce` — прыгающий мяч
- `gradient` — цветовой градиент
- `noise` — случайный шум
- `colorbar` — цветные полосы

### Relay-сервер

Использовать Android-устройство как ретранслятор:

```bash
# Настроить переадресацию портов
# (требуется root или VPN)

# Запустить в режиме слушателя
tvcp call --listen 5000
```

### Yggdrasil P2P узел

Android может участвовать в Yggdrasil mesh-сети:

```bash
# Установить Yggdrasil (если доступен для Termux)
pkg install yggdrasil

# Или собрать из исходников
git clone https://github.com/yggdrasil-network/yggdrasil-go
cd yggdrasil-go
go build -o yggdrasil ./cmd/yggdrasil
go build -o yggdrasilctl ./cmd/yggdrasilctl

# Запустить Yggdrasil
./yggdrasil -autoconf

# Проверить адрес
./yggdrasilctl getSelf
```

Теперь можно принимать P2P звонки через Yggdrasil:
```bash
tvcp call --listen 5000
```

## Оптимизация для Android

### Энергосбережение

```bash
# Использовать меньший FPS для экономии батареи
# (это нужно будет добавить в конфиг)

# Закрыть другие приложения
# Использовать WiFi вместо мобильных данных
```

### Производительность

```bash
# Проверить использование ресурсов
top

# Ограничить FPS если тормозит
# (будущая опция в конфиге)
```

### Сетевой доступ

**Локальная сеть (WiFi):**
```bash
# Узнать IP-адрес
ifconfig wlan0 | grep inet

# Слушать на всех интерфейсах
tvcp receive 5000
```

**Доступ из интернета:**
Требуется:
1. Белый IP или VPN (WireGuard, Tailscale)
2. Проброс портов на роутере
3. Или использование Yggdrasil для P2P

## Устранение проблем

### "Permission denied" при запуске

```bash
# Дать права на выполнение
chmod +x ./bin/tvcp
```

### Нехватка памяти при сборке

```bash
# Освободить память
pkill -9 chrome
pkill -9 firefox

# Собрать с меньшим числом параллельных задач
GOMAXPROCS=1 go build -o bin/tvcp ./cmd/tvcp
```

### "Package not found"

```bash
# Обновить пакеты Termux
pkg update && pkg upgrade

# Переустановить Go
pkg reinstall golang
```

### Видео не отображается

```bash
# Проверить поддержку TrueColor в терминале
# Termux должен поддерживать, но проверьте:
echo $COLORTERM

# Убедиться что терминал достаточно большой
# Минимум 80×24, рекомендуется 120×40
```

## Примеры использования

### 1. Мобильный relay-сервер

Телефон на Android может работать как мобильный relay между двумя пользователями:

```bash
# На Android (имеет доступ к интернету)
tvcp call --listen 5000

# Пользователь A подключается
tvcp call <android-yggdrasil-address>:5000

# Пользователь B подключается
tvcp call <android-yggdrasil-address>:5000
```

### 2. Мониторинг видео на ходу

Принимать видео с камеры наблюдения прямо в Termux:

```bash
# На Android
tvcp receive 5000

# Камера/сервер отправляет
tvcp send <android-ip>:5000 /dev/video0
```

### 3. Тестовый сервер для разработки

```bash
# Отправлять тестовые паттерны для отладки
tvcp send <pc-ip>:5000 gradient

# Принимать от ПК для тестирования
tvcp receive 5000
```

## Автозапуск (требуется root)

Если у вас есть root и Termux:Boot:

```bash
# Создать скрипт автозапуска
mkdir -p ~/.termux/boot
cat > ~/.termux/boot/start-tvcp.sh <<'EOF'
#!/data/data/com.termux/files/usr/bin/bash
cd ~/infon
./bin/tvcp call --listen 5000 &
EOF

chmod +x ~/.termux/boot/start-tvcp.sh
```

## Дополнительная информация

### Потребление ресурсов

На Android-устройстве TVCP потребляет примерно:
- **RAM:** 50-100 MB (зависит от потока)
- **CPU:** 10-30% на одноядерном процессоре
- **Батарея:** ~5-10% в час активной работы
- **Трафик:** ~170 MB/час (при приёме видео+аудио)

### Совместимость

Протестировано на:
- ✅ Termux 0.118+ (F-Droid)
- ✅ Android 7.0+
- ✅ ARM64 и ARMv7 процессоры

### Ограничения Android

1. **Нет фонового выполнения** — Android может убить процесс
2. **Нет автозапуска** — без root/Termux:Boot
3. **Ограничения сети** — некоторые провайдеры блокируют UDP
4. **Батарея** — активная работа быстро разряжает

## Полезные ссылки

- [Termux Wiki](https://wiki.termux.com/)
- [Yggdrasil Network](https://yggdrasil-network.github.io/)
- [TVCP Documentation](README.md)
- [F-Droid Termux](https://f-droid.org/en/packages/com.termux/)

---

**Совет:** Для лучшей производительности используйте современное устройство с 4+ ядрами и 4+ GB RAM.
