// Mock Prisma generated client to avoid ESM import.meta.url issues in Jest
jest.mock('../generated/prisma/client', () => ({
  VerificationStatus: {
    PENDING: 'PENDING',
    VERIFIED: 'VERIFIED',
    UNVERIFIED: 'UNVERIFIED',
    REJECTED: 'REJECTED',
  },
  PrismaClient: jest.fn(),
}));

// Mock PrismaService to avoid transitive Prisma client import
jest.mock('../prisma/prisma.service', () => ({
  PrismaService: jest.fn(),
}));

import { Test, TestingModule } from '@nestjs/testing';
import { VerificationService } from './verification.service';
import { PrismaService } from '../prisma/prisma.service';
import { GeocodingService } from './geocoding.service';

// Mirror Prisma enum for testing
const VerificationStatus = {
  PENDING: 'PENDING',
  VERIFIED: 'VERIFIED',
  UNVERIFIED: 'UNVERIFIED',
  REJECTED: 'REJECTED',
} as const;

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
      prisma.user.upsert.mockResolvedValue({
        id: '1',
        verificationStatus: VerificationStatus.VERIFIED,
      });

      const result = await service.verifyByLocation(
        BigInt(12345),
        34.05,
        -118.24,
      );

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

      const result = await service.verifyByLocation(BigInt(12345), 52.52, 13.4);

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
