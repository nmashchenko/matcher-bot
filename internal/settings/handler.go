package settings

import (
	"context"
	"fmt"
	"log/slog"

	"matcher-bot/internal/database"
	"matcher-bot/internal/events"
	"matcher-bot/internal/messages"

	tele "gopkg.in/telebot.v4"
)

type Handler struct {
	users database.UserRepository
}

func NewHandler(users database.UserRepository) *Handler {
	return &Handler{users: users}
}

func (h *Handler) Register(b *tele.Bot) {
	b.Handle("\fsf", h.onFilterSelect)
	b.Handle("\fsbk", h.onBackToSettings)
}

// CmdSettings shows the settings view with a reply keyboard widget button.
func (h *Handler) CmdSettings(c tele.Context) error {
	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return c.Send(messages.StartPrompt)
	}

	return h.sendSettingsView(c, user)
}

// OnText handles the reply keyboard button tap. Returns true if handled.
func (h *Handler) OnText(c tele.Context) bool {
	if c.Text() != messages.SettingsFilterBtn {
		return false
	}

	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		_ = c.Send(messages.StartPrompt)
		return true
	}

	_ = h.sendFilterPicker(c, user)
	return true
}

// sendSettingsView sends the settings text with a reply keyboard widget button.
func (h *Handler) sendSettingsView(c tele.Context, user *database.User) error {
	text := h.settingsText(user)

	markup := &tele.ReplyMarkup{ResizeKeyboard: true}
	markup.Reply(
		markup.Row(markup.Text(messages.SettingsFilterBtn)),
	)

	return c.Send(text, markup)
}

// sendFilterPicker sends the inline filter options.
func (h *Handler) sendFilterPicker(c tele.Context, user *database.User) error {
	currentFilter := ""
	if user.PreferredEventType != nil {
		currentFilter = *user.PreferredEventType
	}

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	allLabel := fmt.Sprintf("📋 %s", messages.SettingsFilterAll)
	if currentFilter == "" {
		allLabel = fmt.Sprintf("✅ %s", messages.SettingsFilterAll)
	}
	rows = append(rows, markup.Row(markup.Data(allLabel, "sf", "all")))

	for _, opt := range events.EventTypeOptions {
		label := fmt.Sprintf("%s %s", opt.Emoji, opt.Label)
		if string(opt.Key) == currentFilter {
			label = fmt.Sprintf("✅ %s", opt.Label)
		}
		rows = append(rows, markup.Row(markup.Data(label, "sf", string(opt.Key))))
	}

	btnBack := markup.Data("◀️ Назад", "sbk")
	rows = append(rows, markup.Row(btnBack))

	markup.Inline(rows...)
	return c.Send("🎯 Выбери тип событий:", markup)
}

// onFilterSelect handles inline filter selection callback.
func (h *Handler) onFilterSelect(c tele.Context) error {
	selected := c.Callback().Data

	var update database.UserUpdateData
	if selected == "all" {
		empty := ""
		update.PreferredEventType = &empty
	} else {
		update.PreferredEventType = &selected
	}

	if err := h.users.Update(context.Background(), c.Sender().ID, &update); err != nil {
		slog.Error("settings: update filter", "error", err)
		return c.Respond(&tele.CallbackResponse{Text: messages.GenericError})
	}

	_ = c.Respond(&tele.CallbackResponse{Text: messages.SettingsFilterUpdated})

	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return nil
	}

	// Update the inline message to show the new picker state.
	currentFilter := ""
	if user.PreferredEventType != nil {
		currentFilter = *user.PreferredEventType
	}

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	allLabel := fmt.Sprintf("📋 %s", messages.SettingsFilterAll)
	if currentFilter == "" {
		allLabel = fmt.Sprintf("✅ %s", messages.SettingsFilterAll)
	}
	rows = append(rows, markup.Row(markup.Data(allLabel, "sf", "all")))

	for _, opt := range events.EventTypeOptions {
		label := fmt.Sprintf("%s %s", opt.Emoji, opt.Label)
		if string(opt.Key) == currentFilter {
			label = fmt.Sprintf("✅ %s", opt.Label)
		}
		rows = append(rows, markup.Row(markup.Data(label, "sf", string(opt.Key))))
	}

	btnBack := markup.Data("◀️ Назад", "sbk")
	rows = append(rows, markup.Row(btnBack))

	markup.Inline(rows...)
	return c.Edit("🎯 Выбери тип событий:", markup)
}

// onBackToSettings handles the back button from the filter picker.
func (h *Handler) onBackToSettings(c tele.Context) error {
	_ = c.Respond()
	_ = c.Delete()

	user, err := h.users.GetByTelegramID(context.Background(), c.Sender().ID)
	if err != nil {
		return nil
	}

	return h.sendSettingsView(c, user)
}

// settingsText builds the settings message text.
func (h *Handler) settingsText(user *database.User) string {
	filterLabel := messages.SettingsFilterAll
	if user.PreferredEventType != nil {
		et := database.EventType(*user.PreferredEventType)
		emoji := events.EventTypeEmoji(et)
		label := events.EventTypeLabel(et)
		filterLabel = fmt.Sprintf("%s %s", emoji, label)
	}

	return messages.SettingsView(filterLabel)
}
