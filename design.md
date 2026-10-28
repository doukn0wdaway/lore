# lore — дизайн

> **Исторический документ.** Не обновляется. Описывает дизайн на 2026-10-27 (этапы 1–3). Актуальное поведение и решения — в `lore/` (зоны и ADR, `lore show`, `lore adr list`); где этот файл расходится с ними, верны они. Например, хук `UserPromptSubmit`, двухэтапный отбор и `ask` на правки зон заменены ADR `2026-10-27-agent-pulls-knowledge-with-ctx` и `2026-10-27-edits-ask-only-for-adrs`. Журнал решений (§15) сохранён: на него ссылаются ADR через `ref:`.

Статус на тот момент: **этапы 1–3 реализованы** (формат, навигация, bootstrap, ADR-операции, changes с воротами, модель решений: хук, `change status`, `change check`; 11 workflow). Дата: 2026-10-27.
Репозиторий: `~/code/personal/lore`.

Документ описывает всё, о чём договорились: зачем инструмент, из каких документов состоит, как идёт работа, где участвует модель решений, какие у бинарника команды и что входит в MVP. В конце — журнал решений с причинами: lore сам должен отвечать на вопрос «почему так».

---

## 1. Зачем

**Главная боль — непонятно, почему сделано так, а не иначе.** Агент смотрит на код, видит «странное» и «исправляет» осознанное решение. Или путается в собственных прошлых решениях: старое и новое противоречат друг другу, и непонятно, какое действует.

**Вторая боль — контекст.** Не хочется каждый раз прогревать весь проект и не хочется, чтобы агент грепал и читал всё подряд.

**Третья, редкая — агент не находит нужное**, когда проект сильно разрастается.

lore — это переосмысление OpenSpec:

- **берём из OpenSpec** формат требований (`### Requirement:` + сценарии WHEN/THEN), дельты (`ADDED / MODIFIED / REMOVED / RENAMED`), тасклист, «умное слияние» дельт агентом;
- **добавляем**:
  - ADR как отдельную сущность (в OpenSpec решения живут в `design.md` и теряются в архиве);
  - автоматическую доставку контекста через модель решений (хук на промпт и `status` перед задачей);
  - code-first режим (код в реалтайме, а доки потом) с теми же воротами, что у spec-first;
  - bootstrap существующего проекта с интервью владельца по решениям;
  - продуктовый слой (`product.md`) отдельно от технического.

### Уроки старого kit (nexus)

Из `nexus/kit/v2-plan.md` и `v2-design.md`. Учтены как дополнительная информация, а не как основа:

- **Дорого число шагов, а не объём чтения.** В CH-022 было ~160 вызовов при контексте 170–200k: каждый вызов заново читает накопленный контекст. Отсюда: контекст приходит к агенту сам.
- **Одно правило лежало в 2–3 местах**, и набралось 11 расхождений. Отсюда: workflow — один файл, к агенту подключается симлинком.
- **Процесс был скопирован с команды людей**: 9 ролей, 7 этапов, подробные планы наперёд, которые устарели до кода. Отсюда: ролей нет, будущее живёт как `planned`-сценарий, а не как готовый change.
- **Решения терялись в архивных `design.md`.** Отсюда: ADR — отдельная живая сущность.

---

## 2. Принципы

1. **Контекст приходит к агенту сам**: хук на промпт, `lore change status` перед задачей. Агент не ходит за ним по файлам.
2. **У каждого факта одно место.** Действие описывается в зоне объекта, реакция — в зоне того, кто реагирует.
3. **«Закон» (зоны, ADR, product, architecture) меняется только через ворота**, то есть через архив change с апрувом владельца.
4. **Пишет агент, владелец апрувит** в двух точках: proposal с черновиками ADR и diff доков при архиве.
5. **«Почему» не выдумывается.** Причину не знаешь — спроси. При bootstrap неизвестное помечается явно.
6. **Противоречие с ADR всегда решает владелец**: новое это решение или ошибка в коде.
7. **Модель решений видит только текст.** Код в текст переводит агент.
8. **Минимум типов и обязательных полей.** Формат проверяет `lore lint`.
9. **Сначала рабочий флоу, оптимизация потом.**

---

## 3. Структура репозитория

```
<repo>/
  lore.yaml                    # конфиг lore
  AGENTS.md                    # блок протокола (вписывает lore init)
  CLAUDE.md                    # содержит строку @AGENTS.md
  .claude/
    settings.json              # permissions.ask на доки lore, allow Bash(lore:*), хуки
    skills/lore-<name>  →  ../../lore/workflows/<name>    (симлинки)
  lore/
    INDEX.md                   # ГЕНЕРИТСЯ: список зон с tldr
    product.md                 # что и зачем, сценарии live/planned
    architecture.md            # зоны и связи (пишется руками/агентом)
    zones/<zone-id>.md
    adr/<YYYY-MM-DD>-<slug>.md
    changes/<name>/            # активные изменения
    changes/archive/<YYYY-MM-DD>-<name>/
    workflows/<name>/SKILL.md  # инструкции агенту (bootstrap, propose, apply, archive, document)
    prompts/*.txt              # тексты вопросов к модели решений
    .env                       # gitignored: ключ lore для этого проекта (не .env приложения!)
    .local/                    # gitignored: локальное состояние
      sessions/<session_id>.json  # что уже подмешано в сессии
      log.jsonl                # лог вызовов модели решений
      bootstrap.json           # на каком шаге bootstrap
```

- Папка с доками **видимая** и отдельная: не `docs/` (там часто чужие документы) и не скрытая (её пропускают люди и агенты).
- Имя папки задаётся в `lore.yaml` (`root`), по умолчанию `lore`.
- **Одна папка на всё.** Локальное некоммитимое (ключ, лог, сессии, прогресс bootstrap) живёт внутри неё в `lore/.env` и `lore/.local/`, оба в `.gitignore`. Старая схема с отдельной `.lore/` мигрирует при `lore upgrade`.
- Корень проекта — директория с `lore.yaml`. CLI ищет её вверх от текущей, как git ищет `.git`. В монорепо корней может быть несколько, специальной поддержки монорепо нет.
- **Язык всех доков lore — английский**: зоны, ADR, product, architecture, workflow, `tldr`, промпты к модели. Этот `design.md` написан по-русски, потому что он для владельца, а не часть lore.

---

## 4. Сущности

Два типа «знания» — **zone** (что это и как себя ведёт) и **adr** (почему так) — плюс два обзорных документа (`product.md`, `architecture.md`) и рабочая сущность **change**.

| | Вопрос | Время | Меняется |
|---|---|---|---|
| `product.md` | Что приложение должно уметь и для кого? | текущее + планы | через ворота |
| `architecture.md` | Как связаны зоны? | текущее | через ворота |
| `zones/*.md` | Что это и как себя ведёт? | текущее | через ворота |
| `adr/*.md` | Почему так, а не иначе? | момент решения | неизменяем, меняется только статус |
| `changes/<name>/` | Что делаем сейчас? | в работе | агент свободно |

Отдельного типа `flow` (сценарий через несколько зон) **нет**: правила сценария живут в зонах, у логики «между шагами» почти всегда есть владелец, а маршрут сценария описан в `product.md`. Отдельного типа `entity` **тоже нет**: сущность — это либо зона (`users`, `orders`), либо раздел внутри зоны.

### 4.1 `INDEX.md` (генерируется)

Всегда в контексте агента через `@lore/INDEX.md`. Только зоны, одна строка на зону. ADR сюда не попадают: их число растёт без ограничений.

```markdown
<!-- generated by lore index — do not edit -->
# lore index

Zones (open with `lore show <id>`):

- **transport** — How users reach the bot: Telegram adapter, commands, message intake.
- **notifications** — Scheduled and event-driven notifications; crons only enqueue.
- **storage** — Files, OCR/whisper, dialogue history, long-term facts.
- **finance** — Expenses, recurring payments, payment reminders.
```

Пересобирается командой `lore index`, а также автоматически при `init` и завершении архива. `lore lint` проверяет, что индекс свежий.

### 4.2 `product.md`

Продуктовый слой: что это, для кого, какие сценарии есть и какие хочется. Это отдельный документ, а не README: README — про запуск и сборку.

```markdown
# Product

## What it is
Personal AI assistant in Telegram and web: holds a dialogue, remembers facts,
reminds about payments.

## Who it is for
<1–3 lines>

## Scenarios
- [live] User sends a voice message → gets a text answer
  zones: transport, storage, ai-core
- [live] Bot reminds about a recurring payment one day before the charge
  zones: finance, notifications, transport
- [planned] User sends a photo of a receipt → the expense lands in finance
  zones: transport, storage, finance
```

- Статусы: `live` (работает) и `planned` (хочется). Файл одновременно описывает текущее состояние и служит роадмапом.
- Proposal change в разделе `## Scenarios` перечисляет, как меняется `product.md`: `planned → live`, изменённые сценарии, удалённые (с причиной). При архиве агент это применяет.
- **`product.md` описывает настоящее, а не историю.** Изменившийся сценарий переписывается на месте, неактуальный удаляется; статуса `[dropped]` нет. История — в архивных change и git. Если причина отказа неочевидна и агент может предложить сценарий снова — это ADR («we don't do X because …»).
- Сценарий — это маршрут по зонам, **без правил**. Правила живут в зонах.

### 4.3 `architecture.md`

Технический обзор: список зон, кто от кого зависит, основные потоки (mermaid по желанию), ключевые технологии, конвенции. Обычный документ без генерации, пишется агентом при bootstrap и меняется через ворота. Граф событий (`emits`) **не делаем**: держать структурное поле в синхроне с текстом — лишний источник расхождений.

### 4.4 Зона — `zones/<id>.md`

**Что такое зона.** Область ответственности со своими правилами, а не папка и не сущность. Признаки:

- назначение описывается в 2–3 строках, без «и» между несвязанными вещами;
- у неё есть свои WHEN/THEN или инварианты, которые не объяснить через другую зону;
- изменение внутри неё обычно требует понимать только **контракты** других зон, а не их устройство.

Зона бывает **вокруг сущности** (`users`, `orders`) или **вокруг возможности** (`notifications`, `transport`, `ai-core`, `design-system`). Ориентир по размеру — 100–300 строк. Больше — кандидат на разделение, меньше ~50 — скорее раздел соседней зоны.

**Правило актора.** Действие живёт в зоне **объекта**, над которым оно совершается, а не в зоне того, кто его совершает. «Юзер создаёт заказ» описывается в `orders`, а не в `users`. Иначе зона по актору становится god-зоной. В `users` остаётся то, что касается самого юзера: регистрация, профиль, роли, его настройки, удаление.

**Правило реакции.** Связь между зонами описывается у того, кто реагирует. `users` в `Contracts` пишет «emits `user.deleted`», а `orders` в требованиях пишет «WHEN `user.deleted` THEN cancel active orders».

**Формат:**

```markdown
---
id: notifications
tldr: Scheduled and event-driven notifications; crons only enqueue, a worker delivers via transport.
---
# Notifications

## Purpose
What the zone is responsible for, 2–3 lines.

## Boundaries
What is NOT here and where it lives instead.

## Requirements

### Requirement: Payment reminder
The system SHALL remind the user about a recurring payment before the charge.

#### Scenario: Reminder three days before
- **WHEN** a recurring payment is due in 3 days
- **THEN** a reminder is enqueued for the payment owner

## Invariants
- ...

## Contracts
- Consumes `payment.scheduled` from finance.
- Delivers only through transport's `Send` contract.

## Entry points
- `internal/notify/scheduler.go` — cron registration
- `notify.Enqueue()` — the only way to send

## Gotchas
- ...
```

**Обязательно** (проверяет lint):

- frontmatter: `id` (совпадает с именем файла) и `tldr` (одна конкретная фраза: по ней работает модель решений);
- раздел `## Purpose`.

**Рекомендуется:** `Boundaries`, `Requirements`, `Invariants`, `Contracts`, `Entry points`, `Gotchas`.

**Разрешены любые свои разделы.** Например, у `design-system` будут `Tokens`, `Spacing`, `Components`: «spacing is a multiple of 4», «for lists use `DataTable`, never hand-roll a table».

**Требования:**

- заголовок `### Requirement: <name>` — это **id требования**, по нему адресуются дельты и модель решений;
- имя уникально внутри зоны;
- 1–4 сценария `#### Scenario:` с **WHEN / THEN** (SHALL, а не should);
- описывается поведение, а не реализация: никаких имён функций, пропсов, форм таблиц.

**Точки входа** — это 3–5 мест, с которых начинать: ориентир, а не полный маппинг «зона → файлы». Полный маппинг (globs) **не делаем**: код размазан, globs протухают при каждом рефакторинге и создают ложное ощущение полноты. lint проверяет только, что указанные пути существуют.

Сгенерированных блоков внутри зоны **нет**. Действующие ADR и связи показывает `lore show` при чтении.

### 4.5 ADR — `adr/<YYYY-MM-DD>-<slug>.md`

**Когда решение заслуживает ADR.** Выполнены оба условия:

1. у решения есть неочевидное «почему»;
2. агент через месяц, глядя на код, может захотеть его «исправить», и это будет ошибкой.

Были ли рассмотрены альтернативы, **не важно**: решение без рассмотренных альтернатив — тоже ADR.

Примеры: «кроны только ставят задания в очередь» — ADR. «Даты в UTC» — инвариант в зоне. «Кнопки синие» — дизайн-гайд в зоне `design-system`.

**Id — дата и slug**, без порядкового номера, чтобы параллельные ветки не конфликтовали: `2026-10-27-crons-enqueue-only`. Имя файла совпадает с id.

**Формат:**

```markdown
---
id: 2026-10-27-crons-enqueue-only
status: accepted            # proposed | accepted | superseded
zones: [notifications, transport]   # только существующие зоны или [global]
tldr: Crons only enqueue jobs; a single worker delivers them.
rules:
  - "DO: enqueue from cron handlers"
  - "DON'T: call transport.Send() from a cron handler"
supersedes: []              # id ADR, которые этот заменяет целиком
superseded_by: null         # заполняется, когда этот заменён
inferred: false             # true — выведено из кода при bootstrap
ref: null                   # опционально: тикет / change
---
# Crons only enqueue jobs

## Context
What problem forced the decision.

## Decision
What we do.

## Why
Why this way. REQUIRED. If unknown: "Unknown (inferred from code)."

## Rejected alternatives
### Send directly from the cron
Why not: a Telegram outage failed the whole cron; retries would be duplicated in every cron.

## Cost
What we knowingly accept: up to N seconds delay, one more table.

## Revisit if
- Load grows past X messages/min → consider a separate delivery service.
```

**Разделы:**

- `Context`, `Decision`, **`Why`** — обязательные, `Why` не может быть пустым (lint);
- `Rejected alternatives`, `Cost`, `Revisit if` — опциональные.

**Как агент трактует альтернативы:**

- альтернатива **отвергнута** (есть в `Rejected alternatives`) → агент не предлагает её снова, пока не сработало условие `Revisit if`;
- альтернатив **не рассматривали** (раздела нет) → агент **может предложить** другой вариант, если видит, что он лучше, но спрашивает владельца, а не переделывает. Решение действует, пока владелец его не заменит.

**`zones`:**

- только id существующих зон (lint), свободных тегов нет: смысловой поиск делает модель решений;
- `[global]` — общее решение («all dates in UTC», «single Go binary»). Такие ADR кандидаты всегда, независимо от выбранных зон.

**Жизненный цикл:**

- `proposed` → `accepted`: переводит только владелец (при апруве proposal или ответом в code-first);
- `accepted` → `superseded`: только новым ADR с `supersedes: [<id>]`.

**Частичной замены нет.** Если новое решение отменяет одно правило из трёх, новый ADR заменяет старый **целиком** и переносит в себя правила, которые остаются в силе. Иначе агенту придётся собирать актуальное решение из нескольких документов.

Заменённые ADR остаются в `adr/` со статусом `superseded` и ссылкой `superseded_by`. В выборку модели решений и в `lore show` по умолчанию не попадают.

**Черновики в change.** `lore adr new … --change <name>` кладёт черновик в `changes/<name>/adr/`. `lore adr accept` переводит его в `accepted` (апрув владельца), но черновик остаётся в change; `lore change archive --finish` переносит его в `lore/adr/` и только тогда помечает заменённые ADR как `superseded`. Для ADR вне change (`/lore-adr`) `accept` делает замену сразу. Статусы руками не правятся — только через `lore adr accept`.

**ADR создаются только в двух местах:**

1. на этапе proposal (черновик лежит в `changes/<name>/adr/`, владелец апрувит вместе с proposal);
2. когда агент упёрся в развилку посреди работы: он **останавливается и спрашивает**. Ответ владельца становится ADR.

Молча, посреди реализации, агент ADR не создаёт. В code-first агент предлагает кандидатов на воротах, владелец подтверждает.

**Мелкие «почему»** (странный кусок кода, неочевидный обход), не дотягивающие до ADR, пишутся обычным комментарием в коде. lore для этого не нужен.

### 4.6 Change — `changes/<name>/`

```
changes/voice-messages/
  .meta.yaml          # mode: spec|code, base: <commit>, created: <date>
  proposal.md
  deltas/<zone>.md    # по файлу на задетую зону
  adr/<id>.md         # черновики ADR (status: proposed)
  tasks.md            # только в spec-first
  design.md           # опционально: технические заметки
```

**`proposal.md`:**

```markdown
# Voice messages

## Why
What problem / which scenario from product.md.

## What changes
- Bullet list of behaviour changes.

## Zones
storage, transport

## Scenarios
- live: "User sends a voice message → gets a text answer" (was planned)
- changed: "Bot reminds 1 day before a payment" → "Bot reminds 3 days before a payment"
- removed: "User exports notes to PDF" — why: nobody used it

## ADR drafts
- 2026-10-27-async-transcription

## Owner notes
> "Let's not transcribe messages longer than 5 minutes" — owner, 2026-10-27
```

`## Owner notes` — ответы владельца по ходу работы, с цитатой. На них опирается сверка на воротах: так видно, что отклонение от плана намеренное. Ответ, который должен действовать и после change, становится кандидатом в ADR.

**`deltas/<zone>.md`** — формат OpenSpec:

```markdown
## ADDED Requirements

### Requirement: Voice messages are transcribed
The system SHALL transcribe incoming voice messages in the background.

#### Scenario: Voice message received
- **WHEN** a user sends a voice message
- **THEN** the file is stored and a whisper transcription job is enqueued

## MODIFIED Requirements

### Requirement: File upload size limit
The system SHALL reject uploads over the size limit.

#### Scenario: File too big
- **WHEN** a file is larger than 20 MB
- **THEN** it is rejected

#### Scenario: Voice message too long
- **WHEN** a voice message is longer than 5 minutes
- **THEN** it is rejected with a hint to split it

## REMOVED Requirements

### Requirement: Text-only intake

## RENAMED Requirements
- FROM: `### Requirement: Old name`
- TO: `### Requirement: New name`

## MODIFIED Sections

### Section: Components
<full new text of the free-form section `## Components`>
```

- `MODIFIED` несёт требование **целиком**, как в OpenSpec: описание и все сценарии, которые остаются в силе, изменённые и нет. При слиянии блок требования в зоне заменяется блоком дельты, поэтому не упомянутый сценарий удаляется (предпроверка архива предупреждает о каждом таком).
- `MODIFIED Sections` — для свободных разделов (не требований): раздел заменяется целиком по заголовку `##`.
- **Новая зона** — дельта на файл, которого ещё нет в `zones/`. Она должна содержать frontmatter (`id`, `tldr`) и `## Purpose`. Архив создаёт зону.

**`tasks.md`** — чистый формат OpenSpec, без добавок:

```markdown
## 1. Storage
- [ ] 1.1 voice_messages table and migration
- [ ] 1.2 Background whisper transcription

## 2. Transport
- [ ] 2.1 Accept voice messages in Telegram
```

- Порядок задаёт нумерация.
- Поля `ctx:`, `verify:`, `depends:` **не делаем**. Контекст для задачи собирает `lore change status` (зоны из дельт и модель решений по тексту задачи).
- Исполняет основной агент, по порядку. Сабагентов на задачу нет.

**`.meta.yaml`:**

```yaml
mode: spec            # spec | code
base: 3f2a91c         # коммит, от которого считается дифф change
created: 2026-10-27
```

---

## 5. Процессы

### 5.1 Spec-first

```
propose → [апрув владельца] → apply (tasks по одной) → ворота → архив
```

1. **propose** (`/lore-propose`): агент выясняет, что делаем, создаёт `lore change new <name>`, пишет `proposal.md`, дельты, `tasks.md`, черновики ADR.
   - Факты из proposal и черновики ADR прогоняются через модель решений против активных ADR (этап 3). Противоречие → вопрос владельцу ещё до апрува.
2. **Апрув**: владелец подтверждает proposal и черновики ADR. ADR → `accepted`, но остаются в папке change до архива.
3. **apply** (`/lore-apply`): перед каждой задачей `lore change status <name>`, выполнить, отметить `[x]`, повторить.
   - Упёрся в развилку → остановиться и спросить. Ответ пишется в `Owner notes`, а если он долгоживущий — становится черновиком ADR.
4. Все задачи закрыты → `status` говорит «run `lore change archive <name>`».
5. **Ворота** (см. 5.3).

### 5.2 Code-first

Владелец пишет код в реалтайме с агентом, без спек. Доки в это время читаются как обычно (хук подмешивает контекст), но не пишутся.

```
код в реалтайме → /lore-document → change --from-diff → ворота → архив
```

1. Владелец говорит «задокументируй» (`/lore-document`).
2. `lore change new <name> --from-diff [--base <ref>]`: создаёт change с `mode: code`, `base` = merge-base с `main_branch` (или `--base`), заготовку `proposal.md`.
3. Агент читает `git diff base..HEAD` и пишет **сводку изменений фактами**: по одному поведенческому факту на строку, обычным языком. Сводка становится основой `proposal.md`.
4. Дальше — ворота, начиная с проверки фактов моделью решений.

Решения в code-first принимал владелец в реалтайме, поэтому агент **не выводит ADR молча**, а предлагает кандидатов на воротах: «ты сделал X, а не Y — записать как ADR? Почему так?»

### 5.3 Ворота (общие для обоих режимов)

Один выход для обоих режимов: если разные режимы будут обновлять доки разными путями, со временем у них разойдутся правила проверки.

`/lore-archive` → `lore change archive <name>`:

1. **Сверка с кодом** (spec-first): агент сравнивает дельты с фактическим `git diff base..HEAD` и правит их под то, что реально сделано: добавляет появившееся по ходу, убирает выкинутое. Опирается на `Owner notes`. В архив попадает **факт, а не план**.
2. **Сводка фактами** (code-first пишет её в 5.2; spec-first — по результату сверки).
3. **Модель решений** (этап 3):
   - факты → зоны: какие зоны задеты. Если задетая зона без дельты, агент дописывает дельту или объясняет, почему её нет;
   - факты → активные ADR: **противоречит ли факт решению**. На каждое противоречие агент спрашивает владельца: «Факт X противоречит ADR Y. Это **новое решение** или **ошибка в коде**?»
     - новое решение → новый ADR с `supersedes: [Y]`, Y → `superseded`;
     - ошибка → change не архивируется, пока код не исправлен.
4. **Кандидаты в ADR** (особенно в code-first): агент предлагает, владелец подтверждает или отклоняет. «Почему» спрашивается, не выдумывается.
5. **`lore lint <name>`** — детерминированно, без модели: `MODIFIED / REMOVED / RENAMED FROM` указывают на существующие заголовки в **текущем** тексте зоны, формат верный, ADR в порядке; предупреждение, если требование трогает и другой активный change.
6. **Слияние**: агент вливает дельты в зоны по workflow (как `openspec-sync-specs`):
   - ADDED: добавить (если такое требование уже есть — заменить);
   - MODIFIED: заменить блок требования целиком блоком дельты;
   - REMOVED: удалить блок;
   - RENAMED: переименовать заголовок;
   - MODIFIED Sections: заменить раздел целиком;
   - новая зона: создать файл;
   - сценарий в `product.md`: `planned` → `live`; при необходимости обновить `architecture.md`;
   - черновики ADR: перенести в `lore/adr/`;
   - операция идемпотентна.

   Правки доков lore идут через `permissions.ask`: владелец видит каждую, на время архива можно разрешить до конца сессии.
7. **Diff доков**: `git diff -- lore/` владельцу на апрув.
8. **`lore change archive [<name>] --finish`**: проверить слияние детерминированно — блок каждого ADDED/MODIFIED-требования и MODIFIED-раздела в зоне **совпадает с блоком дельты** строка в строку и по порядку (комментарии, пустые строки и пробелы не считаются) — ни одной недостающей строки и ни одной оставшейся от старого текста, REMOVED исчезли, переименования применены, новые зоны созданы, все черновики ADR `accepted`; затем пересобрать `INDEX.md`, перенести черновики ADR в `lore/adr/` и проставить `superseded` заменённым, перенести change в `changes/archive/<date>-<name>/` (в `.meta.yaml` пишутся `archived` и `head`).

### 5.4 Bootstrap существующего проекта

Полноценная часть lore, а не «потом». Workflow `/lore-bootstrap`, шаги по порядку, каждый опирается на предыдущий. Апрув стоит там, где ошибка дороже всего:

| # | Шаг | Кто | Апрув |
|---|---|---|---|
| 1 | `product.md`: сценарии из кода, README, роутов, команд | агент | да |
| 2 | **Карта зон**: список, границы, `tldr` (без текста зон) | агент | **да, главный** |
| 3 | `architecture.md` | агент | да |
| 4 | Зоны по одной: Purpose, Requirements (WHEN/THEN), Invariants, Contracts, Entry points | агент | каждая |
| 5 | **Интервью по ADR** | агент спрашивает, владелец отвечает | по ходу |
| 6 | Якоря в коде: 5–10 критичных мест | агент | да |

- **Карта зон — главные ворота.** Если границы нарезаны криво, всё дальше будет кривым. Дешевле всего поправить именно здесь: это одна страница списка.
- **Интервью, а не генерация.** Из кода агент выводит **что** решено, но не **почему**: причина в голове у владельца. Агент находит кандидатов (странные обходы, неочевидные ограничения, «зачем тут очередь») и задаёт вопросы по одному.
  - Владелец ответил → ADR с настоящим `Why`, `accepted`.
  - «Не помню» → ADR с `inferred: true` и `Why: Unknown (inferred from code).` Он всё равно полезен: «сделано намеренно, без вопроса не трогать».
- **Прерываемость**: прогресс в `lore/.local/bootstrap.json` (шаг, готовые зоны). Проект на 20 зон за одну сессию не пройти.
- Дополнительные источники, если есть: существующие доки, спеки, ADR (например, `nexus/docs/`), но только как подсказка. Главный источник — код и ответы владельца.
- Миграцию с OpenSpec отдельно **пока не делаем**.

### 5.5 Параллельная работа

- **Id ADR — дата и slug**, коллизии между ветками практически невозможны.
- **Два change трогают одно требование** — ничего не блокируется:
  - `lore lint` предупреждает о пересечении активных change;
  - при архиве второго агент вливает дельту в **текущий** текст зоны (умное слияние так и работает), расхождение видно в diff на воротах.

---

## 6. Модель решений

### 6.1 Что это и какая

Модель, которая не пишет текст, а отвечает на типизированные вопросы (да/нет, выбор, шкала) откалиброванными вероятностями. Это общий контракт OpenRouter (Decisions API / System One), под ним много моделей, поэтому модель меняется одной строкой конфига.

**Берём `perplexity/pplx-decider-v1.2`** (27B, контекст 262k, $0.02 за 1M входных токенов, выход бесплатный). Выбор по `~/code/personal/jev-bench`:

- **adr-all** (все 49 ADR кандидатами): формулировка `relates` даёт **R 95.6%** (F1 67.0) — лучшая полнота среди моделей. Для подсказки «прочитай эти ADR» полнота важнее точности. У Jev 1.14 R 61.8%;
- **touches** (какие зоны задевает изменение): F1 **89.7**, R 94.5 — лучший результат;
- flash-версия (4B) не подходит: на adr-all R 47–50%;
- минус — скорость: p50 ~4.1 с на 49 кандидатов одним запросом. Отсюда два этапа (6.4).

Найдено в бенче: **формулировка вопроса меняет полноту с 52.9% до 95.6%**. Поэтому тексты вопросов лежат в `lore/prompts/`, а не в Go-коде.

### 6.2 API

```
POST https://openrouter.ai/api/alpha/decisions
Authorization: Bearer $OPENROUTER_API_KEY

{
  "model": "perplexity/pplx-decider-v1.2",
  "state": "<текст: промпт владельца / задача / факт>",
  "questions": {
    "z_notifications": {
      "type": "noul",
      "instructions": "Does zone «notifications» relate to this work — does the work implement, rely on or have to respect it? Zone: Scheduled and event-driven notifications; crons only enqueue.",
      "criteria": {
        "true": "<описание кандидата>",
        "false": "Zone «notifications» does not relate to this work."
      }
    },
    "z_finance": { ... }
  }
}
```

Ответ: `{"answers": {"z_notifications": {"type": "noul", "noul": 0.91}, ...}, "usage": {...}}`. Типы вопросов: `noul` (вероятность «да»), `choice`, `score`. Эталонная реализация запроса — `jev-bench/internal/bench/jev.go` (`jevRequest`). Лимит числа вопросов в одном вызове проверить при реализации.

### 6.3 Что видит модель

**Только текст.** Кандидаты:

| Кандидат | Текст для модели |
|---|---|
| зона | `id` + `tldr` |
| требование | `zone › name` + первая фраза + сценарии |
| свободный раздел зоны | `zone › section` + первые строки |
| ADR | `id` + `tldr` + `rules` |

Состояние (`state`): промпт владельца (хук), текст задачи (`status`), факт из сводки (ворота). Код модели не показывается никогда: его в факты переводит агент.

Промпты владельца русские, доки английские, то есть сопоставление идёт между языками. В бенче такого случая не было. Проверить на пилоте.

### 6.4 Два этапа

1. **Зоны.** ~20 кандидатов с короткими `tldr`, ~0.65 с. Порог **низкий**: лучше взять лишнюю зону, чем потерять нужную. Полнота второго этапа ограничена полнотой первого.
2. **Требования, свободные разделы и ADR из выбранных зон**, плюс ADR с `zones: [global]`. Кандидатов десяток-два вместо всех, ~1–2 с.

Итого ~2–3 с на содержательный промпт.

### 6.5 Где используется

| Место | state | Кандидаты | Результат |
|---|---|---|---|
| хук `UserPromptSubmit` | промпт владельца | 2 этапа | блок `[lore]` в контексте |
| `lore change status` | текст следующей задачи | 2 этапа (+ зоны из дельт без модели) | контекст задачи |
| ворота, факты → зоны | каждый факт сводки | зоны | задетые зоны |
| ворота, факты → ADR | каждый факт сводки | активные ADR | противоречия → вопрос владельцу |
| propose | факты proposal, черновики ADR | активные ADR | противоречия до апрува |

Для проверки противоречия отдельный вопрос (`prompts/contradicts.txt`): «Contradicts the decision: does this fact violate the decision or its rules?»

### 6.6 Что подмешивается

Три корзины по вероятности (пороги в `lore.yaml`):

- **высокая** → **содержимое**: текст требования со сценариями, `tldr` + `rules` ADR;
- **средняя** → указатель: «maybe relevant: `finance › Payment reminder`»;
- **низкая** → ничего.

Плюс **бюджет** на один промпт (~2–3k токенов): если кандидатов много, берутся лучшие по вероятности.

Пример блока:

```
[lore] relevant to this prompt:

ADR 2026-10-27-crons-enqueue-only (0.88)
  Crons only enqueue jobs; a single worker delivers them.
  DON'T: call transport.Send() from a cron handler

finance › Payment reminder (0.81)
  The system SHALL remind the user about a recurring payment before the charge.
  - WHEN a recurring payment is due in 3 days THEN a reminder is enqueued

maybe relevant: notifications › Delivery retries (0.44), zone storage (0.41)
```

### 6.7 Защита от шума и сбоев

- **Короткие промпты** («ок», «продолжай», «да») не прогоняются: порог по словам (`hook.min_words`).
- **Без повторов в сессии**: что уже подмешано (по `session_id`), второй раз не шлётся. Состояние в `lore/.local/sessions/`.
- **Таймаут** хука (~6 с): не успели — подмешивается то, что дал этап 1 (список зон), или ничего. Промпт не блокируется.
- **Нет ключа / API недоступен** → хук молча ничего не подмешивает. Страховка — `INDEX.md` в постоянном контексте и `lore show`.
- **Лог** каждого вызова в `lore/.local/log.jsonl`: state, кандидаты, вероятности, что подмешано. Нужен для отладки и будущего подбора порогов.

---

## 7. Команды CLI

Go, один бинарник. Вывод по умолчанию — markdown для агента, `--json` — для скриптов. Код возврата ≠ 0 при ошибке lint или проверки.

### Этап 1 (реализован)

**`lore init`**
- создаёт `lore.yaml`, `lore/` со скелетом (`product.md`, `architecture.md`, пустые `zones/`, `adr/`, `changes/`), встроенные `workflows/` и `prompts/`;
- вписывает блок протокола в `AGENTS.md` (между маркерами `<!-- lore:begin -->` / `<!-- lore:end -->`), создаёт или дополняет `CLAUDE.md` строкой `@AGENTS.md`;
- `.claude/settings.json`: `permissions.allow: Bash(lore:*)`, `permissions.ask` на Edit/Write доков lore, хук `UserPromptSubmit` (на этапе 3);
- симлинки skills (`lore agents sync`);
- `lore/.env` и `lore/.local/` в `.gitignore`.

**`lore index`** — пересобирает `lore/INDEX.md`.

**`lore lint [<change>]`** — проверки из раздела 10. Без аргумента — весь lore и все активные change; с аргументом — фокус на одном change.

**`lore show <zone>`** — печатает зону, а под ней действующие ADR этой зоны и `global` (`tldr` + `rules` + id). Рекомендуемый способ открыть зону.

```
$ lore show notifications
<текст зоны>

--- Active decisions (notifications, global) ---
2026-10-27-crons-enqueue-only
  Crons only enqueue jobs; a single worker delivers them.
  DON'T: call transport.Send() from a cron handler
2026-09-02-all-times-utc  [global]
  ...
```

**`lore adr new <slug> --zones a,b [--title "..."] [--supersedes id,...] [--change <name>]`** — создаёт ADR из шаблона (`status: proposed`, id с сегодняшней датой). С `--change` — черновик в папке change (зоны могут быть новыми зонами этого change).

**`lore adr accept <id>`** — апрув владельца: `proposed` → `accepted`. Отказывает, пока у ADR есть ошибки lint (пустой `Why` и т.п.). Для ADR в `adr/` сразу помечает заменённые (`supersedes`) как `superseded`; для черновика в change — откладывает это до архива.

**`lore adr list [--zone z] [--status s]`** — список ADR: id, статус, зоны, tldr.

**`lore agents sync`** — создаёт или чинит симлинки `.claude/skills/lore-<name>` → `lore/workflows/<name>`.

**`lore upgrade`** — обновляет встроенные workflow и prompts до версии бинарника (локально правленные не затирает: показывает дифф, перезапись только с `--force`), дописывает в `lore.yaml` ключи, появившиеся в новых версиях (рядом с соседями из шаблона, с комментариями; существующие значения и форматирование не трогает; вложенные `hook.*`, `thresholds.*`, `prompts.*` — внутрь своего блока), переписывает блок `AGENTS.md`, чинит симлинки и переносит старую `.lore/` в `lore/.local/`.

### Этап 2 (реализован)

**`lore change new <name> [--from-diff] [--base <ref>]`** — создаёт `changes/<name>/` (`.meta.yaml`, заготовка `proposal.md`; без `--from-diff` ещё и `tasks.md`).

**`lore change list`** — активные change: имя, режим, прогресс задач.

Во всех `lore change …` имя можно опустить, если активный change один («текущий»).

**`lore change status [<name>]`** — следующая незакрытая задача и её контекст:

```
change: voice-messages   mode: spec   progress: 1/3

NEXT TASK
  1.2 Background whisper transcription

DELTAS IN THIS CHANGE
  storage:   ADDED "Voice messages are transcribed"
  transport: ADDED "Telegram voice messages are accepted"
  <полный текст дельт>

RELEVANT DECISIONS            (этап 3; до него — ADR зон из дельт)
  2026-10-27-crons-enqueue-only
    Crons only enqueue jobs. DON'T: call transport.Send() from a cron handler

RELEVANT REQUIREMENTS         (этап 3)
  storage › File upload size limit
    WHEN a file exceeds 20 MB THEN reject ...
```

Когда всё закрыто: `all tasks done → run lore change archive voice-messages`.

**`lore change archive [<name>]`** — предпроверка до слияния: задачи закрыты (spec-first), `MODIFIED / REMOVED / RENAMED FROM / MODIFIED Sections` указывают на существующие заголовки текущей зоны, `ADDED` ещё не существует (иначе предупреждение), `MODIFIED` не теряет сценарии зоны (иначе предупреждение), новая зона — полноценный док, черновики ADR без ошибок (не принятые — предупреждение); печатает шаги ворот.
**`lore change archive [<name>] --finish`** — после слияния и апрува: проверка слияния, перенос черновиков ADR, index, архив (см. 5.3 п.8).

### Этап 3 (реализован)

**`lore ctx "<text>"`** — двухэтапный отбор по тексту, печатает блок `[lore]`.
**`lore ctx --hook`** — то же для хука `UserPromptSubmit`: читает JSON из stdin (`prompt` или `prompt_text`, `session_id`, `cwd`), пропускает короткие промпты и slash-команды, не повторяет подмешанное в сессии, укладывается в таймаут, отдаёт `hookSpecificOutput.additionalContext` (≤ 9.5k символов). Никогда не валит промпт: любая проблема — в лог, вывод пустой.
**`lore change check [<name>]`** — каждый факт из `## What changes` против зон и активных ADR: задетые зоны (и зоны без дельты), возможные противоречия. Используется в workflow ворот, propose и document. Результат — кандидаты для вопроса владельцу, не вердикт.
**`lore change status`** с ключом дополнительно подбирает требования и ADR под текст следующей задачи.

Замеры вживую (pplx-decider-v1.2, 3 зоны, 2 ADR): `ctx` ~1.0–1.2 с, `change check` на 2 факта ~2 с. Русский промпт против английских доков сработал (требование 0.96–1.00, ADR 0.83–0.93); `check` поймал оба подложенных противоречия («шлём из крона» против ADR очереди — 0.61, «храним по Москве» против UTC — 1.00).

### Потом

`lore status` (детектор отставания доков по коммитам с последней синхронизации), `lore eval`, `lore why`.

---

## 8. Интеграция с агентами

Сейчас поддерживается только Claude Code. Другие агенты — идея на потом: для них появится другой генератор в `lore agents sync`, а формат workflow не изменится.

### 8.1 Блок в `AGENTS.md`

Загружается в каждую сессию, поэтому только то, без чего агент сломает систему. Процессы и форматы живут в workflow.

```markdown
<!-- lore:begin -->
## lore

Project knowledge lives in `lore/`. Zone list: @lore/INDEX.md

- Context for your task is injected automatically (`[lore]` block). Use it.
  Need more: `lore ctx "<what you need>"`; a whole zone: `lore show <zone>`.
- Accepted ADRs are binding. If your work contradicts one, stop and ask
  the owner — never work around it, never "fix" it silently.
- ADR without "Rejected alternatives": you may suggest an alternative —
  ask, don't act.
- Never invent the "why" of a decision. Unknown → ask.
- Don't create ADRs during implementation. Hit a fork → ask the owner.
- Docs in `lore/` (zones, ADRs, product, architecture) are the project's
  law. They change only at the gate (`lore change archive`) or through a lore
  workflow the owner invoked (bootstrap, zone, adr, product) — never on your
  own initiative.
- Working on a change: `lore change status <name>` before each task.
- Workflows: /lore-propose, /lore-apply, /lore-archive, /lore-document,
  /lore-bootstrap.
<!-- lore:end -->
```

Блок генерируется из шаблона: строки про хук и changes появляются, только когда бинарь их поддерживает, а список `Workflows:` собирается из установленных `lore/workflows/*`. `lore upgrade` переписывает блок между маркерами.

`CLAUDE.md` содержит `@AGENTS.md`.

### 8.2 Workflow

Инструкции для шагов, которые делает агент. Источник правды — `lore/workflows/<name>/SKILL.md` в репо, в формате skill Claude Code (frontmatter `name`, `description`). Их можно править под проект.

| Workflow | Что делает |
|---|---|
| `bootstrap` | 5.4: шесть шагов, интервью, прогресс в `lore/.local/bootstrap.json` |
| `propose` | 5.1 п.1: proposal, дельты, tasks, черновики ADR, проверка против ADR |
| `apply` | 5.1 п.3: цикл `status` → задача → `[x]`, развилки → вопрос + Owner notes |
| `document` | 5.2: code-first, change из диффа, сводка фактами, кандидаты в ADR |
| `archive` | 5.3: сверка, факты, противоречия, ADR, слияние, diff, `--finish` |
| `adr` | записать решение: критерий, поиск конфликтов, «почему» от владельца, замена старого, `accept`, якоря |
| `why` | ответить «почему так?» по ADR и зонам с цитатой; нет записи — так и сказать, догадки помечать как догадки |
| `zone` | создать / обновить / разделить / слить зону, поправить ссылки в ADR, product, architecture |
| `product` | сценарии: добавить `[planned]` (лёгкий бэклог), `[live]`, изменить, удалить |
| `audit` | сверить доки с кодом по зонам: stale / missing / нарушения ADR / устаревшие сценарии; только отчёт |
| `explore` | режим размышления до (или во время) change: опирается на `lore ctx`, зоны, ADR и сценарии; поднимает столкновения идей с решениями; без кода и правок доков; в конце предлагает propose / adr / product |

Операции вне change (`adr`, `zone`, `product`) — для этапа «между фичами»: правки идут с апрувом владельца и `lore lint`. Фичи меняют доки через ворота.

**Подключение — симлинками**: `.claude/skills/lore-archive` → `../../lore/workflows/archive`. Копий нет, рассинхрон невозможен, правка workflow действует сразу.

Симлинк на директорию skill **официально поддерживается** (документация Claude Code, раздел про skills). Скилл вызывается по полю `name` из frontmatter, поэтому оно равно `lore-<name>`. При `git clone` без `core.symlinks` (Windows) симлинки превращаются в файлы — тогда `agents sync` будет делать копию.

`lore upgrade` обновляет встроенные workflow; локальные правки не затираются молча.

### 8.3 `.claude/settings.json`

```json
{
  "permissions": {
    "allow": ["Bash(lore:*)"],
    "ask": [
      "Edit(/lore/zones/**)",
      "Edit(/lore/adr/**)",
      "Edit(/lore/product.md)",
      "Edit(/lore/architecture.md)"
    ]
  },
  "hooks": {
    "UserPromptSubmit": [
      { "hooks": [{ "type": "command", "command": "lore ctx --hook", "timeout": 10 }] }
    ]
  }
}
```

- Пути вида `/lore/...` считаются от корня проекта (а не от cwd), поэтому правило работает, откуда бы ни запустили сессию. Отдельные правила `Write(...)` не нужны: Claude Code сверяет все правки только с `Edit(...)`.
- Хук `UserPromptSubmit`: `timeout` в секундах; контекст добавляется через stdout или `hookSpecificOutput.additionalContext` (до 10k символов). Имя поля с текстом промпта во входном JSON — уточнить на этапе 3 (справка называет `prompt_text`, по памяти — `prompt`).
- **Читать** доки lore агент может как угодно, `lore show` просто удобнее.
- **Править** может, но каждый раз со спросом (`ask`). Никаких окон записи и флагов.
- Свою папку `lore/changes/<name>/` агент правит без вопросов.

---

## 9. Якоря в коде

Комментарий только там, где правило ADR легко нарушить: 5–10 мест на проект, а не разметка каждого файла.

```go
// ADR 2026-10-27-crons-enqueue-only: never call transport.Send() here, only enqueue.
```

- Агент видит правило ровно в момент, когда собирается его нарушить, без всяких хуков.
- Якорь переезжает вместе с кодом при рефакторинге.
- lint: id существует и ADR не `superseded`.

---

## 10. `lore lint`

**Ошибки:**

- зона: нет `id` / `tldr`, `id` ≠ имени файла, нет `## Purpose`, дубли имён `### Requirement:`;
- ADR: нет `id` / `status` / `zones` / `tldr`, `id` ≠ имени файла, пустой или отсутствующий `## Why`, `zones` ссылается на несуществующую зону, `supersedes` / `superseded_by` указывают в никуда, `superseded` без `superseded_by`;
- дельта: `MODIFIED` / `REMOVED` / `RENAMED FROM` / `MODIFIED Sections` указывают на несуществующий заголовок в текущей зоне; новая зона без `id` / `tldr` / `Purpose`;
- якорь в коде ссылается на несуществующий или `superseded` ADR;
- `Entry points`: путь не существует;
- `INDEX.md` устарел;
- `product.md`: сценарий ссылается на несуществующую зону.

**Предупреждения:**

- два активных change трогают одно требование;
- зона больше ~300 строк или меньше ~50;
- `tldr` слишком короткий или общий (меньше N слов);
- ADR `inferred: true` (напоминание, что «почему» неизвестно);
- в spec-first change при архиве не все задачи закрыты.

---

## 11. `lore.yaml`

```yaml
root: lore                       # папка с доками
main_branch: main                # база для --from-diff

decisions_model: perplexity/pplx-decider-v1.2
decisions_api_key_env: LORE_OPENROUTER_API_KEY   # ИМЯ переменной, не ключ
decisions_endpoint: https://openrouter.ai/api/alpha/decisions
decisions_provider: [Perplexity]                  # порядок провайдеров, без фолбэков

hook:
  enabled: true
  min_words: 4                   # короче — не прогоняем
  timeout_ms: 6000
  budget_tokens: 2500            # сколько контента подмешивать на промпт

thresholds:
  zones: 0.3                     # этап 1 — низкий, чтобы не терять зоны
  include_content: 0.7           # выше — подмешать содержимое
  include_pointer: 0.4           # выше — подмешать указатель
  contradiction: 0.5             # выше — вопрос владельцу на воротах

prompts:                         # тексты вопросов к модели
  relates: lore/prompts/relates.txt
  contradicts: lore/prompts/contradicts.txt

anchors_ignore: []               # пути, где lint не ищет якоря ADR: файлы, папки, globs;
                                 # шаблон без `/` совпадает с именем файла на любой глубине
```

Значения порогов стартовые и будут подбираться на пилоте.

**Ключ — собственный ключ lore, отдельно от env приложения.** Поиск по порядку: переменная окружения `LORE_OPENROUTER_API_KEY` → `lore/.env` проекта (gitignored) → `~/.config/lore/.env` (на все проекты). `.env` проекта lore не читает: он принадлежит приложению. Если в `decisions_api_key_env` вписан сам ключ, а не имя, `lint` и все команды с моделью выдают ошибку, не показывая значение.

---

## 12. MVP

Каждый этап полезен сам по себе.

### Этап 1 — формат, навигация, bootstrap (без модели)

- `init`, `index`, `lint`, `show`, `adr new`, `adr list`, `agents sync`, `upgrade`;
- шаблоны зоны, ADR, product, architecture;
- блок `AGENTS.md`, `settings.json` (`ask`, `allow`);
- workflow `bootstrap`;
- **пилот: bootstrap на `~/code/personal/nexus`**. Существующие `nexus/docs/` (specs, decisions) и старый `nexus/kit/` — дополнительный источник, не основа: старый kit сложный и плохо работает, а ADR там плохие.

Готово, когда: у nexus есть `lore/` с картой зон, зонами и ADR из интервью, `lint` чистый, агент ходит по `INDEX.md` и `lore show`.

### Этап 2 — changes (реализован)

- `change new` (+ `--from-diff`), `change list`, `change status` (без модели: задача + дельты + ADR зон из дельт), `change archive` (+ `--finish`);
- `adr accept`, `adr new --supersedes / --change`;
- workflow `propose`, `apply`, `document`, `archive`, а также `adr`, `why`, `zone`, `product`, `audit`.

Готово, когда: одна фича прошла через spec-first и одна через code-first, доки обновились через ворота.

### Этап 3 — модель решений (реализован)

- клиент Decisions API, промпты в `lore/prompts/`;
- `ctx`, `ctx --hook`, хук `UserPromptSubmit`, два этапа, корзины, бюджет, дедуп, таймаут, лог;
- модель в `change status` и `check-facts` (ворота и propose).

Готово, когда: хук подмешивает осмысленный контекст на живых промптах, ворота ловят противоречие с ADR.

### Реализация

- Go, один статический бинарник; nix flake для сборки (как в nexus и jev-bench);
- минимум зависимостей: CLI-роутер (cobra или stdlib `flag`), `gopkg.in/yaml.v3`, разбор frontmatter и заголовков markdown (своими силами или goldmark);
- встроенные шаблоны, workflow и промпты — через `embed`;
- базы и кэша нет: файлы разбираются при каждом вызове.

---

## 13. Не делаем пока

| Что | Почему отложено |
|---|---|
| `lore eval`, хук на Read, перебор порогов | сначала рабочий флоу; лог копится уже с этапа 3 |
| Тип `flow` | правила живут в зонах, маршрут — в `product.md` |
| `emits` и граф событий | структурное поле разойдётся с текстом |
| Маппинг зона → globs кода | код размазан, globs протухают |
| Связь сценариев с тестами (`// covers:`) | хорошая вторая итерация, ляжет как `zone › requirement` |
| Уровни риска, отдельный проверяющий | в старом kit именно такие этапы раздули процесс |
| Миграция с OpenSpec | не нужна сейчас |
| Отдельный `sync` дельт без архива | слияние — часть архива |
| `ctx:` / `verify:` / `depends:` в задачах | контекст собирает `status`, порядок задаёт нумерация |
| Сабагенты на задачу, оптимизация задержки | потом |
| Поддержка агентов, кроме Claude Code | идея, формат к ней готов |
| `lore status` (детектор отставания), `lore why` | потом |

---

## 14. Проверить на пилоте

- ~~Claude Code подхватывает skills через симлинк на директорию~~ — да, по документации.
- Decisions API: лимит вопросов в одном вызове, реальная задержка pplx-decider на 20 зонах и на 20–40 кандидатах этапа 2.
- ~~Кросс-язычное сопоставление~~ — работает на первых живых прогонах; подтвердить на nexus.
- Удобна ли длина id ADR в якорях кода.
- Не слишком ли много запросов `ask` при архиве.
- Хватает ли `INDEX.md` + `lore show`, когда модель недоступна.
- ~~Синтаксис путей в `permissions.ask`~~ — `Edit(/lore/zones/**)`, от корня проекта.

---

## 15. Журнал решений

Почему lore устроен именно так. Порядок — как принималось.

1. **Кто пишет доки.** Пишет агент, владелец апрувит proposal и ADR. *Почему:* руками владелец бросит через месяц; без апрува агент начнёт «узаконивать» свои ошибки через ADR.
2. **Bootstrap — полноценная часть**, а не «потом». *Почему:* lore нужен в первую очередь для существующих проектов.
3. **Зона — область ответственности со своими правилами**, не папка. *Почему:* такой критерий переживает рефакторинг и размазанный код.
4. **Сущность может быть зоной; действие живёт у объекта, реакция у реагирующего; тип `entity` убран.** *Почему:* иначе зона по актору (`users`) становится god-зоной.
5. **Убран тип `flow`, сценарии ушли в `product.md`** со статусами `live` / `planned`. *Почему:* flow без своих правил — это проекция, которая протухает; продуктовый слой отдельно от технического.
6. **Критерий ADR и когда их создавать.** Только на proposal или по ответу владельца на развилку. *Почему:* иначе апрувов слишком много (их перестанут читать), а агент задним числом оформляет свои решения как договорённости.
7. **Дельты — формат OpenSpec, сливает агент.** *Почему:* заголовок требования уже работает как id, формат знаком, слияние надёжно в OpenSpec.
8. **Code-first и spec-first сходятся в одних воротах.** *Почему:* два разных пути обновления доков со временем разойдутся в правилах проверки.
9. **Модель решений видит только текст**: код переводится в сводку фактами. *Почему:* так устроены модели решений.
10. **Факты проверяются и против ADR; противоречие решает владелец; частичной замены ADR нет.** *Почему:* новое решение и ошибка в коде снаружи выглядят одинаково; частичная замена заставляет собирать решение из кусков.
11. **Главная боль — «почему так, а не иначе», а не происхождение.** `origin` и `lore why` убраны из обязательного.
12. **`Why` обязателен, альтернативы опциональны.** *Почему:* решение без рассмотренных альтернатив тоже ценно; отвергнутая альтернатива и нерассмотренная по-разному ограничивают агента.
13. **Отдельная видимая папка, имя `lore`.** *Почему:* `docs/` занят, скрытую папку пропускают.
14. **Workflow — файлы в репо, к агенту подключаются симлинками.** *Почему:* в старом kit копии правил разошлись в 11 местах.
15. **Модель решений работает постоянно (хук), подмешивает содержимое, а не «прочитай X»; единица — требование.** *Почему:* дорого число шагов, а не чтение; целая зона на каждый промпт слишком дорога.
16. **Модель — `pplx-decider-v1.2` через OpenRouter, одна, без flash.** *Почему:* лучшая полнота по бенчу; flash теряет половину ADR.
17. **Два этапа отбора, таймаут, без оптимизации.** *Почему:* pplx медленный на большом числе кандидатов; сначала рабочий флоу.
18. **В задачах нет `ctx:`, контекст собирает `status`.** *Почему:* ручные ссылки протухают и дублируют модель; хук не срабатывает на «продолжай».
19. **Один шаблон зоны: минимум обязателен, остальное свободно.** *Почему:* дизайн-гайд не укладывается в WHEN/THEN.
20. **Генерируемого блока в зонах нет, есть `lore show`.** *Почему:* после появления хука блок дублирует доставку ADR и шумит в diff.
21. **`zones` в ADR вместо свободных тегов, плюс `global`.** *Почему:* смысловой поиск делает модель, свободные теги размножаются.
22. **Id ADR — дата + slug; параллельные change не блокируются.** *Почему:* номера конфликтуют между ветками; блокировки дороже редких пересечений.
23. **Читать доки можно свободно, править — со спросом.** *Почему:* жёсткий запрет (как в старом kit) порождает обходы и лишние шаги.
24. **Тесты (`covers:`) и уровни риска — потом.**
25. **Перед слиянием дельты сверяются с кодом.** *Почему:* в архив должен попадать факт, а не план.
26. **`emits` и граф событий не делаем.**
27. **Протокол в `AGENTS.md` — около 15 строк**, всё остальное в workflow.
28. **MVP: формат и bootstrap → changes → модель решений; eval не делаем; пилот — nexus.**
29. **Агентские команды на каждую повседневную операцию** (`adr`, `why`, `zone`, `product`, `audit`), а механику (статусы, замена ADR, архив) делает CLI. *Почему:* без команд агент делает это «как получится», а ручная правка статусов ломает цепочки замен.
30. **Статус ADR меняет только `lore adr accept`; черновики в change заменяют старые ADR только при архиве.** *Почему:* брошенный change не должен успеть отменить действующее решение.
31. **`--finish` сверяет слияние построчно.** *Почему:* для MODIFIED одного наличия заголовка мало — первая версия архивировала change без слияния.
32. **В `product.md` нет статуса `[dropped]`: изменённое переписывается, неактуальное удаляется, неочевидная причина отказа — ADR.** *Почему:* product описывает настоящее; история — в архиве changes.
33. **Ключ lore отдельно от env приложения** (`LORE_OPENROUTER_API_KEY`, `lore/.env`, `~/.config/lore/.env`), защита от ключа в `lore.yaml`. *Почему:* `.env` проекта принадлежит приложению; ключ, вписанный в конфиг, утекает в git.
34. **Одна папка `lore/`, локальное — в `lore/.env` и `lore/.local/`.** *Почему:* две папки (`lore/` и `.lore/`) ничего не давали, кроме путаницы.
35. **`lore change check` вместо `check-facts`; хук пропускает slash-команды.**
36. **`/lore-explore` — режим размышления поверх знаний lore.** *Почему:* идеи сталкиваются с решениями ещё до proposal; дешевле увидеть это в разговоре.
37. **`lore upgrade` дописывает новые ключи в `lore.yaml` текстом, не пересобирая файл.** *Почему:* конфиг правят руками; пересборка через YAML-энкодер ломает форматирование и выравнивание комментариев.
38. **`MODIFIED` несёт требование целиком, `--finish` требует точного совпадения блока** (вместо частичного `MODIFIED` и проверки «строки дельты есть в зоне»). *Почему:* при частичном `MODIFIED` проверка не замечала строки старого текста, оставшиеся в зоне после слияния, — зона тихо портилась; так же сейчас устроен OpenSpec. Цена — в дельту копируются и неизменённые сценарии.
39. **`anchors_ignore` в `lore.yaml`.** *Почему:* в репо самого lore примеры якорей в документации и тестах ломали `lore lint`.
