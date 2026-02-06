# Matcher Bot MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Telegram bot for CIS folks in the US that shows profile cards, interprets free-text reactions via LLM, learns user preferences, creates bot-mediated 48-hour match chats, and tracks behavior badges.

**Architecture:** NestJS monolith with Telegraf (via `nestjs-telegraf`) for Telegram integration, Prisma ORM with PostgreSQL for persistence, OpenAI/Anthropic API for natural language reaction parsing, and `@nestjs/schedule` for cron-based chat expiry. Match chats are bot-mediated private conversations (not Telegram groups) because the Bot API cannot create group chats programmatically.

**Tech Stack:** NestJS 10+, TypeScript 5+, Telegraf 4 + nestjs-telegraf, Prisma 6+, PostgreSQL 16, @nestjs/schedule, @nestjs/config, OpenAI SDK (for text reaction parsing), node-geocoder (for reverse geocoding)

---

## Critical Architecture Decision: Match Chat Design

Telegram Bot API **cannot** create group chats. Two options exist:

**Option A (Recommended): Bot-Mediated Private Chat**
- After match, the bot sends messages to both users in their private bot chat.
- The bot relays messages between the two users (User A writes to bot -> bot forwards to User B, and vice versa).
- After 48h, the bot stops relaying and notifies both users.
- Pros: No extra infrastructure, no userbot, fully within Bot API.
- Cons: Users chat "through" the bot, not directly.

**Option B: MTProto Userbot (GramJS)**
- A separate user account creates a group, adds both users + bot.
- Requires a real phone number, risks ToS violations.
- Not recommended for MVP.

**Decision: Option A.** The bot acts as a relay. This is clean, simple, and within Telegram's rules. The UX includes the bot's personality (icebreakers, closing messages), which makes relay feel natural.

---

## Task 1: Project Scaffolding

**Files:**
- Create: entire NestJS project via CLI
- Create: `.env`, `.env.example`
- Create: `docker-compose.yml` (PostgreSQL)
- Modify: `package.json` (add dependencies)
- Modify: `tsconfig.json` (BigInt support)

**Step 1: Scaffold NestJS project**

```bash
cd /Users/nmashchenko/Documents/matcher-bot
npx @nestjs/cli new . --package-manager npm --skip-git
```

**Step 2: Install core dependencies**

```bash
npm install nestjs-telegraf telegraf @nestjs/config @nestjs/schedule @prisma/client node-geocoder
npm install -D prisma @types/node-geocoder
```

**Step 3: Create `.env.example`**

```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
DATABASE_URL=postgresql://matcher:matcher@localhost:5432/matcher_bot?schema=public
OPENAI_API_KEY=your_openai_key_here
NODE_ENV=development
```

**Step 4: Create `.env` from example**

```bash
cp .env.example .env
# Fill in actual values
```

**Step 5: Create `docker-compose.yml`**

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: matcher
      POSTGRES_PASSWORD: matcher
      POSTGRES_DB: matcher_bot
    ports:
      - '5432:5432'
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

**Step 6: Add BigInt JSON serialization to `src/main.ts`**

Add before `NestFactory.create`:
```typescript
(BigInt.prototype as any).toJSON = function () {
  return this.toString();
};
```

**Step 7: Start PostgreSQL and verify**

```bash
docker-compose up -d
```

**Step 8: Initialize Prisma**

```bash
npx prisma init
```

**Step 9: Commit**

```bash
git init
git add .
git commit -m "chore: scaffold NestJS project with Telegraf, Prisma, and PostgreSQL"
```

---

## Task 2: Prisma Schema & Database Setup

**Files:**
- Create: `prisma/schema.prisma`
- Create: `src/prisma/prisma.module.ts`
- Create: `src/prisma/prisma.service.ts`

**Step 1: Write the Prisma schema**

```prisma
generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

enum Gender {
  MALE
  FEMALE
  OTHER
}

enum Goal {
  FRIENDS
  HANGOUTS
  DATING
  MIXED
}

enum VerificationStatus {
  VERIFIED
  UNVERIFIED
  PENDING
}

model User {
  telegramId         BigInt             @id
  username           String?
  firstName          String
  lastName           String?
  bio                String?
  age                Int?
  gender             Gender?
  goal               Goal?
  city               String?
  state              String?
  neighborhood       String?
  latitude           Float?
  longitude          Float?
  verificationStatus VerificationStatus @default(PENDING)
  languages          String[]           @default([])
  redFlags           String[]           @default([])
  photos             String[]           @default([])
  isActive           Boolean            @default(true)
  lastActiveAt       DateTime           @default(now())
  createdAt          DateTime           @default(now())
  updatedAt          DateTime           @updatedAt

  // Preferences memory
  preferencesSummary String?            // LLM-generated summary of what user likes
  rejectionPatterns  Json?              // { "too serious": 5, "boring": 3, ... }
  sessionMood        String?            // current session mood
  yesterdaySummary    String?            // "Yesterday didn't click. Reduced serious profiles."

  // Relations
  userTags       UserTag[]
  likesGiven     Like[]      @relation("LikesGiven")
  likesReceived  Like[]      @relation("LikesReceived")
  matchesAsUser1 Match[]     @relation("MatchesAsUser1")
  matchesAsUser2 Match[]     @relation("MatchesAsUser2")
  messages       Message[]
  impressions    Impression[] @relation("ImpressionViewer")
  impressionsOf  Impression[] @relation("ImpressionTarget")

  // Behavioral metrics (denormalized for fast access)
  totalImpressions   Int @default(0)
  totalLikesReceived Int @default(0)
  totalLikesGiven    Int @default(0)
  totalMatches       Int @default(0)
  totalChatsStarted  Int @default(0)
  totalResponses     Int @default(0)

  // State machine
  currentState   String @default("NEW") // NEW, ONBOARDING, ACTIVE, BROWSING, IN_CHAT
  currentCardId  BigInt?                // telegramId of profile currently being shown
  activeChatMatchId String?             // matchId of active relay chat

  @@index([city, state])
  @@index([isActive, verificationStatus])
}

model Tag {
  id       String    @id @default(cuid())
  name     String    @unique
  category String?
  userTags UserTag[]
}

model UserTag {
  userId BigInt
  tagId  String
  weight Float  @default(1.0)

  user User @relation(fields: [userId], references: [telegramId], onDelete: Cascade)
  tag  Tag  @relation(fields: [tagId], references: [id], onDelete: Cascade)

  @@id([userId, tagId])
  @@index([tagId])
}

model Like {
  id        String   @id @default(cuid())
  fromId    BigInt
  toId      BigInt
  reason    String?  // LLM-extracted reason
  createdAt DateTime @default(now())

  from User @relation("LikesGiven", fields: [fromId], references: [telegramId], onDelete: Cascade)
  to   User @relation("LikesReceived", fields: [toId], references: [telegramId], onDelete: Cascade)

  @@unique([fromId, toId])
  @@index([toId])
}

model Pass {
  id        String   @id @default(cuid())
  fromId    BigInt
  toId      BigInt
  reason    String?
  createdAt DateTime @default(now())

  @@unique([fromId, toId])
  @@index([toId])
}

model Match {
  id        String    @id @default(cuid())
  user1Id   BigInt
  user2Id   BigInt
  createdAt DateTime  @default(now())
  isActive  Boolean   @default(true)

  user1    User      @relation("MatchesAsUser1", fields: [user1Id], references: [telegramId], onDelete: Cascade)
  user2    User      @relation("MatchesAsUser2", fields: [user2Id], references: [telegramId], onDelete: Cascade)
  chatRoom ChatRoom?

  @@unique([user1Id, user2Id])
  @@index([user2Id])
}

model ChatRoom {
  id              String    @id @default(cuid())
  matchId         String    @unique
  telegramGroupId BigInt?   // If we ever use real groups
  createdAt       DateTime  @default(now())
  expiresAt       DateTime
  closedAt        DateTime?
  icebreakersComplete Boolean @default(false)

  match    Match     @relation(fields: [matchId], references: [id], onDelete: Cascade)
  messages Message[]

  @@index([expiresAt, closedAt])
}

model Message {
  id         String   @id @default(cuid())
  chatRoomId String
  senderId   BigInt
  text       String
  createdAt  DateTime @default(now())

  chatRoom ChatRoom @relation(fields: [chatRoomId], references: [id], onDelete: Cascade)
  sender   User     @relation(fields: [senderId], references: [telegramId], onDelete: Cascade)

  @@index([chatRoomId, createdAt])
}

model Impression {
  id        String   @id @default(cuid())
  viewerId  BigInt
  targetId  BigInt
  createdAt DateTime @default(now())

  viewer User @relation("ImpressionViewer", fields: [viewerId], references: [telegramId], onDelete: Cascade)
  target User @relation("ImpressionTarget", fields: [targetId], references: [telegramId], onDelete: Cascade)

  @@unique([viewerId, targetId])
  @@index([targetId])
}
```

**Step 2: Create PrismaService**

```typescript
// src/prisma/prisma.service.ts
import { Injectable, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';

@Injectable()
export class PrismaService extends PrismaClient implements OnModuleInit, OnModuleDestroy {
  async onModuleInit() {
    await this.$connect();
  }

  async onModuleDestroy() {
    await this.$disconnect();
  }
}
```

**Step 3: Create PrismaModule**

```typescript
// src/prisma/prisma.module.ts
import { Global, Module } from '@nestjs/common';
import { PrismaService } from './prisma.service';

@Global()
@Module({
  providers: [PrismaService],
  exports: [PrismaService],
})
export class PrismaModule {}
```

**Step 4: Run migration**

```bash
npx prisma migrate dev --name init
```

**Step 5: Verify with Prisma Studio**

```bash
npx prisma studio
```

**Step 6: Seed default tags**

Create `prisma/seed.ts`:

```typescript
import { PrismaClient } from '@prisma/client';

const prisma = new PrismaClient();

const tags = [
  // Hobbies
  { name: 'hiking', category: 'hobby' },
  { name: 'cooking', category: 'hobby' },
  { name: 'gaming', category: 'hobby' },
  { name: 'reading', category: 'hobby' },
  { name: 'fitness', category: 'hobby' },
  { name: 'photography', category: 'hobby' },
  { name: 'travel', category: 'hobby' },
  { name: 'music', category: 'hobby' },
  { name: 'art', category: 'hobby' },
  { name: 'sports', category: 'hobby' },
  { name: 'movies', category: 'hobby' },
  { name: 'dancing', category: 'hobby' },
  { name: 'yoga', category: 'hobby' },
  { name: 'coding', category: 'hobby' },
  // Social
  { name: 'bars-clubs', category: 'social' },
  { name: 'coffee-meetups', category: 'social' },
  { name: 'outdoor-adventures', category: 'social' },
  { name: 'board-games', category: 'social' },
  { name: 'concerts', category: 'social' },
  { name: 'food-scene', category: 'social' },
  // Vibes
  { name: 'humor', category: 'vibe' },
  { name: 'chill', category: 'vibe' },
  { name: 'ambitious', category: 'vibe' },
  { name: 'nerdy', category: 'vibe' },
  { name: 'creative', category: 'vibe' },
  { name: 'extrovert', category: 'vibe' },
  { name: 'introvert', category: 'vibe' },
];

async function main() {
  for (const tag of tags) {
    await prisma.tag.upsert({
      where: { name: tag.name },
      update: {},
      create: tag,
    });
  }
  console.log(`Seeded ${tags.length} tags`);
}

main()
  .catch(console.error)
  .finally(() => prisma.$disconnect());
```

Add to `package.json`:
```json
"prisma": {
  "seed": "ts-node prisma/seed.ts"
}
```

Run:
```bash
npx prisma db seed
```

**Step 7: Commit**

```bash
git add .
git commit -m "feat: add Prisma schema with users, tags, likes, matches, chat rooms"
```

---

## Task 3: Bot Module & Telegraf Integration

**Files:**
- Create: `src/bot/bot.module.ts`
- Create: `src/bot/bot.update.ts`
- Modify: `src/app.module.ts`

**Step 1: Create BotUpdate handler**

```typescript
// src/bot/bot.update.ts
import { Update, Start, Ctx, On, Hears } from 'nestjs-telegraf';
import { Context } from 'telegraf';
import { Injectable } from '@nestjs/common';

@Update()
@Injectable()
export class BotUpdate {
  @Start()
  async onStart(@Ctx() ctx: Context) {
    await ctx.reply(
      'Hey! I\'m Matcher Bot - your wingman for meeting CIS folks in the US.\n\n' +
      'Let\'s get you set up real quick. First, I need to verify you\'re in the US.\n\n' +
      'Tap the button below to share your location (one-time only):',
      {
        reply_markup: {
          keyboard: [
            [{ text: '📍 Share my location', request_location: true }],
            [{ text: '🏙 Enter city manually' }],
          ],
          resize_keyboard: true,
          one_time_keyboard: true,
        },
      },
    );
  }
}
```

**Step 2: Create BotModule**

```typescript
// src/bot/bot.module.ts
import { Module } from '@nestjs/common';
import { BotUpdate } from './bot.update';

@Module({
  providers: [BotUpdate],
})
export class BotModule {}
```

**Step 3: Wire up AppModule**

```typescript
// src/app.module.ts
import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TelegrafModule } from 'nestjs-telegraf';
import { ScheduleModule } from '@nestjs/schedule';
import { PrismaModule } from './prisma/prisma.module';
import { BotModule } from './bot/bot.module';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    ScheduleModule.forRoot(),
    TelegrafModule.forRootAsync({
      imports: [ConfigModule],
      useFactory: (configService: ConfigService) => ({
        token: configService.getOrThrow<string>('TELEGRAM_BOT_TOKEN'),
      }),
      inject: [ConfigService],
    }),
    PrismaModule,
    BotModule,
  ],
})
export class AppModule {}
```

**Step 4: Run the bot and test /start**

```bash
npm run start:dev
```

Open Telegram, send `/start` to your bot. Verify the location keyboard appears.

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add Telegraf bot module with /start command and location prompt"
```

---

## Task 4: Geolocation Verification

**Files:**
- Create: `src/geo/geo.module.ts`
- Create: `src/geo/geo.service.ts`
- Create: `src/geo/geo.service.spec.ts`
- Modify: `src/bot/bot.update.ts`

**Step 1: Write the failing test for GeoService**

```typescript
// src/geo/geo.service.spec.ts
import { Test } from '@nestjs/testing';
import { GeoService } from './geo.service';

describe('GeoService', () => {
  let service: GeoService;

  beforeEach(async () => {
    const module = await Test.createTestingModule({
      providers: [GeoService],
    }).compile();
    service = module.get(GeoService);
  });

  it('should return US state for coordinates in New York', async () => {
    // Times Square coordinates
    const result = await service.reverseGeocode(40.758, -73.9855);
    expect(result.country).toBe('US');
    expect(result.state).toBeDefined();
    expect(result.city).toBeDefined();
  });

  it('should return non-US for coordinates in Moscow', async () => {
    const result = await service.reverseGeocode(55.7558, 37.6173);
    expect(result.country).not.toBe('US');
  });

  it('should validate US location', async () => {
    const isUS = await service.isInUS(40.758, -73.9855);
    expect(isUS).toBe(true);
  });

  it('should reject non-US location', async () => {
    const isUS = await service.isInUS(55.7558, 37.6173);
    expect(isUS).toBe(false);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
npx jest src/geo/geo.service.spec.ts
```

Expected: FAIL - module not found.

**Step 3: Implement GeoService**

```typescript
// src/geo/geo.service.ts
import { Injectable, Logger } from '@nestjs/common';
import * as NodeGeocoder from 'node-geocoder';

export interface GeoResult {
  country: string | null;
  state: string | null;
  city: string | null;
}

@Injectable()
export class GeoService {
  private readonly logger = new Logger(GeoService.name);
  private readonly geocoder: NodeGeocoder.Geocoder;

  constructor() {
    this.geocoder = NodeGeocoder({
      provider: 'openstreetmap',
    });
  }

  async reverseGeocode(latitude: number, longitude: number): Promise<GeoResult> {
    try {
      const results = await this.geocoder.reverse({ lat: latitude, lon: longitude });
      if (results.length === 0) {
        return { country: null, state: null, city: null };
      }
      const result = results[0];
      return {
        country: result.countryCode?.toUpperCase() ?? null,
        state: result.state ?? null,
        city: result.city ?? null,
      };
    } catch (error) {
      this.logger.error('Reverse geocoding failed', error);
      return { country: null, state: null, city: null };
    }
  }

  async isInUS(latitude: number, longitude: number): Promise<boolean> {
    const result = await this.reverseGeocode(latitude, longitude);
    return result.country === 'US';
  }
}
```

**Step 4: Create GeoModule**

```typescript
// src/geo/geo.module.ts
import { Module } from '@nestjs/common';
import { GeoService } from './geo.service';

@Module({
  providers: [GeoService],
  exports: [GeoService],
})
export class GeoModule {}
```

**Step 5: Run tests to verify they pass**

```bash
npx jest src/geo/geo.service.spec.ts
```

**Step 6: Handle location in BotUpdate**

Add to `bot.update.ts`:

```typescript
@On('location')
async onLocation(@Ctx() ctx: Context) {
  const location = (ctx.message as any).location;
  const { latitude, longitude } = location;

  const geo = await this.geoService.reverseGeocode(latitude, longitude);

  if (geo.country !== 'US') {
    await ctx.reply(
      'Looks like you\'re not in the US right now. This bot is for people based in the US.\n\n' +
      'If you\'re actually here but location is off, you can enter your city manually:',
      {
        reply_markup: {
          keyboard: [[{ text: '🏙 Enter city manually' }]],
          resize_keyboard: true,
        },
      },
    );
    return;
  }

  // Save location and mark verified
  await this.prismaService.user.upsert({
    where: { telegramId: BigInt(ctx.from.id) },
    update: {
      latitude,
      longitude,
      city: geo.city,
      state: geo.state,
      verificationStatus: 'VERIFIED',
    },
    create: {
      telegramId: BigInt(ctx.from.id),
      firstName: ctx.from.first_name,
      lastName: ctx.from.last_name,
      username: ctx.from.username,
      latitude,
      longitude,
      city: geo.city,
      state: geo.state,
      verificationStatus: 'VERIFIED',
    },
  });

  await ctx.reply(
    `Got it! You're in ${geo.city}, ${geo.state}. Verified ✓\n\n` +
    'Let\'s set up your profile. What should I call you?',
    { reply_markup: { remove_keyboard: true } },
  );

  // Transition to onboarding state
  await this.prismaService.user.update({
    where: { telegramId: BigInt(ctx.from.id) },
    data: { currentState: 'ONBOARDING' },
  });
}
```

Import GeoModule in BotModule and inject GeoService + PrismaService in BotUpdate constructor.

**Step 7: Commit**

```bash
git add .
git commit -m "feat: add geolocation verification with reverse geocoding"
```

---

## Task 5: Onboarding Flow (Wizard Scene)

**Files:**
- Create: `src/bot/scenes/onboarding.scene.ts`
- Modify: `src/bot/bot.module.ts`
- Modify: `src/app.module.ts` (add session middleware)

**Step 1: Install session middleware**

The Telegraf session is needed for scenes/wizards:

```bash
npm install @telegraf/session
```

**Step 2: Create onboarding scene**

```typescript
// src/bot/scenes/onboarding.scene.ts
import { Wizard, WizardStep, Ctx, On, Message } from 'nestjs-telegraf';
import { Context } from 'telegraf';
import { PrismaService } from '../../prisma/prisma.service';
import { Injectable } from '@nestjs/common';

interface OnboardingSession {
  name?: string;
  age?: number;
  languages?: string[];
  goal?: string;
  interests?: string[];
  redFlags?: string[];
}

@Wizard('onboarding')
@Injectable()
export class OnboardingScene {
  constructor(private readonly prisma: PrismaService) {}

  // Step 1: Name (already asked in bot.update.ts after geo verification)
  @WizardStep(1)
  async askAge(@Ctx() ctx: any, @Message() msg: any) {
    const name = msg?.text?.trim();
    if (!name || name.length < 1 || name.length > 50) {
      await ctx.reply('Just give me a first name (1-50 chars):');
      return;
    }
    ctx.wizard.state.name = name;
    await ctx.reply(`${name}, cool. How old are you?`);
    ctx.wizard.next();
  }

  // Step 2: Age
  @WizardStep(2)
  async askLanguages(@Ctx() ctx: any, @Message() msg: any) {
    const age = parseInt(msg?.text?.trim(), 10);
    if (isNaN(age) || age < 18 || age > 99) {
      await ctx.reply('Need a real age (18-99):');
      return;
    }
    ctx.wizard.state.age = age;
    await ctx.reply(
      'What languages do you speak? Pick or type:\n' +
      '🇷🇺 Russian\n🇺🇦 Ukrainian\n🇺🇸 English\n\n' +
      'Type like: "ru, en" or "all"',
    );
    ctx.wizard.next();
  }

  // Step 3: Languages
  @WizardStep(3)
  async askGoal(@Ctx() ctx: any, @Message() msg: any) {
    const text = msg?.text?.trim().toLowerCase();
    if (!text) {
      await ctx.reply('Type your languages (ru, ua, en):');
      return;
    }

    const langMap: Record<string, string> = {
      ru: 'RU', russian: 'RU', rus: 'RU',
      ua: 'UA', ukrainian: 'UA', ukr: 'UA',
      en: 'EN', english: 'EN', eng: 'EN',
      all: 'ALL',
    };

    let languages: string[];
    if (text === 'all') {
      languages = ['RU', 'UA', 'EN'];
    } else {
      languages = text.split(/[,\s]+/)
        .map(l => langMap[l])
        .filter(Boolean);
    }

    if (languages.length === 0) {
      await ctx.reply('Didn\'t catch that. Try: "ru, en" or "all"');
      return;
    }

    ctx.wizard.state.languages = [...new Set(languages)];
    await ctx.reply(
      'What are you here for?\n\n' +
      '1. Friends\n' +
      '2. Hangouts\n' +
      '3. Dating\n' +
      '4. Mixed (all of the above)\n\n' +
      'Just type the number or word:',
    );
    ctx.wizard.next();
  }

  // Step 4: Goal
  @WizardStep(4)
  async askInterests(@Ctx() ctx: any, @Message() msg: any) {
    const text = msg?.text?.trim().toLowerCase();
    const goalMap: Record<string, string> = {
      '1': 'FRIENDS', friends: 'FRIENDS',
      '2': 'HANGOUTS', hangouts: 'HANGOUTS',
      '3': 'DATING', dating: 'DATING',
      '4': 'MIXED', mixed: 'MIXED', all: 'MIXED',
    };

    const goal = goalMap[text];
    if (!goal) {
      await ctx.reply('Pick one: friends / hangouts / dating / mixed');
      return;
    }

    ctx.wizard.state.goal = goal;
    await ctx.reply(
      'Pick 5-7 interests that describe you (type them comma-separated):\n\n' +
      'Examples: hiking, cooking, gaming, music, fitness, photography, ' +
      'travel, art, sports, movies, dancing, yoga, coding, ' +
      'bars-clubs, coffee-meetups, outdoor-adventures, board-games, ' +
      'concerts, food-scene, humor, chill, ambitious, nerdy, creative\n\n' +
      'Type like: "hiking, humor, gaming, music, chill"',
    );
    ctx.wizard.next();
  }

  // Step 5: Interests
  @WizardStep(5)
  async askRedFlags(@Ctx() ctx: any, @Message() msg: any) {
    const text = msg?.text?.trim().toLowerCase();
    if (!text) {
      await ctx.reply('Type at least 3 interests, comma-separated:');
      return;
    }

    const interests = text.split(/[,]+/).map(s => s.trim()).filter(s => s.length > 0);
    if (interests.length < 3) {
      await ctx.reply('Give me at least 3 interests:');
      return;
    }

    ctx.wizard.state.interests = interests.slice(0, 7);
    await ctx.reply(
      'Last thing: any hard-no red flags? (1-3 things you absolutely don\'t want)\n\n' +
      'Examples: "smoking, heavy drinking, no humor"\n\n' +
      'Or type "none" to skip:',
    );
    ctx.wizard.next();
  }

  // Step 6: Red flags -> finish
  @WizardStep(6)
  async finish(@Ctx() ctx: any, @Message() msg: any) {
    const text = msg?.text?.trim().toLowerCase();
    let redFlags: string[] = [];

    if (text && text !== 'none' && text !== 'skip') {
      redFlags = text.split(/[,]+/).map(s => s.trim()).filter(s => s.length > 0).slice(0, 3);
    }

    ctx.wizard.state.redFlags = redFlags;

    const state: OnboardingSession = ctx.wizard.state;
    const telegramId = BigInt(ctx.from.id);

    // Save to database
    await this.prisma.user.update({
      where: { telegramId },
      data: {
        firstName: state.name,
        age: state.age,
        languages: state.languages,
        goal: state.goal as any,
        redFlags: state.redFlags,
        currentState: 'ACTIVE',
      },
    });

    // Create/link interest tags
    for (const interest of state.interests ?? []) {
      const tag = await this.prisma.tag.upsert({
        where: { name: interest },
        update: {},
        create: { name: interest, category: 'user-defined' },
      });
      await this.prisma.userTag.upsert({
        where: { userId_tagId: { userId: telegramId, tagId: tag.id } },
        update: {},
        create: { userId: telegramId, tagId: tag.id, weight: 1.0 },
      });
    }

    await ctx.reply(
      'Got it. Starting your picks. 🎯\n\n' +
      'I\'ll show you profiles one by one. Just tell me what you think in your own words.\n\n' +
      'Type /browse to start, or /profile to edit your profile.',
    );

    await ctx.scene.leave();
  }
}
```

**Step 3: Add session middleware to Telegraf config**

Update `app.module.ts` TelegrafModule config:

```typescript
import { session } from 'telegraf';

// In useFactory:
{
  token: configService.getOrThrow<string>('TELEGRAM_BOT_TOKEN'),
  middlewares: [session()],
}
```

Register the scene in BotModule.

**Step 4: Enter scene after geo verification**

In `bot.update.ts`, after saving user location and transitioning to ONBOARDING, enter the wizard:

```typescript
await (ctx as any).scene.enter('onboarding');
```

**Step 5: Run the bot, test full onboarding flow**

```bash
npm run start:dev
```

Test: /start -> share location -> name -> age -> languages -> goal -> interests -> red flags.

**Step 6: Commit**

```bash
git add .
git commit -m "feat: add onboarding wizard with name, age, languages, goal, interests, red flags"
```

---

## Task 6: Profile Card Display

**Files:**
- Create: `src/profile/profile.module.ts`
- Create: `src/profile/profile.service.ts`
- Create: `src/profile/profile.service.spec.ts`
- Modify: `src/bot/bot.update.ts`

**Step 1: Write the failing test**

```typescript
// src/profile/profile.service.spec.ts
import { Test } from '@nestjs/testing';
import { ProfileService } from './profile.service';
import { PrismaService } from '../prisma/prisma.service';

describe('ProfileService', () => {
  let service: ProfileService;

  beforeEach(async () => {
    const module = await Test.createTestingModule({
      providers: [
        ProfileService,
        {
          provide: PrismaService,
          useValue: {
            user: {
              findUnique: jest.fn(),
            },
            userTag: {
              findMany: jest.fn(),
            },
          },
        },
      ],
    }).compile();
    service = module.get(ProfileService);
  });

  it('should format profile card caption', () => {
    const caption = service.formatCardCaption({
      firstName: 'Masha',
      age: 23,
      city: 'Brooklyn',
      state: 'NY',
      bio: 'Love coffee and long walks',
      interests: ['hiking', 'coffee', 'music'],
    });

    expect(caption).toContain('Masha, 23');
    expect(caption).toContain('Brooklyn, NY');
    expect(caption).toContain('hiking');
  });
});
```

**Step 2: Run test to verify it fails**

```bash
npx jest src/profile/profile.service.spec.ts
```

**Step 3: Implement ProfileService**

```typescript
// src/profile/profile.service.ts
import { Injectable } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';

interface CardData {
  firstName: string;
  age: number | null;
  city: string | null;
  state: string | null;
  bio: string | null;
  interests: string[];
}

@Injectable()
export class ProfileService {
  constructor(private readonly prisma: PrismaService) {}

  formatCardCaption(data: CardData): string {
    const lines: string[] = [];

    const agePart = data.age ? `, ${data.age}` : '';
    lines.push(`<b>${data.firstName}${agePart}</b>`);

    if (data.city || data.state) {
      const location = [data.city, data.state].filter(Boolean).join(', ');
      lines.push(`📍 ${location}`);
    }

    if (data.bio) {
      lines.push('');
      lines.push(data.bio);
    }

    if (data.interests.length > 0) {
      lines.push('');
      lines.push(data.interests.map(i => `#${i.replace(/\s+/g, '_')}`).join(' '));
    }

    return lines.join('\n');
  }

  async getCardData(telegramId: bigint): Promise<CardData | null> {
    const user = await this.prisma.user.findUnique({
      where: { telegramId },
      include: { userTags: { include: { tag: true } } },
    });

    if (!user) return null;

    return {
      firstName: user.firstName,
      age: user.age,
      city: user.city,
      state: user.state,
      bio: user.bio,
      interests: user.userTags.map(ut => ut.tag.name),
    };
  }

  async sendProfileCard(
    bot: any,
    chatId: number,
    targetTelegramId: bigint,
  ): Promise<void> {
    const data = await this.getCardData(targetTelegramId);
    if (!data) return;

    const user = await this.prisma.user.findUnique({
      where: { telegramId: targetTelegramId },
    });
    if (!user) return;

    const caption = this.formatCardCaption(data);

    if (user.photos.length > 0) {
      await bot.telegram.sendPhoto(chatId, user.photos[0], {
        caption,
        parse_mode: 'HTML',
      });
    } else {
      await bot.telegram.sendMessage(chatId, caption, {
        parse_mode: 'HTML',
      });
    }

    await bot.telegram.sendMessage(chatId, 'Thoughts?');
  }
}
```

**Step 4: Run tests**

```bash
npx jest src/profile/profile.service.spec.ts
```

**Step 5: Create ProfileModule and export**

```typescript
// src/profile/profile.module.ts
import { Module } from '@nestjs/common';
import { ProfileService } from './profile.service';

@Module({
  providers: [ProfileService],
  exports: [ProfileService],
})
export class ProfileModule {}
```

**Step 6: Commit**

```bash
git add .
git commit -m "feat: add profile card rendering with photo + caption"
```

---

## Task 7: Discovery Feed & Profile Ranking

**Files:**
- Create: `src/discovery/discovery.module.ts`
- Create: `src/discovery/discovery.service.ts`
- Create: `src/discovery/discovery.service.spec.ts`
- Modify: `src/bot/bot.update.ts`

**Step 1: Write the failing test**

```typescript
// src/discovery/discovery.service.spec.ts
import { Test } from '@nestjs/testing';
import { DiscoveryService } from './discovery.service';
import { PrismaService } from '../prisma/prisma.service';

describe('DiscoveryService', () => {
  let service: DiscoveryService;
  let prisma: any;

  beforeEach(async () => {
    prisma = {
      $queryRaw: jest.fn(),
      user: { findUnique: jest.fn() },
      impression: { findMany: jest.fn() },
      like: { findMany: jest.fn() },
      pass: { findMany: jest.fn() },
    };

    const module = await Test.createTestingModule({
      providers: [
        DiscoveryService,
        { provide: PrismaService, useValue: prisma },
      ],
    }).compile();
    service = module.get(DiscoveryService);
  });

  it('should return candidates excluding already seen users', async () => {
    prisma.impression.findMany.mockResolvedValue([
      { targetId: BigInt(100) },
    ]);
    prisma.like.findMany.mockResolvedValue([]);
    prisma.pass.findMany.mockResolvedValue([]);
    prisma.$queryRaw.mockResolvedValue([
      { telegramId: BigInt(200), score: 5.0 },
      { telegramId: BigInt(300), score: 3.0 },
    ]);

    const candidates = await service.getNextCandidates(BigInt(1), 10);
    expect(candidates).toBeDefined();
    expect(Array.isArray(candidates)).toBe(true);
  });
});
```

**Step 2: Implement DiscoveryService**

```typescript
// src/discovery/discovery.service.ts
import { Injectable, Logger } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { Prisma } from '@prisma/client';

@Injectable()
export class DiscoveryService {
  private readonly logger = new Logger(DiscoveryService.name);

  constructor(private readonly prisma: PrismaService) {}

  async getNextCandidates(
    userId: bigint,
    limit: number = 10,
  ): Promise<{ telegramId: bigint; score: number }[]> {
    // Get user's data for filtering
    const user = await this.prisma.user.findUnique({
      where: { telegramId: userId },
      include: { userTags: true },
    });

    if (!user) return [];

    // Get already-interacted user IDs (seen, liked, passed)
    const [impressions, likes, passes] = await Promise.all([
      this.prisma.impression.findMany({
        where: { viewerId: userId },
        select: { targetId: true },
      }),
      this.prisma.like.findMany({
        where: { fromId: userId },
        select: { toId: true },
      }),
      this.prisma.pass.findMany({
        where: { fromId: userId },
        select: { toId: true },
      }),
    ]);

    const excludeIds = new Set<bigint>([
      userId,
      ...impressions.map(i => i.targetId),
      ...likes.map(l => l.toId),
      ...passes.map(p => p.toId),
    ]);

    const excludeArray = Array.from(excludeIds);

    if (excludeArray.length === 0) {
      excludeArray.push(userId);
    }

    // Weighted tag matching query
    const candidates = await this.prisma.$queryRaw<
      { telegramId: bigint; score: number }[]
    >`
      SELECT
        u."telegramId",
        COALESCE(SUM(a."weight" * b."weight"), 0) AS score
      FROM "User" u
      LEFT JOIN "UserTag" b ON b."userId" = u."telegramId"
      LEFT JOIN "UserTag" a ON a."tagId" = b."tagId" AND a."userId" = ${userId}
      WHERE u."telegramId" NOT IN (${Prisma.join(excludeArray)})
        AND u."isActive" = true
        AND u."verificationStatus" != 'PENDING'
        AND u."currentState" NOT IN ('NEW', 'ONBOARDING')
      GROUP BY u."telegramId"
      ORDER BY score DESC, u."lastActiveAt" DESC
      LIMIT ${limit}
    `;

    return candidates;
  }

  async recordImpression(viewerId: bigint, targetId: bigint): Promise<void> {
    await this.prisma.impression.upsert({
      where: {
        viewerId_targetId: { viewerId, targetId },
      },
      update: {},
      create: { viewerId, targetId },
    });

    await this.prisma.user.update({
      where: { telegramId: targetId },
      data: { totalImpressions: { increment: 1 } },
    });
  }
}
```

**Step 3: Create DiscoveryModule**

```typescript
// src/discovery/discovery.module.ts
import { Module } from '@nestjs/common';
import { DiscoveryService } from './discovery.service';

@Module({
  providers: [DiscoveryService],
  exports: [DiscoveryService],
})
export class DiscoveryModule {}
```

**Step 4: Add /browse command to BotUpdate**

```typescript
@Command('browse')
async onBrowse(@Ctx() ctx: Context) {
  const userId = BigInt(ctx.from.id);
  const user = await this.prisma.user.findUnique({ where: { telegramId: userId } });

  if (!user || user.currentState === 'NEW' || user.currentState === 'ONBOARDING') {
    await ctx.reply('Finish your profile first! Send /start');
    return;
  }

  const candidates = await this.discoveryService.getNextCandidates(userId, 1);
  if (candidates.length === 0) {
    await ctx.reply('No more profiles right now. Check back later!');
    return;
  }

  const candidate = candidates[0];

  // Record impression
  await this.discoveryService.recordImpression(userId, candidate.telegramId);

  // Update user state
  await this.prisma.user.update({
    where: { telegramId: userId },
    data: { currentState: 'BROWSING', currentCardId: candidate.telegramId },
  });

  // Send profile card
  await this.profileService.sendProfileCard(ctx.telegram, ctx.chat.id, candidate.telegramId);
}
```

**Step 5: Run tests**

```bash
npx jest src/discovery/discovery.service.spec.ts
```

**Step 6: Commit**

```bash
git add .
git commit -m "feat: add discovery feed with weighted tag-based profile ranking"
```

---

## Task 8: LLM-Powered Text Reaction Parsing

**Files:**
- Create: `src/reaction/reaction.module.ts`
- Create: `src/reaction/reaction.service.ts`
- Create: `src/reaction/reaction.service.spec.ts`
- Modify: `src/bot/bot.update.ts`

**Step 1: Install OpenAI SDK**

```bash
npm install openai
```

**Step 2: Write the failing test**

```typescript
// src/reaction/reaction.service.spec.ts
import { Test } from '@nestjs/testing';
import { ReactionService, ReactionResult } from './reaction.service';
import { ConfigService } from '@nestjs/config';

describe('ReactionService', () => {
  let service: ReactionService;

  beforeEach(async () => {
    const module = await Test.createTestingModule({
      providers: [
        ReactionService,
        {
          provide: ConfigService,
          useValue: {
            getOrThrow: jest.fn().mockReturnValue('test-key'),
          },
        },
      ],
    }).compile();
    service = module.get(ReactionService);
  });

  it('should parse "ok let\'s go" as a like', () => {
    // This tests the fallback parser (when LLM is unavailable)
    const result = service.fallbackParse('ok let\'s go');
    expect(result.action).toBe('like');
  });

  it('should parse "not my vibe" as a pass', () => {
    const result = service.fallbackParse('not my vibe');
    expect(result.action).toBe('pass');
  });

  it('should parse "idk" as a maybe', () => {
    const result = service.fallbackParse('idk');
    expect(result.action).toBe('maybe');
  });

  it('should parse "report" as a report', () => {
    const result = service.fallbackParse('report this person');
    expect(result.action).toBe('report');
  });
});
```

**Step 3: Implement ReactionService**

```typescript
// src/reaction/reaction.service.ts
import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import OpenAI from 'openai';

export interface ReactionResult {
  action: 'like' | 'pass' | 'maybe' | 'report';
  reasons: string[];
  confirmation: string;
}

@Injectable()
export class ReactionService {
  private readonly logger = new Logger(ReactionService.name);
  private readonly openai: OpenAI;

  constructor(private readonly configService: ConfigService) {
    this.openai = new OpenAI({
      apiKey: this.configService.getOrThrow<string>('OPENAI_API_KEY'),
    });
  }

  async parseReaction(
    userText: string,
    profileContext: { name: string; interests: string[] },
  ): Promise<ReactionResult> {
    try {
      const response = await this.openai.chat.completions.create({
        model: 'gpt-4o-mini',
        response_format: { type: 'json_object' },
        messages: [
          {
            role: 'system',
            content: `You are a dating app assistant. Parse the user's text reaction to a profile and return JSON with:
- "action": one of "like", "pass", "maybe", "report"
- "reasons": array of short reason tags (e.g., "too serious", "funny", "shared interests", "no common ground")
- "confirmation": a short, casual one-sentence confirmation message for the user that feels personalized

The profile they're reacting to is: ${profileContext.name}, interests: ${profileContext.interests.join(', ')}.

Examples:
Input: "ok let's go" -> {"action": "like", "reasons": ["general appeal"], "confirmation": "Got it: like. Saving that you're feeling the vibe."}
Input: "too serious, want more fun" -> {"action": "pass", "reasons": ["too serious"], "confirmation": "Got it: pass, reason 'too serious'. I'll tilt the feed toward more easy-going profiles."}
Input: "boring" -> {"action": "pass", "reasons": ["boring", "no engagement"], "confirmation": "Noted: pass. Adding more engaging profiles to your feed."}
Input: "idk, unsure" -> {"action": "maybe", "reasons": ["unsure"], "confirmation": "Saved as maybe. I'll circle back to similar profiles later."}
Input: "this person is fake" -> {"action": "report", "reasons": ["fake profile"], "confirmation": "Reported. I'll review this profile. Moving on."}

Return ONLY valid JSON.`,
          },
          { role: 'user', content: userText },
        ],
        max_tokens: 200,
        temperature: 0.3,
      });

      const content = response.choices[0]?.message?.content;
      if (!content) throw new Error('Empty LLM response');

      return JSON.parse(content) as ReactionResult;
    } catch (error) {
      this.logger.warn('LLM parsing failed, using fallback', error);
      return this.fallbackParse(userText);
    }
  }

  fallbackParse(text: string): ReactionResult {
    const lower = text.toLowerCase().trim();

    if (/report|fake|spam|scam|abuse/i.test(lower)) {
      return {
        action: 'report',
        reasons: ['reported'],
        confirmation: 'Reported. Moving on.',
      };
    }

    if (/^(ok|yes|let'?s go|sure|yeah|cool|nice|like|down|interested|match)/i.test(lower)) {
      return {
        action: 'like',
        reasons: ['general appeal'],
        confirmation: 'Got it: like!',
      };
    }

    if (/idk|unsure|maybe|not sure|hmm|dunno|don'?t know/i.test(lower)) {
      return {
        action: 'maybe',
        reasons: ['unsure'],
        confirmation: 'Saved as maybe.',
      };
    }

    // Default to pass
    const reasons: string[] = [];
    if (/boring/i.test(lower)) reasons.push('boring');
    if (/serious/i.test(lower)) reasons.push('too serious');
    if (/flirt/i.test(lower)) reasons.push('too flirty');
    if (/old|young|age/i.test(lower)) reasons.push('age mismatch');
    if (/far|distance/i.test(lower)) reasons.push('too far');
    if (reasons.length === 0) reasons.push('not interested');

    return {
      action: 'pass',
      reasons,
      confirmation: `Got it: pass. Reason: ${reasons.join(', ')}.`,
    };
  }
}
```

**Step 4: Create ReactionModule**

```typescript
// src/reaction/reaction.module.ts
import { Module } from '@nestjs/common';
import { ReactionService } from './reaction.service';

@Module({
  providers: [ReactionService],
  exports: [ReactionService],
})
export class ReactionModule {}
```

**Step 5: Run tests**

```bash
npx jest src/reaction/reaction.service.spec.ts
```

**Step 6: Wire reaction handling into BotUpdate**

Add a text message handler that activates when user is in BROWSING state (viewing a profile card):

```typescript
@On('text')
async onText(@Ctx() ctx: Context) {
  const userId = BigInt(ctx.from.id);
  const user = await this.prisma.user.findUnique({ where: { telegramId: userId } });

  if (!user) return;

  // Handle text based on user state
  if (user.currentState === 'BROWSING' && user.currentCardId) {
    await this.handleBrowsingReaction(ctx, user);
    return;
  }

  if (user.activeChatMatchId) {
    await this.handleChatRelay(ctx, user);
    return;
  }
}

private async handleBrowsingReaction(ctx: Context, user: any) {
  const text = (ctx.message as any).text;
  const currentCardId = user.currentCardId;

  // Get profile context for LLM
  const target = await this.prisma.user.findUnique({
    where: { telegramId: currentCardId },
    include: { userTags: { include: { tag: true } } },
  });

  if (!target) return;

  const reaction = await this.reactionService.parseReaction(text, {
    name: target.firstName,
    interests: target.userTags.map(ut => ut.tag.name),
  });

  // Send confirmation
  await ctx.reply(reaction.confirmation);

  // Process action
  switch (reaction.action) {
    case 'like':
      await this.matchingService.handleLike(user.telegramId, currentCardId, reaction.reasons);
      break;
    case 'pass':
      await this.matchingService.handlePass(user.telegramId, currentCardId, reaction.reasons);
      break;
    case 'maybe':
      // Don't record pass, just move on
      break;
    case 'report':
      // TODO: report handling
      break;
  }

  // Update user preferences based on reaction
  await this.personalizationService.updatePreferences(user.telegramId, reaction);

  // Show next profile
  await this.showNextProfile(ctx, user.telegramId);
}
```

**Step 7: Commit**

```bash
git add .
git commit -m "feat: add LLM-powered text reaction parsing with fallback"
```

---

## Task 9: Matching Service (Like/Pass/Mutual Match)

**Files:**
- Create: `src/matching/matching.module.ts`
- Create: `src/matching/matching.service.ts`
- Create: `src/matching/matching.service.spec.ts`

**Step 1: Write the failing test**

```typescript
// src/matching/matching.service.spec.ts
import { Test } from '@nestjs/testing';
import { MatchingService } from './matching.service';
import { PrismaService } from '../prisma/prisma.service';

describe('MatchingService', () => {
  let service: MatchingService;
  let prisma: any;

  beforeEach(async () => {
    prisma = {
      like: {
        create: jest.fn(),
        findUnique: jest.fn(),
      },
      pass: { create: jest.fn() },
      match: { create: jest.fn() },
      user: { update: jest.fn() },
    };

    const module = await Test.createTestingModule({
      providers: [
        MatchingService,
        { provide: PrismaService, useValue: prisma },
      ],
    }).compile();
    service = module.get(MatchingService);
  });

  it('should create a like and check for mutual match', async () => {
    prisma.like.create.mockResolvedValue({ id: '1', fromId: BigInt(1), toId: BigInt(2) });
    prisma.like.findUnique.mockResolvedValue(null); // No reciprocal like
    prisma.user.update.mockResolvedValue({});

    const result = await service.handleLike(BigInt(1), BigInt(2), ['funny']);
    expect(result.matched).toBe(false);
    expect(prisma.like.create).toHaveBeenCalledWith({
      data: { fromId: BigInt(1), toId: BigInt(2), reason: 'funny' },
    });
  });

  it('should detect mutual match when reciprocal like exists', async () => {
    prisma.like.create.mockResolvedValue({ id: '1', fromId: BigInt(1), toId: BigInt(2) });
    prisma.like.findUnique.mockResolvedValue({ id: '2', fromId: BigInt(2), toId: BigInt(1) });
    prisma.match.create.mockResolvedValue({ id: 'match-1', user1Id: BigInt(1), user2Id: BigInt(2) });
    prisma.user.update.mockResolvedValue({});

    const result = await service.handleLike(BigInt(1), BigInt(2), ['funny']);
    expect(result.matched).toBe(true);
  });
});
```

**Step 2: Run test to verify it fails**

```bash
npx jest src/matching/matching.service.spec.ts
```

**Step 3: Implement MatchingService**

```typescript
// src/matching/matching.service.ts
import { Injectable, Logger } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';

@Injectable()
export class MatchingService {
  private readonly logger = new Logger(MatchingService.name);

  constructor(private readonly prisma: PrismaService) {}

  async handleLike(
    fromId: bigint,
    toId: bigint,
    reasons: string[],
  ): Promise<{ matched: boolean; matchId?: string }> {
    // Create the like
    await this.prisma.like.create({
      data: {
        fromId,
        toId,
        reason: reasons.join(', '),
      },
    });

    // Update metrics
    await Promise.all([
      this.prisma.user.update({
        where: { telegramId: fromId },
        data: { totalLikesGiven: { increment: 1 } },
      }),
      this.prisma.user.update({
        where: { telegramId: toId },
        data: { totalLikesReceived: { increment: 1 } },
      }),
    ]);

    // Check for reciprocal like (mutual match)
    const reciprocal = await this.prisma.like.findUnique({
      where: {
        fromId_toId: { fromId: toId, toId: fromId },
      },
    });

    if (reciprocal) {
      // Create match (normalize order: smaller ID is user1)
      const [u1, u2] = fromId < toId ? [fromId, toId] : [toId, fromId];

      const match = await this.prisma.match.create({
        data: {
          user1Id: u1,
          user2Id: u2,
        },
      });

      // Update match counts
      await Promise.all([
        this.prisma.user.update({
          where: { telegramId: fromId },
          data: { totalMatches: { increment: 1 } },
        }),
        this.prisma.user.update({
          where: { telegramId: toId },
          data: { totalMatches: { increment: 1 } },
        }),
      ]);

      this.logger.log(`Match created: ${u1} <-> ${u2}`);

      return { matched: true, matchId: match.id };
    }

    return { matched: false };
  }

  async handlePass(fromId: bigint, toId: bigint, reasons: string[]): Promise<void> {
    await this.prisma.pass.create({
      data: {
        fromId,
        toId,
        reason: reasons.join(', '),
      },
    });
  }

  async getPeopleWhoLikedMe(userId: bigint): Promise<bigint[]> {
    const likes = await this.prisma.like.findMany({
      where: {
        toId: userId,
        // Exclude already matched
        from: {
          matchesAsUser1: { none: { user2Id: userId } },
          matchesAsUser2: { none: { user1Id: userId } },
        },
      },
      select: { fromId: true },
      orderBy: { createdAt: 'desc' },
    });

    return likes.map(l => l.fromId);
  }
}
```

**Step 4: Create MatchingModule**

```typescript
// src/matching/matching.module.ts
import { Module } from '@nestjs/common';
import { MatchingService } from './matching.service';

@Module({
  providers: [MatchingService],
  exports: [MatchingService],
})
export class MatchingModule {}
```

**Step 5: Run tests**

```bash
npx jest src/matching/matching.service.spec.ts
```

**Step 6: Commit**

```bash
git add .
git commit -m "feat: add matching service with like, pass, and mutual match detection"
```

---

## Task 10: Bot-Mediated Match Chat (Relay)

**Files:**
- Create: `src/chat/chat.module.ts`
- Create: `src/chat/chat.service.ts`
- Create: `src/chat/chat.service.spec.ts`
- Modify: `src/bot/bot.update.ts`

**Step 1: Write the failing test**

```typescript
// src/chat/chat.service.spec.ts
import { Test } from '@nestjs/testing';
import { ChatService } from './chat.service';
import { PrismaService } from '../prisma/prisma.service';

describe('ChatService', () => {
  let service: ChatService;
  let prisma: any;

  beforeEach(async () => {
    prisma = {
      chatRoom: {
        create: jest.fn(),
        findFirst: jest.fn(),
        update: jest.fn(),
        findMany: jest.fn(),
      },
      match: { findUnique: jest.fn() },
      user: { update: jest.fn(), findUnique: jest.fn() },
      message: { create: jest.fn() },
    };

    const module = await Test.createTestingModule({
      providers: [
        ChatService,
        { provide: PrismaService, useValue: prisma },
      ],
    }).compile();
    service = module.get(ChatService);
  });

  it('should create a chat room with 48h expiry', async () => {
    const now = new Date();
    prisma.chatRoom.create.mockResolvedValue({
      id: 'room-1',
      matchId: 'match-1',
      expiresAt: new Date(now.getTime() + 48 * 60 * 60 * 1000),
    });

    const room = await service.createChatRoom('match-1');
    expect(room).toBeDefined();
    expect(prisma.chatRoom.create).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          matchId: 'match-1',
        }),
      }),
    );
  });

  it('should generate icebreaker messages', () => {
    const icebreakers = service.generateIcebreakers('Masha', 'Dima', ['music', 'hiking']);
    expect(icebreakers.length).toBeGreaterThanOrEqual(2);
    expect(icebreakers.length).toBeLessThanOrEqual(4);
  });
});
```

**Step 2: Implement ChatService**

```typescript
// src/chat/chat.service.ts
import { Injectable, Logger } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { Telegraf } from 'telegraf';
import { InjectBot } from 'nestjs-telegraf';

@Injectable()
export class ChatService {
  private readonly logger = new Logger(ChatService.name);

  constructor(
    private readonly prisma: PrismaService,
    @InjectBot() private readonly bot: Telegraf,
  ) {}

  async createChatRoom(matchId: string): Promise<any> {
    const expiresAt = new Date(Date.now() + 48 * 60 * 60 * 1000);

    return this.prisma.chatRoom.create({
      data: {
        matchId,
        expiresAt,
      },
      include: {
        match: {
          include: {
            user1: true,
            user2: true,
          },
        },
      },
    });
  }

  generateIcebreakers(
    name1: string,
    name2: string,
    sharedInterests: string[],
  ): string[] {
    const messages: string[] = [];

    messages.push(
      `Yo, ${name1} and ${name2} -- you two matched! ` +
      `I'm just here to prevent "hi how are you" from killing this chat.`,
    );

    messages.push(
      'Each of you: in one sentence, what does a perfect evening in this city look like?',
    );

    messages.push(
      'Now one thing you genuinely enjoy lately (food/music/hobby).',
    );

    if (sharedInterests.length > 0) {
      messages.push(
        `Nice. I see overlap on ${sharedInterests.slice(0, 3).join(', ')}. ` +
        `You're on your own now. I'm going quiet. This chat closes in 48 hours.`,
      );
    } else {
      messages.push(
        'You\'re on your own now. I\'m going quiet. This chat closes in 48 hours.',
      );
    }

    return messages;
  }

  async startMatchChat(matchId: string): Promise<void> {
    const chatRoom = await this.createChatRoom(matchId);
    const { user1, user2 } = chatRoom.match;

    // Find shared interests
    const user1Tags = await this.prisma.userTag.findMany({
      where: { userId: user1.telegramId },
      include: { tag: true },
    });
    const user2Tags = await this.prisma.userTag.findMany({
      where: { userId: user2.telegramId },
      include: { tag: true },
    });
    const user1TagNames = new Set(user1Tags.map(ut => ut.tag.name));
    const sharedInterests = user2Tags
      .filter(ut => user1TagNames.has(ut.tag.name))
      .map(ut => ut.tag.name);

    const icebreakers = this.generateIcebreakers(
      user1.firstName,
      user2.firstName,
      sharedInterests,
    );

    // Set both users into chat mode
    await Promise.all([
      this.prisma.user.update({
        where: { telegramId: user1.telegramId },
        data: { activeChatMatchId: matchId, currentState: 'IN_CHAT' },
      }),
      this.prisma.user.update({
        where: { telegramId: user2.telegramId },
        data: { activeChatMatchId: matchId, currentState: 'IN_CHAT' },
      }),
    ]);

    // Send icebreakers to both users
    for (const msg of icebreakers) {
      await this.bot.telegram.sendMessage(Number(user1.telegramId), `🤝 ${msg}`);
      await this.bot.telegram.sendMessage(Number(user2.telegramId), `🤝 ${msg}`);
      // Small delay between messages for natural feel
      await new Promise(r => setTimeout(r, 1500));
    }

    this.logger.log(`Match chat started: ${user1.firstName} <-> ${user2.firstName}`);
  }

  async relayMessage(
    senderId: bigint,
    matchId: string,
    text: string,
  ): Promise<void> {
    // Find the match and determine recipient
    const match = await this.prisma.match.findUnique({
      where: { id: matchId },
      include: {
        user1: true,
        user2: true,
        chatRoom: true,
      },
    });

    if (!match || !match.chatRoom || match.chatRoom.closedAt) {
      await this.bot.telegram.sendMessage(
        Number(senderId),
        'This chat has ended.',
      );
      return;
    }

    // Check expiry
    if (match.chatRoom.expiresAt <= new Date()) {
      await this.closeChatRoom(match.chatRoom.id);
      return;
    }

    const recipientId = match.user1Id === senderId
      ? match.user2Id
      : match.user1Id;
    const senderUser = match.user1Id === senderId ? match.user1 : match.user2;

    // Save message
    await this.prisma.message.create({
      data: {
        chatRoomId: match.chatRoom.id,
        senderId,
        text,
      },
    });

    // Relay to recipient
    await this.bot.telegram.sendMessage(
      Number(recipientId),
      `💬 ${senderUser.firstName}: ${text}`,
    );

    // Update engagement metrics
    await this.prisma.user.update({
      where: { telegramId: senderId },
      data: {
        totalResponses: { increment: 1 },
        totalChatsStarted: { increment: 1 }, // simplified; ideally track first message only
      },
    });
  }

  async closeChatRoom(chatRoomId: string): Promise<void> {
    const chatRoom = await this.prisma.chatRoom.update({
      where: { id: chatRoomId },
      data: { closedAt: new Date() },
      include: {
        match: {
          include: { user1: true, user2: true },
        },
      },
    });

    const { user1, user2 } = chatRoom.match;

    // Notify both users
    const closeMsg = '⏰ This chat has closed after 48 hours. Hope you had a good one!\n\nType /browse to keep discovering.';
    await Promise.all([
      this.bot.telegram.sendMessage(Number(user1.telegramId), closeMsg),
      this.bot.telegram.sendMessage(Number(user2.telegramId), closeMsg),
    ]);

    // Reset user states
    await Promise.all([
      this.prisma.user.update({
        where: { telegramId: user1.telegramId },
        data: { activeChatMatchId: null, currentState: 'ACTIVE' },
      }),
      this.prisma.user.update({
        where: { telegramId: user2.telegramId },
        data: { activeChatMatchId: null, currentState: 'ACTIVE' },
      }),
    ]);

    this.logger.log(`Chat room ${chatRoomId} closed`);
  }
}
```

**Step 3: Create ChatModule**

```typescript
// src/chat/chat.module.ts
import { Module } from '@nestjs/common';
import { ChatService } from './chat.service';

@Module({
  providers: [ChatService],
  exports: [ChatService],
})
export class ChatModule {}
```

**Step 4: Add relay handler to BotUpdate**

```typescript
private async handleChatRelay(ctx: Context, user: any) {
  const text = (ctx.message as any).text;

  // Handle /endchat command
  if (text === '/endchat') {
    const match = await this.prisma.match.findUnique({
      where: { id: user.activeChatMatchId },
      include: { chatRoom: true },
    });
    if (match?.chatRoom) {
      await this.chatService.closeChatRoom(match.chatRoom.id);
    }
    return;
  }

  // Relay the message
  await this.chatService.relayMessage(
    user.telegramId,
    user.activeChatMatchId,
    text,
  );
}
```

**Step 5: Run tests**

```bash
npx jest src/chat/chat.service.spec.ts
```

**Step 6: Commit**

```bash
git add .
git commit -m "feat: add bot-mediated match chat with relay, icebreakers, and 48h expiry"
```

---

## Task 11: Chat Expiry Cron Job

**Files:**
- Create: `src/chat/chat-cleanup.service.ts`
- Create: `src/chat/chat-cleanup.service.spec.ts`

**Step 1: Write the failing test**

```typescript
// src/chat/chat-cleanup.service.spec.ts
import { Test } from '@nestjs/testing';
import { ChatCleanupService } from './chat-cleanup.service';
import { ChatService } from './chat.service';
import { PrismaService } from '../prisma/prisma.service';

describe('ChatCleanupService', () => {
  let service: ChatCleanupService;
  let prisma: any;
  let chatService: any;

  beforeEach(async () => {
    prisma = {
      chatRoom: {
        findMany: jest.fn(),
      },
    };
    chatService = {
      closeChatRoom: jest.fn(),
    };

    const module = await Test.createTestingModule({
      providers: [
        ChatCleanupService,
        { provide: PrismaService, useValue: prisma },
        { provide: ChatService, useValue: chatService },
      ],
    }).compile();
    service = module.get(ChatCleanupService);
  });

  it('should close expired chat rooms', async () => {
    prisma.chatRoom.findMany.mockResolvedValue([
      { id: 'room-1' },
      { id: 'room-2' },
    ]);
    chatService.closeChatRoom.mockResolvedValue(undefined);

    await service.handleExpiredChats();

    expect(chatService.closeChatRoom).toHaveBeenCalledTimes(2);
    expect(chatService.closeChatRoom).toHaveBeenCalledWith('room-1');
    expect(chatService.closeChatRoom).toHaveBeenCalledWith('room-2');
  });
});
```

**Step 2: Implement ChatCleanupService**

```typescript
// src/chat/chat-cleanup.service.ts
import { Injectable, Logger } from '@nestjs/common';
import { Cron, CronExpression } from '@nestjs/schedule';
import { PrismaService } from '../prisma/prisma.service';
import { ChatService } from './chat.service';

@Injectable()
export class ChatCleanupService {
  private readonly logger = new Logger(ChatCleanupService.name);

  constructor(
    private readonly prisma: PrismaService,
    private readonly chatService: ChatService,
  ) {}

  @Cron(CronExpression.EVERY_5_MINUTES)
  async handleExpiredChats() {
    const expiredRooms = await this.prisma.chatRoom.findMany({
      where: {
        expiresAt: { lte: new Date() },
        closedAt: null,
      },
      select: { id: true },
    });

    if (expiredRooms.length > 0) {
      this.logger.log(`Closing ${expiredRooms.length} expired chat rooms`);
    }

    for (const room of expiredRooms) {
      try {
        await this.chatService.closeChatRoom(room.id);
      } catch (error) {
        this.logger.error(`Failed to close room ${room.id}`, error);
      }
    }
  }
}
```

**Step 3: Add to ChatModule providers**

**Step 4: Run tests**

```bash
npx jest src/chat/chat-cleanup.service.spec.ts
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add cron job to close expired chat rooms every 5 minutes"
```

---

## Task 12: Personalization Service (Preference Learning)

**Files:**
- Create: `src/personalization/personalization.module.ts`
- Create: `src/personalization/personalization.service.ts`
- Create: `src/personalization/personalization.service.spec.ts`

**Step 1: Write the failing test**

```typescript
// src/personalization/personalization.service.spec.ts
import { Test } from '@nestjs/testing';
import { PersonalizationService } from './personalization.service';
import { PrismaService } from '../prisma/prisma.service';

describe('PersonalizationService', () => {
  let service: PersonalizationService;
  let prisma: any;

  beforeEach(async () => {
    prisma = {
      user: { findUnique: jest.fn(), update: jest.fn() },
      userTag: { upsert: jest.fn(), findMany: jest.fn() },
      tag: { upsert: jest.fn() },
      like: { count: jest.fn() },
      pass: { count: jest.fn() },
    };

    const module = await Test.createTestingModule({
      providers: [
        PersonalizationService,
        { provide: PrismaService, useValue: prisma },
      ],
    }).compile();
    service = module.get(PersonalizationService);
  });

  it('should boost tag weights for liked profile interests', async () => {
    prisma.user.findUnique.mockResolvedValue({
      telegramId: BigInt(1),
      rejectionPatterns: {},
    });
    prisma.userTag.findMany.mockResolvedValue([
      { tagId: 'tag-1', tag: { name: 'humor' } },
    ]);
    prisma.tag.upsert.mockResolvedValue({ id: 'tag-1' });
    prisma.userTag.upsert.mockResolvedValue({});
    prisma.user.update.mockResolvedValue({});

    await service.updatePreferences(BigInt(1), {
      action: 'like',
      reasons: ['funny'],
      confirmation: '',
    }, BigInt(2));

    expect(prisma.userTag.upsert).toHaveBeenCalled();
  });

  it('should track rejection patterns', async () => {
    prisma.user.findUnique.mockResolvedValue({
      telegramId: BigInt(1),
      rejectionPatterns: { 'too serious': 2 },
    });
    prisma.user.update.mockResolvedValue({});
    prisma.userTag.findMany.mockResolvedValue([]);

    await service.updatePreferences(BigInt(1), {
      action: 'pass',
      reasons: ['too serious'],
      confirmation: '',
    }, BigInt(2));

    expect(prisma.user.update).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          rejectionPatterns: { 'too serious': 3 },
        }),
      }),
    );
  });

  it('should detect rejection streak', () => {
    expect(service.getStreakMessage(5)).toContain('strict mode');
  });
});
```

**Step 2: Implement PersonalizationService**

```typescript
// src/personalization/personalization.service.ts
import { Injectable, Logger } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { ReactionResult } from '../reaction/reaction.service';

@Injectable()
export class PersonalizationService {
  private readonly logger = new Logger(PersonalizationService.name);

  // In-memory rejection streak tracking (per session)
  private rejectionStreaks = new Map<string, number>();

  constructor(private readonly prisma: PrismaService) {}

  async updatePreferences(
    userId: bigint,
    reaction: ReactionResult,
    targetId?: bigint,
  ): Promise<void> {
    const user = await this.prisma.user.findUnique({
      where: { telegramId: userId },
    });
    if (!user) return;

    if (reaction.action === 'like') {
      // Reset rejection streak
      this.rejectionStreaks.delete(userId.toString());

      // Boost weights of the liked user's tags
      if (targetId) {
        const targetTags = await this.prisma.userTag.findMany({
          where: { userId: targetId },
          include: { tag: true },
        });

        for (const tt of targetTags) {
          await this.prisma.userTag.upsert({
            where: { userId_tagId: { userId, tagId: tt.tagId } },
            update: { weight: { increment: 0.3 } },
            create: { userId, tagId: tt.tagId, weight: 1.3 },
          });
        }
      }
    }

    if (reaction.action === 'pass') {
      // Increment rejection streak
      const key = userId.toString();
      const current = this.rejectionStreaks.get(key) ?? 0;
      this.rejectionStreaks.set(key, current + 1);

      // Track rejection patterns
      const patterns = (user.rejectionPatterns as Record<string, number>) ?? {};
      for (const reason of reaction.reasons) {
        patterns[reason] = (patterns[reason] ?? 0) + 1;
      }
      await this.prisma.user.update({
        where: { telegramId: userId },
        data: { rejectionPatterns: patterns },
      });

      // Decrease weights of the passed user's tags
      if (targetId) {
        const targetTags = await this.prisma.userTag.findMany({
          where: { userId: targetId },
          include: { tag: true },
        });

        for (const tt of targetTags) {
          const existing = await this.prisma.userTag.findMany({
            where: { userId, tagId: tt.tagId },
          });
          if (existing.length > 0) {
            await this.prisma.userTag.upsert({
              where: { userId_tagId: { userId, tagId: tt.tagId } },
              update: { weight: { increment: -0.1 } },
              create: { userId, tagId: tt.tagId, weight: 0.9 },
            });
          }
        }
      }
    }
  }

  getStreakMessage(streak: number): string | null {
    if (streak === 3) {
      return "You're in strict mode today. Got it. Tweaking the feed.";
    }
    if (streak === 5) {
      return "Okay, we're in strict mode. I'll tighten the picks.";
    }
    if (streak === 8) {
      return "Tough crowd today. I'm reshuffling hard. Maybe take a break?";
    }
    if (streak >= 10) {
      return "At this point I'm basically filtering out everyone. Let's revisit your preferences or try again tomorrow.";
    }
    return null;
  }

  getRejectionStreak(userId: bigint): number {
    return this.rejectionStreaks.get(userId.toString()) ?? 0;
  }

  async generateYesterdaySummary(userId: bigint): Promise<string | null> {
    const user = await this.prisma.user.findUnique({
      where: { telegramId: userId },
    });
    if (!user) return null;

    const patterns = (user.rejectionPatterns as Record<string, number>) ?? {};
    const topReasons = Object.entries(patterns)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 2)
      .map(([reason]) => reason);

    // Count yesterday's likes/passes
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    yesterday.setHours(0, 0, 0, 0);
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const [likesCount, passesCount] = await Promise.all([
      this.prisma.like.count({
        where: { fromId: userId, createdAt: { gte: yesterday, lt: today } },
      }),
      this.prisma.pass.count({
        where: { fromId: userId, createdAt: { gte: yesterday, lt: today } },
      }),
    ]);

    if (likesCount === 0 && passesCount === 0) return null;

    if (passesCount > likesCount * 3 && topReasons.length > 0) {
      return `Yesterday was picky. I reduced "${topReasons[0]}" profiles and added more variety.`;
    }

    if (likesCount > passesCount) {
      return `Yesterday went well -- you were vibing. Keeping the feed in that direction.`;
    }

    return `Based on yesterday, I adjusted the feed. Let's see if today clicks better.`;
  }
}
```

**Step 3: Create PersonalizationModule**

```typescript
// src/personalization/personalization.module.ts
import { Module } from '@nestjs/common';
import { PersonalizationService } from './personalization.service';

@Module({
  providers: [PersonalizationService],
  exports: [PersonalizationService],
})
export class PersonalizationModule {}
```

**Step 4: Run tests**

```bash
npx jest src/personalization/personalization.service.spec.ts
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add personalization service with preference learning and streak detection"
```

---

## Task 13: "People Who Liked You" Feature

**Files:**
- Modify: `src/bot/bot.update.ts`
- Modify: `src/matching/matching.service.ts`

**Step 1: Add /likes command to BotUpdate**

```typescript
@Command('likes')
async onLikes(@Ctx() ctx: Context) {
  const userId = BigInt(ctx.from.id);

  const likerIds = await this.matchingService.getPeopleWhoLikedMe(userId);

  if (likerIds.length === 0) {
    await ctx.reply('No one has liked you yet. Keep browsing and building your profile!');
    return;
  }

  await ctx.reply(`${likerIds.length} people liked you! Here's who:\n`);

  // Show first profile from likers
  const firstLikerId = likerIds[0];
  await this.discoveryService.recordImpression(userId, firstLikerId);
  await this.prisma.user.update({
    where: { telegramId: userId },
    data: { currentState: 'BROWSING', currentCardId: firstLikerId },
  });
  await this.profileService.sendProfileCard(ctx.telegram, ctx.chat.id, firstLikerId);

  await ctx.reply('This person already liked you. What do you think?');
}
```

**Step 2: Test manually**

```bash
npm run start:dev
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add /likes command to show people who liked you"
```

---

## Task 14: Badges & Ranking Service

**Files:**
- Create: `src/ranking/ranking.module.ts`
- Create: `src/ranking/ranking.service.ts`
- Create: `src/ranking/ranking.service.spec.ts`

**Step 1: Write the failing test**

```typescript
// src/ranking/ranking.service.spec.ts
import { Test } from '@nestjs/testing';
import { RankingService, Badge } from './ranking.service';
import { PrismaService } from '../prisma/prisma.service';

describe('RankingService', () => {
  let service: RankingService;

  beforeEach(async () => {
    const module = await Test.createTestingModule({
      providers: [
        RankingService,
        {
          provide: PrismaService,
          useValue: { user: { findUnique: jest.fn() } },
        },
      ],
    }).compile();
    service = module.get(RankingService);
  });

  it('should assign "Attention Hunter" badge', () => {
    const badges = service.computeBadges({
      totalImpressions: 100,
      totalLikesReceived: 50,
      totalLikesGiven: 5,
      totalMatches: 1,
      totalChatsStarted: 0,
      totalResponses: 0,
    });
    expect(badges).toContain('Attention Hunter');
  });

  it('should assign "Like Sprayer" badge', () => {
    const badges = service.computeBadges({
      totalImpressions: 100,
      totalLikesReceived: 5,
      totalLikesGiven: 80,
      totalMatches: 2,
      totalChatsStarted: 1,
      totalResponses: 1,
    });
    expect(badges).toContain('Like Sprayer');
  });

  it('should assign "Selective but Real" badge', () => {
    const badges = service.computeBadges({
      totalImpressions: 100,
      totalLikesReceived: 30,
      totalLikesGiven: 10,
      totalMatches: 7,
      totalChatsStarted: 6,
      totalResponses: 20,
    });
    expect(badges).toContain('Selective but Real');
  });

  it('should assign "Ghost" badge', () => {
    const badges = service.computeBadges({
      totalImpressions: 100,
      totalLikesReceived: 20,
      totalLikesGiven: 20,
      totalMatches: 10,
      totalChatsStarted: 1,
      totalResponses: 0,
    });
    expect(badges).toContain('Ghost');
  });
});
```

**Step 2: Implement RankingService**

```typescript
// src/ranking/ranking.service.ts
import { Injectable } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';

export type Badge =
  | 'Attention Hunter'
  | 'Selective but Real'
  | 'Like Sprayer'
  | 'Ghost'
  | 'Solid Communicator'
  | 'Starter'
  | 'Empty Profile'
  | 'Clear Profile';

interface UserMetrics {
  totalImpressions: number;
  totalLikesReceived: number;
  totalLikesGiven: number;
  totalMatches: number;
  totalChatsStarted: number;
  totalResponses: number;
}

@Injectable()
export class RankingService {
  constructor(private readonly prisma: PrismaService) {}

  computeBadges(metrics: UserMetrics): Badge[] {
    const badges: Badge[] = [];

    const likeReceivedRate = metrics.totalImpressions > 0
      ? metrics.totalLikesReceived / metrics.totalImpressions
      : 0;
    const likeGivenRate = metrics.totalImpressions > 0
      ? metrics.totalLikesGiven / metrics.totalImpressions
      : 0;
    const mutualMatchRate = metrics.totalLikesGiven > 0
      ? metrics.totalMatches / metrics.totalLikesGiven
      : 0;
    const chatStartRate = metrics.totalMatches > 0
      ? metrics.totalChatsStarted / metrics.totalMatches
      : 0;
    const responseRate = metrics.totalMatches > 0
      ? metrics.totalResponses / metrics.totalMatches
      : 0;

    // Attention Hunter: gets many likes, rarely likes back, low engagement
    if (likeReceivedRate > 0.3 && likeGivenRate < 0.1 && chatStartRate < 0.3) {
      badges.push('Attention Hunter');
    }

    // Selective but Real: few likes given but high mutual + good response
    if (likeGivenRate < 0.2 && mutualMatchRate > 0.4 && responseRate > 1) {
      badges.push('Selective but Real');
    }

    // Like Sprayer: likes way too many, low mutual rate
    if (likeGivenRate > 0.6 && mutualMatchRate < 0.1) {
      badges.push('Like Sprayer');
    }

    // Ghost: has matches but doesn't engage
    if (metrics.totalMatches >= 3 && chatStartRate < 0.2 && responseRate < 1) {
      badges.push('Ghost');
    }

    // Solid Communicator: responds consistently
    if (metrics.totalMatches >= 3 && responseRate > 3) {
      badges.push('Solid Communicator');
    }

    // Starter: often sends first message
    if (metrics.totalMatches >= 3 && chatStartRate > 0.7) {
      badges.push('Starter');
    }

    return badges;
  }

  async getUserBadges(telegramId: bigint): Promise<Badge[]> {
    const user = await this.prisma.user.findUnique({
      where: { telegramId },
    });
    if (!user) return [];

    const metrics: UserMetrics = {
      totalImpressions: user.totalImpressions,
      totalLikesReceived: user.totalLikesReceived,
      totalLikesGiven: user.totalLikesGiven,
      totalMatches: user.totalMatches,
      totalChatsStarted: user.totalChatsStarted,
      totalResponses: user.totalResponses,
    };

    return this.computeBadges(metrics);
  }
}
```

**Step 3: Create RankingModule**

```typescript
// src/ranking/ranking.module.ts
import { Module } from '@nestjs/common';
import { RankingService } from './ranking.service';

@Module({
  providers: [RankingService],
  exports: [RankingService],
})
export class RankingModule {}
```

**Step 4: Run tests**

```bash
npx jest src/ranking/ranking.service.spec.ts
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add ranking service with behavioral badges"
```

---

## Task 15: Profile Management Commands

**Files:**
- Modify: `src/bot/bot.update.ts`

**Step 1: Add /profile command**

```typescript
@Command('profile')
async onProfile(@Ctx() ctx: Context) {
  const userId = BigInt(ctx.from.id);
  const user = await this.prisma.user.findUnique({
    where: { telegramId: userId },
    include: { userTags: { include: { tag: true } } },
  });

  if (!user) {
    await ctx.reply('No profile found. Send /start to get started.');
    return;
  }

  const badges = await this.rankingService.getUserBadges(userId);
  const badgeStr = badges.length > 0
    ? `\nBadges: ${badges.join(', ')}`
    : '';

  const interests = user.userTags.map(ut => `#${ut.tag.name}`).join(' ');

  await ctx.reply(
    `<b>Your Profile</b>\n\n` +
    `Name: ${user.firstName}\n` +
    `Age: ${user.age ?? 'not set'}\n` +
    `Location: ${user.city ?? '?'}, ${user.state ?? '?'}\n` +
    `Goal: ${user.goal ?? 'not set'}\n` +
    `Bio: ${user.bio ?? 'not set'}\n` +
    `Interests: ${interests || 'none'}\n` +
    `Photos: ${user.photos.length}\n` +
    `Verification: ${user.verificationStatus}` +
    badgeStr +
    `\n\nTo update:\n` +
    `/setbio <your bio>\n` +
    `/setphoto (send a photo)\n` +
    `/setage <age>`,
    { parse_mode: 'HTML' },
  );
}

@Command('setbio')
async onSetBio(@Ctx() ctx: Context) {
  const text = (ctx.message as any).text.replace('/setbio', '').trim();
  if (!text) {
    await ctx.reply('Usage: /setbio Your bio here (1-3 sentences)');
    return;
  }

  await this.prisma.user.update({
    where: { telegramId: BigInt(ctx.from.id) },
    data: { bio: text.slice(0, 500) },
  });

  await ctx.reply('Bio updated!');
}

@On('photo')
async onPhoto(@Ctx() ctx: Context) {
  const photos = (ctx.message as any).photo;
  if (!photos || photos.length === 0) return;

  // Get highest resolution photo
  const photo = photos[photos.length - 1];
  const fileId = photo.file_id;

  const user = await this.prisma.user.findUnique({
    where: { telegramId: BigInt(ctx.from.id) },
  });

  if (!user) return;

  const currentPhotos = user.photos ?? [];
  if (currentPhotos.length >= 3) {
    await ctx.reply('Max 3 photos. Remove old ones first with /removephoto');
    return;
  }

  await this.prisma.user.update({
    where: { telegramId: BigInt(ctx.from.id) },
    data: { photos: [...currentPhotos, fileId] },
  });

  await ctx.reply(`Photo ${currentPhotos.length + 1}/3 saved!`);
}
```

**Step 2: Test manually**

```bash
npm run start:dev
```

**Step 3: Commit**

```bash
git add .
git commit -m "feat: add profile management commands (/profile, /setbio, photo upload)"
```

---

## Task 16: Session Memory ("Yesterday Summary")

**Files:**
- Modify: `src/bot/bot.update.ts`
- Modify: `src/personalization/personalization.service.ts`

**Step 1: Add greeting with yesterday context on /browse or /start (returning users)**

In the `/browse` command handler, before showing the first card:

```typescript
// Check if user hasn't been active today
const today = new Date();
today.setHours(0, 0, 0, 0);
if (user.lastActiveAt < today) {
  const summary = await this.personalizationService.generateYesterdaySummary(userId);
  if (summary) {
    await ctx.reply(`📝 ${summary}`);
  }
}

// Update last active
await this.prisma.user.update({
  where: { telegramId: userId },
  data: { lastActiveAt: new Date() },
});
```

**Step 2: Add streak quips after reactions**

In `handleBrowsingReaction`, after processing the reaction:

```typescript
if (reaction.action === 'pass') {
  const streak = this.personalizationService.getRejectionStreak(user.telegramId);
  const streakMsg = this.personalizationService.getStreakMessage(streak);
  if (streakMsg) {
    await ctx.reply(`💭 ${streakMsg}`);
  }
}
```

**Step 3: Test manually**

**Step 4: Commit**

```bash
git add .
git commit -m "feat: add yesterday summary and rejection streak quips"
```

---

## Task 17: Wire Everything Together in AppModule

**Files:**
- Modify: `src/app.module.ts`
- Modify: `src/bot/bot.module.ts`

**Step 1: Update AppModule with all modules**

```typescript
// src/app.module.ts
import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TelegrafModule } from 'nestjs-telegraf';
import { ScheduleModule } from '@nestjs/schedule';
import { session } from 'telegraf';
import { PrismaModule } from './prisma/prisma.module';
import { BotModule } from './bot/bot.module';
import { GeoModule } from './geo/geo.module';
import { ProfileModule } from './profile/profile.module';
import { DiscoveryModule } from './discovery/discovery.module';
import { ReactionModule } from './reaction/reaction.module';
import { MatchingModule } from './matching/matching.module';
import { ChatModule } from './chat/chat.module';
import { PersonalizationModule } from './personalization/personalization.module';
import { RankingModule } from './ranking/ranking.module';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    ScheduleModule.forRoot(),
    TelegrafModule.forRootAsync({
      imports: [ConfigModule],
      useFactory: (configService: ConfigService) => ({
        token: configService.getOrThrow<string>('TELEGRAM_BOT_TOKEN'),
        middlewares: [session()],
      }),
      inject: [ConfigService],
    }),
    PrismaModule,
    GeoModule,
    ProfileModule,
    DiscoveryModule,
    ReactionModule,
    MatchingModule,
    ChatModule,
    PersonalizationModule,
    RankingModule,
    BotModule,
  ],
})
export class AppModule {}
```

**Step 2: Update BotModule to import required modules**

```typescript
// src/bot/bot.module.ts
import { Module } from '@nestjs/common';
import { BotUpdate } from './bot.update';
import { OnboardingScene } from './scenes/onboarding.scene';
import { GeoModule } from '../geo/geo.module';
import { ProfileModule } from '../profile/profile.module';
import { DiscoveryModule } from '../discovery/discovery.module';
import { ReactionModule } from '../reaction/reaction.module';
import { MatchingModule } from '../matching/matching.module';
import { ChatModule } from '../chat/chat.module';
import { PersonalizationModule } from '../personalization/personalization.module';
import { RankingModule } from '../ranking/ranking.module';

@Module({
  imports: [
    GeoModule,
    ProfileModule,
    DiscoveryModule,
    ReactionModule,
    MatchingModule,
    ChatModule,
    PersonalizationModule,
    RankingModule,
  ],
  providers: [BotUpdate, OnboardingScene],
})
export class BotModule {}
```

**Step 3: Run all tests**

```bash
npx jest --passWithNoTests
```

**Step 4: Run the full bot**

```bash
npm run start:dev
```

**Step 5: Commit**

```bash
git add .
git commit -m "feat: wire all modules together in AppModule"
```

---

## Task 18: End-to-End Manual Testing Checklist

This is a manual test pass. Go through each flow in Telegram:

1. **Onboarding**
   - Send /start to bot
   - Share location -> verify US check works
   - Complete all onboarding steps (name, age, languages, goal, interests, red flags)
   - Verify user appears in database with correct data

2. **Profile**
   - Send /profile -> verify all fields shown
   - Send /setbio -> verify bio updates
   - Send a photo -> verify it's stored
   - Send /profile again -> verify photo count

3. **Discovery**
   - Send /browse -> verify profile card appears
   - Type a reaction ("cool, let's go") -> verify like is recorded
   - Type a pass reaction ("boring") -> verify pass is recorded
   - Type several passes -> verify streak messages appear
   - Verify next profile loads after each reaction

4. **Matching**
   - Create two test users
   - Have User A like User B
   - Have User B like User A -> verify match notification
   - Verify icebreaker messages sent to both users
   - Send message as User A -> verify relay to User B
   - Send message as User B -> verify relay to User A

5. **Chat Expiry**
   - Create a match with short expiry (modify for testing)
   - Wait for cron to fire -> verify chat closes
   - Verify both users notified

6. **People Who Liked You**
   - Have User B like User A (no reciprocal yet)
   - As User A, send /likes -> verify User B's card shown
   - Like User B -> verify instant match

7. **Personalization**
   - Browse multiple profiles, pass with reasons
   - Verify rejection patterns stored in database
   - Close bot, reopen next day -> verify yesterday summary

---

## Summary: Module Dependency Graph

```
AppModule
├── ConfigModule (global)
├── ScheduleModule
├── TelegrafModule
├── PrismaModule (global)
├── GeoModule
├── ProfileModule
├── DiscoveryModule
├── ReactionModule
├── MatchingModule
├── ChatModule (includes ChatCleanupService cron)
├── PersonalizationModule
├── RankingModule
└── BotModule
    ├── BotUpdate (main handler)
    └── OnboardingScene (wizard)
```

## File Count: ~35 files (including tests)

## Commands Reference

| Command | Description |
|---------|-------------|
| `/start` | Begin onboarding or show welcome back |
| `/browse` | Start discovery feed |
| `/likes` | Show people who liked you |
| `/profile` | View your profile |
| `/setbio <text>` | Update bio |
| `/setage <age>` | Update age |
| `/endchat` | End current match chat early |
| (send photo) | Add profile photo |
| (send text while browsing) | React to current profile |
| (send text while in chat) | Relay message to match |
