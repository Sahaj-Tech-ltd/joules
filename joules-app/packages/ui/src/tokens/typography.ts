export const fontFamilies = {
  sans: 'Inter',
  mono: 'JetBrainsMono',
} as const;

export const fontSizes = {
  xs: 12,
  sm: 14,
  md: 16,
  lg: 18,
  xl: 20,
  '2xl': 24,
  '3xl': 30,
  '4xl': 36,
} as const;

export type FontSizeKey = keyof typeof fontSizes;

export const fontWeights = {
  regular: '400' as const,
  medium: '500' as const,
  semibold: '600' as const,
  bold: '700' as const,
};

export type FontWeightKey = keyof typeof fontWeights;

export const lineHeights = {
  tight: 1.2,
  normal: 1.5,
  relaxed: 1.75,
} as const;

export type LineHeightKey = keyof typeof lineHeights;
