import React, { useState, useCallback, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  Pressable,
  ScrollView,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { identifyMealFromText, fetchTopFavorites, createMeal } from '@joules/api-client';
import CoachAvatar from '@/components/CoachAvatar';
import type { FoodFavorite, FoodItem } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

type MealType = 'breakfast' | 'lunch' | 'dinner' | 'snack';

const MEAL_LABELS: Record<MealType, string> = {
  breakfast: 'Breakfast',
  lunch: 'Lunch',
  dinner: 'Dinner',
  snack: 'Snack',
};

const MEAL_TYPES: MealType[] = ['breakfast', 'lunch', 'dinner', 'snack'];

function getDefaultMealType(): MealType {
  const hour = new Date().getHours();
  if (hour < 11) return 'breakfast';
  if (hour < 15) return 'lunch';
  if (hour < 19) return 'dinner';
  return 'snack';
}

export default function ManualScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();

  const [text, setText] = useState('');
  const [mealType, setMealType] = useState<MealType>(getDefaultMealType);
  const [identifying, setIdentifying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [favorites, setFavorites] = useState<FoodFavorite[]>([]);

  useEffect(() => {
    fetchTopFavorites(5)
      .then(setFavorites)
      .catch(() => setFavorites([]));
  }, []);

  const handleIdentify = useCallback(async () => {
    if (!text.trim() || identifying) return;
    setError(null);
    setIdentifying(true);
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    try {
      const result = await identifyMealFromText(text.trim());
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      router.replace({
        pathname: '/log/confirm',
        params: { results: JSON.stringify(result) },
      });
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
      setError('Could not identify your meal. Try being more specific.');
    } finally {
      setIdentifying(false);
    }
  }, [text, identifying, router]);

  const handleQuickAdd = useCallback(async (fav: FoodFavorite) => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    try {
      const foods: Partial<FoodItem>[] = [{
        name: fav.name,
        calories: fav.calories,
        protein_g: fav.protein_g,
        carbs_g: fav.carbs_g,
        fat_g: fav.fat_g,
        fiber_g: fav.fiber_g,
        serving_size: fav.serving_size,
        source: fav.source,
      }];
      await createMeal({ meal_type: mealType, foods });
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      router.back();
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  }, [mealType, router]);

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <Pressable onPress={() => router.back()} style={styles.closeBtn} hitSlop={12}>
          <Ionicons name="close" size={24} color={colors.textPrimary} />
        </Pressable>
        <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Describe Your Meal</Text>
      </View>

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
      >
        <TextInput
          style={[
            styles.textInput,
            {
              color: colors.textPrimary,
              backgroundColor: colors.surface,
              borderColor: error ? colors.error : colors.border,
            },
          ]}
          placeholder="e.g., 1 cup of oatmeal with banana and honey"
          placeholderTextColor={colors.textTertiary}
          value={text}
          onChangeText={setText}
          multiline
          textAlignVertical="top"
        />
        <Text style={[styles.charCount, { color: colors.textTertiary }]}>
          {text.length}
        </Text>

        <View style={styles.mealTypeRow}>
          {MEAL_TYPES.map(type => (
            <Pressable
              key={type}
              onPress={() => {
                Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                setMealType(type);
              }}
              style={[
                styles.mealPill,
                {
                  backgroundColor: mealType === type ? colors.primary : colors.surface,
                  borderColor: mealType === type ? colors.primary : colors.border,
                },
              ]}
            >
              <Text
                style={[
                  styles.mealPillText,
                  { color: mealType === type ? '#fff' : colors.textSecondary },
                ]}
              >
                {MEAL_LABELS[type]}
              </Text>
            </Pressable>
          ))}
        </View>

        {error && (
          <View style={[styles.errorBox, { backgroundColor: colors.error + '15', borderColor: colors.error }]}>
            <Ionicons name="alert-circle-outline" size={18} color={colors.error} />
            <Text style={[styles.errorText, { color: colors.error }]}>{error}</Text>
          </View>
        )}

        <Pressable
          onPress={handleIdentify}
          disabled={!text.trim() || identifying}
          style={({ pressed }) => [
            styles.identifyBtn,
            {
              backgroundColor: colors.primary,
              opacity: !text.trim() || identifying ? 0.5 : 1,
              transform: [{ scale: pressed ? 0.97 : 1 }],
            },
          ]}
        >
          {identifying ? (
            <View style={styles.identifyLoading}>
              <CoachAvatar size={28} />
              <ActivityIndicator size="small" color="#fff" style={styles.identifySpinner} />
            </View>
          ) : (
            <Text style={styles.identifyBtnText}>Identify</Text>
          )}
        </Pressable>

        {favorites.length > 0 && (
          <View style={styles.quickSection}>
            <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Quick Add</Text>
            <ScrollView horizontal showsHorizontalScrollIndicator={false}>
              {favorites.map((fav, i) => (
                <Pressable
                  key={fav.id ?? `${fav.name}-${i}`}
                  onPress={() => handleQuickAdd(fav)}
                  style={({ pressed }) => [
                    styles.quickChip,
                    {
                      backgroundColor: colors.surfaceElevated,
                      borderColor: colors.border,
                      transform: [{ scale: pressed ? 0.95 : 1 }],
                    },
                  ]}
                >
                  <Text style={[styles.quickName, { color: colors.textPrimary }]} numberOfLines={1}>
                    {fav.name}
                  </Text>
                  <Text style={[styles.quickCals, { color: colors.textTertiary }]}>
                    {fav.calories} cal
                  </Text>
                </Pressable>
              ))}
            </ScrollView>
          </View>
        )}
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
  },
  header: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
    gap: spacing.sm,
  },
  closeBtn: {
    padding: spacing.xs,
  },
  headerTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '700' as const,
  },
  scroll: {
    flex: 1,
  },
  content: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.lg,
    paddingBottom: spacing['4xl'],
  },
  textInput: {
    minHeight: 120,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    padding: spacing.md,
    fontSize: fontSizes.md,
    lineHeight: 22,
  },
  charCount: {
    fontSize: fontSizes.xs,
    textAlign: 'right' as const,
    marginTop: spacing.xs,
    marginBottom: spacing.md,
  },
  mealTypeRow: {
    flexDirection: 'row' as const,
    gap: spacing.sm,
    marginBottom: spacing.lg,
  },
  mealPill: {
    flex: 1,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.full,
    borderWidth: 1,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
  },
  mealPillText: {
    fontSize: fontSizes.xs,
    fontWeight: '600' as const,
  },
  errorBox: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    padding: spacing.md,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    marginBottom: spacing.md,
    gap: spacing.sm,
  },
  errorText: {
    fontSize: fontSizes.sm,
    flex: 1,
  },
  identifyBtn: {
    height: 52,
    borderRadius: borderRadius.lg,
    justifyContent: 'center' as const,
    alignItems: 'center' as const,
    marginBottom: spacing.xl,
  },
  identifyBtnText: {
    color: '#fff',
    fontSize: fontSizes.lg,
    fontWeight: '700' as const,
  },
  identifyLoading: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    gap: spacing.sm,
  },
  identifySpinner: {
    marginLeft: spacing.xs,
  },
  quickSection: {
    gap: spacing.sm,
  },
  sectionTitle: {
    fontSize: fontSizes.sm,
    fontWeight: '700' as const,
  },
  quickChip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    marginRight: spacing.sm,
    minWidth: 90,
  },
  quickName: {
    fontSize: fontSizes.xs,
    fontWeight: '600' as const,
  },
  quickCals: {
    fontSize: 10,
    marginTop: 1,
  },
});
