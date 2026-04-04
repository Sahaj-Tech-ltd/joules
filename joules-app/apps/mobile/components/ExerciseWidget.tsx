import React, { useState } from 'react';
import { View, Text, StyleSheet, Pressable, TextInput, ScrollView } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface ExerciseEntry {
  id: string;
  name: string;
  duration_min: number;
  calories_burned: number;
  timestamp: string;
}

interface ExerciseWidgetProps {
  exercises: ExerciseEntry[];
  onLog?: (exercise: { name: string; duration_min: number; calories_burned?: number }) => void;
}

export default function ExerciseWidget({ exercises, onLog }: ExerciseWidgetProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [duration, setDuration] = useState('');

  const totalCalories = exercises.reduce((sum, e) => sum + e.calories_burned, 0);
  const estimatedCalories = duration ? parseInt(duration, 10) * 5 : 0;

  const handleLog = () => {
    const durationMin = parseInt(duration, 10);
    if (!name.trim() || !durationMin || durationMin <= 0) return;

    const cal = durationMin * 5;
    onLog?.({ name: name.trim(), duration_min: durationMin, calories_burned: cal });

    Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);

    setName('');
    setDuration('');
    setShowForm(false);
  };

  return (
    <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
      <View style={styles.header}>
        <View style={styles.headerLeft}>
          <Ionicons name="fitness-outline" size={18} color={colors.primary} />
          <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Exercise</Text>
        </View>
        <Text style={[styles.headerCalories, { color: colors.textSecondary }]}>
          {totalCalories} cal burned
        </Text>
      </View>

      {exercises.length > 0 ? (
        <ScrollView style={styles.list} scrollEnabled={exercises.length > 3} nestedScrollEnabled>
          {exercises.slice(0, exercises.length > 3 ? exercises.length : 3).map((exercise) => (
            <View key={exercise.id} style={styles.exerciseRow}>
              <Ionicons name="fitness-outline" size={16} color={colors.textTertiary} />
              <Text style={[styles.exerciseName, { color: colors.textPrimary }]} numberOfLines={1}>
                {exercise.name}
              </Text>
              <Text style={[styles.exerciseDetail, { color: colors.textSecondary }]}>
                {exercise.duration_min} min
              </Text>
              <Text style={[styles.exerciseCalories, { color: colors.textSecondary }]}>
                {exercise.calories_burned} cal
              </Text>
            </View>
          ))}
        </ScrollView>
      ) : (
        <Text style={[styles.emptyText, { color: colors.textTertiary }]}>No exercises today</Text>
      )}

      {showForm ? (
        <View style={[styles.form, { borderTopColor: colors.border }]}>
          <TextInput
            style={[styles.input, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
            placeholder="Exercise name"
            placeholderTextColor={colors.textTertiary}
            value={name}
            onChangeText={setName}
            returnKeyType="next"
          />
          <TextInput
            style={[styles.input, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
            placeholder="Duration (min)"
            placeholderTextColor={colors.textTertiary}
            value={duration}
            onChangeText={setDuration}
            keyboardType="numeric"
            returnKeyType="done"
          />
          {estimatedCalories > 0 && (
            <Text style={[styles.estimate, { color: colors.textSecondary }]}>
              ~{estimatedCalories} cal
            </Text>
          )}
          <View style={styles.formActions}>
            <Pressable
              style={[styles.cancelBtn, { borderColor: colors.border }]}
              onPress={() => { setShowForm(false); setName(''); setDuration(''); }}
            >
              <Text style={[styles.cancelBtnText, { color: colors.textSecondary }]}>Cancel</Text>
            </Pressable>
            <Pressable
              style={[styles.logBtn, { backgroundColor: colors.primary }]}
              onPress={handleLog}
              disabled={!name.trim() || !duration}
            >
              <Text style={styles.logBtnText}>Log</Text>
            </Pressable>
          </View>
        </View>
      ) : (
        <Pressable
          style={[styles.addBtn, { borderColor: colors.border }]}
          onPress={() => setShowForm(true)}
        >
          <Ionicons name="add" size={18} color={colors.primary} />
          <Text style={[styles.addBtnText, { color: colors.primary }]}>Add Exercise</Text>
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
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: spacing.md,
  },
  headerLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
  },
  headerTitle: {
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  headerCalories: {
    fontSize: fontSizes.sm,
    fontWeight: '500',
  },
  list: {
    maxHeight: 156,
  },
  exerciseRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
    paddingVertical: spacing.xs,
  },
  exerciseName: {
    flex: 1,
    fontSize: fontSizes.sm,
    fontWeight: '500',
  },
  exerciseDetail: {
    fontSize: fontSizes.sm,
  },
  exerciseCalories: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  emptyText: {
    fontSize: fontSizes.sm,
    textAlign: 'center',
    paddingVertical: spacing.md,
  },
  addBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.xs,
    marginTop: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    borderStyle: 'dashed',
  },
  addBtnText: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  form: {
    marginTop: spacing.md,
    paddingTop: spacing.md,
    borderTopWidth: 1,
    gap: spacing.sm,
  },
  input: {
    borderRadius: borderRadius.md,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    fontSize: fontSizes.sm,
  },
  estimate: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  formActions: {
    flexDirection: 'row',
    gap: spacing.sm,
    marginTop: spacing.xs,
  },
  cancelBtn: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.md,
    borderWidth: 1,
  },
  cancelBtnText: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  logBtn: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.md,
  },
  logBtnText: {
    fontSize: fontSizes.sm,
    fontWeight: '700',
    color: '#fff',
  },
});
