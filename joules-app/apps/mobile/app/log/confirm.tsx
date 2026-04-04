import { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Pressable,
  ScrollView,
  Image,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter, useLocalSearchParams } from 'expo-router';
import * as Haptics from 'expo-haptics';
import { Ionicons } from '@expo/vector-icons';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { createMeal } from '@joules/api-client';
import type { FoodItem, MealIdentifyResponse } from '@joules/api-client';
import FoodItemCard from '@/components/FoodItemCard';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

const MEAL_TYPES = [
  { key: 'breakfast', label: 'Breakfast' },
  { key: 'lunch', label: 'Lunch' },
  { key: 'dinner', label: 'Dinner' },
  { key: 'snack', label: 'Snack' },
] as const;

function getDefaultMealType(): string {
  const hour = new Date().getHours();
  if (hour < 11) return 'breakfast';
  if (hour < 15) return 'lunch';
  if (hour < 19) return 'dinner';
  return 'snack';
}

function getConfidenceBadge(confidence: string, colors: ReturnType<typeof getColors>) {
  switch (confidence) {
    case 'high':
      return { label: 'High confidence', bg: `${colors.success}20`, text: colors.success };
    case 'medium':
      return { label: 'Medium confidence', bg: `${colors.warning}20`, text: colors.warning };
    default:
      return { label: 'Low confidence', bg: `${colors.error}20`, text: colors.error };
  }
}

export default function ConfirmScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();
  const { photo, results } = useLocalSearchParams<{ photo?: string; results?: string }>();

  const [foods, setFoods] = useState<FoodItem[]>([]);
  const [mealType, setMealType] = useState<string>(getDefaultMealType());
  const [logging, setLogging] = useState(false);
  const [identifyResult, setIdentifyResult] = useState<MealIdentifyResponse | null>(null);

  useEffect(() => {
    if (!results) {
      router.replace('/log/camera');
      return;
    }

    try {
      const parsed: MealIdentifyResponse = JSON.parse(results);
      if (!parsed.foods || !Array.isArray(parsed.foods)) {
        router.replace('/log/camera');
        return;
      }
      setIdentifyResult(parsed);
      setFoods(parsed.foods);
    } catch {
      router.replace('/log/camera');
    }
  }, [results]);

  const handleLog = async () => {
    if (logging) return;

    setLogging(true);

    try {
      await createMeal({
        meal_type: mealType,
        foods: foods.map((f) => ({
          name: f.name,
          calories: f.calories,
          protein_g: f.protein_g,
          carbs_g: f.carbs_g,
          fat_g: f.fat_g,
          fiber_g: f.fiber_g,
          serving_size: f.serving_size,
          source: f.source,
        })),
        photo: photo ? decodeURIComponent(photo) : undefined,
      });

      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      router.replace('/(tabs)');
    } catch {
      setLogging(false);
    }
  };

  const handleClose = () => {
    if (router.canGoBack()) {
      router.back();
    } else {
      router.replace('/(tabs)');
    }
  };

  const handleSuggestionPress = (suggestion: string) => {
    setFoods([
      {
        id: `suggested-${Date.now()}`,
        name: suggestion,
        calories: 0,
        protein_g: 0,
        carbs_g: 0,
        fat_g: 0,
        fiber_g: 0,
        serving_size: '1 serving',
        source: 'suggestion',
      },
    ]);
  };

  const confidenceBadge = identifyResult
    ? getConfidenceBadge(identifyResult.confidence, colors)
    : null;

  const totalCalories = foods.reduce((sum, f) => sum + f.calories, 0);

  const decodedPhoto = photo ? decodeURIComponent(photo) : null;

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={styles.header}>
        <Pressable onPress={handleClose} style={styles.closeButton}>
          <Ionicons name="close" size={24} color={colors.textPrimary} />
        </Pressable>
        <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>
          Confirm Your Meal
        </Text>
        <View style={styles.headerRight} />
      </View>

      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {decodedPhoto && (
          <View style={[styles.photoPreview, { borderColor: colors.border }]}>
            <Image
              source={{ uri: `data:image/jpeg;base64,${decodedPhoto}` }}
              style={styles.photo}
              resizeMode="cover"
            />
          </View>
        )}

        {confidenceBadge && (
          <View style={[styles.confidenceBadge, { backgroundColor: confidenceBadge.bg }]}>
            <Text style={[styles.confidenceText, { color: confidenceBadge.text }]}>
              {confidenceBadge.label}
            </Text>
          </View>
        )}

        {foods.length > 0 && (
          <View style={styles.foodsSection}>
            {foods.map((food) => (
              <FoodItemCard key={food.id} food={food} />
            ))}
            <View style={[styles.totalRow, { borderColor: colors.border }]}>
              <Text style={[styles.totalLabel, { color: colors.textSecondary }]}>Total</Text>
              <Text style={[styles.totalValue, { color: colors.textPrimary }]}>
                {totalCalories} cal
              </Text>
            </View>
          </View>
        )}

        {identifyResult?.suggestions && identifyResult.suggestions.length > 0 && (
          <View style={styles.suggestionsSection}>
            <Text style={[styles.suggestionsTitle, { color: colors.textSecondary }]}>
              Did you mean?
            </Text>
            <View style={styles.suggestionsRow}>
              {identifyResult.suggestions.map((suggestion) => (
                <Pressable
                  key={suggestion}
                  onPress={() => handleSuggestionPress(suggestion)}
                  style={[
                    styles.suggestionChip,
                    {
                      backgroundColor: colors.surfaceElevated,
                      borderColor: colors.border,
                    },
                  ]}
                >
                  <Text style={[styles.suggestionText, { color: colors.textPrimary }]}>
                    {suggestion}
                  </Text>
                </Pressable>
              ))}
            </View>
          </View>
        )}

        <View style={styles.mealTypeSection}>
          <Text style={[styles.sectionLabel, { color: colors.textSecondary }]}>Meal Type</Text>
          <View style={styles.mealTypeRow}>
            {MEAL_TYPES.map((mt) => (
              <Pressable
                key={mt.key}
                onPress={() => setMealType(mt.key)}
                style={[
                  styles.mealTypePill,
                  {
                    backgroundColor: mealType === mt.key ? colors.primary : colors.surfaceElevated,
                    borderColor: mealType === mt.key ? colors.primary : colors.border,
                  },
                ]}
              >
                <Text
                  style={[
                    styles.mealTypeText,
                    {
                      color: mealType === mt.key ? '#fff' : colors.textSecondary,
                    },
                  ]}
                >
                  {mt.label}
                </Text>
              </Pressable>
            ))}
          </View>
        </View>
      </ScrollView>

      <View style={[styles.footer, { backgroundColor: colors.background }]}>
        <Pressable
          onPress={handleLog}
          disabled={logging}
          style={[
            styles.logButton,
            { backgroundColor: logging ? colors.primary + '60' : colors.primary },
          ]}
        >
          {logging ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.logButtonText}>Log It</Text>
          )}
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
  },
  closeButton: {
    width: 40,
    height: 40,
    justifyContent: 'center',
    alignItems: 'center',
  },
  headerTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '600',
  },
  headerRight: {
    width: 40,
  },
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    paddingHorizontal: spacing.lg,
    paddingBottom: spacing['3xl'],
  },
  photoPreview: {
    width: '100%',
    height: 180,
    borderRadius: borderRadius.lg,
    overflow: 'hidden',
    borderWidth: 1,
    marginBottom: spacing.lg,
  },
  photo: {
    width: '100%',
    height: '100%',
  },
  confidenceBadge: {
    alignSelf: 'flex-start',
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderRadius: borderRadius.full,
    marginBottom: spacing.lg,
  },
  confidenceText: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  foodsSection: {
    gap: spacing.sm,
    marginBottom: spacing.lg,
  },
  totalRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingTop: spacing.md,
    borderTopWidth: 1,
  },
  totalLabel: {
    fontSize: fontSizes.md,
    fontWeight: '600',
  },
  totalValue: {
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  suggestionsSection: {
    marginBottom: spacing.lg,
  },
  suggestionsTitle: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
    marginBottom: spacing.sm,
  },
  suggestionsRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.sm,
  },
  suggestionChip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.full,
    borderWidth: 1,
  },
  suggestionText: {
    fontSize: fontSizes.sm,
  },
  mealTypeSection: {
    marginBottom: spacing.lg,
  },
  sectionLabel: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
    marginBottom: spacing.sm,
  },
  mealTypeRow: {
    flexDirection: 'row',
    gap: spacing.sm,
  },
  mealTypePill: {
    flex: 1,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.full,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
  },
  mealTypeText: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  footer: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: spacing.lg,
  },
  logButton: {
    height: 52,
    borderRadius: borderRadius.lg,
    justifyContent: 'center',
    alignItems: 'center',
  },
  logButtonText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
});
