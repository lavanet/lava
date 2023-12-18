export enum Geolocation {
  GLS = 0,
  USC = 1,
  EU = 2,
  USE = 4,
  USW = 8,
  AF = 16,
  AS = 32,
  AU = 64,
  GL = 65535,
}

const GeolocationName: Record<number, string> = {
  0: "GLS",
  1: "USC",
  2: "EU",
  4: "USE",
  8: "USW",
  16: "AF",
  32: "AS",
  64: "AU",
  65535: "GL",
};

const GeolocationValue: Record<string, number> = {
  GLS: 0,
  USC: 1,
  EU: 2,
  USE: 4,
  USW: 8,
  AF: 16,
  AS: 32,
  AU: 64,
  GL: 65535,
};

type GeoLatencyMap = Record<Geolocation, Record<Geolocation, number>>;
const maxGeoLatency = 10000;
const GEO_LATENCY_MAP: GeoLatencyMap = {
  [Geolocation.AS]: {
    [Geolocation.AU]: 146,
    [Geolocation.EU]: 155,
  },
  [Geolocation.USE]: {
    [Geolocation.USC]: 42,
    [Geolocation.USW]: 68,
    [Geolocation.EU]: 116,
  },
  [Geolocation.USW]: {
    [Geolocation.USC]: 45,
    [Geolocation.USE]: 68,
  },
  [Geolocation.USC]: {
    [Geolocation.USE]: 42,
    [Geolocation.USW]: 45,
    [Geolocation.EU]: 170,
  },
  [Geolocation.EU]: {
    [Geolocation.USE]: 116,
    [Geolocation.AF]: 138,
    [Geolocation.AS]: 155,
    [Geolocation.USC]: 170,
  },
  [Geolocation.AF]: {
    [Geolocation.EU]: 138,
    [Geolocation.USE]: 203,
    [Geolocation.AS]: 263,
  },
  [Geolocation.AU]: {
    [Geolocation.AS]: 146,
    [Geolocation.USW]: 179,
  },
} as GeoLatencyMap;

export function calcGeoLatency(reqGeo: Geolocation, pGeo: Geolocation): number {
  let minLatency: number = maxGeoLatency;

  if (pGeo === reqGeo) {
    minLatency = 1;
  }

  const inner = GEO_LATENCY_MAP[reqGeo];
  if (inner && inner[pGeo] !== undefined) {
    const latency = inner[pGeo];
    if (latency < minLatency) {
      minLatency = latency;
    }
  }

  return minLatency;
}

export function GeolocationFromString(geolocation: string): Geolocation {
  const parsedValue = parseInt(geolocation, 10);

  if (!isNaN(parsedValue) && GeolocationName.hasOwnProperty(parsedValue)) {
    return parsedValue as Geolocation;
  }

  const upperCaseGeo = geolocation.toUpperCase();
  if (GeolocationValue.hasOwnProperty(upperCaseGeo)) {
    return GeolocationValue[upperCaseGeo] as Geolocation;
  }

  throw new Error(`Invalid Geolocation: ${geolocation}`);
}
