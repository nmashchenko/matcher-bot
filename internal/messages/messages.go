package messages

import "fmt"

// Errors
const (
	GenericError   = "\u274C Произошла ошибка. Попробуй ещё раз."
	RestartError   = "\u274C Произошла ошибка. Попробуй /start заново."
	GeocodingError = "\u274C Не удалось определить местоположение. Попробуй ещё раз."
	StartPrompt    = "Напиши /start чтобы начать."
	UnknownCommand = "Не понял тебя. Выбери, что хочешь сделать, или напиши /start."
)

// Verification
const (
	VerificationIntro = "Привет! Я — Matcher Bot. Помогу найти интересных людей из СНГ рядом с тобой в США.\n\n" +
		"Для начала мне нужно убедиться, что ты в США. Поделись геолокацией — это одноразово и безопасно."
	VerificationButton = "\U0001F4CD Поделиться геолокацией"
	CheckingLocation   = "\u23F3 Проверяю твою геолокацию..."
	NotInUSA           = "\u274C Похоже, ты не в США. Этот бот пока работает только для людей в Штатах.\n\n" +
		"Если ты считаешь, что это ошибка — попробуй отправить геолокацию ещё раз."
)

// Sticker
const WelcomeSticker = "CAACAgIAAxkBAAIBKWmdA6Gwvt5-6TpbRIyrKAdxPl0vAAL5HgACv2nBSsMzoEX8LyRUOgQ" // send any sticker to bot

// Onboarding
const (
	InvalidAge     = "Введи возраст от 16 до 99."
	TooShort       = "Слишком коротко — напиши хотя бы 20 символов."
	OnboardingDone = "" // unused, kept for reference
	StepAlready    = "Этот шаг уже пройден."
	UnknownGoal    = "Неизвестный вариант."
	CallbackError  = "Ошибка, попробуй /start."
)

// Formatted messages

func Verified(city, state string) string {
	return fmt.Sprintf("\u2705 Подтверждено! Ты в %s, %s.", city, state)
}

func WelcomeBack(city, state string) string {
	return fmt.Sprintf("С возвращением! Ты уже зарегистрирован (%s, %s). Скоро здесь будет подбор.", city, state)
}

func AgeFromTelegram(age int, city, state string) string {
	return fmt.Sprintf("Ты в %s, %s — и тебе %d. Что ищешь?", city, state, age)
}

func AskAge(city, state string) string {
	return fmt.Sprintf("Круто, ты в %s, %s! Сколько тебе лет?", city, state)
}

func AskGoal(age int, city string) string {
	return fmt.Sprintf("Отлично, %d лет в %s — расскажи, что ищешь?", age, city)
}

func AskBio(goalLabel string) string {
	return fmt.Sprintf("Цель: %s — отлично! Расскажи о себе в 2-3 предложениях:", goalLabel)
}

func AskLookingFor(name string) string {
	return fmt.Sprintf("Спасибо, %s! Теперь опиши, кого ищешь или что для тебя важно в общении:", name)
}

func ProfileComplete(name string, age int, goal, bio, lookingFor, city, state string) string {
	return fmt.Sprintf(
		"\u2728 Профиль готов! Уже ищу идеальные совпадения для тебя...\n\n"+
			"\U0001F464 %s, %d\n"+
			"\U0001F4CD %s, %s\n"+
			"\U0001F3AF %s\n\n"+
			"\U0001F4AC О себе:\n%s\n\n"+
			"\U0001F50D Ищу:\n%s\n\n"+
			"\U0001F525 Мы уже нашли несколько отличных совпадений!\n\n"+
			"Жми /match чтобы посмотреть, кого мы подобрали \U0001F447",
		name, age, city, state, goal, bio, lookingFor,
	)
}
