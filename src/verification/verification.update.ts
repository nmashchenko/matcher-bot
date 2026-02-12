import { Logger } from '@nestjs/common';
import { Update, Ctx, Start, On, Action, Hears } from 'nestjs-telegraf';
import { Context, Markup } from 'telegraf';
import { VerificationService } from './verification.service.js';
import { US_STATES } from './us-states.js';
import { VerificationStatus } from '../../prisma/generated/client.js';

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

  @On('location')
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
    const buttons = US_STATES.map((state) =>
      Markup.button.callback(state, `state:${state}`),
    );

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

    (ctx as any).session = (ctx as any).session || {};
    (ctx as any).session.selectedState = state;

    await ctx.editMessageText(
      `Штат: ${state}\n\nТеперь напиши название своего города:`,
    );
  }

  @On('text')
  async onText(@Ctx() ctx: Context) {
    const from = ctx.from;
    if (!from) return;

    const text = (ctx.message as any)?.text;
    if (!text) return;

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
