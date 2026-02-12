import { Module } from '@nestjs/common';
import { GeocodingService } from './geocoding.service.js';
import { VerificationService } from './verification.service.js';
import { VerificationUpdate } from './verification.update.js';

@Module({
  providers: [GeocodingService, VerificationService, VerificationUpdate],
  exports: [VerificationService, VerificationUpdate],
})
export class VerificationModule {}
