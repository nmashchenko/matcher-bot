package messages

import (
	"fmt"
	"strings"
	"time"
)

const (
	GenericError   = "\u274c Произошла ошибка. Попробуй ещё раз."
	RestartError   = "\u274c Произошла ошибка. Попробуй /start заново."
	GeocodingError = "\u274c Не удалось определить местоположение. Попробуй ещё раз."
	StartPrompt    = "Напиши /start чтобы начать."
	ShareLocationFirst    = "Сначала поделись геолокацией, чтобы я мог тебя верифицировать.\n\n📍 Точные координаты не сохраняются — только город и штат."
	FinishOnboardingFirst = "Сначала заверши регистрацию — напиши /start."
	UnknownCommand        = "Не понял тебя. Напиши /events, /create, /myevents или /settings."
)

const (
	VerificationIntro = "Привет! Я — бот для событий. Помогу найти интересные мероприятия рядом с тобой в США.\n\n" +
		"Для начала мне нужно убедиться, что ты в США. Поделись геолокацией — это одноразово и безопасно.\n\n" +
		"📍 Точные координаты не сохраняются — мы определяем только город и штат для подбора событий рядом.\n\n" +
		"Мы же не хотим чтобы это было как в Давинчике, где 80% обманывают 😉"
	VerificationButton = "\U0001f4cd Поделиться геолокацией"
	CheckingLocation   = "\u23f3 Проверяю твою геолокацию..."
	NotInUSA           = "\u274c Похоже, ты не в США. Этот бот пока работает только для людей в Штатах.\n\n" +
		"Если ты считаешь, что это ошибка — попробуй отправить геолокацию ещё раз."
)

const WelcomeSticker = "CAACAgIAAxkBAAIBKWmdA6Gwvt5-6TpbRIyrKAdxPl0vAAL5HgACv2nBSsMzoEX8LyRUOgQ"

const InvalidAge = "Введи возраст от 16 до 99."

func Verified(city, state string) string {
	return fmt.Sprintf("\u2705 Подтверждено! Ты в %s, %s.", city, state)
}

func AskAge(city, state string) string {
	return fmt.Sprintf("Круто, ты в %s, %s! Сколько тебе лет?", city, state)
}

func AgeFromTelegram(age int, city, state string) string {
	return fmt.Sprintf("Ты в %s, %s — и тебе %d. Добро пожаловать!", city, state, age)
}

func OnboardingComplete(name string, age int, city, state string) string {
	return fmt.Sprintf(
		"\u2728 Готово, %s! Тебе %d, ты в %s, %s.\n\n"+
			"Вот что можно делать:\n"+
			"/events — смотреть события рядом\n"+
			"/create — создать своё событие\n"+
			"/myevents — мои события\n"+
			"/settings — настройки",
		name, age, city, state,
	)
}

func MainMenu(city, state string) string {
	return fmt.Sprintf(
		"С возвращением! Ты в %s, %s.\n\n"+
			"/events — смотреть события рядом\n"+
			"/create — создать своё событие\n"+
			"/myevents — мои события\n"+
			"/settings — настройки",
		city, state,
	)
}

const (
	CreateStart       = "Создаём событие!"
	CreatePickType    = "Выбери тип:"
	CreateAskTitle    = "Придумай название для события:"
	CreateAskDesc     = "Добавь описание (или отправь \"-\" чтобы пропустить):"
	CreateAskTime     = "Когда начало? Формат: ДД.ММ ЧЧ:ММ\nНапример: 15.03 20:00"
	CreateAskLocation = "\U0001f4cd Отправь геолокацию места проведения через \U0001f4ce (скрепку) → Геопозиция."
	CreateAskCapacity = "Сколько участников (не считая тебя)? Максимум — 50."
	CreateCancelled   = "Создание события отменено."
	CreateInProgress  = "Ты сейчас создаёшь событие. Заверши или отмени создание."
	CreateCancelBtn    = "❌ Отменить создание"
	CreateGamingNotice = "🎮 Игровые события видны всем пользователям по всей стране — геолокация не нужна."
	CreateAskAge       = "Укажи возраст участников (например: 18-30) или отправь \"-\" чтобы пропустить:"
	InvalidAgeRange    = "Неверный формат. Используй: 18-30, или отправь \"-\" чтобы пропустить."
	InvalidTime       = "Неверный формат. Используй: ДД.ММ ЧЧ:ММ (например: 15.03 20:00)"
	TimePast          = "Дата уже прошла. Укажи будущую дату."
)

func CreateSuccess(title, city string, startsAt time.Time, ageRestriction string) string {
	ageLine := ""
	if ageRestriction != "" {
		ageLine = fmt.Sprintf("\n\U0001f464 %s лет", ageRestriction)
	}
	return fmt.Sprintf(
		"\u2705 Событие \"%s\" создано!\n"+
			"\U0001f4cd %s\n"+
			"\U0001f4c5 %s%s\n\n"+
			"Жди заявок — я уведомлю тебя о каждой.",
		title, city, startsAt.Format("02.01 15:04"), ageLine,
	)
}

func CreateConfirm(emoji, typeLabel, title, desc, city string, startsAt time.Time, capacity int, ageRestriction string) string {
	descLine := ""
	if desc != "" {
		descLine = fmt.Sprintf("\n\U0001f4ac %s", desc)
	}
	ageLine := ""
	if ageRestriction != "" {
		ageLine = fmt.Sprintf("\n\U0001f464 %s лет", ageRestriction)
	}
	return fmt.Sprintf(
		"Всё верно?\n\n"+
			"%s %s\n"+
			"\U0001f3af %s%s\n"+
			"\U0001f4cd %s\n"+
			"\U0001f4c5 %s\n"+
			"\U0001f465 до %d чел.%s",
		emoji, typeLabel, title, descLine, city,
		startsAt.Format("02.01 15:04"), capacity, ageLine,
	)
}

const (
	BrowseEmpty   = "Пока нет событий рядом с тобой. Создай первое — /create"
	BrowseEnd     = "Это все события на данный момент."
	BrowseExpired = "Напиши /events заново."
	AlreadyJoined = "Ты уже подал заявку на это событие."
	JoinSent      = "\u2705 Заявка отправлена! Организатор получит уведомление."
	EventFull     = "К сожалению, все места заняты."
)

func BrowseFilterNotice(emoji, typeLabel, city string, isGaming bool) string {
	if isGaming {
		return fmt.Sprintf("🔍 Ищу: %s %s — по всем городам", emoji, typeLabel)
	}
	return fmt.Sprintf("🔍 Ищу: %s %s — в %s", emoji, typeLabel, city)
}

func BrowseFilterAll(city string) string {
	return fmt.Sprintf("🔍 Ищу: все типы — в %s", city)
}

const (
	SessionExpired    = "Сессия истекла. Начни /create заново."
	UnknownEventType  = "Неизвестный тип."
	InvalidNumber     = "Неверное число."
	NotHost           = "Ты не организатор."
	Approved          = "\u2705 Принят!"
	Rejected          = "\u274c Отклонён."
	EventCancelledCb  = "\u2705 Событие отменено."
	ParticipantRemovedCb = "\u2705 Участник удалён."
	EventNoLongerActive  = "Это событие уже неактивно."
	AlreadyRemoved       = "Участник уже убран."
)

func EventCard(emoji, typeLabel, title, desc, city string, startsAt time.Time, approved, max int, ageRestriction string) string {
	descLine := ""
	if desc != "" {
		descLine = fmt.Sprintf("\n\U0001f4ac %s", desc)
	}
	ageLine := ""
	if ageRestriction != "" {
		ageLine = fmt.Sprintf("\n\U0001f464 %s лет", ageRestriction)
	}
	return fmt.Sprintf(
		"%s %s\n"+
			"\U0001f3af %s%s\n"+
			"\U0001f4cd %s\n"+
			"\U0001f4c5 %s\n"+
			"\U0001f465 %d/%d%s",
		emoji, typeLabel, title, descLine, city,
		startsAt.Format("02.01 15:04"), approved, max, ageLine,
	)
}

func JoinRequest(eventTitle string, telegramID int64, name, username string, age int, city, state string) string {
	usernameStr := "(юзернейм не указан)"
	if username != "" {
		usernameStr = fmt.Sprintf("(@%s)", username)
	}
	return fmt.Sprintf(
		"\U0001f514 Новая заявка на \"%s\"!\n\n"+
			"\U0001f464 <a href=\"tg://user?id=%d\">%s</a> %s, %d\n"+
			"\U0001f4cd %s, %s",
		eventTitle, telegramID, name, usernameStr, age, city, state,
	)
}

func ParticipantApproved(eventTitle, hostName, hostUsername, city string, participantNames []string) string {
	contactLine := hostName
	if hostUsername != "" {
		contactLine = fmt.Sprintf("%s (@%s)", hostName, hostUsername)
	}
	participantsLine := ""
	if len(participantNames) > 0 {
		participantsLine = fmt.Sprintf("\n\U0001f465 Участники: %s", strings.Join(participantNames, ", "))
	}
	return fmt.Sprintf(
		"\u2705 Тебя приняли на \"%s\"!\n\n"+
			"\U0001f464 Организатор: %s\n"+
			"\U0001f4cd %s%s",
		eventTitle, contactLine, city, participantsLine,
	)
}

func ParticipantRejected(eventTitle string) string {
	return fmt.Sprintf("\u274c К сожалению, организатор отклонил заявку на \"%s\".", eventTitle)
}

func EventCancelled(eventTitle string) string {
	return fmt.Sprintf("\u274c Событие \"%s\" было отменено организатором.", eventTitle)
}

func ParticipantRemoved(eventTitle string) string {
	return fmt.Sprintf("\u274c Организатор убрал тебя из события \"%s\".", eventTitle)
}

const ParticipantLeft = "Ты покинул событие."

func HostNotifyLeft(eventTitle, name string) string {
	return fmt.Sprintf("👤 %s покинул событие \"%s\".", name, eventTitle)
}

const EventAlreadyStarted = "Событие уже началось, покинуть нельзя."
const NotParticipant = "Ты не участник этого события."

func HostApprovedConfirm(name, username, eventTitle string) string {
	if username != "" {
		return fmt.Sprintf("✅ %s (@%s) принят на \"%s\".", name, username, eventTitle)
	}
	return fmt.Sprintf("✅ %s принят на \"%s\".", name, eventTitle)
}

func HostRemovedConfirm(name, eventTitle string) string {
	return fmt.Sprintf("❌ %s убран из \"%s\".", name, eventTitle)
}

func EventCancelledConfirm(eventTitle string) string {
	return fmt.Sprintf("🚫 Событие \"%s\" отменено. Участники уведомлены.", eventTitle)
}

const UsernameWarning = "⚠️ У тебя не установлен юзернейм в Telegram. Организаторы событий не смогут связаться с тобой напрямую. Рекомендуем установить его в настройках Telegram."

const SettingsTitle = "⚙️ Настройки"
const SettingsFilterAll = "Все типы"
const SettingsFilterUpdated = "Фильтр обновлён."
const SettingsFilterBtn = "🎯 Фильтр событий"

func SettingsView(currentFilter string) string {
	return fmt.Sprintf("%s\n\n🎯 Фильтр событий: %s", SettingsTitle, currentFilter)
}

const MyEventsEmpty = "У тебя пока нет событий.\n\n/create — создать событие\n/events — найти событие"

func MyEventRow(emoji, title string, startsAt time.Time, approved, max, pending int) string {
	pendingStr := ""
	if pending > 0 {
		pendingStr = fmt.Sprintf(" (+%d \U0001f514)", pending)
	}
	return fmt.Sprintf(
		"%s %s — %s [%d/%d]%s",
		emoji, title, startsAt.Format("02.01 15:04"), approved, max, pendingStr,
	)
}
