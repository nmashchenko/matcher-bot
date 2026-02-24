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

// Onboarding
const (
	AskAge         = "Сколько тебе лет?"
	AskGoal        = "Что ищешь?"
	AskBio         = "Расскажи о себе в 2-3 предложениях:"
	AskLookingFor  = "Опиши, кого ищешь или что для тебя важно в общении:"
	InvalidAge     = "Введи возраст от 16 до 99."
	TooShort       = "Слишком коротко — напиши хотя бы 20 символов."
	OnboardingDone = "Окей, я запомнил. Скоро начнём подбор!"
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

func AgeFromTelegram(age int) string {
	return fmt.Sprintf("Мне удалось определить твой возраст из Telegram: %d. Что ищешь?", age)
}
