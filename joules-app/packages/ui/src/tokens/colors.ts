export type ColorToken =
  | 'background'
  | 'surface'
  | 'surfaceElevated'
  | 'textPrimary'
  | 'textSecondary'
  | 'textTertiary'
  | 'border'
  | 'primary'
  | 'primaryHover'
  | 'success'
  | 'warning'
  | 'error'
  | 'ring'
  | 'macroProtein'
  | 'macroCarbs'
  | 'macroFat'
  | 'macroFiber'
  | 'calorieRing'
  | 'calorieRingBg';

export type ColorPalette = Record<ColorToken, string>;

export const light: ColorPalette = {
  background: '#f8fafc',
  surface: '#ffffff',
  surfaceElevated: '#f1f5f9',
  textPrimary: '#0f172a',
  textSecondary: '#475569',
  textTertiary: '#94a3b8',
  border: '#e2e8f0',
  primary: '#2563eb',
  primaryHover: '#1d4ed8',
  success: '#16a34a',
  warning: '#d97706',
  error: '#dc2626',
  ring: '#3b82f6',
  macroProtein: '#ef4444',
  macroCarbs: '#f59e0b',
  macroFat: '#8b5cf6',
  macroFiber: '#22c55e',
  calorieRing: '#3b82f6',
  calorieRingBg: '#e2e8f0',
};

export const dark: ColorPalette = {
  background: '#0f172a',
  surface: '#1e293b',
  surfaceElevated: '#334155',
  textPrimary: '#f8fafc',
  textSecondary: '#cbd5e1',
  textTertiary: '#64748b',
  border: '#334155',
  primary: '#3b82f6',
  primaryHover: '#60a5fa',
  success: '#22c55e',
  warning: '#f59e0b',
  error: '#ef4444',
  ring: '#60a5fa',
  macroProtein: '#f87171',
  macroCarbs: '#fbbf24',
  macroFat: '#a78bfa',
  macroFiber: '#4ade80',
  calorieRing: '#60a5fa',
  calorieRingBg: '#334155',
};

export const oled: ColorPalette = {
  background: '#000000',
  surface: '#0a0a0a',
  surfaceElevated: '#171717',
  textPrimary: '#fafafa',
  textSecondary: '#a1a1aa',
  textTertiary: '#52525b',
  border: '#1e1e1e',
  primary: '#3b82f6',
  primaryHover: '#60a5fa',
  success: '#22c55e',
  warning: '#f59e0b',
  error: '#ef4444',
  ring: '#60a5fa',
  macroProtein: '#f87171',
  macroCarbs: '#fbbf24',
  macroFat: '#a78bfa',
  macroFiber: '#4ade80',
  calorieRing: '#60a5fa',
  calorieRingBg: '#1e1e1e',
};

export const palettes: Record<string, ColorPalette> = {
  light,
  dark,
  oled,
};
