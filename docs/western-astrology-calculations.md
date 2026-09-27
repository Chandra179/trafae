# Western Astrology Calculations

How Western ("Roman") astrology computes a chart: the actual math, not the
interpretation. Companion document: [Vedic Astrology Calculations](./vedic-astrology-calculations.md)
— the two systems share the entire astronomy pipeline and differ in one
load-bearing choice (which point in the sky is 0° Aries).

> **Terminology.** "Roman astrology" usually means the horoscopic tradition
> systematized under the Roman Empire (Hellenistic astrology: Ptolemy,
> Manilius, Firmicus Maternus) as carried into modern practice. That tradition
> is **tropical** — it anchors the zodiac to the seasons, not to the stars.
>
> All positions in the worked example were computed with Swiss Ephemeris and
> rounded to the arcminute.

## 1. Inputs and time normalization

Every calculation starts from four inputs:

| Input | Example |
|---|---|
| Date | 1990-11-15 |
| Clock time + timezone | 09:00 IST (UTC+5:30) → **03:30 UTC** |
| Latitude φ | 28.6139° N (New Delhi) |
| Longitude λ | 77.2090° E |

Time is converted to UTC, then to a **Julian Day** (JD), a continuous day
count that all ephemeris math consumes. For a Gregorian date (Meeus):

```
if month <= 2:  Y = Y - 1;  M = M + 12
A = floor(Y / 100)
B = 2 - A + floor(A / 4)
JD = floor(365.25 * (Y + 4716)) + floor(30.6001 * (M + 1)) + D + B - 1524.5
```

where `D` is the day of month plus the UT fraction of a day. For the example:
JD(UT) = **2448210.645833**.

## 2. Planetary positions (the ephemeris step)

For the JD, an ephemeris theory computes each body's **apparent geocentric
ecliptic longitude** λ (0–360°):

- **Theory:** VSOP87 for the planets, ELP2000 for the Moon (JPL-grade suites
  like DE440 refine this further). These are big truncated Fourier series
  derived from celestial mechanics, not table lookups.
- **Corrections applied on top:** light-time (~8 min for the Sun's rays),
  annual aberration (~20.5″), nutation of Earth's axis, and the UT↔TT offset
  (ΔT, ≈ 57 s in 1990).
- **Output per body:** λ (longitude), β (ecliptic latitude), Δ (distance),
  plus daily speed — negative speed = retrograde.

Everything downstream (signs, houses, aspects) is geometry applied to these
longitudes. **Both Western and Vedic astrology compute this identically.**

## 3. The tropical zodiac

This is the defining choice of Western astrology:

- **0° Aries = the vernal equinox point** — where the ecliptic crosses the
  celestial equator northward each March. The zodiac is anchored to the
  **seasons**: Aries always begins at the spring equinox, Cancer at the
  summer solstice, and so on.
- Earth's axis precesses (wobbles) at **50.29″ per year** — the equinox point
  drifts ~1° every 71.6 years against the fixed stars, a full cycle in
  ~25,772 years. The tropical zodiac **ignores this drift** by design: the
  sign boundaries stay glued to the equinoxes and solstices, even though the
  background constellations slide underneath. (The "Astrological Ages" idea
  comes from this same drift.)

## 4. Ascendant and houses

### Sidereal time → RAMC

The ascendant depends on *when and where* on Earth, expressed as sidereal
time:

```
T = (JD - 2451545.0) / 36525
GMST (deg) = 280.46061837 + 360.98564736629 * (JD - 2451545.0)
             + 0.000387933 * T² - T³ / 38710000
LST = GMST + λ_east          (mod 360°)
RAMC = LST                   (Right Ascension of the Midheaven)
```

### Midheaven and Ascendant

With ε = true obliquity of the ecliptic (≈ 23.44°, slowly decreasing) and φ =
latitude:

```
λ_MC  = atan2( sin(RAMC),  cos(RAMC) * cos(ε) )
λ_Asc = atan2( cos(RAMC),  -( sin(RAMC) * cos(ε) + tan(φ) * sin(ε) ) )
```

The `atan2` quadrant handling is essential — it encodes the fact that the
ascendant must be within 180° "east" of the MC.

**Worked numbers** (the example birth: RAMC = 183.6755°, ε = 23.4418°,
φ = 28.6139°):

- MC = **184.0051°** = 4°00′ Libra
- Asc = **260.9917°** = 21°00′ Sagittarius
  (matches Swiss Ephemeris to the arcsecond)

### House systems

Houses divide the sky between the Ascendant (1st cusp) and MC (10th cusp).
Common systems:

| System | Method | Notes |
|---|---|---|
| **Placidus** | Each diurnal/nocturnal semi-arc divided into three; solved iteratively | Modern default; undefined above ~66° latitude |
| **Koch** | Birth-horizon semi-arc variant | Popular in German-speaking countries |
| **Equal** | Every cusp = Asc + 30° × k | Simplest; used with Uranian astrology |
| **Whole Sign** | The Asc's sign is the 1st house; each sign = next house | The Hellenistic original; also the Vedic standard |
| **Regiomontanus** | Equator split into 30° arcs, projected via house circles | Traditional/horary workhorse |

Placidus cusps for the example chart: 2nd 23°35′ Capricorn, 3rd 29°24′
Aquarius, 4th 4°00′ Aries (IC), 5th 3°33′ Taurus, 6th 28°22′ Taurus, 7th
21°00′ Gemini (Descendant), 8th 23°35′ Cancer, 9th 29°24′ Leo, 10th 4°00′
Libra (MC), 11th 3°33′ Scorpio, 12th 28°22′ Scorpio.

## 5. Bodies on the chart

Standard modern set: **Sun, Moon, Mercury, Venus, Mars, Jupiter, Saturn,
Uranus, Neptune, Pluto** (10 bodies — the Vedic chart uses 9, without the
outers; see the companion doc). Frequently added: Chiron, Lilith, the lunar
nodes, and lots such as the **Part of Fortune**:

```
PoF = Asc + Moon - Sun      (day birth, Sun above horizon)
PoF = Asc + Sun - Moon      (night birth)
```

Example (day birth): 260.99 + 207.90 − 232.50 = 236.39° = 26°23′ Scorpio.

## 6. Aspects

Aspects are angular separations between bodies, each with an allowed **orb**
(tolerance):

| Aspect | Angle | Typical orb |
|---|---|---|
| Conjunction | 0° | 8–10° (luminaries), 4–6° (planets) |
| Sextile | 60° | 4–6° |
| Square | 90° | 5–8° |
| Trine | 120° | 6–8° |
| Opposition | 180° | 6–10° |
| Minor (semisextile 30°, semisquare 45°, sesquiquadrate 135°, quincunx 150°) | — | 1–3° |

With 10 bodies there are C(10,2) = **45 pairs** to check. An aspect is
*applying* if the faster body's separation is shrinking (its daily motion
closes the gap) and *separating* otherwise.

## 7. Worked example — full tropical chart

**1990-11-15, 09:00 IST, New Delhi** (JD 2448210.645833):

| Body | Longitude | Sign position |
|---|---|---|
| Sun | 232.50° | 22°30′ Scorpio |
| Moon | 207.90° | 27°54′ Libra |
| Mercury | 246.25° | 6°15′ Sagittarius |
| Venus | 235.89° | 25°54′ Scorpio |
| Mars | 70.01° | 10°01′ Gemini |
| Jupiter | 133.23° | 13°14′ Leo |
| Saturn | 290.92° | 20°55′ Capricorn |
| Uranus | 277.13° | 7°08′ Capricorn |
| Neptune | 282.53° | 12°32′ Capricorn |
| Pluto | 227.91° | 17°54′ Scorpio |
| **Asc** | 260.99° | **21°00′ Sagittarius** |
| **MC** | 184.01° | **4°00′ Libra** |

Sample aspect calculations:

- Sun–Venus: |232.50 − 235.89| = 3.39° → **conjunction, orb 3°23′**
- Mercury–Mars: |246.25 − 70.01| = 176.24° → **opposition, orb 3°46′**
- Jupiter–Mars: |133.23 − 70.01| = 63.22° → **sextile, orb 3°13′**

## 8. How many combinations are there?

This has a precise answer once you fix what you're counting. Two regimes:

**Continuous reality — infinite.** Longitudes are real numbers, so strictly
there are uncountably infinitely many distinct charts. No software keeps that
precision meaningful beyond ~0.001° anyway.

**Discretized — finite and exactly countable.** Fix a grid and count:

| Discretization level | Count for the Western chart |
|---|---|
| Sign placements of 10 bodies | 12¹⁰ = 6.19 × 10¹⁰ |
| … plus Ascendant and MC | 12¹² = **8.92 × 10¹²** |
| 1°-of-longitude bins (12 bodies) | 360¹² = 4.74 × 10³⁰ |
| … with aspect state per pair (45 pairs, ~7 states each) | × ~10³⁸ more |

**Does the count grow as time passes? No.** The state space is fixed by the
framework's *definitions* (10 bodies, 12 signs, house system). Time only moves
planets around inside a space that never gets bigger. Two things do change:

1. **Realized charts accumulate** — every birth and every moment adds one
   point in the space — but even trillions of events never dent a space of
   10¹²–10³⁰ states.
2. **The count only jumped when the framework itself changed** — when a new
   body was admitted, the state space multiplied by 12 overnight:
   Uranus (1781), Neptune (1846), Pluto (1930), Chiron (1977), Eris (2006).
   That is definitional change, not time growth.

So: **exact number — yes, per discretization choice; universal number — no;
growth with time — no.**

## 9. Implementation notes

- **Swiss Ephemeris** is the de facto standard library (AGPL / commercial):
  `swe_calc_ut` for positions, `swe_houses` for cusps. Its Moshier fallback
  needs no data files and is plenty accurate (~0.1″) for modern dates.
  Prototype in Python with `pyswisseph`; from Go, call it via CGo.
- Pure-Python alternative: `skyfield` (loads JPL SPICE files).
- Practical cautions: always pass UT (never local time) to the ephemeris;
  keep `atan2` quadrants for MC/Asc; remember Placidus breaks near the poles.
- In this repo, a future feature would live in `server/internal/<feature>/`
  (handler + service + types) wired into `server/router/dependencies.go`,
  following the existing `example` module pattern.

## 10. References and accuracy

Astronomical sources the calculations in this document trace back to:

- **Swiss Ephemeris** (Astrodienst, `github.com/aloistr/swisseph`) — produced
  every position in the worked example (v2.10, Moshier fileless mode). It is
  the de facto standard library in commercial astrology software and is
  itself derived from **JPL's Development Ephemerides** (DE431) — the same
  NASA-fitted solar-system solution built from radar ranging, spacecraft
  telemetry, and VLBI observations. Moshier mode agrees with JPL to around
  the arcsecond for the modern era — far finer than the arcminute shown here.
- **J. Meeus, *Astronomical Algorithms*, 2nd ed. (Willmann-Bell, 1998)** —
  the Julian Day formula (§1) and the GMST/sidereal-time formula (§4), plus
  the nutation and obliquity models; the standard reference across astronomy
  software. The underlying precession–nutation theory is the IAU
  2000A/2006 model.
- **VSOP87** (Bretagnon & Francou, Bureau des Longitudes, 1988) and
  **ELP2000** (Chapront-Touzé & Chapront, 1983) — the analytic planetary and
  lunar theories named in §2.

Cross-checks performed while writing: the hand-coded MC/Ascendant formulas
in §4 reproduce Swiss Ephemeris `swe_houses` to the arcsecond for the worked
example; the Julian Day matches `swe_julday` exactly; the aspect orbs in §7
were recomputed from raw longitudes by script.

What is **convention rather than fact**: the house system, aspect orbs, and
the 10-body set are school choices, not measurements — different reputable
software legitimately disagrees on them. And the scope note bears
repeating: the *astronomy* here is exact and independently verifiable
(e.g., against astro.com or NASA's Horizons at ssd.jpl.nasa.gov/horizons);
astrology's interpretive claims are outside this document and are not
scientifically established.
