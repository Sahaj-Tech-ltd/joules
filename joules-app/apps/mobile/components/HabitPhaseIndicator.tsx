import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

const phaseLabels: Record<string, string> = {
  scaffolding: 'Building Habits',
  identity_building: 'Identity Building',
  intrinsic: 'Strengthening',
  maintenance: 'Thriving',
};

const phaseColors: Record<string, string> = {
  scaffolding: '#3b82f6',
  identity_building: '#8b5cf6',
  intrinsic: '#f59e0b',
  maintenance: '#22c55e',
};

interface HabitPhaseIndicatorProps {
  phase: string;
  totalDays: number;
}

export default function HabitPhaseIndicator({ phase, totalDays }: HabitPhaseIndicatorProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const label = phaseLabels[phase] ?? phase;
  const color = phaseColors[phase] ?? colors.primary;

  return (
    <View style={[styles.badge, { backgroundColor: `${color}18`, borderColor: `${color}40` }]}>
      <Text style={[styles.text, { color }]}>
        Day {totalDays} · {label}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    paddingHorizontal: spacing.sm + 2,
    paddingVertical: spacing.xs + 1,
    borderRadius: borderRadius.full,
    borderWidth: 1,
  },
  text: {
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 0.3,
  },
});
