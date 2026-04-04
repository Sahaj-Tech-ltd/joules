import React, { useState } from 'react';
import { View, Text, StyleSheet, Pressable, TextInput } from 'react-native';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

const formatNumber = (n: number) => n.toLocaleString('en-US');

interface StepsWidgetProps {
  steps: number;
  goal?: number;
  source?: string;
  onLog?: (steps: number) => void;
}

export default function StepsWidget({ steps, goal = 10000, source = 'Manual', onLog }: StepsWidgetProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const progress = goal > 0 ? Math.min(steps / goal, 1) : 0;
  const [showInput, setShowInput] = useState(false);
  const [inputValue, setInputValue] = useState('');

  const handleAdd = () => {
    const value = parseInt(inputValue, 10);
    if (isNaN(value) || value <= 0) return;
    onLog?.(value);
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    setInputValue('');
    setShowInput(false);
  };

  const sourceLabel = source === 'Health Kit' ? 'Health' : source;

  return (
    <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
      <View style={styles.header}>
        <Text style={[styles.label, { color: colors.textPrimary }]}>Steps</Text>
        <View style={[styles.sourceBadge, { backgroundColor: colors.surfaceElevated }]}>
          <Text style={[styles.sourceText, { color: colors.textTertiary }]}>{sourceLabel}</Text>
        </View>
      </View>

      <View style={styles.countSection}>
        <Text style={[styles.count, { color: colors.textPrimary }]}>{formatNumber(steps)}</Text>
        <Text style={[styles.goalText, { color: colors.textSecondary }]}>
          / {formatNumber(goal)}
        </Text>
      </View>

      <View style={[styles.progressBg, { backgroundColor: colors.border }]}>
        <View
          style={[
            styles.progressFill,
            { backgroundColor: colors.primary, width: `${progress * 100}%` },
          ]}
        />
      </View>

      {showInput ? (
        <View style={styles.inputRow}>
          <TextInput
            style={[
              styles.input,
              {
                color: colors.textPrimary,
                borderColor: colors.border,
                backgroundColor: colors.surfaceElevated,
              },
            ]}
            value={inputValue}
            onChangeText={setInputValue}
            placeholder="Number of steps"
            placeholderTextColor={colors.textTertiary}
            keyboardType="number-pad"
            returnKeyType="done"
            autoFocus
          />
          <Pressable
            style={[styles.addBtn, { backgroundColor: colors.primary }]}
            onPress={handleAdd}
          >
            <Text style={styles.addBtnText}>Add</Text>
          </Pressable>
          <Pressable
            style={[styles.cancelBtn, { borderColor: colors.border }]}
            onPress={() => {
              setInputValue('');
              setShowInput(false);
            }}
          >
            <Text style={[styles.cancelBtnText, { color: colors.textSecondary }]}>Cancel</Text>
          </Pressable>
        </View>
      ) : (
        <Pressable
          style={[styles.logBtn, { borderColor: colors.primary }]}
          onPress={() => setShowInput(true)}
        >
          <Text style={[styles.logBtnText, { color: colors.primary }]}>Log Steps</Text>
        </Pressable>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    padding: spacing.lg,
    gap: spacing.sm,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  label: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  sourceBadge: {
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
    borderRadius: borderRadius.full,
  },
  sourceText: {
    fontSize: 11,
    fontWeight: '500',
  },
  countSection: {
    alignItems: 'center',
    marginTop: spacing.xs,
    marginBottom: spacing.xs,
  },
  count: {
    fontSize: fontSizes['2xl'],
    fontWeight: '700',
  },
  goalText: {
    fontSize: fontSizes.sm,
    marginTop: 2,
  },
  progressBg: {
    height: 6,
    borderRadius: 3,
    overflow: 'hidden',
    marginTop: spacing.xs,
  },
  progressFill: {
    height: 6,
    borderRadius: 3,
  },
  logBtn: {
    borderWidth: 1,
    borderRadius: borderRadius.md,
    paddingVertical: spacing.sm,
    alignItems: 'center',
    marginTop: spacing.xs,
  },
  logBtnText: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
    marginTop: spacing.xs,
  },
  input: {
    flex: 1,
    borderWidth: 1,
    borderRadius: borderRadius.md,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.sm,
    fontSize: fontSizes.sm,
  },
  addBtn: {
    borderRadius: borderRadius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
  },
  addBtnText: {
    color: '#ffffff',
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  cancelBtn: {
    borderWidth: 1,
    borderRadius: borderRadius.md,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
  },
  cancelBtnText: {
    fontSize: fontSizes.sm,
    fontWeight: '500',
  },
});
