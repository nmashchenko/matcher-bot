import { Injectable, Logger } from '@nestjs/common';
import { VerificationStatus } from '../../prisma/generated/client';
import { PrismaService } from '../prisma/prisma.service';
import { GeocodingService } from './geocoding.service';

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
