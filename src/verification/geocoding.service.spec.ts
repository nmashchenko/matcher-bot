import { Test, TestingModule } from '@nestjs/testing';
import { GeocodingService } from './geocoding.service';

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
        json: () => ({
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
        json: () => ({
          address: {
            country: 'Germany',
            country_code: 'de',
            state: 'Berlin',
            city: 'Berlin',
          },
        }),
      });

      const result = await service.reverseGeocode(52.52, 13.405);

      expect(result!.isUSA).toBe(false);
    });

    it('should return null when API fails', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 500 });

      const result = await service.reverseGeocode(0, 0);

      expect(result).toBeNull();
    });
  });
});
