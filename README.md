Idea

I’m building a Telegram bot for CIS youth in the US (mostly young people) that works like a personal assistant: it matches people, learns from my text reactions (no buttons), remembers “how yesterday went,” and after a like/match it automatically creates a private chat for two, starts the dialogue, and deletes the room after 48 hours.

⸻

Main MVP Goals
	•	Fast onboarding and quick start (so users don’t drop off).
	•	Personalization for each user.
	•	Matches don’t turn into endless texting: a two-person chat lives exactly 48 hours and always closes.
	•	Reputation/rating system isn’t dumb, but motivates normal behavior (not “farming attention,” not ghosting, not liking everyone).
	•	Matching is fair: if someone likes someone, we don’t hide it as a “secret behind a donation” (at least at the core level).

⸻

Initial Geolocation Validation (Filtering Non-US Users)

It’s important that the bot is for people who are actually in the US, otherwise there will be tons of fakes/random users and it will turn into garbage.

How it works on first launch
	1.	The bot asks to share location (one-time) via Telegram’s standard “Share location” button.
	2.	The bot checks country = United States and activates the user.
	3.	If the user doesn’t want to share exact location:
	•	The bot offers an alternative: manually choose state/city,
	•	But the account gets “unverified” status, and visibility/likes are limited until US presence is confirmed.

Re-verification
	•	Periodically (for example every N days), the bot can gently ask to refresh verification if the user was inactive for a long time or suddenly changed “city.”

⸻

Format (Asynchronous)
	•	I enter the bot and start matching.
	•	The bot shows profiles one by one (as cards).
	•	I reply with text, saying what I think.
	•	The bot interprets: like/pass/maybe/report + reasons.
	•	Matching works in two ways (below).

⸻

Onboarding (Fast, No Life Story Questionnaire)

Goal: collect the minimum needed to personalize immediately and filter out junk.
	1.	City (and neighborhood/county)
	2.	Age range
	3.	Languages (RU/UA/EN)
	4.	Goal: friends / parties / dating / mixed
	5.	Interests (5–7 tags)
	6.	A few “definitely don’t want” (1–3 red flags)

Final message: “Okay, I’ve got it. Showing your matches.”

⸻

Profile Card (MVP Display)
	•	Photo (1–3)
	•	Name, age
	•	City/area
	•	Short bio (1–3 sentences)
	•	Interests (tags)

And a bot question: “What do you think?”

I respond with text.

⸻

Text Reactions (No Buttons)

I can write anything:
	•	“fine, let’s go”
	•	“not my type”
	•	“too serious, want more fun”
	•	“too flirty”
	•	“boring”
	•	“idk, not sure”

The bot must:
	1.	Understand the action: like/pass/maybe/report
	2.	Extract reasons (tags)
	3.	Confirm in one sentence what it understood (so I see personalization working)

Example confirmations:
	•	“Got it: pass, reason ‘too serious.’ Shifting results toward more light profiles.”
	•	“Okay: like. Noted that you’re into [humor/sports/etc.].”

⸻

Personalization (Simple but Feels Smart)

I’m not training a separate model per user. I’m making it feel smart.

What the bot stores per user
	•	Preference tag weights (what they like / what turns them off)
	•	Statistics of rejection reasons (“too serious,” “no common interests,” “too flirty”)
	•	Mood/session dynamics (for example, streak of rejections)
	•	Short “yesterday summary” (1–2 sentences)

How the bot uses it
	•	Ranks new profiles by interest match + learned preferences
	•	If I often say “boring/too serious,” it reduces those in the feed
	•	If I like a certain vibe, it increases similar profiles

⸻

“Tomorrow Memory” + Light Humor

The bot feels alive because it remembers yesterday.

Inside a session

If I reject many profiles in a row, the bot adds short playful comments:
	•	“You’re strict today. Got it. Adjusting.”
	•	“Okay, we’re in ‘not impressed’ mode. Let me try sharper.”

Next day (or after a pause)

Bot opens with context:
	•	“Yesterday didn’t go great. I removed some ‘too serious’ profiles and added more [humor/events/etc.].”
	•	“Based on yesterday, vibe matters more to you than a perfect profile. Let’s go.”

⸻

Matching Options (2 Scenarios)

Two paths so there are no artificial secrets and people reach contact faster.

1) Classic: Mutual Like
	•	I like someone.
	•	They like me.
	•	Bot registers a match and creates a 48-hour private chat.

2) Like via “Who Liked You” (Nothing Hidden)

If someone liked me, the bot doesn’t hide it. There’s a mode:
“Who liked you” — the bot shows profiles of people who liked me.

Then I decide:
	•	If I like them back → instant match → 48-hour chat.
	•	If I pass → rejection is logged (used for personalization).

Important: “Who liked you” shows real interest, not blind algorithmic guessing.

⸻

Match: Creating a Two-Person Chat + Starting Dialogue

How chat works
	•	After a match, the bot creates a private two-person chat (plus the bot).
	•	The bot starts the conversation to avoid “hi how are you.”
	•	Then users talk on their own.
	•	After 48 hours, the bot always closes and deletes the chat.

Host Script (Short, 2–4 Messages)
	1.	“Yo. I matched you. I’m just here to kick things off so you don’t drown in ‘how are you.’”
	2.	“Each of you: in one sentence, what’s your ideal evening in this city?”
	3.	“Now one by one: name one thing you genuinely like (food/music/hobby).”
	4.	“Okay, I see overlap in [X]. You’re on your own now. I’m silent.”

⸻

Rating & Badges (Smarter and More Interesting)

I want the rating system to encourage healthy behavior and filter people properly.

Two Levels: “Quality Signals” + “Behavioral Badges”
	•	Quality Signals (indirect indicators of attractiveness/adequacy)
	•	Behavioral Badges (how the person behaves inside the product)

Metrics Tracked
	1.	Like Received Rate: likes received per 100 impressions
	2.	Like Given Rate: likes given per 100 impressions
	3.	Mutual Match Rate: mutual matches from given likes
	4.	Chat Start Rate: % of matches where user actually enters/replies
	5.	Response Rate: whether they reply during the 48-hour chat
	6.	Selectivity Balance: balance between receiving and giving likes
	7.	Consistency: regularity of activity

⸻

Badges (Examples)

Attention Balance
	•	“Attention Hunter”
Gets many likes, rarely likes back + low chat start/response rate.
	•	“Selective but Honest”
Gives few likes but has high mutual match rate and decent response rate.
	•	“Like Machine”
Likes too many profiles in a row, low mutual match rate.

Communication
	•	“Ghost”
Gets matches but often stays silent or doesn’t start chats.
	•	“Solid Communicator”
Replies consistently, chats aren’t empty.
	•	“Starter”
Frequently initiates and replies first.

Profile Quality
	•	“Empty Profile”
Minimal info. Motivates filling it in (otherwise shown less).
	•	“Clear Profile”
Bio + interests filled, normal activity, good engagement.

⸻

Monetization (Later)

Monetization without aggressive “pay to message.” Logic is simple: free version has limits, donation (via Telegram Stars or similar) expands options.

Free Limits
	•	Profile views per day: limited (e.g., 25–40)
	•	Likes per day: limited (e.g., 10–20)
	•	“Who liked you”: available, but possibly limited daily views (e.g., 10)
	•	Re-show skipped profiles: rare or once per day

Paid Expansion (Stars / Donation)
	1.	+Limits
	•	+X profile views today
	•	+X likes today
	•	+X “who liked you” views today
	2.	Rewind / Second Chance
	•	Restore last N skipped profiles (“oops, rejected by accident”)
	3.	Profile Boost (Carefully)
	•	Short-term boost (e.g., 1–3 hours), profile shown more often
	4.	Personalization Pro
	•	Fine preference tuning (“more humor/sports,” “less flirting”)
	•	Short bot explanation of why certain profiles are shown

⸻

MVP Success Criteria
	•	User sees the bot “learning” (feed actually improves).
	•	Matches lead to real conversations.
	•	48-hour chats create tempo and prevent dead dialogues from piling up.
	•	Badges/rating improve quality, not just “cool/not cool.”
	•	“Who liked you” speeds up matches and keeps mechanics honest.

⸻

UX Notes
	•	Photos stay at the start (for conversion).
	•	Personalization through text reactions + preferences.
	•	Bot feels “alive” via memory and humor.
	•	Chat always closes after 48 hours.
	•	Priority entry for people actually in the US (geolocation validation).