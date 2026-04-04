import { light, dark, oled } from '@joules/ui';
import type { ColorPalette } from '@joules/ui';

export default {
  light: {
    text: light.textPrimary,
    background: light.background,
    tint: light.primary,
    tabIconDefault: light.textTertiary,
    tabIconSelected: light.primary,
  },
  dark: {
    text: dark.textPrimary,
    background: dark.background,
    tint: dark.primary,
    tabIconDefault: dark.textTertiary,
    tabIconSelected: dark.primary,
  },
  oled: {
    text: oled.textPrimary,
    background: oled.background,
    tint: oled.primary,
    tabIconDefault: oled.textTertiary,
    tabIconSelected: oled.primary,
  },
};
