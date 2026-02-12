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
