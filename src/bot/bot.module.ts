import { Module } from '@nestjs/common';
import { BotUpdate } from './bot.update.js';
import { VerificationModule } from '../verification/verification.module.js';

@Module({
  imports: [VerificationModule],
  providers: [BotUpdate],
})
export class BotModule {}
