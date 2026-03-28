// Unit conversion utilities for Joules.
// The DB always stores metric: cm, kg, kcal.
// These functions convert for display based on user preference.

// ─── Weight ────────────────────────────────────────────────────────────────

export function kgToLbs(kg: number): number {
  return kg * 2.20462;
}

export function lbsToKg(lbs: number): number {
  return lbs / 2.20462;
}

export function formatWeight(kg: number, unit: 'kg' | 'lbs'): string {
  if (unit === 'lbs') return `${kgToLbs(kg).toFixed(1)} lbs`;
  return `${kg.toFixed(1)} kg`;
}

export function displayWeight(kg: number, unit: 'kg' | 'lbs'): number {
  return unit === 'lbs' ? parseFloat(kgToLbs(kg).toFixed(1)) : parseFloat(kg.toFixed(1));
}

export function inputToKg(value: number, unit: 'kg' | 'lbs'): number {
  return unit === 'lbs' ? lbsToKg(value) : value;
}

// ─── Height ────────────────────────────────────────────────────────────────

export function cmToFtIn(cm: number): { ft: number; inches: number } {
  const totalInches = cm / 2.54;
  return { ft: Math.floor(totalInches / 12), inches: Math.round(totalInches % 12) };
}

export function ftInToCm(ft: number, inches: number): number {
  return (ft * 12 + inches) * 2.54;
}

export function formatHeight(cm: number, unit: 'cm' | 'ft'): string {
  if (unit === 'ft') {
    const { ft, inches } = cmToFtIn(cm);
    return `${ft}'${inches}"`;
  }
  return `${Math.round(cm)} cm`;
}

export function inputToCm(value: number, unit: 'cm' | 'ft', inchesValue = 0): number {
  return unit === 'ft' ? ftInToCm(value, inchesValue) : value;
}

// ─── Energy ────────────────────────────────────────────────────────────────

export function kcalToKj(kcal: number): number {
  return kcal * 4.184;
}

export function kjToKcal(kj: number): number {
  return kj / 4.184;
}

export function formatEnergy(kcal: number, unit: 'kcal' | 'kJ'): string {
  if (unit === 'kJ') return `${Math.round(kcalToKj(kcal))} kJ`;
  return `${Math.round(kcal)} kcal`;
}

export function displayEnergy(kcal: number, unit: 'kcal' | 'kJ'): number {
  return unit === 'kJ' ? Math.round(kcalToKj(kcal)) : Math.round(kcal);
}

export function energyLabel(unit: 'kcal' | 'kJ'): string {
  return unit;
}

// ─── Preference types ──────────────────────────────────────────────────────

export interface UnitPrefs {
  height_unit: 'cm' | 'ft';
  weight_unit: 'kg' | 'lbs';
  energy_unit: 'kcal' | 'kJ';
}

export const defaultUnits: UnitPrefs = {
  height_unit: 'cm',
  weight_unit: 'kg',
  energy_unit: 'kcal',
};
