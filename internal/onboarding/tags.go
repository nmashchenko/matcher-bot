package onboarding

import "matcher-bot/internal/database"

type GoalOption struct {
	Key   database.Goal
	Label string
}

var GoalOptions = []GoalOption{
	{Key: database.GoalFriends, Label: "Друзья"},
	{Key: database.GoalHangouts, Label: "Тусовки"},
	{Key: database.GoalDating, Label: "Свидания"},
	{Key: database.GoalMixed, Label: "Всё сразу"},
}

// ValidGoal returns true if g is a recognized goal value.
func ValidGoal(g database.Goal) bool {
	for _, opt := range GoalOptions {
		if opt.Key == g {
			return true
		}
	}
	return false
}
