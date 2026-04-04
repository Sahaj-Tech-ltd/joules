import React, { useState, useCallback, useEffect, useMemo } from 'react';
import { View, Text, StyleSheet, Pressable, Modal, ActivityIndicator } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { useRouter } from 'expo-router';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { fetchMeals, deleteMeal } from '@joules/api-client';
import type { Meal } from '@joules/api-client';
import MealTimeline from '@/components/MealTimeline';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

const ADD_OPTIONS = [
  { key: 'camera', label: 'Take Photo', icon: 'camera-outline' as const, route: '/log/camera' },
  { key: 'barcode', label: 'Scan Barcode', icon: 'barcode-outline' as const, route: '/log/barcode' },
  { key: 'text', label: 'Describe Meal', icon: 'create-outline' as const, route: '/log/manual' },
  { key: 'search', label: 'Search Foods', icon: 'search-outline' as const, route: '/log/search' },
] as const;

export default function LogTabScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();

  const [meals, setMeals] = useState<Meal[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [showAddSheet, setShowAddSheet] = useState(false);

  const loadMeals = useCallback(async () => {
    try {
      const data = await fetchMeals();
      setMeals(data);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadMeals();
  }, [loadMeals]);

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    loadMeals();
  }, [loadMeals]);

  const handleDeleteMeal = useCallback(async (id: string) => {
    try {
      await deleteMeal(id);
      setMeals((prev) => prev.filter((m) => m.id !== id));
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  }, []);

  const totals = useMemo(() => {
    return meals.reduce(
      (acc, meal) => {
        for (const food of meal.foods) {
          acc.calories += food.calories;
          acc.protein += food.protein_g;
          acc.carbs += food.carbs_g;
          acc.fat += food.fat_g;
        }
        return acc;
      },
      { calories: 0, protein: 0, carbs: 0, fat: 0 }
    );
  }, [meals]);

  const handleOptionPress = (route: string) => {
    setShowAddSheet(false);
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    router.push(route as any);
  };

  if (loading) {
    return (
      <View style={[styles.loadingWrap, { backgroundColor: colors.background }]}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Today's Log</Text>
      </View>

      <View style={[styles.totalsCard, { backgroundColor: colors.surface, borderColor: colors.border }]}>
        <Text style={[styles.totalsCalories, { color: colors.textPrimary }]}>
          {Math.round(totals.calories)}
          <Text style={[styles.totalsUnit, { color: colors.textTertiary }]}> cal</Text>
        </Text>
        <View style={styles.macroRow}>
          <View style={styles.macroItem}>
            <Text style={[styles.macroValue, { color: colors.macroProtein }]}>
              {Math.round(totals.protein)}g
            </Text>
            <Text style={[styles.macroLabel, { color: colors.textTertiary }]}>Protein</Text>
          </View>
          <View style={styles.macroItem}>
            <Text style={[styles.macroValue, { color: colors.macroCarbs }]}>
              {Math.round(totals.carbs)}g
            </Text>
            <Text style={[styles.macroLabel, { color: colors.textTertiary }]}>Carbs</Text>
          </View>
          <View style={styles.macroItem}>
            <Text style={[styles.macroValue, { color: colors.macroFat }]}>
              {Math.round(totals.fat)}g
            </Text>
            <Text style={[styles.macroLabel, { color: colors.textTertiary }]}>Fat</Text>
          </View>
        </View>
      </View>

      <MealTimeline
        meals={meals}
        onDeleteMeal={handleDeleteMeal}
        refreshing={refreshing}
        onRefresh={handleRefresh}
      />

      <Pressable
        onPress={() => {
          Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
          setShowAddSheet(true);
        }}
        style={({ pressed }) => [
          styles.fab,
          {
            backgroundColor: colors.primary,
            transform: [{ scale: pressed ? 0.92 : 1 }],
          },
        ]}
        android_ripple={{ color: 'rgba(255,255,255,0.2)', borderless: true }}
      >
        <Ionicons name="add" size={28} color="#fff" />
      </Pressable>

      <Modal
        visible={showAddSheet}
        transparent
        animationType="fade"
        onRequestClose={() => setShowAddSheet(false)}
      >
        <Pressable
          style={styles.sheetBackdrop}
          onPress={() => setShowAddSheet(false)}
        />
        <View style={[styles.sheetContainer, { backgroundColor: colors.surface }]}>
          <View style={[styles.sheetHandle, { backgroundColor: colors.border }]} />
          <Text style={[styles.sheetTitle, { color: colors.textPrimary }]}>Add Meal</Text>
          {ADD_OPTIONS.map((option) => (
            <Pressable
              key={option.key}
              onPress={() => handleOptionPress(option.route)}
              style={({ pressed }) => [
                styles.sheetOption,
                { backgroundColor: pressed ? colors.surfaceElevated : 'transparent' },
              ]}
            >
              <Ionicons name={option.icon} size={22} color={colors.primary} />
              <Text style={[styles.sheetOptionLabel, { color: colors.textPrimary }]}>
                {option.label}
              </Text>
              <Ionicons name="chevron-forward" size={18} color={colors.textTertiary} />
            </Pressable>
          ))}
        </View>
      </Modal>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
  },
  loadingWrap: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  header: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: 1,
  },
  headerTitle: {
    fontSize: fontSizes.xl,
    fontWeight: '700',
  },
  totalsCard: {
    marginHorizontal: spacing.lg,
    marginTop: spacing.md,
    padding: spacing.lg,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
  },
  totalsCalories: {
    fontSize: fontSizes['3xl'],
    fontWeight: '700',
  },
  totalsUnit: {
    fontSize: fontSizes.md,
    fontWeight: '500',
  },
  macroRow: {
    flexDirection: 'row',
    marginTop: spacing.sm,
    gap: spacing.lg,
  },
  macroItem: {
    flex: 1,
  },
  macroValue: {
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  macroLabel: {
    fontSize: fontSizes.xs,
    marginTop: 1,
  },
  fab: {
    position: 'absolute',
    bottom: spacing['2xl'],
    right: spacing.lg,
    width: 56,
    height: 56,
    borderRadius: borderRadius.full,
    justifyContent: 'center',
    alignItems: 'center',
    elevation: 6,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 3 },
    shadowOpacity: 0.3,
    shadowRadius: 6,
  },
  sheetBackdrop: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0,0,0,0.5)',
  },
  sheetContainer: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    borderTopLeftRadius: borderRadius.xl,
    borderTopRightRadius: borderRadius.xl,
    paddingBottom: 40,
    paddingTop: spacing.md,
    paddingHorizontal: spacing.lg,
  },
  sheetHandle: {
    width: 36,
    height: 4,
    borderRadius: 2,
    alignSelf: 'center',
    marginBottom: spacing.md,
  },
  sheetTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '700',
    marginBottom: spacing.md,
  },
  sheetOption: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: spacing.md,
    paddingHorizontal: spacing.sm,
    borderRadius: borderRadius.md,
    gap: spacing.md,
  },
  sheetOptionLabel: {
    flex: 1,
    fontSize: fontSizes.md,
    fontWeight: '500',
  },
});
