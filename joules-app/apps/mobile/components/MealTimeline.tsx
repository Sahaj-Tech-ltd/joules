import React, { useState, useRef, useCallback } from 'react';
import { View, Text, StyleSheet, Pressable, FlatList, Animated, PanResponder, ActivityIndicator } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import FoodItemCard from '@/components/FoodItemCard';
import type { Meal } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface MealTimelineProps {
  meals: Meal[];
  onDeleteMeal: (id: string) => void;
  refreshing: boolean;
  onRefresh: () => void;
}

function MealCard({
  meal,
  colors,
  onDelete,
}: {
  meal: Meal;
  colors: ReturnType<typeof getColors>;
  onDelete: (id: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const swipeX = useRef(new Animated.Value(0)).current;
  const swipeThreshold = -80;

  const panResponder = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (_, gs) => Math.abs(gs.dx) > 10 && Math.abs(gs.dx) > Math.abs(gs.dy) * 2,
      onPanResponderMove: (_, gs) => {
        if (gs.dx < 0) {
          swipeX.setValue(Math.max(gs.dx, -120));
        } else {
          swipeX.setValue(Math.min(gs.dx, 0));
        }
      },
      onPanResponderRelease: (_, gs) => {
        if (gs.dx < swipeThreshold) {
          Animated.spring(swipeX, { toValue: -90, useNativeDriver: false }).start();
        } else {
          Animated.spring(swipeX, { toValue: 0, useNativeDriver: false }).start();
        }
      },
    })
  ).current;

  const totalCalories = meal.foods.reduce((sum, f) => sum + f.calories, 0);
  const itemCount = meal.foods.length;
  const timeStr = new Date(meal.timestamp).toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
  });
  const mealTypeLabel = meal.meal_type.charAt(0).toUpperCase() + meal.meal_type.slice(1);

  const handleDelete = () => {
    Animated.spring(swipeX, { toValue: 0, useNativeDriver: false }).start(() => {
      onDelete(meal.id);
    });
  };

  return (
    <View style={styles.swipeContainer}>
      <Pressable
        onPress={handleDelete}
        style={[styles.deleteBtn, { backgroundColor: colors.error }]}
      >
        <Ionicons name="trash" size={20} color="#fff" />
      </Pressable>
      <Animated.View
        style={{ transform: [{ translateX: swipeX }] }}
        {...panResponder.panHandlers}
      >
        <Pressable
          onPress={() => setExpanded((e) => !e)}
          style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}
        >
          <View style={styles.cardHeader}>
            <View style={styles.timeCol}>
              <Text style={[styles.timeText, { color: colors.textSecondary }]}>{timeStr}</Text>
            </View>
            <View style={styles.infoCol}>
              <View style={styles.infoRow}>
                <View style={[styles.badge, { backgroundColor: `${colors.primary}20` }]}>
                  <Text style={[styles.badgeText, { color: colors.primary }]}>{mealTypeLabel}</Text>
                </View>
                <Text style={[styles.caloriesText, { color: colors.textPrimary }]}>
                  {Math.round(totalCalories)} cal
                </Text>
              </View>
              <Text style={[styles.itemCount, { color: colors.textTertiary }]}>
                {itemCount} {itemCount === 1 ? 'item' : 'items'}
              </Text>
            </View>
            <Ionicons
              name={expanded ? 'chevron-up' : 'chevron-down'}
              size={18}
              color={colors.textTertiary}
            />
          </View>

          {expanded && (
            <View style={styles.expandedContent}>
              {meal.foods.map((food) => (
                <FoodItemCard key={food.id} food={food} />
              ))}
            </View>
          )}
        </Pressable>
      </Animated.View>
    </View>
  );
}

export default function MealTimeline({ meals, onDeleteMeal, refreshing, onRefresh }: MealTimelineProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  const sortedMeals = [...meals].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
  );

  const renderMeal = useCallback(
    ({ item }: { item: Meal }) => (
      <MealCard meal={item} colors={colors} onDelete={onDeleteMeal} />
    ),
    [colors, onDeleteMeal]
  );

  if (meals.length === 0) {
    return (
      <View style={[styles.emptyWrap, { backgroundColor: colors.background }]}>
        <Ionicons name="restaurant-outline" size={48} color={colors.textTertiary} />
        <Text style={[styles.emptyText, { color: colors.textTertiary }]}>No meals logged today</Text>
      </View>
    );
  }

  return (
    <FlatList
      data={sortedMeals}
      keyExtractor={(item) => item.id}
      renderItem={renderMeal}
      refreshing={refreshing}
      onRefresh={onRefresh}
      contentContainerStyle={styles.listContent}
    />
  );
}

const styles = StyleSheet.create({
  swipeContainer: {
    overflow: 'hidden',
    marginBottom: spacing.sm,
    position: 'relative',
  },
  deleteBtn: {
    position: 'absolute',
    right: 0,
    top: 0,
    bottom: spacing.sm,
    width: 80,
    justifyContent: 'center',
    alignItems: 'center',
    borderRadius: borderRadius.lg,
    marginVertical: 0,
  },
  card: {
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    padding: spacing.lg,
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  timeCol: {
    marginRight: spacing.md,
  },
  timeText: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  infoCol: {
    flex: 1,
  },
  infoRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: borderRadius.full,
  },
  badgeText: {
    fontSize: 11,
    fontWeight: '700',
  },
  caloriesText: {
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  itemCount: {
    fontSize: fontSizes.xs,
    marginTop: 2,
  },
  expandedContent: {
    marginTop: spacing.md,
    paddingTop: spacing.md,
    borderTopWidth: 1,
    borderTopColor: 'rgba(150,150,150,0.2)',
  },
  emptyWrap: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingTop: 80,
  },
  emptyText: {
    fontSize: fontSizes.md,
    marginTop: spacing.md,
    fontWeight: '500',
  },
  listContent: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: 100,
  },
});
