import React, { useEffect } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import Svg, { Circle } from 'react-native-svg';
import Animated, { useSharedValue, useAnimatedProps, withTiming } from 'react-native-reanimated';
import { light, dark, oled, spacing, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

const AnimatedCircle = Animated.createAnimatedComponent(Circle);

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface CalorieRingProps {
  consumed: number;
  target: number;
  size?: number;
  strokeWidth?: number;
}

export default function CalorieRing({ consumed, target, size = 180, strokeWidth = 14 }: CalorieRingProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = target > 0 ? Math.min(consumed / target, 1) : 0;

  const animatedProgress = useSharedValue(0);

  useEffect(() => {
    animatedProgress.value = withTiming(progress, { duration: 800 });
  }, [progress]);

  const animatedProps = useAnimatedProps(() => ({
    strokeDashoffset: circumference * (1 - animatedProgress.value),
  }));

  const remaining = Math.max(target - consumed, 0);

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
          <AnimatedCircle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={colors.calorieRing}
            strokeWidth={strokeWidth}
            fill="none"
            strokeDasharray={circumference}
            animatedProps={animatedProps}
            strokeLinecap="round"
            rotation="-90"
            origin={`${size / 2}, ${size / 2}`}
          />
        </Svg>
        <View style={[styles.centerLabel, { width: size, height: size }]}>
          <Text style={[styles.remaining, { color: colors.textPrimary }]}>{remaining}</Text>
          <Text style={[styles.remainingSub, { color: colors.textSecondary }]}>remaining</Text>
        </View>
      </View>
      <View style={styles.statsRow}>
        <View style={styles.statItem}>
          <Text style={[styles.statValue, { color: colors.textPrimary }]}>{consumed}</Text>
          <Text style={[styles.statLabel, { color: colors.textTertiary }]}>eaten</Text>
        </View>
        <View style={[styles.statDivider, { backgroundColor: colors.border }]} />
        <View style={styles.statItem}>
          <Text style={[styles.statValue, { color: colors.textPrimary }]}>{target}</Text>
          <Text style={[styles.statLabel, { color: colors.textTertiary }]}>target</Text>
        </View>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center' as const,
  },
  ringWrap: {
    position: 'relative' as const,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
  },
  centerLabel: {
    position: 'absolute' as const,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
  },
  remaining: {
    fontSize: 32,
    fontWeight: '700',
  },
  remainingSub: {
    fontSize: 13,
    marginTop: -2,
  },
  statsRow: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    marginTop: spacing.md,
    gap: spacing.lg,
  },
  statItem: {
    alignItems: 'center' as const,
  },
  statValue: {
    fontSize: fontSizes.lg,
    fontWeight: '700',
  },
  statLabel: {
    fontSize: 12,
    marginTop: 2,
  },
  statDivider: {
    width: 1,
    height: 24,
  },
});
