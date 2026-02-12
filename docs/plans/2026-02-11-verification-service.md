# Verification Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a geo-verification service that validates users are in the USA upon first interaction with the Telegram bot, stores their verification status in PostgreSQL via Prisma, and provides a fallback manual city/state selection.

**Architecture:** NestJS modular architecture. `ConfigModule` for env vars, `PrismaModule` for DB access, `TelegrafModule` for Telegram bot, `VerificationModule` for geo-verification logic. The bot listens for `/start`, asks user to share location, reverse-geocodes via free Nominatim API (OpenStreetMap), and stores verification result. Users who decline get manual state/city picker with "unverified" status.

**Tech Stack:** NestJS 11, Prisma (latest), PostgreSQL, nestjs-telegraf + telegraf, @nestjs/config, node-fetch (for Nominatim)

---

### Task 1: Install dependencies

**Files:**
- Modify: `package.json`

**Step 1: Install production dependencies**

Run:
```bash
npm install nestjs-telegraf telegraf @nestjs/config @prisma/client
```
Expected: packages added to dependencies

**Step 2: Install dev dependencies**

Run:
```bash
npm install -D prisma
```
Expected: prisma added to devDependencies

**Step 3: Commit**

```bash
git add package.json package-lock.json
git commit -m "chore: add telegraf, prisma, and config dependencies"
```

---

### Task 2: Set up Prisma with User schema

**Files:**
- Create: `prisma/schema.prisma`
- Create: `.env` (do NOT commit — already in .gitignore)

**Step 1: Initialize Prisma**

Run:
```bash
cd /Users/nmashchenko/Documents/matcher-bot && npx prisma init
```
Expected: creates `prisma/schema.prisma` and `.env` with `DATABASE_URL`

**Step 2: Set DATABASE_URL in `.env`**

Edit `.env`:
```
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/matcher_bot?schema=public"
TELEGRAM_BOT_TOKEN="your-bot-token-here"
```

**Step 3: Write the User model in `prisma/schema.prisma`**

Replace the generated schema with:

```prisma
generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

enum VerificationStatus {
  PENDING
  VERIFIED
  UNVERIFIED
  REJECTED
}

model User {
  id        String   @id @default(uuid())
  telegramId BigInt  @unique
  username   String?
  firstName  String?
  lastName   String?

  // Verification
  verificationStatus VerificationStatus @default(PENDING)
  latitude           Float?
  longitude          Float?
  country            String?
  state              String?
  city               String?
  verifiedAt         DateTime?

  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  @@map("users")
}
```

**Step 4: Generate Prisma client**

Run:
```bash
npx prisma generate
```
Expected: Prisma Client generated successfully

**Step 5: Create and apply migration**

Run:
```bash
npx prisma migrate dev --name init-user-verification
```
Expected: Migration applied, `users` table created in PostgreSQL.
NOTE: PostgreSQL must be running locally. If not, the engineer should start it first.

**Step 6: Commit**

```bash
git add prisma/schema.prisma prisma/migrations
git commit -m "feat: add Prisma schema with User model and verification status"
```

---

### Task 3: Create PrismaModule (shared DB access)

**Files:**
- Create: `src/prisma/prisma.service.ts`
- Create: `src/prisma/prisma.module.ts`

**Step 1: Create PrismaService**

Create `src/prisma/prisma.service.ts`:

```typescript
import { Injectable, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { PrismaClient } from '@prisma/client';

@Injectable()
export class PrismaService
  extends PrismaClient
  implements OnModuleInit, OnModuleDestroy
{
  async onModuleInit() {
    await this.$connect();
  }

  async onModuleDestroy() {
    await this.$disconnect();
  }
}
```

**Step 2: Create PrismaModule**

Create `src/prisma/prisma.module.ts`:

```typescript
import { Global, Module } from '@nestjs/common';
import { PrismaService } from './prisma.service.js';

@Global()
@Module({
  providers: [PrismaService],
  exports: [PrismaService],
})
export class PrismaModule {}
```

**Step 3: Commit**

```bash
git add src/prisma
git commit -m "feat: add global PrismaModule for database access"
```

---

### Task 4: Create ConfigModule setup and wire up AppModule

**Files:**
- Create: `src/config/config.module.ts`
- Modify: `src/app.module.ts`
- Modify: `src/main.ts`

**Step 1: Create config module**

Create `src/config/config.module.ts`:

```typescript
import { Module } from '@nestjs/common';
import { ConfigModule as NestConfigModule } from '@nestjs/config';

@Module({
  imports: [
    NestConfigModule.forRoot({
      isGlobal: true,
    }),
  ],
})
export class ConfigModule {}
```

**Step 2: Update AppModule with Prisma, Config, and Telegraf**

Replace `src/app.module.ts`:

```typescript
import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TelegrafModule } from 'nestjs-telegraf';
import { session } from 'telegraf';
import { PrismaModule } from './prisma/prisma.module.js';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    PrismaModule,
    TelegrafModule.forRootAsync({
      inject: [ConfigService],
      useFactory: (configService: ConfigService) => ({
        token: configService.getOrThrow<string>('TELEGRAM_BOT_TOKEN'),
        middlewares: [session()],
      }),
    }),
  ],
})
export class AppModule {}
```

**Step 3: Update main.ts**

Replace `src/main.ts`:

```typescript
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module.js';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  app.enableShutdownHooks();
  await app.listen(process.env.PORT ?? 3000);
}
bootstrap();
```

**Step 4: Delete unused boilerplate files**

Delete these files since we no longer need the default controller/service:
- `src/app.controller.ts`
- `src/app.service.ts`
- `src/app.controller.spec.ts`

**Step 5: Verify it compiles**

Run:
```bash
npx nest build
```
Expected: compiles without errors

**Step 6: Commit**

```bash
git add src/app.module.ts src/main.ts src/config
git rm src/app.controller.ts src/app.service.ts src/app.controller.spec.ts
git commit -m "feat: wire up ConfigModule, PrismaModule, and TelegrafModule"
```

---

### Task 5: Create GeocodingService (Nominatim reverse geocoding)

**Files:**
- Create: `src/verification/geocoding.service.ts`
- Create: `src/verification/geocoding.service.spec.ts`

**Step 1: Write the failing test**

Create `src/verification/geocoding.service.spec.ts`:

```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { GeocodingService } from './geocoding.service.js';

// We mock global fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe('GeocodingService', () => {
  let service: GeocodingService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [GeocodingService],
    }).compile();

    service = module.get<GeocodingService>(GeocodingService);
    mockFetch.mockReset();
  });

  describe('reverseGeocode', () => {
    it('should return US location data for coordinates in the USA', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          address: {
            country: 'United States',
            country_code: 'us',
            state: 'California',
            city: 'Los Angeles',
          },
        }),
      });

      const result = await service.reverseGeocode(34.0522, -118.2437);

      expect(result).toEqual({
        country: 'United States',
        countryCode: 'us',
        state: 'California',
        city: 'Los Angeles',
        isUSA: true,
      });
    });

    it('should return isUSA=false for non-US coordinates', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          address: {
            country: 'Germany',
            country_code: 'de',
            state: 'Berlin',
            city: 'Berlin',
          },
        }),
      });

      const result = await service.reverseGeocode(52.52, 13.405);

      expect(result.isUSA).toBe(false);
    });

    it('should return null when API fails', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 500 });

      const result = await service.reverseGeocode(0, 0);

      expect(result).toBeNull();
    });
  });
});
```

**Step 2: Run test to verify it fails**

Run:
```bash
npx jest --testPathPattern=geocoding.service.spec --verbose
```
Expected: FAIL — `Cannot find module './geocoding.service.js'`

**Step 3: Write the implementation**

Create `src/verification/geocoding.service.ts`:

```typescript
import { Injectable, Logger } from '@nestjs/common';

export interface GeocodingResult {
  country: string;
  countryCode: string;
  state: string;
  city: string;
  isUSA: boolean;
}

@Injectable()
export class GeocodingService {
  private readonly logger = new Logger(GeocodingService.name);

  async reverseGeocode(
    latitude: number,
    longitude: number,
  ): Promise<GeocodingResult | null> {
    try {
      const url = `https://nominatim.openstreetmap.org/reverse?format=json&lat=${latitude}&lon=${longitude}&addressdetails=1`;

      const response = await fetch(url, {
        headers: {
          'User-Agent': 'MatcherBot/1.0',
        },
      });

      if (!response.ok) {
        this.logger.warn(
          `Nominatim API returned ${response.status} for (${latitude}, ${longitude})`,
        );
        return null;
      }

      const data = await response.json();
      const address = data.address;

      return {
        country: address.country ?? '',
        countryCode: address.country_code ?? '',
        state: address.state ?? '',
        city: address.city ?? address.town ?? address.village ?? '',
        isUSA: address.country_code === 'us',
      };
    } catch (error) {
      this.logger.error('Reverse geocoding failed', error);
      return null;
    }
  }
}
```

**Step 4: Run test to verify it passes**

Run:
```bash
npx jest --testPathPattern=geocoding.service.spec --verbose
```
Expected: 3 tests PASS

**Step 5: Commit**

```bash
git add src/verification/geocoding.service.ts src/verification/geocoding.service.spec.ts
git commit -m "feat: add GeocodingService with Nominatim reverse geocoding"
```

---

### Task 6: Create VerificationService (business logic)

**Files:**
- Create: `src/verification/verification.service.ts`
- Create: `src/verification/verification.service.spec.ts`

**Step 1: Write the failing test**

Create `src/verification/verification.service.spec.ts`:

```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { VerificationService } from './verification.service.js';
import { PrismaService } from '../prisma/prisma.service.js';
import { GeocodingService } from './geocoding.service.js';
import { VerificationStatus } from '@prisma/client';

describe('VerificationService', () => {
  let service: VerificationService;
  let prisma: { user: { upsert: jest.Mock; findUnique: jest.Mock } };
  let geocoding: { reverseGeocode: jest.Mock };

  beforeEach(async () => {
    prisma = {
      user: {
        upsert: jest.fn(),
        findUnique: jest.fn(),
      },
    };
    geocoding = {
      reverseGeocode: jest.fn(),
    };

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        VerificationService,
        { provide: PrismaService, useValue: prisma },
        { provide: GeocodingService, useValue: geocoding },
      ],
    }).compile();

    service = module.get<VerificationService>(VerificationService);
  });

  describe('findOrCreateUser', () => {
    it('should upsert a user by telegramId', async () => {
      const mockUser = {
        id: '1',
        telegramId: BigInt(12345),
        verificationStatus: VerificationStatus.PENDING,
      };
      prisma.user.upsert.mockResolvedValue(mockUser);

      const result = await service.findOrCreateUser({
        telegramId: 12345,
        username: 'testuser',
        firstName: 'Test',
      });

      expect(prisma.user.upsert).toHaveBeenCalledWith({
        where: { telegramId: BigInt(12345) },
        update: { username: 'testuser', firstName: 'Test' },
        create: {
          telegramId: BigInt(12345),
          username: 'testuser',
          firstName: 'Test',
        },
      });
      expect(result).toEqual(mockUser);
    });
  });

  describe('verifyByLocation', () => {
    it('should mark user as VERIFIED when in USA', async () => {
      geocoding.reverseGeocode.mockResolvedValue({
        country: 'United States',
        countryCode: 'us',
        state: 'California',
        city: 'Los Angeles',
        isUSA: true,
      });
      const mockUser = {
        id: '1',
        verificationStatus: VerificationStatus.VERIFIED,
      };
      prisma.user.upsert.mockResolvedValue(mockUser);

      const result = await service.verifyByLocation(BigInt(12345), 34.05, -118.24);

      expect(result.verified).toBe(true);
      expect(result.state).toBe('California');
      expect(result.city).toBe('Los Angeles');
    });

    it('should mark user as REJECTED when outside USA', async () => {
      geocoding.reverseGeocode.mockResolvedValue({
        country: 'Germany',
        countryCode: 'de',
        state: 'Berlin',
        city: 'Berlin',
        isUSA: false,
      });
      prisma.user.upsert.mockResolvedValue({
        id: '1',
        verificationStatus: VerificationStatus.REJECTED,
      });

      const result = await service.verifyByLocation(BigInt(12345), 52.52, 13.40);

      expect(result.verified).toBe(false);
    });

    it('should return verified=false when geocoding fails', async () => {
      geocoding.reverseGeocode.mockResolvedValue(null);

      const result = await service.verifyByLocation(BigInt(12345), 0, 0);

      expect(result.verified).toBe(false);
      expect(result.error).toBe('geocoding_failed');
    });
  });

  describe('verifyManually', () => {
    it('should set user as UNVERIFIED with manual state/city', async () => {
      const mockUser = {
        id: '1',
        verificationStatus: VerificationStatus.UNVERIFIED,
        state: 'New York',
        city: 'New York City',
      };
      prisma.user.upsert.mockResolvedValue(mockUser);

      const result = await service.verifyManually(
        BigInt(12345),
        'New York',
        'New York City',
      );

      expect(result.status).toBe(VerificationStatus.UNVERIFIED);
      expect(result.state).toBe('New York');
    });
  });

  describe('getVerificationStatus', () => {
    it('should return user verification status', async () => {
      prisma.user.findUnique.mockResolvedValue({
        verificationStatus: VerificationStatus.VERIFIED,
        state: 'California',
        city: 'LA',
      });

      const result = await service.getVerificationStatus(BigInt(12345));

      expect(result?.verificationStatus).toBe(VerificationStatus.VERIFIED);
    });

    it('should return null for non-existent user', async () => {
      prisma.user.findUnique.mockResolvedValue(null);

      const result = await service.getVerificationStatus(BigInt(99999));

      expect(result).toBeNull();
    });
  });
});
```

**Step 2: Run test to verify it fails**

Run:
```bash
npx jest --testPathPattern=verification.service.spec --verbose
```
Expected: FAIL — `Cannot find module './verification.service.js'`

**Step 3: Write the implementation**

Create `src/verification/verification.service.ts`:

```typescript
import { Injectable, Logger } from '@nestjs/common';
import { VerificationStatus } from '@prisma/client';
import { PrismaService } from '../prisma/prisma.service.js';
import { GeocodingService } from './geocoding.service.js';

interface VerifyByLocationResult {
  verified: boolean;
  state?: string;
  city?: string;
  error?: string;
}

interface ManualVerifyResult {
  status: VerificationStatus;
  state: string;
  city: string;
}

@Injectable()
export class VerificationService {
  private readonly logger = new Logger(VerificationService.name);

  constructor(
    private readonly prisma: PrismaService,
    private readonly geocoding: GeocodingService,
  ) {}

  async findOrCreateUser(data: {
    telegramId: number;
    username?: string;
    firstName?: string;
    lastName?: string;
  }) {
    return this.prisma.user.upsert({
      where: { telegramId: BigInt(data.telegramId) },
      update: {
        username: data.username,
        firstName: data.firstName,
      },
      create: {
        telegramId: BigInt(data.telegramId),
        username: data.username,
        firstName: data.firstName,
      },
    });
  }

  async verifyByLocation(
    telegramId: bigint,
    latitude: number,
    longitude: number,
  ): Promise<VerifyByLocationResult> {
    const geoResult = await this.geocoding.reverseGeocode(latitude, longitude);

    if (!geoResult) {
      return { verified: false, error: 'geocoding_failed' };
    }

    const status = geoResult.isUSA
      ? VerificationStatus.VERIFIED
      : VerificationStatus.REJECTED;

    await this.prisma.user.upsert({
      where: { telegramId },
      update: {
        verificationStatus: status,
        latitude,
        longitude,
        country: geoResult.country,
        state: geoResult.state,
        city: geoResult.city,
        verifiedAt: geoResult.isUSA ? new Date() : null,
      },
      create: {
        telegramId,
        verificationStatus: status,
        latitude,
        longitude,
        country: geoResult.country,
        state: geoResult.state,
        city: geoResult.city,
        verifiedAt: geoResult.isUSA ? new Date() : null,
      },
    });

    if (!geoResult.isUSA) {
      this.logger.log(
        `User ${telegramId} rejected: location is in ${geoResult.country}`,
      );
      return { verified: false };
    }

    this.logger.log(
      `User ${telegramId} verified: ${geoResult.city}, ${geoResult.state}`,
    );
    return {
      verified: true,
      state: geoResult.state,
      city: geoResult.city,
    };
  }

  async verifyManually(
    telegramId: bigint,
    state: string,
    city: string,
  ): Promise<ManualVerifyResult> {
    await this.prisma.user.upsert({
      where: { telegramId },
      update: {
        verificationStatus: VerificationStatus.UNVERIFIED,
        state,
        city,
        country: 'United States',
      },
      create: {
        telegramId,
        verificationStatus: VerificationStatus.UNVERIFIED,
        state,
        city,
        country: 'United States',
      },
    });

    return {
      status: VerificationStatus.UNVERIFIED,
      state,
      city,
    };
  }

  async getVerificationStatus(telegramId: bigint) {
    return this.prisma.user.findUnique({
      where: { telegramId },
      select: {
        verificationStatus: true,
        state: true,
        city: true,
        verifiedAt: true,
      },
    });
  }
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
npx jest --testPathPattern=verification.service.spec --verbose
```
Expected: 5 tests PASS

**Step 5: Commit**

```bash
git add src/verification/verification.service.ts src/verification/verification.service.spec.ts
git commit -m "feat: add VerificationService with location and manual verification"
```

---

### Task 7: Create US states data for manual selection

**Files:**
- Create: `src/verification/us-states.ts`

**Step 1: Create the states data file**

Create `src/verification/us-states.ts`:

```typescript
export const US_STATES = [
  'Alabama', 'Alaska', 'Arizona', 'Arkansas', 'California',
  'Colorado', 'Connecticut', 'Delaware', 'Florida', 'Georgia',
  'Hawaii', 'Idaho', 'Illinois', 'Indiana', 'Iowa',
  'Kansas', 'Kentucky', 'Louisiana', 'Maine', 'Maryland',
  'Massachusetts', 'Michigan', 'Minnesota', 'Mississippi', 'Missouri',
  'Montana', 'Nebraska', 'Nevada', 'New Hampshire', 'New Jersey',
  'New Mexico', 'New York', 'North Carolina', 'North Dakota', 'Ohio',
  'Oklahoma', 'Oregon', 'Pennsylvania', 'Rhode Island', 'South Carolina',
  'South Dakota', 'Tennessee', 'Texas', 'Utah', 'Vermont',
  'Virginia', 'Washington', 'West Virginia', 'Wisconsin', 'Wyoming',
] as const;

export type USState = (typeof US_STATES)[number];
```

**Step 2: Commit**

```bash
git add src/verification/us-states.ts
git commit -m "feat: add US states data for manual verification fallback"
```

---

### Task 8: Create VerificationUpdate (Telegram bot handler)

This is the main Telegram interaction handler. It handles `/start`, location sharing, and the manual state/city selection flow.

**Files:**
- Create: `src/verification/verification.update.ts`

**Step 1: Write the bot update handler**

Create `src/verification/verification.update.ts`:

```typescript
import { Logger } from '@nestjs/common';
import { Update, Ctx, Start, On, Action, Hears } from 'nestjs-telegraf';
import { Context, Markup } from 'telegraf';
import { message } from 'telegraf/filters';
import { VerificationService } from './verification.service.js';
import { US_STATES } from './us-states.js';
import { VerificationStatus } from '@prisma/client';

@Update()
export class VerificationUpdate {
  private readonly logger = new Logger(VerificationUpdate.name);

  constructor(private readonly verificationService: VerificationService) {}

  @Start()
  async onStart(@Ctx() ctx: Context) {
    const from = ctx.from;
    if (!from) return;

    await this.verificationService.findOrCreateUser({
      telegramId: from.id,
      username: from.username,
      firstName: from.first_name,
      lastName: from.last_name,
    });

    const status = await this.verificationService.getVerificationStatus(
      BigInt(from.id),
    );

    if (
      status?.verificationStatus === VerificationStatus.VERIFIED ||
      status?.verificationStatus === VerificationStatus.UNVERIFIED
    ) {
      await ctx.reply(
        `С возвращением! Ты уже зарегистрирован (${status.city}, ${status.state}). Скоро здесь будет подбор.`,
      );
      return;
    }

    await ctx.reply(
      'Привет! Я — Matcher Bot. Помогу найти интересных людей из СНГ рядом с тобой в США.\n\n' +
        'Для начала мне нужно убедиться, что ты в США. Поделись геолокацией — это одноразово и безопасно.',
      Markup.keyboard([
        [Markup.button.locationRequest('📍 Поделиться геолокацией')],
        ['🏙 Выбрать город вручную'],
      ])
        .oneTime()
        .resize(),
    );
  }

  @On(message('location'))
  async onLocation(@Ctx() ctx: Context) {
    const from = ctx.from;
    if (!from) return;

    const location = (ctx.message as any)?.location;
    if (!location) return;

    await ctx.reply('⏳ Проверяю твою геолокацию...');

    const result = await this.verificationService.verifyByLocation(
      BigInt(from.id),
      location.latitude,
      location.longitude,
    );

    if (result.verified) {
      await ctx.reply(
        `✅ Подтверждено! Ты в ${result.city}, ${result.state}.\n\n` +
          'Отлично, теперь можно переходить к настройке профиля. (Скоро будет доступно)',
        Markup.removeKeyboard(),
      );
    } else if (result.error === 'geocoding_failed') {
      await ctx.reply(
        '❌ Не удалось определить местоположение. Попробуй ещё раз или выбери город вручную.',
        Markup.keyboard([
          [Markup.button.locationRequest('📍 Поделиться геолокацией')],
          ['🏙 Выбрать город вручную'],
        ])
          .oneTime()
          .resize(),
      );
    } else {
      await ctx.reply(
        '❌ Похоже, ты не в США. Этот бот пока работает только для людей в Штатах.\n\n' +
          'Если ты считаешь, что это ошибка — попробуй ещё раз или выбери город вручную.',
        Markup.keyboard([
          [Markup.button.locationRequest('📍 Попробовать снова')],
          ['🏙 Выбрать город вручную'],
        ])
          .oneTime()
          .resize(),
      );
    }
  }

  @Hears('🏙 Выбрать город вручную')
  async onManualSelect(@Ctx() ctx: Context) {
    // Show state selection in chunks (inline keyboard, 3 columns)
    const buttons = US_STATES.map((state) =>
      Markup.button.callback(state, `state:${state}`),
    );

    // Split into rows of 3
    const rows: ReturnType<typeof Markup.button.callback>[][] = [];
    for (let i = 0; i < buttons.length; i += 3) {
      rows.push(buttons.slice(i, i + 3));
    }

    await ctx.reply(
      '📍 Выбери свой штат:',
      Markup.inlineKeyboard(rows),
    );
  }

  @Action(/^state:(.+)$/)
  async onStateSelected(@Ctx() ctx: Context) {
    const from = ctx.from;
    if (!from) return;

    const callbackQuery = ctx.callbackQuery;
    if (!callbackQuery || !('data' in callbackQuery)) return;

    const state = callbackQuery.data.replace('state:', '');

    await ctx.answerCbQuery();

    // Store selected state in session for the next step
    (ctx as any).session = (ctx as any).session || {};
    (ctx as any).session.selectedState = state;

    await ctx.editMessageText(
      `Штат: ${state}\n\nТеперь напиши название своего города:`,
    );
  }

  @On(message('text'))
  async onText(@Ctx() ctx: Context) {
    const from = ctx.from;
    if (!from) return;

    const text = (ctx.message as any)?.text;
    if (!text) return;

    // Check if user is in manual city input mode
    const session = (ctx as any).session;
    if (!session?.selectedState) return;

    const state = session.selectedState;
    const city = text.trim();
    delete session.selectedState;

    const result = await this.verificationService.verifyManually(
      BigInt(from.id),
      state,
      city,
    );

    await ctx.reply(
      `📍 Записал: ${city}, ${state}\n` +
        `⚠️ Статус: не подтверждён (показы будут ограничены, пока не подтвердишь геолокацию)\n\n` +
        'Можно переходить к настройке профиля. (Скоро будет доступно)',
      Markup.removeKeyboard(),
    );
  }
}
```

**Step 2: Commit**

```bash
git add src/verification/verification.update.ts
git commit -m "feat: add VerificationUpdate Telegram handler with geo and manual flows"
```

---

### Task 9: Create VerificationModule and wire into AppModule

**Files:**
- Create: `src/verification/verification.module.ts`
- Modify: `src/app.module.ts`

**Step 1: Create VerificationModule**

Create `src/verification/verification.module.ts`:

```typescript
import { Module } from '@nestjs/common';
import { GeocodingService } from './geocoding.service.js';
import { VerificationService } from './verification.service.js';
import { VerificationUpdate } from './verification.update.js';

@Module({
  providers: [GeocodingService, VerificationService, VerificationUpdate],
  exports: [VerificationService],
})
export class VerificationModule {}
```

**Step 2: Import VerificationModule in AppModule**

Update `src/app.module.ts` — add the import:

```typescript
import { Module } from '@nestjs/common';
import { ConfigModule, ConfigService } from '@nestjs/config';
import { TelegrafModule } from 'nestjs-telegraf';
import { session } from 'telegraf';
import { PrismaModule } from './prisma/prisma.module.js';
import { VerificationModule } from './verification/verification.module.js';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    PrismaModule,
    TelegrafModule.forRootAsync({
      inject: [ConfigService],
      useFactory: (configService: ConfigService) => ({
        token: configService.getOrThrow<string>('TELEGRAM_BOT_TOKEN'),
        middlewares: [session()],
      }),
    }),
    VerificationModule,
  ],
})
export class AppModule {}
```

**Step 3: Verify build**

Run:
```bash
npx nest build
```
Expected: compiles without errors

**Step 4: Commit**

```bash
git add src/verification/verification.module.ts src/app.module.ts
git commit -m "feat: create VerificationModule and register in AppModule"
```

---

### Task 10: Run all tests and verify end-to-end

**Files:** None (verification only)

**Step 1: Run unit tests**

Run:
```bash
npx jest --verbose
```
Expected: All tests pass (geocoding + verification service tests)

**Step 2: Run build**

Run:
```bash
npx nest build
```
Expected: Compiles without errors

**Step 3: Manual smoke test**

1. Ensure PostgreSQL is running
2. Ensure `.env` has a valid `TELEGRAM_BOT_TOKEN` and `DATABASE_URL`
3. Run: `npx nest start`
4. Open Telegram, send `/start` to the bot
5. Verify: bot replies with the welcome message and location sharing keyboard
6. Share location → verify bot confirms USA or rejects
7. Try "🏙 Выбрать город вручную" → verify state selection → type city → verify confirmation

**Step 4: Final commit (if any test fixes needed)**

```bash
git add -A
git commit -m "fix: test/build fixes for verification service"
```

---

## Summary

| Task | What it does |
|------|-------------|
| 1 | Install all dependencies |
| 2 | Set up Prisma schema with User model + verification enum |
| 3 | Create shared PrismaModule |
| 4 | Wire up ConfigModule + TelegrafModule in AppModule |
| 5 | GeocodingService (Nominatim reverse geocode, tested) |
| 6 | VerificationService (business logic, tested) |
| 7 | US states data for manual fallback |
| 8 | VerificationUpdate (Telegram bot handler — /start, location, manual flow) |
| 9 | VerificationModule + wire into AppModule |
| 10 | Run all tests + manual smoke test |
