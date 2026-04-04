import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import Svg, { Circle } from 'react-native-svg';
import { light, dark, oled, spacing, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface ConsistencyRingProps {
  percentage: number;
  graceUsed: number;
  graceMax: number;
}

export default function ConsistencyRing({ percentage, graceUsed, graceMax }: ConsistencyRingProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const size = 48;
  const strokeWidth = 5;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = Math.min(percentage / 100, 1);

  return (
    <View style={styles.container}>
      <View style={styles.ringWrap}>
        <Svg width={size} height={size}>
          <Circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={colors.calorieRingBg}
            strokeWidth={strokeWidth}
            fill="none"
          />
          <Circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={colors.success}
            strokeWidth={strokeWidth}
            fill="none"
            strokeDasharray={circumference}
            strokeDashoffset={circumference * (1 - progress)}
            strokeLinecap="round"
            rotation="-90"
            origin={`${size / 2}, ${size / 2}`}
          />
        </Svg>
        <Text style={[styles.pctText, { color: colors.textPrimary }]}>
          {Math.round(percentage)}%
        </Text>
      </View>
      <View style={styles.info}>
        <Text style={[styles.label, { color: colors.textSecondary }]}>Consistency</Text>
        <Text style={[styles.grace, { color: colors.textTertiary }]}>
          {graceMax - graceUsed} grace days left
        </Text>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    gap: spacing.sm,
  },
  ringWrap: {
    position: 'relative' as const,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
  },
  pctText: {
    position: 'absolute' as const,
    fontSize: 11,
    fontWeight: '700',
  },
  info: {
    flex: 1,
  },
  label: {
    fontSize: 13,
    fontWeight: '600',
  },
  grace: {
    fontSize: 11,
    marginTop: 1,
  },
});
