# Vedic Astrology Calculations

How Vedic astrology (Jyotisha) computes a chart: the actual math, not the
interpretation. Companion document: [Western Astrology Calculations](./western-astrology-calculations.md)
— the two systems share the entire astronomy pipeline and differ in one
load-bearing choice (which point in the sky is 0° Aries).

> Vedic astrology is **sidereal** — it anchors the zodiac to the fixed stars,
> and additionally layers nakshatras, whole-sign houses, and time-period
> (dasha) systems on top. It uses **nine grahas** (no Uranus/Neptune/Pluto).
>
> All positions in the worked example were computed with Swiss Ephemeris and
> rounded to the arcminute.

## 1. Inputs and time normalization

Identical to the Western pipeline — four inputs, UTC conversion, Julian Day:

| Input | Example |
|---|---|
| Date | 1990-11-15 |
| Clock time + timezone | 09:00 IST (UTC+5:30) → **03:30 UTC** |
| Latitude φ | 28.6139° N (New Delhi) |
| Longitude λ | 77.2090° E |

Julian Day (Meeus formula, see companion doc §1): JD(UT) = **2448210.645833**.

## 2. Planetary positions (the ephemeris step)

Identical to the Western pipeline: VSOP87/ELP2000 (or Swiss Ephemeris / JPL)
gives each body's **apparent geocentric ecliptic longitude** λ, with
light-time, aberration, nutation and ΔT corrections. **Vedic astrology starts
from these tropical longitudes and then converts them.**

## 3. The sidereal zodiac and the ayanamsa

The defining choice of Jyotisha: the zodiac is anchored to the **fixed
stars**, not to the seasons. The conversion is one subtraction:

```
sidereal longitude = tropical longitude − ayanamsa     (mod 360°)
```

The **ayanamsa** is the angle between the vernal-equinox point and the fixed
sidereal zero point. Because Earth's axis precesses at **50.29″ per year**
(1° per ≈ 71.6 years), the ayanamsa grows steadily: it was ~23°44′ at our
1990 example moment and is **≈ 24°14′ in late 2026**.

The standard is the **Lahiri (Chitrapaksha) ayanamsa** — official since
India's Calendar Reform Committee (1956), defined so the star Spica (Chitra)
sits at exactly 180°. Other schools differ by fractions of a degree up to
~1.5°:

| Ayanamsa | Value on 2026-09-27 | Notes |
|---|---|---|
| Fagan–Bradley | 25.11° | Western sidereal school |
| **Lahiri** | **24.23° (24°14′)** | De facto standard for Vedic |
| Krishnamurti (KP) | 24.13° | KP system |
| Raman | 22.78° | B.V. Raman school |

## 4. Ascendant and houses

Same formulas as the Western pipeline (repeated here for completeness). With
GMST/RAMC from the Julian Day, ε = true obliquity ≈ 23.44°, φ = latitude:

```
T = (JD - 2451545.0) / 36525
GMST (deg) = 280.46061837 + 360.98564736629 * (JD - 2451545.0)
             + 0.000387933 * T² - T³ / 38710000
LST   = GMST + λ_east                        (mod 360°)
RAMC  = LST
λ_MC  = atan2( sin(RAMC),  cos(RAMC) * cos(ε) )
λ_Asc = atan2( cos(RAMC),  -( sin(RAMC) * cos(ε) + tan(φ) * sin(ε) ) )
```

Then the ascendant is converted to sidereal like any other longitude.

**Worked numbers** (the example birth): RAMC = 183.6755°, ε = 23.4418° →
Asc = 260.9917° tropical = 21°00′ Sagittarius; minus ayanamsa 23.7296° →
**27°16′ Scorpio sidereal**.

**Houses are whole-sign (bhava):** the sign containing the sidereal ascendant
is the 1st house, and every following sign is the next house — no cusp math
at all in the standard rashi chart. (Some jyotishis also cast the *bhava
chalit*, a Sripati/Placidus-like cusp overlay, for finer placement.)

## 5. The nine grahas

| Graha | Body | Note |
|---|---|---|
| Surya | Sun | |
| Chandra | Moon | |
| Mangala | Mars | |
| Budha | Mercury | |
| Guru | Jupiter | |
| Shukra | Venus | |
| Shani | Saturn | |
| Rahu | Moon's ascending node | Mean or true node (differ ≤ ~1.5°) |
| Ketu | Moon's descending node | **Exactly Rahu + 180°** |

Rahu and Ketu are the two points where the Moon's orbit crosses the ecliptic.
Because Ketu is by definition exactly opposite Rahu, **the chart has only 8
independent positions**, and no ephemeris call is needed for Ketu. Outer
planets are not part of the classical framework.

## 6. Nakshatras and padas

On top of the 12 signs, the sidereal zodiac is divided into **27 nakshatras**
of exactly 13°20′ each (27 × 13°20′ = 360°). Each nakshatra has a lord from a
fixed 9-lord cycle:

```
lord(nakshatra i) = [Ketu, Venus, Sun, Moon, Mars, Rahu, Jupiter, Saturn, Mercury][(i-1) mod 9]
```

(so Ashwini 1–9 run Ketu→Mercury, Magha 10–18 run Ketu→Mercury again, Mula
19–27 the same cycle a third time). Each nakshatra splits into 4 **padas** of
3°20′ — **108 padas** across the zodiac — and each pada maps to one navamsa
sign (see §8).

Example: Moon sidereal = 184.1663° → 184.1663 / 13.3333 = nakshatra **14,
Chitra, lord Mars**, within it 81.25% → **pada 4**.

## 7. Vimshottari dasha (planetary periods)

The famous 120-year timing system is pure arithmetic on the Moon's sidereal
position. The nakshatra the Moon occupies names the **mahadasha lord ruling
at birth**, and the fraction of the nakshatra already traversed determines
how much of that lord's period remains:

```
i       = floor( λ_moon_sidereal / 13.3333° )
lord    = nakshatra lord(i)
f       = (λ - i * 13.3333°) / 13.3333°        (elapsed fraction)
balance = (1 - f) * years(lord)
```

| Lord | Years |
|---|---|
| Ketu | 7 |
| Venus | 20 |
| Sun | 6 |
| Moon | 10 |
| Mars | 7 |
| Rahu | 18 |
| Jupiter | 16 |
| Saturn | 19 |
| Mercury | 17 |
| **Total** | **120** |

After the balance runs out, periods follow the fixed cycle above in sequence.
(Convention note: "years" are usually 365.2425-day solar years in software;
some schools use 360-day years, which shifts dates by ~1.5%.)

**Worked numbers** (example birth): Moon in Chitra (lord Mars), f = 81.25% →
balance = 0.1875 × 7 = **1.31 years of Mars**, ending ≈ March 1992; then Rahu
18 y (≈ to March 2010), Jupiter 16 y (≈ to March 2026), Saturn 19 y.

## 8. Vargas (divisional charts)

Vedic astrology re-projects the same longitudes into divisional charts. The
most used is the **navamsa (D9)**: each 3°20′ pada of the zodiac maps to one
sign, cycling continuously from Aries:

```
navamsa sign = floor( λ_sidereal / 3.3333° )  mod 12
```

(Example: 4°10′ Libra Moon = absolute 184.17° → floor(55.25) mod 12 = 7 →
Scorpio navamsa.) Other vargas (D2 hora, D3 drekkana, D10 dashamsa, … D60)
use similar fixed-division rules. Vargas are pure functions of the sidereal
longitude — they add interpretive layers, not new degrees of freedom.

## 9. Worked example — full sidereal chart

**1990-11-15, 09:00 IST, New Delhi** (JD 2448210.645833), Lahiri ayanamsa
23.7296°:

| Graha | Tropical | Sidereal | Sign | House (whole-sign) |
|---|---|---|---|---|
| Sun | 22°30′ Scorpio | 28°46′ Libra | Libra | 12 |
| Moon | 27°54′ Libra | 4°10′ Libra | Libra | 12 |
| Mercury | 6°15′ Sagittarius | 12°31′ Scorpio | Scorpio | 1 |
| Venus | 25°54′ Scorpio | 2°10′ Scorpio | Scorpio | 1 |
| Mars | 10°01′ Gemini | 16°17′ Taurus | Taurus | 7 |
| Jupiter | 13°14′ Leo | 19°30′ Cancer | Cancer | 9 |
| Saturn | 20°55′ Capricorn | 27°12′ Sagittarius | Sagittarius | 2 |
| Rahu (true) | 0°45′ Aquarius | 7°01′ Capricorn | Capricorn | 3 |
| Ketu (true) | 0°45′ Leo | 7°01′ Cancer | Cancer | 9 |
| **Asc** | 21°00′ Sagittarius | **27°16′ Scorpio** | Scorpio | 1 |

Note the teaching point in row 1: the Sun moves from Scorpio (Western) to
Libra (Vedic) — same body, same ephemeris, ~24° shift from the ayanamsa.

Nakshatra/dasha summary for the chart:

| Graha | Nakshatra | Pada | Lord |
|---|---|---|---|
| Sun (28°46′ Libra = 208.77°) | Vishakha (#16) | 3 | Jupiter |
| Moon (4°10′ Libra = 184.17°) | Chitra (#14) | 4 | Mars |

Vimshottari at birth: Mars balance 1.31 y → Rahu 18 y → Jupiter 16 y →
Saturn 19 y.

## 10. How many combinations are there?

Same logic as the Western case, with Vedic's own discretizations. This
section is also available in machine-readable form — every domain enumerated
and the counts expressed as data — in
[vedic-astrology-combinations.yaml](./vedic-astrology-combinations.yaml).

**Continuous reality — infinite.** Sidereal longitudes are real numbers.

**Discretized — finite and exactly countable.** The pada (3°20′ grid) is the
finest standard grid, so it is the natural "pixel":

| Discretization level | Count for the Vedic chart |
|---|---|
| Sign placements: 8 independent grahas (Rahu fixes Ketu) × 12 ascendants | 12⁸ × 12 = 12⁹ = **5.16 × 10⁹** |
| Nakshatra placements (27) | 27⁸ × 27 = 27⁹ = 7.63 × 10¹² |
| Pada placements (108) — the finest standard grid | 108⁸ × 108 = 108⁹ = **2.0 × 10¹⁸** |
| 1°-of-longitude bins (9 bodies incl. Ketu) | 360⁹ = 1.02 × 10²³ |
| Distinct Vimshottari dasha timelines | 27 (one per Moon nakshatra; continuous balance within) |

Note that houses and vargas add nothing to these counts: whole-sign houses
are derived from the ascendant, and vargas are functions of the longitude.

**Does the count grow as time passes? No.** Same two-part answer as the
Western doc:

1. The state space is fixed by definitions (9 grahas, 12 signs, 27
   nakshatras). Planets cycle through it; realized charts accumulate but stay
   astronomically far below the space.
2. Time does change the **ayanamsa** (+50.29″ per year), which slowly slides
   the tropical↔sidereal mapping — but that moves every planet together; the
   *count* of possible charts is untouched. The count only jumped
   historically when the framework admitted new bodies (e.g., if one adopted
   the outer planets, the sign-level space would multiply by 12).

So: **exact number — yes, per discretization choice (5.16 × 10⁹ at sign
level, 2 × 10¹⁸ at pada level); universal number — no; growth with time — no.**

## 11. Implementation notes

- **Swiss Ephemeris** supports all of this natively:
  `swe_set_sid_mode(SE_SIDM_LAHIRI)` + `FLG_SIDEREAL` flag returns sidereal
  positions directly; `swe_get_ayanamsa_ut` exposes the ayanamsa; other
  schools are constants (`SE_SIDM_FAGAN_BRADLEY`, `SE_SIDM_RAMAN`,
  `SE_SIDM_KRISHNAMURTI`). Prototype with `pyswisseph`; call via CGo from Go.
- Choose **mean vs true node** deliberately — Rahu differs by up to ~1.5°
  between the two, which can flip a nakshatra or dasha lord near a boundary.
- Same UT/ΔT and `atan2` quadrant cautions as the Western pipeline.
- In this repo, a future feature would live in `server/internal/<feature>/`
  (handler + service + types) wired into `server/router/dependencies.go`,
  following the existing `example` module pattern.

## 12. References and accuracy

Astronomical sources the calculations in this document trace back to:

- **Swiss Ephemeris** (Astrodienst, `github.com/aloistr/swisseph`) — produced
  every position, ayanamsa, and sidereal conversion in the worked example
  (v2.10, Moshier fileless mode, `FLG_SIDEREAL` with `SE_SIDM_LAHIRI`). It
  is the de facto standard library in commercial astrology software and is
  itself derived from **JPL's Development Ephemerides** (DE431) — the same
  NASA-fitted solar-system solution built from radar ranging, spacecraft
  telemetry, and VLBI observations. Moshier mode agrees with JPL to around
  the arcsecond for the modern era — far finer than the arcminute shown here.
- **J. Meeus, *Astronomical Algorithms*, 2nd ed. (Willmann-Bell, 1998)** —
  the Julian Day and GMST/sidereal-time formulas (§1, §4) and the
  nutation/obliquity models; the standard reference across astronomy
  software (IAU 2000A/2006 precession–nutation underneath).
- **Lahiri (Chitrapaksha) ayanamsa** — defined by India's Calendar Reform
  Committee (1956), anchoring the star Spica (Chitra) at exactly 180°; the
  values in §3 were computed directly from the Swiss Ephemeris
  implementation.
- **Brihat Parashara Hora Shastra** (Parashara) — the classical source for
  the Vimshottari dasha sequence and periods in §7, implemented identically
  by mainstream Vedic software (Jagannatha Hora, Parashara's Light).

Cross-checks performed while writing: the hand-coded MC/Ascendant formulas
in §4 reproduce Swiss Ephemeris `swe_houses` to the arcsecond for the worked
example; the sidereal positions match `FLG_SIDEREAL` output exactly; the
dasha balance was recomputed by hand (Moon 81.25% through Chitra →
0.1875 × 7 = 1.3125 years of Mars); the four ayanamsa values in §3 were each
computed independently.

What is **convention rather than fact**: the ayanamsa school (up to ~1.5°
spread between Lahiri and Raman), mean vs true lunar node (up to ~1.5° on
Rahu), and dasha year length (365.2425-day vs 360-day years) are documented
school choices, not errors — different reputable software legitimately
disagrees on them. And the scope note bears repeating: the *astronomy* here
is exact and independently verifiable (e.g., against astro.com, NASA's
Horizons at ssd.jpl.nasa.gov/horizons, or the free Jagannatha Hora for the
Vedic side); astrology's interpretive claims are outside this document and
are not scientifically established.
