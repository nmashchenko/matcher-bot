# Personalization

Simple but feels smart. No separate model per user — just smart tracking that makes the feed visibly improve.

## What the Bot Stores Per User

- Preference tag weights (what they like / what turns them off)
- Statistics of rejection reasons ("too serious", "no common interests", "too flirty")
- Mood / session dynamics (e.g. streak of rejections)
- Short "yesterday summary" (1-2 sentences)

## How the Bot Uses It

- Ranks new profiles by interest match + learned preferences
- If a user often says "boring / too serious", it reduces those in the feed
- If a user likes a certain vibe, it increases similar profiles

## Session Dynamics

### Inside a Session

If the user rejects many profiles in a row, the bot adds short playful comments:

- "You're strict today. Got it. Adjusting."
- "Okay, we're in 'not impressed' mode. Let me try sharper."

### Next Day (or After a Pause)

Bot opens with context:

- "Yesterday didn't go great. I removed some 'too serious' profiles and added more [humor/events/etc.]."
- "Based on yesterday, vibe matters more to you than a perfect profile. Let's go."
