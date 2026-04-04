import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface MacroBarProps {
  label: string;
  consumed: number;
  target: number;
  color: string;
  unit?: string;
}

export default function MacroBar({ label, consumed, target, color, unit = 'g' }: MacroBarProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const progress = target > 0 ? Math.min(consumed / target, 1) : 0;

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={[styles.label, { color: colors.textPrimary }]}>{label}</Text>
        <Text style={[styles.values, { color: colors.textSecondary }]}>
          {Math.round(consumed)}{unit} / {Math.round(target)}{unit}
        </Text>
      </View>
      <View style={[styles.track, { backgroundColor: colors.surfaceElevated }]}>
        <View
          style={[
            styles.fill,
            { width: `${progress * 100}%`, backgroundColor: color },
          ]}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: spacing.sm,
  },
  header: {
    flexDirection: 'row' as const,
    justifyContent: 'space-between' as const,
    marginBottom: 4,
  },
  label: {
    fontSize: 13,
    fontWeight: '600',
  },
  values: {
    fontSize: 12,
  },
  track: {
    height: 8,
    borderRadius: borderRadius.full,
    overflow: 'hidden' as const,
  },
  fill: {
    height: '100%' as const,
    borderRadius: borderRadius.full,
    minWidth: 4,
  },
});
