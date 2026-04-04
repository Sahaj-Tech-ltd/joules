import React from 'react';
import { View, Text, ScrollView, StyleSheet, ActivityIndicator } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery } from '@tanstack/react-query';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { fetchDashboardSummary, fetchHabitPhase, fetchIdentityQuote, fetchWeightLogs, fetchCoachMessages, fetchWaterLogs, fetchTopFavorites } from '@joules/api-client';
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

export default function HomeScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  const { data: dashboard, isLoading: dashLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => fetchDashboardSummary(),
  });

  const { data: phase } = useQuery({
    queryKey: ['habits-phase'],
    queryFn: fetchHabitPhase,
  });

  const { data: weightLogs } = useQuery({
    queryKey: ['weight-7d'],
    queryFn: () => fetchWeightLogs(7),
  });

  const { data: coachMsgs } = useQuery({
    queryKey: ['coach-latest'],
    queryFn: () => fetchCoachMessages(1),
  });

  const { data: waterLogs } = useQuery({
    queryKey: ['water-today'],
    queryFn: () => fetchWaterLogs(),
  });

  const { data: favorites } = useQuery({
    queryKey: ['favorites-top'],
    queryFn: () => fetchTopFavorites(6),
  });

  if (dashLoading) {
    return (
      <View style={[styles.loadingWrap, { backgroundColor: colors.background }]}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  const consumed = dashboard?.calories_consumed ?? 0;
  const target = dashboard?.calorie_target ?? 2000;
  const waterMl = dashboard?.water_ml ?? 0;
  const waterTarget = 2000;

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.content}
        showsVerticalScrollIndicator={false}
      >
        <Text style={[styles.greeting, { color: colors.textPrimary }]}>Good morning</Text>

        <View style={styles.section}>
          <IdentityQuoteCard />
        </View>

        <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <CalorieRing consumed={consumed} target={target} />
        </View>

        <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Macros</Text>
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

        {weightLogs && weightLogs.length >= 2 && (
          <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Weight</Text>
            <View style={styles.sparkline}>
              {weightLogs.map((log, i) => {
                const weights = weightLogs.map(w => w.weight_kg);
                const min = Math.min(...weights);
                const max = Math.max(...weights);
                const range = max - min || 1;
                const height = ((log.weight_kg - min) / range) * 40 + 4;
                return (
                  <View
                    key={log.id}
                    style={[
                      styles.sparkBar,
                      {
                        height,
                        backgroundColor: i === weightLogs.length - 1 ? colors.primary : colors.textTertiary + '40',
                      },
                    ]}
                  />
                );
              })}
            </View>
            <Text style={[styles.sparkLabel, { color: colors.textSecondary }]}>
              {weightLogs[weightLogs.length - 1]?.weight_kg?.toFixed(1)} kg
            </Text>
          </View>
        )}

        {coachMsgs && coachMsgs.length > 0 && (
          <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Coach</Text>
            <Text style={[styles.coachMsg, { color: colors.textSecondary }]} numberOfLines={3}>
              {coachMsgs[0].content ?? ''}
            </Text>
          </View>
        )}

        <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Water</Text>
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
            <Text style={[styles.waterLabel, { color: colors.textSecondary }]}>
              {waterMl}ml / {waterTarget}ml
            </Text>
          </View>
        </View>

        {Array.isArray(favorites) && favorites.length > 0 && (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, { color: colors.textPrimary }]}>Quick Add</Text>
            <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.favScroll}>
              {favorites.slice(0, 6).map((fav: any) => (
                <View
                  key={fav.id ?? fav.name}
                  style={[styles.favChip, { backgroundColor: colors.surfaceElevated, borderColor: colors.border }]}
                >
                  <Text style={[styles.favName, { color: colors.textPrimary }]} numberOfLines={1}>
                    {fav.name}
                  </Text>
                  <Text style={[styles.favCals, { color: colors.textTertiary }]}>
                    {fav.calories}cal
                  </Text>
                </View>
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
  },
  loadingWrap: {
    flex: 1,
    justifyContent: 'center' as const,
    alignItems: 'center' as const,
  },
  greeting: {
    fontSize: 28,
    fontWeight: '700',
    marginTop: spacing.md,
    marginBottom: spacing.md,
  },
  section: {
    marginBottom: spacing.md,
  },
  card: {
    padding: spacing.lg,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    marginBottom: spacing.md,
  },
  rowCard: {
    padding: spacing.lg,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    marginBottom: spacing.md,
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    justifyContent: 'space-between' as const,
  },
  sectionTitle: {
    fontSize: 15,
    fontWeight: '700',
    marginBottom: spacing.sm,
  },
  sparkline: {
    flexDirection: 'row' as const,
    alignItems: 'flex-end' as const,
    height: 48,
    gap: 6,
  },
  sparkBar: {
    flex: 1,
    borderRadius: 3,
    minHeight: 4,
  },
  sparkLabel: {
    fontSize: 13,
    marginTop: spacing.xs,
    fontWeight: '600',
  },
  coachMsg: {
    fontSize: 14,
    lineHeight: 20,
  },
  waterRow: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    gap: spacing.sm,
  },
  waterTrack: {
    flex: 1,
    height: 10,
    borderRadius: borderRadius.full,
    overflow: 'hidden' as const,
  },
  waterFill: {
    height: '100%' as const,
    borderRadius: borderRadius.full,
    minWidth: 4,
  },
  waterLabel: {
    fontSize: 12,
    fontWeight: '600',
    minWidth: 90,
    textAlign: 'right' as const,
  },
  favScroll: {
    marginTop: spacing.xs,
  },
  favChip: {
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.md,
    borderWidth: 1,
    marginRight: spacing.sm,
    minWidth: 80,
  },
  favName: {
    fontSize: 12,
    fontWeight: '600',
  },
  favCals: {
    fontSize: 10,
    marginTop: 1,
  },
  bottomPadding: {
    height: 100,
  },
});
