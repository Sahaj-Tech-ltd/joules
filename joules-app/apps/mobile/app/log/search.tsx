import React, { useState, useRef, useCallback, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  Pressable,
  ScrollView,
  StyleSheet,
  ActivityIndicator,
  Keyboard,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { searchFoods, fetchTopFavorites, createMeal } from '@joules/api-client';
import type { FoodSearchResult, FoodFavorite, FoodItem } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

export default function SearchScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();

  const [query, setQuery] = useState('');
  const [results, setResults] = useState<FoodSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [selected, setSelected] = useState<FoodSearchResult[]>([]);
  const [favorites, setFavorites] = useState<FoodFavorite[]>([]);
  const [submitting, setSubmitting] = useState(false);

  const inputRef = useRef<TextInput>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();
  const hasSearched = useRef(false);

  useEffect(() => {
    fetchTopFavorites(10)
      .then(setFavorites)
      .catch(() => setFavorites([]));
    setTimeout(() => inputRef.current?.focus(), 300);
  }, []);

  const handleSearch = useCallback((text: string) => {
    setQuery(text);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      if (text.trim().length >= 2) {
        setSearching(true);
        hasSearched.current = true;
        searchFoods(text.trim())
          .then(setResults)
          .catch(() => setResults([]))
          .finally(() => setSearching(false));
      } else {
        hasSearched.current = false;
        setResults([]);
        setSearching(false);
      }
    }, 300);
  }, []);

  const toggleSelect = useCallback((food: FoodSearchResult) => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    setSelected(prev => {
      const exists = prev.find(f => f.name === food.name && f.source === food.source);
      if (exists) return prev.filter(f => f !== exists);
      return [...prev, food];
    });
  }, []);

  const removeSelected = useCallback((index: number) => {
    setSelected(prev => prev.filter((_, i) => i !== index));
  }, []);

  const handleAdd = useCallback(async () => {
    if (selected.length === 0 || submitting) return;
    setSubmitting(true);
    try {
      const foods: Partial<FoodItem>[] = selected.map(f => ({
        name: f.name,
        calories: f.calories,
        protein_g: f.protein_g,
        carbs_g: f.carbs_g,
        fat_g: f.fat_g,
        fiber_g: f.fiber_g,
        serving_size: f.serving_size,
        source: f.source,
      }));
      await createMeal({ meal_type: 'snack', foods });
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      router.back();
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    } finally {
      setSubmitting(false);
    }
  }, [selected, submitting, router]);

  const isSelected = (food: FoodSearchResult) =>
    selected.some(f => f.name === food.name && f.source === food.source);

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <Pressable onPress={() => router.back()} style={styles.closeBtn} hitSlop={12}>
          <Ionicons name="close" size={24} color={colors.textPrimary} />
        </Pressable>
        <TextInput
          ref={inputRef}
          style={[styles.searchInput, { color: colors.textPrimary, backgroundColor: colors.surfaceElevated }]}
          placeholder="Search foods..."
          placeholderTextColor={colors.textTertiary}
          value={query}
          onChangeText={handleSearch}
          returnKeyType="search"
          autoCapitalize="none"
          autoCorrect={false}
        />
      </View>

      {selected.length > 0 && (
        <View style={[styles.selectedWrap, { borderBottomColor: colors.border }]}>
          <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.selectedScroll}>
            {selected.map((food, i) => (
              <View
                key={`${food.name}-${food.source}-${i}`}
                style={[styles.chip, { backgroundColor: colors.primary + '20', borderColor: colors.primary }]}
              >
                <Text style={[styles.chipText, { color: colors.primary }]} numberOfLines={1}>
                  {food.name}
                </Text>
                <Pressable onPress={() => removeSelected(i)} hitSlop={6}>
                  <Ionicons name="close" size={14} color={colors.primary} />
                </Pressable>
              </View>
            ))}
          </ScrollView>
        </View>
      )}

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
      >
        {searching && (
          <View style={styles.loaderRow}>
            <ActivityIndicator size="small" color={colors.primary} />
            <Text style={[styles.loaderText, { color: colors.textSecondary }]}>Searching...</Text>
          </View>
        )}

        {!searching && results.length > 0 && (
          <View style={styles.resultsSection}>
            {results.map((food, i) => {
              const selected = isSelected(food);
              return (
                <Pressable
                  key={`${food.name}-${food.source}-${food.barcode ?? food.id ?? i}`}
                  onPress={() => toggleSelect(food)}
                  style={({ pressed }) => [
                    styles.resultCard,
                    {
                      backgroundColor: selected ? colors.primary + '15' : colors.surface,
                      borderColor: selected ? colors.primary : colors.border,
                      transform: [{ scale: pressed ? 0.98 : 1 }],
                    },
                  ]}
                >
                  <View style={styles.resultTop}>
                    <View style={styles.resultInfo}>
                      <Text style={[styles.resultName, { color: colors.textPrimary }]} numberOfLines={1}>
                        {food.name}
                      </Text>
                      {food.brand ? (
                        <Text style={[styles.resultBrand, { color: colors.textSecondary }]} numberOfLines={1}>
                          {food.brand}
                        </Text>
                      ) : null}
                    </View>
                    <View style={[styles.sourceBadge, { backgroundColor: colors.surfaceElevated }]}>
                      <Text style={[styles.sourceText, { color: colors.textTertiary }]}>
                        {food.source === 'local' ? 'DB' : 'OFF'}
                      </Text>
                    </View>
                  </View>
                  <View style={styles.macroRow}>
                    <Text style={[styles.calText, { color: colors.textPrimary }]}>
                      {food.calories} cal
                    </Text>
                    <Text style={[styles.macroText, { color: colors.textTertiary }]}>
                      P {food.protein_g}g
                    </Text>
                    <Text style={[styles.macroText, { color: colors.textTertiary }]}>
                      C {food.carbs_g}g
                    </Text>
                    <Text style={[styles.macroText, { color: colors.textTertiary }]}>
                      F {food.fat_g}g
                    </Text>
                  </View>
                </Pressable>
              );
            })}
          </View>
        )}

        {!searching && hasSearched.current && results.length === 0 && query.trim().length >= 2 && (
          <View style={styles.emptyWrap}>
            <Ionicons name="search-outline" size={40} color={colors.textTertiary} />
            <Text style={[styles.emptyTitle, { color: colors.textSecondary }]}>
              No foods found for '{query.trim()}'
            </Text>
          </View>
        )}

        {!searching && !hasSearched.current && query.trim().length < 2 && (
          <>
            {favorites.length > 0 && (
              <View style={styles.favSection}>
                <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Favorites</Text>
                {favorites.map((fav, i) => (
                  <Pressable
                    key={fav.id ?? `${fav.name}-${i}`}
                    onPress={() => toggleSelect(fav as any)}
                    style={({ pressed }) => [
                      styles.resultCard,
                      {
                        backgroundColor: isSelected(fav as any) ? colors.primary + '15' : colors.surface,
                        borderColor: isSelected(fav as any) ? colors.primary : colors.border,
                        transform: [{ scale: pressed ? 0.98 : 1 }],
                      },
                    ]}
                  >
                    <View style={styles.resultTop}>
                      <View style={styles.resultInfo}>
                        <Text style={[styles.resultName, { color: colors.textPrimary }]} numberOfLines={1}>
                          {fav.name}
                        </Text>
                      </View>
                      <Text style={[styles.useCount, { color: colors.textTertiary }]}>
                        {fav.use_count}x
                      </Text>
                    </View>
                    <View style={styles.macroRow}>
                      <Text style={[styles.calText, { color: colors.textPrimary }]}>
                        {fav.calories} cal
                      </Text>
                      <Text style={[styles.macroText, { color: colors.textTertiary }]}>
                        P {fav.protein_g}g
                      </Text>
                      <Text style={[styles.macroText, { color: colors.textTertiary }]}>
                        C {fav.carbs_g}g
                      </Text>
                      <Text style={[styles.macroText, { color: colors.textTertiary }]}>
                        F {fav.fat_g}g
                      </Text>
                    </View>
                  </Pressable>
                ))}
              </View>
            )}

            {favorites.length === 0 && (
              <View style={styles.emptyWrap}>
                <Ionicons name="search-outline" size={40} color={colors.textTertiary} />
                <Text style={[styles.emptyTitle, { color: colors.textSecondary }]}>
                  Search for any food
                </Text>
              </View>
            )}
          </>
        )}
      </ScrollView>

      {selected.length > 0 && (
        <View style={[styles.bottomBar, { backgroundColor: colors.background, borderTopColor: colors.border }]}>
          <Pressable
            onPress={handleAdd}
            disabled={submitting}
            style={({ pressed }) => [
              styles.addBtn,
              {
                backgroundColor: colors.primary,
                opacity: submitting ? 0.7 : 1,
                transform: [{ scale: pressed ? 0.97 : 1 }],
              },
            ]}
          >
            {submitting ? (
              <ActivityIndicator size="small" color="#fff" />
            ) : (
              <Text style={styles.addBtnText}>
                Add Selected ({selected.length})
              </Text>
            )}
          </Pressable>
        </View>
      )}
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
  searchInput: {
    flex: 1,
    height: 40,
    borderRadius: borderRadius.lg,
    paddingHorizontal: spacing.md,
    fontSize: fontSizes.md,
  },
  selectedWrap: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderBottomWidth: 1,
  },
  selectedScroll: {
    flexDirection: 'row' as const,
  },
  chip: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderRadius: borderRadius.full,
    borderWidth: 1,
    marginRight: spacing.sm,
    gap: 4,
  },
  chipText: {
    fontSize: fontSizes.xs,
    fontWeight: '600' as const,
    maxWidth: 120,
  },
  scroll: {
    flex: 1,
  },
  content: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: 100,
  },
  loaderRow: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    paddingVertical: spacing.xl,
    gap: spacing.sm,
  },
  loaderText: {
    fontSize: fontSizes.sm,
  },
  resultsSection: {
    gap: spacing.sm,
  },
  resultCard: {
    padding: spacing.md,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
  },
  resultTop: {
    flexDirection: 'row' as const,
    alignItems: 'flex-start' as const,
    justifyContent: 'space-between' as const,
    marginBottom: spacing.xs,
  },
  resultInfo: {
    flex: 1,
    marginRight: spacing.sm,
  },
  resultName: {
    fontSize: fontSizes.sm,
    fontWeight: '600' as const,
  },
  resultBrand: {
    fontSize: fontSizes.xs,
    marginTop: 1,
  },
  sourceBadge: {
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
    borderRadius: borderRadius.sm,
  },
  sourceText: {
    fontSize: 10,
    fontWeight: '700' as const,
    letterSpacing: 0.5,
  },
  macroRow: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    gap: spacing.md,
  },
  calText: {
    fontSize: fontSizes.xs,
    fontWeight: '600' as const,
  },
  macroText: {
    fontSize: fontSizes.xs,
  },
  useCount: {
    fontSize: fontSizes.xs,
    fontWeight: '600' as const,
  },
  favSection: {
    gap: spacing.sm,
  },
  sectionTitle: {
    fontSize: fontSizes.sm,
    fontWeight: '700' as const,
    marginBottom: spacing.xs,
  },
  emptyWrap: {
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    paddingVertical: spacing['4xl'],
    gap: spacing.md,
  },
  emptyTitle: {
    fontSize: fontSizes.md,
    fontWeight: '500' as const,
    textAlign: 'center' as const,
  },
  bottomBar: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderTopWidth: 1,
  },
  addBtn: {
    height: 48,
    borderRadius: borderRadius.lg,
    justifyContent: 'center' as const,
    alignItems: 'center' as const,
  },
  addBtnText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '700' as const,
  },
});
