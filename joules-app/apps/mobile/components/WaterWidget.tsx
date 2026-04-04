import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, Pressable, TextInput } from 'react-native';
import Svg, { Circle } from 'react-native-svg';
import Animated, { useSharedValue, useAnimatedProps, withTiming } from 'react-native-reanimated';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

const AnimatedCircle = Animated.createAnimatedComponent(Circle);

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface WaterWidgetProps {
  consumed: number;
  target?: number;
  onLog?: (amountMl: number) => void;
}

export default function WaterWidget({ consumed, target = 2500, onLog }: WaterWidgetProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const size = 100;
  const strokeWidth = 8;
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = target > 0 ? Math.min(consumed / target, 1) : 0;

  const animatedProgress = useSharedValue(0);

  useEffect(() => {
    animatedProgress.value = withTiming(progress, { duration: 600 });
  }, [progress]);

  const animatedProps = useAnimatedProps(() => ({
    strokeDashoffset: circumference * (1 - animatedProgress.value),
  }));

  const [showCustom, setShowCustom] = useState(false);
  const [customAmount, setCustomAmount] = useState('');

  function formatMl(n: number): string {
    return n.toLocaleString();
  }

  function handleLog(amount: number) {
    if (amount <= 0) return;
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    if (onLog) {
      onLog(amount);
    }
  }

  function handleCustomAdd() {
    const parsed = parseInt(customAmount, 10);
    if (!isNaN(parsed) && parsed > 0) {
      handleLog(parsed);
      setCustomAmount('');
      setShowCustom(false);
    }
  }

  const ringBgColor = colors.primary + '30';

  return (
    <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
      <View style={styles.header}>
        <Text style={[styles.title, { color: colors.textPrimary }]}>Water</Text>
        <Text style={[styles.targetText, { color: colors.textSecondary }]}>
          {formatMl(target)} ml
        </Text>
      </View>

      <View style={styles.ringWrap}>
        <Svg width={size} height={size}>
          <Circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={ringBgColor}
            strokeWidth={strokeWidth}
            fill="none"
          />
          <AnimatedCircle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={colors.primary}
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
          <Text style={[styles.consumedText, { color: colors.textPrimary }]}>
            {formatMl(consumed)}
          </Text>
          <Text style={[styles.unitText, { color: colors.textSecondary }]}>ml</Text>
        </View>
      </View>

      <View style={styles.presets}>
        <Pressable
          style={[styles.presetBtn, { backgroundColor: colors.primary }]}
          onPress={() => handleLog(250)}
        >
          <Text style={styles.presetLabel}>Glass</Text>
          <Text style={styles.presetSub}>250ml</Text>
        </Pressable>
        <Pressable
          style={[styles.presetBtn, { backgroundColor: colors.primary }]}
          onPress={() => handleLog(500)}
        >
          <Text style={styles.presetLabel}>Bottle</Text>
          <Text style={styles.presetSub}>500ml</Text>
        </Pressable>
        <Pressable
          style={[styles.presetBtn, showCustom ? { backgroundColor: colors.primaryHover } : { backgroundColor: colors.primary }]}
          onPress={() => setShowCustom((v) => !v)}
        >
          <Text style={styles.presetLabel}>Custom</Text>
        </Pressable>
      </View>

      {showCustom && (
        <View style={styles.customRow}>
          <TextInput
            style={[styles.customInput, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.surfaceElevated }]}
            keyboardType="number-pad"
            placeholder="0"
            placeholderTextColor={colors.textTertiary}
            value={customAmount}
            onChangeText={setCustomAmount}
            returnKeyType="done"
          />
          <Text style={[styles.customSuffix, { color: colors.textSecondary }]}>ml</Text>
          <Pressable
            style={[styles.addBtn, { backgroundColor: colors.primary }]}
            onPress={handleCustomAdd}
          >
            <Text style={styles.addBtnText}>Add</Text>
          </Pressable>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    padding: spacing.lg,
  },
  header: {
    flexDirection: 'row' as const,
    justifyContent: 'space-between' as const,
    alignItems: 'center' as const,
    marginBottom: spacing.md,
  },
  title: {
    fontSize: fontSizes.lg,
    fontWeight: '700',
  },
  targetText: {
    fontSize: 13,
  },
  ringWrap: {
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    alignSelf: 'center' as const,
    position: 'relative' as const,
  },
  centerLabel: {
    position: 'absolute' as const,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
  },
  consumedText: {
    fontSize: 20,
    fontWeight: '700',
  },
  unitText: {
    fontSize: 11,
    marginTop: -2,
  },
  presets: {
    flexDirection: 'row' as const,
    gap: spacing.sm,
    marginTop: spacing.md,
    justifyContent: 'center' as const,
  },
  presetBtn: {
    flex: 1,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.xs,
    borderRadius: borderRadius.full,
    alignItems: 'center' as const,
  },
  presetLabel: {
    color: '#ffffff',
    fontSize: 13,
    fontWeight: '600',
  },
  presetSub: {
    color: 'rgba(255,255,255,0.8)',
    fontSize: 10,
  },
  customRow: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    gap: spacing.sm,
    marginTop: spacing.sm,
  },
  customInput: {
    flex: 1,
    borderWidth: 1,
    borderRadius: borderRadius.md,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    fontSize: 14,
  },
  customSuffix: {
    fontSize: 13,
  },
  addBtn: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.full,
  },
  addBtnText: {
    color: '#ffffff',
    fontSize: 13,
    fontWeight: '600',
  },
});
