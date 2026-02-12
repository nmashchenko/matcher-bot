import { Module } from '@nestjs/common';
import { GeocodingService } from './geocoding.service';
import { VerificationService } from './verification.service';
import { VerificationUpdate } from './verification.update';

@Module({
  providers: [GeocodingService, VerificationService, VerificationUpdate],
  exports: [VerificationService, VerificationUpdate],
})
export class VerificationModule {}
