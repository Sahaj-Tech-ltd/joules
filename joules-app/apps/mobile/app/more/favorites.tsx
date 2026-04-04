import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react';
import { View, Text, StyleSheet, Pressable, FlatList, TextInput, ActivityIndicator, Modal } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { fetchFavorites, deleteFavorite, useFavorite, createMeal } from '@joules/api-client';
import type { FoodFavorite, FoodItem } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

export default function FavoritesScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();

  const [favorites, setFavorites] = useState<FoodFavorite[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [query, setQuery] = useState('');
  const [actionItem, setActionItem] = useState<FoodFavorite | null>(null);
  const [logging, setLogging] = useState(false);

  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const loadFavorites = useCallback(async () => {
    try {
      const data = await fetchFavorites();
      setFavorites(data.sort((a, b) => b.use_count - a.use_count));
    } catch {
      // silently fail
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    loadFavorites();
  }, [loadFavorites]);

  const handleRefresh = useCallback(() => {
    setRefreshing(true);
    loadFavorites();
  }, [loadFavorites]);

  const filtered = useMemo(() => {
    if (!query.trim()) return favorites;
    const q = query.trim().toLowerCase();
    return favorites.filter((f) => f.name.toLowerCase().includes(q));
  }, [favorites, query]);

  const handleSearch = useCallback((text: string) => {
    setQuery(text);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setQuery(text);
    }, 200);
  }, []);

  const logFavorite = useCallback(async (fav: FoodFavorite) => {
    if (logging) return;
    setLogging(true);
    try {
      await useFavorite(fav.id);
      const foods: Partial<FoodItem>[] = [
        {
          name: fav.name,
          calories: fav.calories,
          protein_g: fav.protein_g,
          carbs_g: fav.carbs_g,
          fat_g: fav.fat_g,
          fiber_g: fav.fiber_g,
          serving_size: fav.serving_size,
          source: fav.source,
        },
      ];
      await createMeal({ meal_type: 'snack', foods });
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      setFavorites((prev) =>
        prev.map((f) => (f.id === fav.id ? { ...f, use_count: f.use_count + 1 } : f)).sort((a, b) => b.use_count - a.use_count)
      );
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    } finally {
      setLogging(false);
      setActionItem(null);
    }
  }, [logging]);

  const handleDelete = useCallback(async (fav: FoodFavorite) => {
    try {
      await deleteFavorite(fav.id);
      setFavorites((prev) => prev.filter((f) => f.id !== fav.id));
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    } finally {
      setActionItem(null);
    }
  }, []);

  const renderItem = useCallback(
    ({ item }: { item: FoodFavorite }) => {
      const onPress = () => {
        Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
        logFavorite(item);
      };
      const onLongPress = () => {
        Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
        setActionItem(item);
      };

      return (
        <Pressable
          onPress={onPress}
          onLongPress={onLongPress}
          delayLongPress={400}
          style={({ pressed }) => [
            styles.card,
            {
              backgroundColor: colors.surface,
              borderColor: colors.border,
              transform: [{ scale: pressed ? 0.98 : 1 }],
            },
          ]}
          android_ripple={{ color: colors.surfaceElevated }}
        >
          <View style={styles.cardTop}>
            <Text style={[styles.cardName, { color: colors.textPrimary }]} numberOfLines={1}>
              {item.name}
            </Text>
            <View style={[styles.useCountBadge, { backgroundColor: colors.primary + '20' }]}>
              <Text style={[styles.useCountText, { color: colors.primary }]}>
                {item.use_count}x
              </Text>
            </View>
          </View>
          <View style={styles.macroRow}>
            <Text style={[styles.calText, { color: colors.textPrimary }]}>
              {item.calories} cal
            </Text>
            <Text style={[styles.macroText, { color: colors.macroProtein }]}>
              P {item.protein_g}g
            </Text>
            <Text style={[styles.macroText, { color: colors.macroCarbs }]}>
              C {item.carbs_g}g
            </Text>
            <Text style={[styles.macroText, { color: colors.macroFat }]}>
              F {item.fat_g}g
            </Text>
          </View>
        </Pressable>
      );
    },
    [colors, logFavorite]
  );

  if (loading) {
    return (
      <View style={[styles.loadingWrap, { backgroundColor: colors.background }]}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  const ListEmptyComponent = () => (
    <View style={styles.emptyWrap}>
      <Ionicons name="heart-outline" size={48} color={colors.textTertiary} />
      <Text style={[styles.emptyTitle, { color: colors.textSecondary }]}>
        No favorites yet
      </Text>
      <Text style={[styles.emptySubtitle, { color: colors.textTertiary }]}>
        Foods you log frequently will appear here
      </Text>
    </View>
  );

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      <View style={[styles.header, { borderBottomColor: colors.border }]}>
        <Pressable onPress={() => router.back()} style={styles.backBtn} hitSlop={12}>
          <Ionicons name="chevron-back" size={24} color={colors.textPrimary} />
        </Pressable>
        <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Favorites</Text>
        <View style={styles.headerSpacer} />
      </View>

      <View style={[styles.searchWrap, { backgroundColor: colors.background }]}>
        <View style={[styles.searchBar, { backgroundColor: colors.surfaceElevated }]}>
          <Ionicons name="search" size={18} color={colors.textTertiary} />
          <TextInput
            style={[styles.searchInput, { color: colors.textPrimary }]}
            placeholder="Search favorites..."
            placeholderTextColor={colors.textTertiary}
            value={query}
            onChangeText={handleSearch}
            returnKeyType="search"
            autoCapitalize="none"
            autoCorrect={false}
          />
          {query.length > 0 ? (
            <Pressable onPress={() => setQuery('')} hitSlop={8}>
              <Ionicons name="close-circle" size={18} color={colors.textTertiary} />
            </Pressable>
          ) : null}
        </View>
      </View>

      <FlatList
        data={filtered}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        contentContainerStyle={filtered.length === 0 ? styles.listContentEmpty : styles.listContent}
        refreshing={refreshing}
        onRefresh={handleRefresh}
        ListEmptyComponent={ListEmptyComponent}
        keyboardShouldPersistTaps="handled"
      />

      <Modal
        visible={actionItem !== null}
        transparent
        animationType="fade"
        onRequestClose={() => setActionItem(null)}
      >
        <Pressable
          style={styles.sheetBackdrop}
          onPress={() => setActionItem(null)}
        />
        <View style={[styles.sheetContainer, { backgroundColor: colors.surface }]}>
          <View style={[styles.sheetHandle, { backgroundColor: colors.border }]} />
          <Text style={[styles.sheetTitle, { color: colors.textPrimary }]} numberOfLines={1}>
            {actionItem?.name}
          </Text>
          <Pressable
            onPress={() => actionItem && logFavorite(actionItem)}
            disabled={logging}
            style={({ pressed }) => [
              styles.sheetOption,
              { backgroundColor: pressed ? colors.surfaceElevated : 'transparent' },
            ]}
          >
            <Ionicons name="restaurant-outline" size={22} color={colors.primary} />
            <Text style={[styles.sheetOptionLabel, { color: colors.textPrimary }]}>
              Log It
            </Text>
          </Pressable>
          <Pressable
            onPress={() => actionItem && handleDelete(actionItem)}
            style={({ pressed }) => [
              styles.sheetOption,
              { backgroundColor: pressed ? colors.surfaceElevated : 'transparent' },
            ]}
          >
            <Ionicons name="trash-outline" size={22} color="#EF4444" />
            <Text style={[styles.sheetOptionLabel, { color: '#EF4444' }]}>
              Delete
            </Text>
          </Pressable>
          <Pressable
            onPress={() => setActionItem(null)}
            style={({ pressed }) => [
              styles.sheetOption,
              { backgroundColor: pressed ? colors.surfaceElevated : 'transparent' },
            ]}
          >
            <Ionicons name="close-outline" size={22} color={colors.textTertiary} />
            <Text style={[styles.sheetOptionLabel, { color: colors.textSecondary }]}>
              Cancel
            </Text>
          </Pressable>
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
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: 1,
  },
  backBtn: {
    padding: spacing.xs,
    marginLeft: -spacing.xs,
  },
  headerTitle: {
    flex: 1,
    textAlign: 'center',
    fontSize: fontSizes.lg,
    fontWeight: '700',
  },
  headerSpacer: {
    width: 32,
  },
  searchWrap: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
  },
  searchBar: {
    flexDirection: 'row',
    alignItems: 'center',
    height: 42,
    borderRadius: borderRadius.lg,
    paddingHorizontal: spacing.md,
    gap: spacing.sm,
  },
  searchInput: {
    flex: 1,
    fontSize: fontSizes.md,
    height: 42,
  },
  listContent: {
    paddingHorizontal: spacing.lg,
    paddingBottom: spacing['2xl'],
  },
  listContentEmpty: {
    flexGrow: 1,
    paddingHorizontal: spacing.lg,
  },
  card: {
    padding: spacing.md,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    marginBottom: spacing.sm,
  },
  cardTop: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: spacing.xs,
  },
  cardName: {
    flex: 1,
    fontSize: fontSizes.md,
    fontWeight: '600',
    marginRight: spacing.sm,
  },
  useCountBadge: {
    paddingHorizontal: spacing.sm,
    paddingVertical: 2,
    borderRadius: borderRadius.full,
  },
  useCountText: {
    fontSize: fontSizes.xs,
    fontWeight: '700',
  },
  macroRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.md,
  },
  calText: {
    fontSize: fontSizes.xs,
    fontWeight: '600',
  },
  macroText: {
    fontSize: fontSizes.xs,
    fontWeight: '500',
  },
  emptyWrap: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.md,
    paddingVertical: spacing['4xl'],
  },
  emptyTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '600',
  },
  emptySubtitle: {
    fontSize: fontSizes.sm,
    textAlign: 'center',
    paddingHorizontal: spacing.xl,
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
    fontSize: fontSizes.md,
    fontWeight: '600',
    marginBottom: spacing.sm,
    paddingHorizontal: spacing.sm,
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
