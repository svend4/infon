# Концепт-карта проекта (смысловой срез)

> Смысловой/концептуальный разбор: какие возможности и фичи уже есть и — главное —
> **как они сочетаются и переиспользуются**. Это не технический аудит кода, а карта
> возможностей и узлов переиспользования. Срез по состоянию репозитория:
> ~120 команд в `cmd/`, ~50 пакетов, ~740 публичных символов в одном `pkg/raydir`.

## 1. Семь «миров» (что вообще умеет система)

| Мир | Суть возможности | Где живёт |
|---|---|---|
| **TVCP** — терминальная видеосвязь | реальная P2P/групповая видео+аудио+экран по UDP в терминале, нейро-аватары, SFU, DTN, semantic-codec | `internal/{network,video,audio,sfu,avatar}`, `tvcp`, `connect`, `sem*`, `fold*` |
| **Ray** — CPU-рендер и 3D-миры | путь-трейсер (NEE/MIS/MLT/каустики/BVH/том. туман), сцены, материалы, террейн, мандалы | `pkg/{raytrace,raydir}`, ~40 `ray*` |
| **Q6 / И-Цзин** — мост «смысл↔мир» | гексаграмма ↔ непрерывный вектор ↔ конкретный мир, и обратно | `raydir`: `Hexagram`, `SceneVector`, `brain.SceneSpec` |
| **tvcp-ai/1** — ИИ-протокол и «мозги» | режиссёр миров, игроки, адаптеры Go↔Python | `pkg/brain`, `ai/adapters`, `brainserver`, `aiplay*` |
| **Живое** — симуляции | экосистема, климат-автомат, сезоны, стаи/боиды, метапопуляция | `Ecosystem`, `Climate`, `SeasonalWorld`, `Flock`, `World` |
| **Игры/генеративка** | tangram, arena, uno, wordle, connect4, дебаты, Arecibo/Lincos, гербы, glyph-QR | `tangram*`, `arena*`, `ai*`, `arecibo`, `lincos` |
| **Флот/мониторинг** | оценка техники/роботов, логистика, sim-to-real гейты | `pkg/fleet`, `rayfleet`, `raycamp`, `raygates`, `raywatch` |

## 2. Переиспользуемые «вещества» (хабы — на чём всё держится)

Ядро карты: ~десяток примитивов, которые тянут к себе десятки фич.

| Хаб | Что это | Кто переиспользует |
|---|---|---|
| `raytrace.PathRender` + `PostProcess` + `Scene`/`Camera` | рендер-движок | **почти все** `ray*` визуальные |
| `brain.SceneSpec` | мир как **данные** (не пиксели) на проводе | author→`BuildScene`→render; `DiffScene`; сеть |
| `raydir.SceneVector` (6 Q6-float) | **мост** гексаграмма↔мир, словарь Go≡Python | режиссёр C, читатель G, тур I, звук J, куратор K, подземелье E |
| `raydir.Hexagram` (куб 6-бит) | Flip/Neighbors/Antipode/Hamming/**GrayWalk** | квест D, тур I, агент H, куратор K, reading, climate-seed |
| `raydir.Quest` (лабиринт на кубе) | стены + BFS-решатель | подземелье E, агент H, куратор K |
| `raydir.Ecosystem` (alife) | фуражировка + миграция + richness + **weather-hook** | raylife B, подземелье E, сезоны L, звук-жизни J |
| `network.Transport` + `SceneDelta` + `Pose`/`Presence` | смысл по UDP, дельты, присутствие | raymeet, raycall, raymetaverse, raystream |
| `AvatarFace` + `Pose` + `avatar.Keypoints` | лицо-в-мире из ключевых точек | raygather, raymeet, raymetaverse, rayface |
| `ScoreParams` / `ScoreWAV` / `sona.WAV` | генеративная музыка → PCM/WAV | rayscore, raysound |
| `brain.Brain` (Local/HTTP) | пер-кадровый «решатель» (внешняя модель за `BRAIN_URL`) | AuthorScene, игроки, режиссёр |
| `vision.Detection` / `ReadImage` | пиксели → смысл | aicam, raydetect, rayread |

## 3. Карта сочетаний (как фичи собираются из фич)

Пять главных «конвейеров», по которым кирпичи стыкуются:

**① Смысл → мир → пиксели** (прямой проход):

```
prompt | hexagram → SceneVector / AuthorScene → SceneSpec → BuildScene → PathRender → PostProcess
```

используют: `rayask`, `raydebate`, `raydirect`, `raymetaverse`, `rayquest`, `raytour`, `rayread`.

**② Смысл по проводу** (вместо пикселей):

```
SceneSpec → DiffScene → scene-delta → network.Transport → ApplyScene → локальный рендер
```

`raycall` (видеозвонок), `raymeet` (общий мир), `raymetaverse` (присутствие+дельта), `raystream`.

**③ Куб Q6 как субстрат-граф:**

```
Hexagram (рёбра = смена 1 черты) → Quest (лабиринт) → { агент H, куратор K, подземелье E }
GrayWalk → { атлас/морф I, тур-звук J }
```

**④ Стек «живого»:**

```
Ecosystem → { raylife B, LivingDungeon E (× Quest + Climate), SeasonalWorld L (weather-hook), звук-жизни J }
```

миграция и richness переиспользуются между E и L.

**⑤ Обратная петля (замыкание смысла):**

```
мир → ReadScene / ReadImage → SceneVector → hexagram + Mood
```

G инвертирует C (а `SceneSpec` в Go **побайтно** равен Python `scene_from_vector`).
И петля вовлечённости: `ViewerModel → { LearningDirector C, Curator K }`.

## 4. Матрица плотного переиспользования (серия блоков A–L)

Прямой ответ на «в каких сочетаниях»: каждая новая фича = композиция старых.

| Фича | Тянет к себе |
|---|---|
| A `raygather` | AuthorScene + Hexagram + AvatarFace/Pose/Keypoints + BuildScene + render |
| B `raylife` | **Ecosystem** + Climate + Plane/PathRender |
| C `raydirect` | **SceneVector** (SceneSpec/ReadScene/Mood) + LearningDirector + ViewerModel + render |
| D `rayquest` | **Hexagram → Quest** + Move + VectorFromHexagram → render |
| E `raydungeon` | **Quest (D) × Ecosystem (B)** + Climate + SceneVector-richness (C) + миграция |
| F `raymetaverse` | **Presence** (Pose + Keypoints) + Transport + **delta** (ApplyScene) + Hexagram + AvatarFace |
| G `rayread` | **ReadImage → SceneVector → hex + Mood** (инверсия C) + vision + re-author render |
| H `rayquestai` | **Quest (D)** + QuestAgent (туман войны, BFS-вера) |
| I `raytour` | **Hexagram.GrayWalk** + VectorFromHexagram + TourMorph → render |
| J `raysound` | **SceneVector → ScoreParams** + GrayCode + **Ecosystem (B)** + ScoreWAV |
| K `raycurator` | **Quest (D) × ViewerModel (C)** + Hexagram.Neighbors |
| L `rayseasons` | **Ecosystem weather-hook** + SeasonalWorld + overhead-render |

Видно: `SceneVector`, `Hexagram`, `Quest`, `Ecosystem` — четыре опоры, на пересечениях
которых рождаются все новые фичи (E = D×B, K = D×C, J = C-mapping × B, и т. д.).

## 5. Сквозные принципы (почему так легко переиспользуется)

- **«Смысл, не пиксели»** — `SceneSpec`/`Pose`/`Presence`/`delta` крошечные; картинка
  пересобирается локально. Поэтому рендер, сеть и ИИ стыкуются без переписывания.
- **Детерминизм от seed** — всё воспроизводимо → тестируемо → безопасно переиспользовать.
- **Один словарь по обе стороны** — `SceneVector` идентичен в Go и Python (tvcp-ai/1),
  так что «мозг» и движок говорят на одном языке. Внешняя модель подключается через
  `BRAIN_URL` / `VISION_API_URL` без правок кода (см. `docs/EXTERNAL_MODELS.md`,
  `docs/TVCP_AI_PROTOCOL.md`).
- **«Всё активировано»** — каждый прежде спящий модуль (каустики, MLT, HexCA, Mandala,
  ScoreWAV, BVH…) получил свою команду; новые фичи специально строятся как *сплав*
  существующих, а не сбоку.

---

*Это смысловой срез, а не поштучный перечень всех команд: возможности сгруппированы по
темам, акцент — на узлах переиспользования. Технические детали API — в коде `pkg/raydir`
и в `docs/`.*
