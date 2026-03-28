# DNS Peer Discovery для TVCP

## Обзор и контекст

Статья [«Doom Over DNS»](https://habr.com/ru/news/1015672/) демонстрирует нетривиальное использование DNS TXT-записей как канала доставки бинарных данных: вся shareware-версия Doom (~4 MB) упакована в ~1964 TXT-записи и загружается без единого HTTP-запроса — только через DNS, с глобальным кэшем через Cloudflare.

Это вдохновило на анализ: **как аналогичные подходы применимы к TVCP?**

---

## Текущая проблема адресации в TVCP

Сейчас для звонка нужно знать Yggdrasil IPv6-адрес и порт собеседника:

```bash
tvcp call [200:abc:def::1]:5000
```

Это создаёт барьер при onboarding: IPv6-адрес Yggdrasil (`200::/7`) — криптографический, не запоминается, не читается человеком. Пользователи сохраняют контакты локально в `~/.tvcp/contacts.json`, но нет механизма **глобального** обнаружения: чтобы позвонить новому человеку, нужно получить его адрес по внешнему каналу (мессенджер, email и т.д.).

Структура контакта сейчас (`internal/contacts/contact_book.go`):

```go
type Contact struct {
    Name    string `json:"name"`
    Address string `json:"address"` // IP:port — ручное добавление
    Email   string `json:"email"`   // Хранится, но не используется для lookup
    // ...
}
```

---

## Идея: DNS TXT как глобальная книга контактов

### Принцип

Вдохновлённый «Doom Over DNS», предлагается использовать DNS TXT-записи для хранения Yggdrasil-адреса пользователя. Тогда для звонка достаточно знать **имя пользователя и его домен**:

```bash
tvcp call alice@example.com
```

Клиент выполняет DNS lookup и узнаёт адрес автоматически.

### Формат DNS TXT-записи

Запись создаётся владельцем домена:

```
_tvcp.alice.example.com.  TXT  "ygg=200:abc:def::1:5000 v=1"
```

| Поле | Описание |
|------|----------|
| `ygg=` | IPv6-адрес Yggdrasil + порт |
| `v=` | Версия формата (для будущей совместимости) |
| `pubkey=` | (опционально) Публичный ключ Yggdrasil для верификации |

Пример полной записи с публичным ключом:

```
_tvcp.alice.example.com.  TXT  "ygg=200:abc:def::1:5000 v=1 pubkey=a1b2c3d4..."
```

### Аналоги в существующих протоколах

Этот подход — не изобретение велосипеда. Он следует устоявшимся стандартам:

| Протокол | DNS-запись | Пример |
|----------|-----------|--------|
| SIP | SRV + NAPTR | `_sip._udp.example.com SRV ...` |
| XMPP | SRV | `_xmpp-client._tcp.example.com SRV ...` |
| Matrix | `.well-known` + SRV | `_matrix._tcp.example.com SRV ...` |
| Email (MX) | MX | `example.com MX ...` |
| **TVCP (предлагается)** | TXT | `_tvcp.user.example.com TXT ...` |

---

## Сравнение: DNS Discovery vs Книга контактов

| Критерий | Текущий подход (contacts.json) | DNS Peer Discovery |
|----------|-------------------------------|-------------------|
| **Обнаружение** | Ручное, локальное | Автоматическое, глобальное |
| **Формат адреса** | `[200:abc::1]:5000` | `alice@example.com` |
| **Требования** | Нет (всё локально) | Домен + DNS-записи |
| **Конфиденциальность** | Высокая (нет DNS) | Ниже (DNS логируется) |
| **Удобство** | Низкое (ручной ввод) | Высокое (email-подобный синтаксис) |
| **Динамическое обновление** | Ручное | Автоматическое (TTL) |
| **Зависимость** | Нет внешних сервисов | DNS-резолвер |
| **Firewall** | UDP может блокироваться | DNS (53/UDP) обычно открыт |

---

## Реализация: пример кода на Go

Функция разбора `user@domain` и DNS lookup:

```go
package contacts

import (
    "context"
    "fmt"
    "net"
    "strings"
    "time"
)

// DNSContact содержит данные, полученные из DNS TXT-записи
type DNSContact struct {
    Address string // Yggdrasil IPv6:port
    PubKey  string // Публичный ключ (опционально)
    Version string // Версия формата
}

// LookupDNS выполняет DNS TXT lookup для адреса вида "user@domain"
// Запись ищется по имени _tvcp.<user>.<domain>
func LookupDNS(target string) (*DNSContact, error) {
    parts := strings.SplitN(target, "@", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid address format, expected user@domain")
    }
    user, domain := parts[0], parts[1]

    fqdn := fmt.Sprintf("_tvcp.%s.%s", user, domain)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    records, err := net.DefaultResolver.LookupTXT(ctx, fqdn)
    if err != nil {
        return nil, fmt.Errorf("DNS lookup failed for %s: %w", fqdn, err)
    }

    for _, record := range records {
        contact := parseTXTRecord(record)
        if contact != nil {
            return contact, nil
        }
    }

    return nil, fmt.Errorf("no TVCP record found at %s", fqdn)
}

// parseTXTRecord разбирает TXT-запись формата "ygg=... v=1 pubkey=..."
func parseTXTRecord(txt string) *DNSContact {
    fields := strings.Fields(txt)
    contact := &DNSContact{}

    for _, field := range fields {
        kv := strings.SplitN(field, "=", 2)
        if len(kv) != 2 {
            continue
        }
        switch kv[0] {
        case "ygg":
            contact.Address = kv[1]
        case "v":
            contact.Version = kv[1]
        case "pubkey":
            contact.PubKey = kv[1]
        }
    }

    if contact.Address == "" {
        return nil
    }
    return contact
}

// ResolveAddress разрешает адрес: если содержит @, делает DNS lookup,
// иначе возвращает как есть (прямой IP:port)
func ResolveAddress(target string) (string, error) {
    if strings.Contains(target, "@") {
        contact, err := LookupDNS(target)
        if err != nil {
            return "", err
        }
        return contact.Address, nil
    }
    return target, nil // Прямой адрес, DNS не нужен
}
```

Интеграция в CLI (`cmd/tvcp/call.go`) — минимальное изменение:

```go
// Было:
addr := args[0] // [200:abc::1]:5000

// Станет:
addr, err := contacts.ResolveAddress(args[0])
if err != nil {
    return fmt.Errorf("failed to resolve address: %w", err)
}
```

---

## Что взято из «Doom Over DNS» — прямая аналогия

| Концепция в Doom Over DNS | Аналог в TVCP DNS Discovery |
|--------------------------|----------------------------|
| DNS TXT-записи как хранилище данных | TXT-запись хранит Yggdrasil-адрес |
| Cloudflare кэширует данные глобально | DNS TTL кэширует адрес у резолверов |
| Данные разбиты на фрагменты (1964 записи) | Один контакт = одна запись |
| Checksum для верификации целостности | `pubkey=` для верификации идентичности |
| Загрузка без HTTP, только DNS | Обнаружение пира без центрального сервера |
| Базовый синтаксис: PowerShell + DNS API | Базовый синтаксис: `tvcp call user@domain` |

---

## Другие применимые идеи из статьи

### DNS Bootstrap для STUN/TURN

Сейчас STUN/TURN серверы могут быть жёстко зашиты в конфиге. DNS SRV-записи позволяют делать это динамически:

```
_stun._udp.tvcp.example.com.  SRV  0 0 3478 stun.example.com.
_turn._udp.tvcp.example.com.  SRV  0 0 3478 turn.example.com.
```

Клиент автоматически находит STUN/TURN через DNS при запуске.

### DNS как fallback-транспорт для сигнализации

В корпоративных сетях UDP часто блокируется, но DNS (порт 53) почти всегда открыт. DNS tunneling (медленный, ~1-5 Kbps) подходит **только для сигнализации** (handshake, установка звонка), но не для медиапотока.

Схема:
```
Сторона A                    DNS-сервер             Сторона B
    |---[CALL REQUEST via DNS TXT]-->|                  |
    |                                |<--[POLL DNS]-----|
    |                                |---[CALL ACCEPT]->|
    |<===[Media: UDP/Yggdrasil]========================>|
```

### Проверочные суммы для file-sharing

Паттерн «докачки с checksum» из статьи применим к модулю `experimental/` (file sharing). При передаче файлов через TVCP: разбить на чанки, хранить checksums, возобновлять с последнего валидного чанка.

---

## Плюсы и минусы DNS Discovery

### Плюсы
- Человекочитаемые адреса (`alice@example.com`)
- Не нужен центральный сервер TVCP
- Динамическое обновление через TTL (сменил IP → обновил TXT)
- Совместимо с существующей инфраструктурой (любой DNS-провайдер)
- Firewall-friendly: DNS работает даже там, где UDP блокируется
- Бесплатно (через Cloudflare, как в статье)

### Минусы
- Требует владения доменом
- DNS-запросы логируются провайдером (снижение конфиденциальности)
- Зависимость от внешнего DNS-резолвера
- Возможны DNS-спуфинг атаки (нужен DNSSEC или `pubkey=` верификация)
- Не подходит для пользователей без домена (только IPv6-адресация)

---

## Рекомендации по реализации

1. **Сделать опциональным**: DNS Discovery — дополнение к текущей системе, не замена. Если адрес содержит `@` → DNS lookup, иначе → прямое подключение.

2. **Верификация через pubkey**: При lookup сравнивать публичный ключ Yggdrasil из DNS с реальным ключом пира — защита от DNS-спуфинга.

3. **Кэширование**: Кэшировать результат DNS lookup локально на время TTL, чтобы не делать запрос при каждом звонке.

4. **Приватный режим**: Добавить флаг `--no-dns` для пользователей, которые хотят избежать DNS-запросов.

5. **Документация для пользователей**: Инструкция по созданию TXT-записи у популярных провайдеров (Cloudflare, Route53, Namecheap).

---

## Критические файлы для изменений

| Файл | Изменение |
|------|-----------|
| `internal/contacts/contact_book.go` | Добавить `LookupDNS()`, `ResolveAddress()` |
| `cmd/tvcp/call.go` | Вызвать `ResolveAddress()` перед dial |
| `cmd/tvcp/receive.go` | Аналогично для входящих соединений |
| `config.example.yaml` | Добавить `dns_discovery: enabled: true` |

---

## Итог

«Doom Over DNS» показывает: DNS — это не только система разрешения имён, но и **универсальный канал распределённого хранения и доставки данных**. Для TVCP самое ценное из этой идеи — использовать DNS TXT-записи как децентрализованную, не требующую серверов «визитную карточку» пользователя. Это решает главную UX-проблему: сложность первого контакта между пользователями.
