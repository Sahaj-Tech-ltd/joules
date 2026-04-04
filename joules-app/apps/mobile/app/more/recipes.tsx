import React, { useState, useCallback, useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Pressable,
  FlatList,
  TextInput,
  ActivityIndicator,
  Animated,
  PanResponder,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { fetchRecipes, createRecipe, deleteRecipe, logFromRecipe } from '@joules/api-client';
import type { Recipe, RecipeFood } from '@joules/api-client';

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

interface IngredientRow {
  id: string;
  name: string;
  calories: string;
  protein_g: string;
  carbs_g: string;
  fat_g: string;
}

function RecipeCard({
  recipe,
  colors,
  onDelete,
  onLog,
}: {
  recipe: Recipe;
  colors: ReturnType<typeof getColors>;
  onDelete: (id: string) => void;
  onLog: (recipe: Recipe) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const swipeX = useRef(new Animated.Value(0)).current;
  const swipeThreshold = -80;

  const panResponder = useRef(
    PanResponder.create({
      onMoveShouldSetPanResponder: (_, gs) =>
        Math.abs(gs.dx) > 10 && Math.abs(gs.dx) > Math.abs(gs.dy) * 2,
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

  const handleDelete = () => {
    Animated.spring(swipeX, { toValue: 0, useNativeDriver: false }).start(() => {
      onDelete(recipe.id);
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
          <View style={styles.cardTop}>
            <View style={styles.cardInfo}>
              <Text style={[styles.recipeName, { color: colors.textPrimary }]} numberOfLines={1}>
                {recipe.name}
              </Text>
              {recipe.description ? (
                <Text style={[styles.recipeDesc, { color: colors.textSecondary }]} numberOfLines={1}>
                  {recipe.description}
                </Text>
              ) : null}
            </View>
            <View style={[styles.calBadge, { backgroundColor: `${colors.primary}20` }]}>
              <Text style={[styles.calBadgeText, { color: colors.primary }]}>
                {Math.round(recipe.calories)} cal
              </Text>
            </View>
          </View>

          <View style={styles.macroRow}>
            <View style={styles.macroItem}>
              <Text style={[styles.macroValue, { color: colors.macroProtein }]}>
                {Math.round(recipe.protein_g)}g
              </Text>
              <Text style={[styles.macroLabel, { color: colors.textTertiary }]}>P</Text>
            </View>
            <View style={styles.macroItem}>
              <Text style={[styles.macroValue, { color: colors.macroCarbs }]}>
                {Math.round(recipe.carbs_g)}g
              </Text>
              <Text style={[styles.macroLabel, { color: colors.textTertiary }]}>C</Text>
            </View>
            <View style={styles.macroItem}>
              <Text style={[styles.macroValue, { color: colors.macroFat }]}>
                {Math.round(recipe.fat_g)}g
              </Text>
              <Text style={[styles.macroLabel, { color: colors.textTertiary }]}>F</Text>
            </View>
          </View>

          {expanded && (
            <View style={[styles.expandedContent, { borderTopColor: colors.border }]}>
              {recipe.foods.map((food, i) => (
                <View
                  key={i}
                  style={[
                    styles.ingredientRow,
                    i < recipe.foods.length - 1 ? { borderBottomWidth: 1, borderBottomColor: colors.border } : {},
                  ]}
                >
                  <View style={styles.ingredientInfo}>
                    <Text style={[styles.ingredientName, { color: colors.textPrimary }]}>
                      {food.name}
                    </Text>
                    {food.serving_size ? (
                      <Text style={[styles.ingredientServing, { color: colors.textTertiary }]}>
                        {food.serving_size}
                      </Text>
                    ) : null}
                  </View>
                  <Text style={[styles.ingredientCal, { color: colors.textSecondary }]}>
                    {Math.round(food.calories)} cal
                  </Text>
                </View>
              ))}

              <Pressable
                onPress={() => onLog(recipe)}
                style={[styles.logBtn, { backgroundColor: colors.primary }]}
              >
                <Ionicons name="restaurant-outline" size={18} color="#fff" />
                <Text style={styles.logBtnText}>Log This Recipe</Text>
              </Pressable>
            </View>
          )}
        </Pressable>
      </Animated.View>
    </View>
  );
}

function LogModal({
  recipe,
  colors,
  onClose,
  onLogged,
}: {
  recipe: Recipe;
  colors: ReturnType<typeof getColors>;
  onClose: () => void;
  onLogged: () => void;
}) {
  const [mealType, setMealType] = useState('lunch');
  const [logging, setLogging] = useState(false);

  const handleLog = async () => {
    if (logging) return;
    setLogging(true);
    try {
      await logFromRecipe(recipe.id, mealType);
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      onLogged();
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
      setLogging(false);
    }
  };

  return (
    <View style={styles.logOverlay}>
      <Pressable style={StyleSheet.absoluteFillObject} onPress={onClose} />
      <View style={[styles.logSheet, { backgroundColor: colors.surface }]}>
        <View style={[styles.sheetHandle, { backgroundColor: colors.border }]} />
        <Text style={[styles.logSheetTitle, { color: colors.textPrimary }]}>
          Log &quot;{recipe.name}&quot;
        </Text>
        <Text style={[styles.logSheetSubtitle, { color: colors.textSecondary }]}>
          {Math.round(recipe.calories)} cal · {recipe.servings} {recipe.servings === 1 ? 'serving' : 'servings'}
        </Text>

        <View style={styles.mealTypeRow}>
          {MEAL_TYPES.map((mt) => (
            <Pressable
              key={mt.key}
              onPress={() => {
                Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                setMealType(mt.key);
              }}
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
                  { color: mealType === mt.key ? '#fff' : colors.textSecondary },
                ]}
              >
                {mt.label}
              </Text>
            </Pressable>
          ))}
        </View>

        <Pressable
          onPress={handleLog}
          disabled={logging}
          style={[
            styles.logConfirmBtn,
            { backgroundColor: logging ? `${colors.primary}60` : colors.primary },
          ]}
        >
          {logging ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.logConfirmBtnText}>Log It</Text>
          )}
        </Pressable>
      </View>
    </View>
  );
}

export default function RecipesScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();

  const [recipes, setRecipes] = useState<Recipe[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [logRecipe, setLogRecipe] = useState<Recipe | null>(null);

  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [newServings, setNewServings] = useState('1');
  const [ingredients, setIngredients] = useState<IngredientRow[]>([]);
  const [saving, setSaving] = useState(false);

  const loadRecipes = useCallback(async () => {
    try {
      const data = await fetchRecipes();
      setRecipes(data);
    } catch {
      // silently fail
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadRecipes();
  }, [loadRecipes]);

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    loadRecipes();
  }, [loadRecipes]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await deleteRecipe(id);
      setRecipes((prev) => prev.filter((r) => r.id !== id));
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  }, []);

  const handleLog = useCallback((recipe: Recipe) => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    setLogRecipe(recipe);
  }, []);

  const handleLogged = useCallback(() => {
    setLogRecipe(null);
    loadRecipes();
  }, [loadRecipes]);

  const addIngredient = () => {
    setIngredients((prev) => [
      ...prev,
      {
        id: `${Date.now()}-${Math.random()}`,
        name: '',
        calories: '',
        protein_g: '',
        carbs_g: '',
        fat_g: '',
      },
    ]);
  };

  const updateIngredient = (id: string, field: keyof IngredientRow, value: string) => {
    setIngredients((prev) =>
      prev.map((ing) => (ing.id === id ? { ...ing, [field]: value } : ing))
    );
  };

  const removeIngredient = (id: string) => {
    setIngredients((prev) => prev.filter((ing) => ing.id !== id));
  };

  const resetForm = () => {
    setNewName('');
    setNewDesc('');
    setNewServings('1');
    setIngredients([]);
    setShowCreate(false);
  };

  const handleSave = async () => {
    if (saving || !newName.trim()) return;

    const foods: RecipeFood[] = ingredients
      .filter((ing) => ing.name.trim())
      .map((ing) => ({
        name: ing.name.trim(),
        calories: parseFloat(ing.calories) || 0,
        protein_g: parseFloat(ing.protein_g) || 0,
        carbs_g: parseFloat(ing.carbs_g) || 0,
        fat_g: parseFloat(ing.fat_g) || 0,
        fiber_g: 0,
        serving_size: '1 serving',
        source: 'custom',
      }));

    const totalCalories = foods.reduce((s, f) => s + f.calories, 0);
    const totalProtein = foods.reduce((s, f) => s + f.protein_g, 0);
    const totalCarbs = foods.reduce((s, f) => s + f.carbs_g, 0);
    const totalFat = foods.reduce((s, f) => s + f.fat_g, 0);

    setSaving(true);
    try {
      await createRecipe({
        name: newName.trim(),
        description: newDesc.trim(),
        servings: parseInt(newServings) || 1,
        calories: totalCalories,
        protein_g: totalProtein,
        carbs_g: totalCarbs,
        fat_g: totalFat,
        foods,
      });
      await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      resetForm();
      loadRecipes();
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <SafeAreaView style={[styles.container, { backgroundColor: colors.background }]}>
        <View style={styles.header}>
          <Pressable onPress={() => router.back()} style={styles.headerBtn}>
            <Ionicons name="arrow-back" size={24} color={colors.textPrimary} />
          </Pressable>
          <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Recipes</Text>
          <View style={styles.headerBtn} />
        </View>
        <View style={styles.loadingWrap}>
          <ActivityIndicator size="large" color={colors.primary} />
        </View>
      </SafeAreaView>
    );
  }

  const renderRecipe = useCallback(
    ({ item }: { item: Recipe }) => (
      <RecipeCard recipe={item} colors={colors} onDelete={handleDelete} onLog={handleLog} />
    ),
    [colors, handleDelete, handleLog]
  );

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.background }]}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} style={styles.headerBtn}>
          <Ionicons name="arrow-back" size={24} color={colors.textPrimary} />
        </Pressable>
        <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Recipes</Text>
        <Pressable
          onPress={() => {
            Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
            setShowCreate((prev) => !prev);
          }}
          style={styles.headerBtn}
        >
          <Ionicons
            name={showCreate ? 'close' : 'add'}
            size={28}
            color={colors.primary}
          />
        </Pressable>
      </View>

      {showCreate && (
        <View style={[styles.createCard, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <Text style={[styles.createTitle, { color: colors.textPrimary }]}>New Recipe</Text>

          <TextInput
            style={[styles.input, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
            placeholder="Recipe name"
            placeholderTextColor={colors.textTertiary}
            value={newName}
            onChangeText={setNewName}
          />

          <TextInput
            style={[styles.input, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
            placeholder="Description (optional)"
            placeholderTextColor={colors.textTertiary}
            value={newDesc}
            onChangeText={setNewDesc}
          />

          <TextInput
            style={[styles.inputSmall, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
            placeholder="Servings"
            placeholderTextColor={colors.textTertiary}
            value={newServings}
            onChangeText={setNewServings}
            keyboardType="numeric"
          />

          <View style={styles.ingredientHeader}>
            <Text style={[styles.ingredientHeaderTitle, { color: colors.textSecondary }]}>
              Ingredients
            </Text>
            <Pressable onPress={addIngredient} style={styles.addIngredientBtn}>
              <Ionicons name="add-circle-outline" size={20} color={colors.primary} />
              <Text style={[styles.addIngredientText, { color: colors.primary }]}>Add</Text>
            </Pressable>
          </View>

          {ingredients.map((ing) => (
            <View key={ing.id} style={styles.ingredientFormRow}>
              <TextInput
                style={[styles.ingInput, styles.ingInputName, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
                placeholder="Food name"
                placeholderTextColor={colors.textTertiary}
                value={ing.name}
                onChangeText={(v) => updateIngredient(ing.id, 'name', v)}
              />
              <TextInput
                style={[styles.ingInput, styles.ingInputSmall, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
                placeholder="Cal"
                placeholderTextColor={colors.textTertiary}
                value={ing.calories}
                onChangeText={(v) => updateIngredient(ing.id, 'calories', v)}
                keyboardType="numeric"
              />
              <TextInput
                style={[styles.ingInput, styles.ingInputSmall, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
                placeholder="P"
                placeholderTextColor={colors.textTertiary}
                value={ing.protein_g}
                onChangeText={(v) => updateIngredient(ing.id, 'protein_g', v)}
                keyboardType="numeric"
              />
              <TextInput
                style={[styles.ingInput, styles.ingInputSmall, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
                placeholder="C"
                placeholderTextColor={colors.textTertiary}
                value={ing.carbs_g}
                onChangeText={(v) => updateIngredient(ing.id, 'carbs_g', v)}
                keyboardType="numeric"
              />
              <TextInput
                style={[styles.ingInput, styles.ingInputSmall, { backgroundColor: colors.surfaceElevated, color: colors.textPrimary, borderColor: colors.border }]}
                placeholder="F"
                placeholderTextColor={colors.textTertiary}
                value={ing.fat_g}
                onChangeText={(v) => updateIngredient(ing.id, 'fat_g', v)}
                keyboardType="numeric"
              />
              <Pressable onPress={() => removeIngredient(ing.id)} style={styles.removeIngBtn}>
                <Ionicons name="close-circle" size={18} color={colors.error} />
              </Pressable>
            </View>
          ))}

          <View style={styles.createActions}>
            <Pressable
              onPress={resetForm}
              style={[styles.cancelBtn, { borderColor: colors.border }]}
            >
              <Text style={[styles.cancelBtnText, { color: colors.textSecondary }]}>Cancel</Text>
            </Pressable>
            <Pressable
              onPress={handleSave}
              disabled={saving || !newName.trim()}
              style={[
                styles.saveBtn,
                {
                  backgroundColor: saving || !newName.trim() ? `${colors.primary}60` : colors.primary,
                },
              ]}
            >
              {saving ? (
                <ActivityIndicator color="#fff" size="small" />
              ) : (
                <Text style={styles.saveBtnText}>Save Recipe</Text>
              )}
            </Pressable>
          </View>
        </View>
      )}

      {recipes.length === 0 && !showCreate ? (
        <View style={styles.emptyWrap}>
          <Ionicons name="book-outline" size={48} color={colors.textTertiary} />
          <Text style={[styles.emptyTitle, { color: colors.textTertiary }]}>No recipes yet</Text>
          <Text style={[styles.emptySubtitle, { color: colors.textTertiary }]}>
            Create a recipe to quickly log meals
          </Text>
        </View>
      ) : (
        <FlatList
          data={recipes}
          keyExtractor={(item) => item.id}
          renderItem={renderRecipe}
          refreshing={refreshing}
          onRefresh={handleRefresh}
          contentContainerStyle={styles.listContent}
          ListHeaderComponent={showCreate ? <View style={{ height: spacing.sm }} /> : null}
        />
      )}

      {logRecipe && (
        <LogModal
          recipe={logRecipe}
          colors={colors}
          onClose={() => setLogRecipe(null)}
          onLogged={handleLogged}
        />
      )}
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
  headerBtn: {
    width: 40,
    height: 40,
    justifyContent: 'center',
    alignItems: 'center',
  },
  headerTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '600',
  },
  loadingWrap: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
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
  },
  card: {
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    padding: spacing.lg,
  },
  cardTop: {
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
  cardInfo: {
    flex: 1,
    marginRight: spacing.sm,
  },
  recipeName: {
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  recipeDesc: {
    fontSize: fontSizes.sm,
    marginTop: 2,
  },
  calBadge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: borderRadius.full,
  },
  calBadgeText: {
    fontSize: 11,
    fontWeight: '700',
  },
  macroRow: {
    flexDirection: 'row',
    marginTop: spacing.sm,
    gap: spacing.lg,
  },
  macroItem: {
    alignItems: 'center',
  },
  macroValue: {
    fontSize: fontSizes.sm,
    fontWeight: '700',
  },
  macroLabel: {
    fontSize: fontSizes.xs,
    marginTop: 1,
  },
  expandedContent: {
    marginTop: spacing.md,
    paddingTop: spacing.md,
    borderTopWidth: 1,
  },
  ingredientRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: spacing.sm,
  },
  ingredientInfo: {
    flex: 1,
    marginRight: spacing.sm,
  },
  ingredientName: {
    fontSize: fontSizes.sm,
    fontWeight: '500',
  },
  ingredientServing: {
    fontSize: fontSizes.xs,
    marginTop: 1,
  },
  ingredientCal: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  logBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: spacing.md,
    paddingVertical: spacing.md,
    borderRadius: borderRadius.lg,
    gap: spacing.sm,
  },
  logBtnText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  emptyWrap: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: spacing['3xl'],
  },
  emptyTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '600',
    marginTop: spacing.md,
  },
  emptySubtitle: {
    fontSize: fontSizes.sm,
    marginTop: spacing.xs,
    textAlign: 'center',
  },
  listContent: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: 100,
  },
  logOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    justifyContent: 'flex-end',
  },
  logSheet: {
    borderTopLeftRadius: borderRadius.xl,
    borderTopRightRadius: borderRadius.xl,
    paddingTop: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingBottom: 40,
  },
  sheetHandle: {
    width: 36,
    height: 4,
    borderRadius: 2,
    alignSelf: 'center',
    marginBottom: spacing.md,
  },
  logSheetTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '700',
  },
  logSheetSubtitle: {
    fontSize: fontSizes.sm,
    marginTop: spacing.xs,
    marginBottom: spacing.lg,
  },
  mealTypeRow: {
    flexDirection: 'row',
    gap: spacing.sm,
    marginBottom: spacing.lg,
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
  logConfirmBtn: {
    height: 52,
    borderRadius: borderRadius.lg,
    justifyContent: 'center',
    alignItems: 'center',
  },
  logConfirmBtnText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  createCard: {
    marginHorizontal: spacing.lg,
    padding: spacing.lg,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    gap: spacing.sm,
  },
  createTitle: {
    fontSize: fontSizes.md,
    fontWeight: '700',
    marginBottom: spacing.xs,
  },
  input: {
    height: 44,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    fontSize: fontSizes.md,
  },
  inputSmall: {
    height: 44,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    fontSize: fontSizes.md,
    width: 120,
  },
  ingredientHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: spacing.sm,
  },
  ingredientHeaderTitle: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  addIngredientBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  addIngredientText: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  ingredientFormRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
    marginTop: spacing.xs,
  },
  ingInput: {
    borderRadius: borderRadius.md,
    borderWidth: 1,
    paddingHorizontal: spacing.sm,
    fontSize: fontSizes.xs,
    height: 36,
  },
  ingInputName: {
    flex: 1,
  },
  ingInputSmall: {
    width: 42,
  },
  removeIngBtn: {
    padding: 2,
  },
  createActions: {
    flexDirection: 'row',
    gap: spacing.sm,
    marginTop: spacing.md,
  },
  cancelBtn: {
    flex: 1,
    height: 44,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  cancelBtnText: {
    fontSize: fontSizes.md,
    fontWeight: '600',
  },
  saveBtn: {
    flex: 1,
    height: 44,
    borderRadius: borderRadius.lg,
    justifyContent: 'center',
    alignItems: 'center',
  },
  saveBtnText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
});
