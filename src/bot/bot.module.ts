import { Module } from '@nestjs/common';
import { BotUpdate } from './bot.update';
import { VerificationModule } from '../verification/verification.module';

@Module({
  imports: [VerificationModule],
  providers: [BotUpdate],
})
export class BotModule {}
