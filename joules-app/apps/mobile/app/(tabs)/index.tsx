import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  StyleSheet,
  ActivityIndicator,
  Pressable,
  RefreshControl,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import {
  fetchDashboardSummary,
  fetchHabitPhase,
  fetchWeightLogs,
  fetchCoachMessages,
  fetchWaterLogs,
  logWater,
  fetchTopFavorites,
  fetchProfile,
  createMeal,
} from '@joules/api-client';
import { useColorScheme } from '@/hooks/useColorScheme';
import CalorieRing from '@/components/CalorieRing';
import MacroBar from '@/components/MacroBar';
import IdentityQuoteCard from '@/components/IdentityQuoteCard';
import ConsistencyRing from '@/components/ConsistencyRing';
import HabitPhaseIndicator from '@/components/HabitPhaseIndicator';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

function getGreeting(name?: string): string {
  const hour = new Date().getHours();
  let timeStr = 'Good morning';
  if (hour >= 12 && hour < 17) timeStr = 'Good afternoon';
  if (hour >= 17) timeStr = 'Good evening';
  const firstName = name ? name.split(' ')[0] : null;
  return firstName ? `${timeStr}, ${firstName}` : timeStr;
}

export default function HomeScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();
  const queryClient = useQueryClient();
  const [refreshing, setRefreshing] = useState(false);

  const { data: profile } = useQuery({
    queryKey: ['userProfile'],
    queryFn: fetchProfile,
    staleTime: 1000 * 60 * 10,
  });

  const { data: dashboard, isLoading: dashLoading, refetch: refetchDash } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => fetchDashboardSummary(),
  });

  const { data: phase, refetch: refetchPhase } = useQuery({
    queryKey: ['habits-phase'],
    queryFn: fetchHabitPhase,
  });

  const { data: weightLogs, refetch: refetchWeight } = useQuery({
    queryKey: ['weight-7d'],
    queryFn: () => fetchWeightLogs(7),
  });

  const { data: coachMsgs, refetch: refetchCoach } = useQuery({
    queryKey: ['coach-latest'],
    queryFn: () => fetchCoachMessages(1),
  });

  const { data: waterLogs, refetch: refetchWater } = useQuery({
    queryKey: ['water-today'],
    queryFn: () => fetchWaterLogs(),
  });

  const { data: favorites, refetch: refetchFavorites } = useQuery({
    queryKey: ['favorites-top'],
    queryFn: () => fetchTopFavorites(6),
  });

  const handleRefresh = async () => {
    setRefreshing(true);
    await Promise.all([
      refetchDash(),
      refetchPhase(),
      refetchWeight(),
      refetchCoach(),
      refetchWater(),
      refetchFavorites(),
    ]);
    setRefreshing(false);
  };

  const handleQuickWater = async () => {
    try {
      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
      await logWater(250);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      queryClient.invalidateQueries({ queryKey: ['water-today'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  };

  const handleQuickLogFavorite = async (fav: any) => {
    try {
      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
      await createMeal({
        meal_type: 'snack',
        foods: [
          {
            name: fav.name,
            calories: fav.calories,
            protein_g: fav.protein_g ?? 0,
            carbs_g: fav.carbs_g ?? 0,
            fat_g: fav.fat_g ?? 0,
            fiber_g: fav.fiber_g ?? 0,
            source: 'quick_fav',
          },
        ],
      });
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      Alert.alert('Logged', `Added "${fav.name}" to today's log.`);
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['meals'] });
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  };

  if (dashLoading) {
    return (
      <View style={[styles.loadingWrap, { backgroundColor: colors.background }]}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  const consumed = dashboard?.calories_consumed ?? 0;
  const target = dashboard?.calorie_target ?? 2000;
  const remaining = Math.max(0, target - consumed);
  const waterMl = (waterLogs ?? []).reduce((acc: number, w: any) => acc + (w.amount_ml || 0), 0) || (dashboard?.water_ml ?? 0);
  const waterTarget = 2500;

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.content}
        showsVerticalScrollIndicator={false}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} tintColor={colors.primary} />
        }
      >
        {/* Header Greeting & Date */}
        <View style={styles.headerRow}>
          <View>
            <Text style={[styles.greeting, { color: colors.textPrimary }]}>
              {getGreeting(profile?.name)}
            </Text>
            <Text style={[styles.dateSubtitle, { color: colors.textSecondary }]}>
              {new Date().toLocaleDateString('en-US', {
                weekday: 'long',
                month: 'short',
                day: 'numeric',
              })}
            </Text>
          </View>
          <Pressable
            style={[styles.profileButton, { backgroundColor: colors.surface, borderColor: colors.border }]}
            onPress={() => router.push('/more')}
          >
            <Ionicons name="person-circle-outline" size={32} color={colors.primary} />
          </Pressable>
        </View>

        {/* Quick Action Chips */}
        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.quickActionsContainer}
        >
          <Pressable
            style={({ pressed }) => [
              styles.quickActionChip,
              { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.7 : 1 },
            ]}
            onPress={() => {
              Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
              router.push('/log/camera');
            }}
          >
            <View style={[styles.chipIcon, { backgroundColor: '#38bdf820' }]}>
              <Ionicons name="camera" size={16} color="#38bdf8" />
            </View>
            <Text style={[styles.chipText, { color: colors.textPrimary }]}>Photo AI</Text>
          </Pressable>

          <Pressable
            style={({ pressed }) => [
              styles.quickActionChip,
              { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.7 : 1 },
            ]}
            onPress={() => {
              Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
              router.push('/log/barcode');
            }}
          >
            <View style={[styles.chipIcon, { backgroundColor: '#a78bfa20' }]}>
              <Ionicons name="barcode" size={16} color="#a78bfa" />
            </View>
            <Text style={[styles.chipText, { color: colors.textPrimary }]}>Barcode</Text>
          </Pressable>

          <Pressable
            style={({ pressed }) => [
              styles.quickActionChip,
              { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.7 : 1 },
            ]}
            onPress={() => {
              Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
              router.push('/log/search');
            }}
          >
            <View style={[styles.chipIcon, { backgroundColor: '#10b98120' }]}>
              <Ionicons name="search" size={16} color="#10b981" />
            </View>
            <Text style={[styles.chipText, { color: colors.textPrimary }]}>Search</Text>
          </Pressable>

          <Pressable
            style={({ pressed }) => [
              styles.quickActionChip,
              { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.7 : 1 },
            ]}
            onPress={handleQuickWater}
          >
            <View style={[styles.chipIcon, { backgroundColor: '#3b82f620' }]}>
              <Ionicons name="water" size={16} color="#3b82f6" />
            </View>
            <Text style={[styles.chipText, { color: colors.textPrimary }]}>+250ml</Text>
          </Pressable>
        </ScrollView>

        {/* Identity Quote Banner */}
        <View style={styles.section}>
          <IdentityQuoteCard />
        </View>

        {/* Calorie Ring Card */}
        <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <View style={styles.cardHeader}>
            <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Calories</Text>
            <View style={[styles.remainingBadge, { backgroundColor: `${colors.primary}20` }]}>
              <Text style={[styles.remainingText, { color: colors.primary }]}>
                {remaining} cal left
              </Text>
            </View>
          </View>
          <CalorieRing consumed={consumed} target={target} />
        </View>

        {/* Macros Breakdown Card */}
        <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Macronutrients</Text>
          <MacroBar
            label="Protein"
            consumed={dashboard?.protein_consumed ?? 0}
            target={dashboard?.protein_target ?? 150}
            color={colors.macroProtein}
          />
          <MacroBar
            label="Carbs"
            consumed={dashboard?.carbs_consumed ?? 0}
            target={dashboard?.carbs_target ?? 200}
            color={colors.macroCarbs}
          />
          <MacroBar
            label="Fat"
            consumed={dashboard?.fat_consumed ?? 0}
            target={dashboard?.fat_target ?? 65}
            color={colors.macroFat}
          />
        </View>

        {/* Consistency & Phase Card */}
        <View style={[styles.rowCard, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <ConsistencyRing
            percentage={phase?.consistency_percentage ?? 0}
            graceUsed={phase?.grace_days_used_this_week ?? 0}
            graceMax={phase?.grace_days_max_per_week ?? 2}
          />
          {phase && (
            <HabitPhaseIndicator phase={phase.phase} totalDays={phase.total_days} />
          )}
        </View>

        {/* AI Coach Preview Card */}
        <Pressable
          style={({ pressed }) => [
            styles.card,
            { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.85 : 1 },
          ]}
          onPress={() => {
            Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
            router.push('/coach');
          }}
        >
          <View style={styles.coachHeaderRow}>
            <View style={styles.coachHeaderLeft}>
              <View style={[styles.coachAvatarIcon, { backgroundColor: `${colors.primary}20` }]}>
                <Ionicons name="sparkles" size={18} color={colors.primary} />
              </View>
              <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>AI Coach</Text>
            </View>
            <View style={styles.coachActionLink}>
              <Text style={[styles.coachLinkText, { color: colors.primary }]}>Chat</Text>
              <Ionicons name="chevron-forward" size={14} color={colors.primary} />
            </View>
          </View>
          <Text style={[styles.coachMsg, { color: colors.textSecondary }]} numberOfLines={2}>
            {coachMsgs && coachMsgs.length > 0 && coachMsgs[0]?.content
              ? coachMsgs[0].content
              : "Ask me anything about today's nutrition, recipes, or meal timing."}
          </Text>
        </Pressable>

        {/* Water Card */}
        <Pressable
          style={({ pressed }) => [
            styles.card,
            { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.85 : 1 },
          ]}
          onPress={() => {
            Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
            router.push('/progress');
          }}
        >
          <View style={styles.cardHeader}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6 }}>
              <Ionicons name="water-outline" size={18} color="#3b82f6" />
              <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Hydration</Text>
            </View>
            <Text style={[styles.waterLabel, { color: colors.textSecondary }]}>
              {waterMl} / {waterTarget} ml
            </Text>
          </View>
          <View style={styles.waterRow}>
            <View style={[styles.waterTrack, { backgroundColor: colors.surfaceElevated }]}>
              <View
                style={[
                  styles.waterFill,
                  {
                    width: `${Math.min((waterMl / waterTarget) * 100, 100)}%`,
                    backgroundColor: '#3b82f6',
                  },
                ]}
              />
            </View>
          </View>
        </Pressable>

        {/* Quick Add Favorites */}
        {Array.isArray(favorites) && favorites.length > 0 && (
          <View style={styles.section}>
            <View style={styles.favHeaderRow}>
              <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Quick Favorites</Text>
              <Pressable onPress={() => router.push('/more/favorites')}>
                <Text style={{ color: colors.primary, fontSize: 13, fontWeight: '600' }}>View All</Text>
              </Pressable>
            </View>
            <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.favScroll}>
              {favorites.slice(0, 6).map((fav: any) => (
                <Pressable
                  key={fav.id ?? fav.name}
                  style={({ pressed }) => [
                    styles.favChip,
                    {
                      backgroundColor: colors.surfaceElevated,
                      borderColor: colors.border,
                      opacity: pressed ? 0.7 : 1,
                    },
                  ]}
                  onPress={() => handleQuickLogFavorite(fav)}
                >
                  <Ionicons name="add-circle" size={16} color={colors.primary} style={{ marginRight: 4 }} />
                  <Text style={[styles.favName, { color: colors.textPrimary }]} numberOfLines={1}>
                    {fav.name}
                  </Text>
                  <Text style={[styles.favCals, { color: colors.textTertiary }]}>
                    {fav.calories}cal
                  </Text>
                </Pressable>
              ))}
            </ScrollView>
          </View>
        )}

        <View style={styles.bottomPadding} />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
  },
  scroll: {
    flex: 1,
  },
  content: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.xs,
  },
  loadingWrap: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  headerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: spacing.sm,
  },
  greeting: {
    fontSize: 26,
    fontWeight: '800',
    letterSpacing: -0.5,
  },
  dateSubtitle: {
    fontSize: 13,
    marginTop: 2,
    fontWeight: '500',
  },
  profileButton: {
    width: 44,
    height: 44,
    borderRadius: 22,
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
  },
  quickActionsContainer: {
    paddingVertical: spacing.sm,
    gap: spacing.sm,
  },
  quickActionChip: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: borderRadius.full,
    borderWidth: 1,
    gap: 6,
  },
  chipIcon: {
    width: 24,
    height: 24,
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  chipText: {
    fontSize: 13,
    fontWeight: '600',
  },
  section: {
    marginVertical: spacing.xs,
  },
  card: {
    borderRadius: borderRadius.xl,
    padding: spacing.lg,
    borderWidth: 1,
    marginVertical: spacing.xs,
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: spacing.xs,
  },
  remainingBadge: {
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: borderRadius.full,
  },
  remainingText: {
    fontSize: 11,
    fontWeight: '700',
  },
  rowCard: {
    flexDirection: 'row',
    borderRadius: borderRadius.xl,
    padding: spacing.md,
    borderWidth: 1,
    alignItems: 'center',
    justifyContent: 'space-around',
    marginVertical: spacing.xs,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: '700',
    letterSpacing: -0.2,
  },
  coachHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 6,
  },
  coachHeaderLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  coachAvatarIcon: {
    width: 28,
    height: 28,
    borderRadius: 14,
    alignItems: 'center',
    justifyContent: 'center',
  },
  coachActionLink: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 2,
  },
  coachLinkText: {
    fontSize: 13,
    fontWeight: '700',
  },
  coachMsg: {
    fontSize: 13,
    lineHeight: 18,
    marginTop: 2,
  },
  waterRow: {
    marginTop: spacing.xs,
  },
  waterTrack: {
    height: 10,
    borderRadius: 5,
    overflow: 'hidden',
  },
  waterFill: {
    height: '100%',
    borderRadius: 5,
  },
  waterLabel: {
    fontSize: 12,
    fontWeight: '600',
  },
  favHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: spacing.xs,
  },
  favScroll: {
    marginVertical: spacing.xs,
  },
  favChip: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: borderRadius.lg,
    paddingHorizontal: 12,
    paddingVertical: 8,
    marginRight: spacing.sm,
    borderWidth: 1,
  },
  favName: {
    fontSize: 13,
    fontWeight: '600',
    maxWidth: 100,
  },
  favCals: {
    fontSize: 11,
    marginLeft: 6,
  },
  bottomPadding: {
    height: 40,
  },
});
