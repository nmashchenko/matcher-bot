# Matching System

## Format

Asynchronous, card-based browsing inside the Telegram chat.

1. User enters the bot and starts matching.
2. Bot shows profiles one by one (as cards).
3. User replies with free text saying what they think.
4. Bot interprets the reaction: like / pass / maybe / report + reasons.

## Profile Card (MVP)

- Photo (1-3)
- Name, age
- City / area
- Short bio (1-3 sentences)
- Interests (tags)

Bot prompt after each card: "What do you think?"

## Text Reactions (No Buttons)

User can write anything:

- "fine, let's go"
- "not my type"
- "too serious, want more fun"
- "too flirty"
- "boring"
- "idk, not sure"

The bot must:

1. Understand the action: like / pass / maybe / report
2. Extract reasons (tags)
3. Confirm in one sentence what it understood

Example confirmations:

- "Got it: pass, reason 'too serious.' Shifting results toward more light profiles."
- "Okay: like. Noted that you're into [humor/sports/etc.]."

## Two Matching Paths

### 1. Classic: Mutual Like

- I like someone.
- They like me.
- Bot registers a match and creates a 48-hour private chat.

### 2. "Who Liked You" (Nothing Hidden)

If someone liked me, the bot doesn't hide it. There's a mode: "Who liked you" — the bot shows profiles of people who liked me.

Then I decide:

- If I like them back -> instant match -> 48-hour chat.
- If I pass -> rejection is logged (used for personalization).

Important: "Who liked you" shows real interest, not blind algorithmic guessing.
