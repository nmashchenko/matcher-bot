import { Logger } from '@nestjs/common';
import { Update, Ctx, Start } from 'nestjs-telegraf';
import { VerificationService } from '../verification/verification.service.js';
import { VerificationUpdate } from '../verification/verification.update.js';
import { VerificationStatus } from '../../prisma/generated/client.js';
import type { BotContext } from './bot-context.js';

@Update()
export class BotUpdate {
  private readonly logger = new Logger(BotUpdate.name);

  constructor(
    private readonly verificationService: VerificationService,
    private readonly verificationUpdate: VerificationUpdate,
  ) {}

  @Start()
  async onStart(@Ctx() ctx: BotContext) {
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

    await this.verificationUpdate.sendVerificationPrompt(ctx);
  }
}
