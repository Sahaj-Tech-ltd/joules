import React, { useState, useMemo } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Pressable,
  RefreshControl,
  Modal,
  TextInput,
  ActivityIndicator,
  Alert,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import {
  fetchWaterLogs,
  logWater,
  fetchSteps,
  logSteps,
  fetchExercises,
  logExercise,
  fetchWeightLogs,
  logWeight,
  fetchFastingStatus,
  fetchGoals,
  useAuthStore,
} from '@joules/api-client';
import WaterWidget from '@/components/WaterWidget';
import StepsWidget from '@/components/StepsWidget';
import ExerciseWidget from '@/components/ExerciseWidget';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

export default function ProgressScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const queryClient = useQueryClient();
  const token = useAuthStore((s) => s.token);

  const [activeTab, setActiveTab] = useState<'activity' | 'body'>('activity');
  const [refreshing, setRefreshing] = useState(false);

  // Weight Log Modal State
  const [weightModalVisible, setWeightModalVisible] = useState(false);
  const [weightInput, setWeightInput] = useState('');
  const [submittingWeight, setSubmittingWeight] = useState(false);

  // Queries
  const { data: waterLogs = [], refetch: refetchWater } = useQuery({
    queryKey: ['water-today'],
    queryFn: () => fetchWaterLogs(),
    enabled: !!token,
  });

  const { data: stepsData, refetch: refetchSteps } = useQuery({
    queryKey: ['steps-today'],
    queryFn: () => fetchSteps(),
    enabled: !!token,
  });

  const { data: exercises = [], refetch: refetchExercises } = useQuery({
    queryKey: ['exercises-today'],
    queryFn: () => fetchExercises(),
    enabled: !!token,
  });

  const { data: weightLogs = [], refetch: refetchWeight } = useQuery({
    queryKey: ['weight-30d'],
    queryFn: () => fetchWeightLogs(30),
    enabled: !!token,
  });

  const { data: fasting, refetch: refetchFasting } = useQuery({
    queryKey: ['fasting-status'],
    queryFn: () => fetchFastingStatus(),
    enabled: !!token,
  });

  const { data: goals } = useQuery({
    queryKey: ['userGoals'],
    queryFn: fetchGoals,
    enabled: !!token,
  });

  const handleRefresh = async () => {
    setRefreshing(true);
    await Promise.all([
      refetchWater(),
      refetchSteps(),
      refetchExercises(),
      refetchWeight(),
      refetchFasting(),
    ]);
    setRefreshing(false);
  };

  // Water calculations
  const totalWaterMl = useMemo(() => {
    if (Array.isArray(waterLogs)) {
      return waterLogs.reduce((sum: number, log: any) => sum + (log.amount_ml || 0), 0);
    }
    return (waterLogs as any)?.total_ml ?? 0;
  }, [waterLogs]);

  const handleLogWater = async (amount: number) => {
    try {
      await logWater(amount);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      queryClient.invalidateQueries({ queryKey: ['water-today'] });
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  };

  const handleLogSteps = async (steps: number) => {
    try {
      await logSteps(steps);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      queryClient.invalidateQueries({ queryKey: ['steps-today'] });
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  };

  const handleLogExercise = async (ex: { name: string; duration_min: number; calories_burned?: number }) => {
    try {
      await logExercise(ex);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      queryClient.invalidateQueries({ queryKey: ['exercises-today'] });
    } catch {
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  };

  const handleSaveWeight = async () => {
    const val = parseFloat(weightInput.trim());
    if (isNaN(val) || val <= 0) {
      Alert.alert('Invalid Weight', 'Please enter a valid weight.');
      return;
    }
    setSubmittingWeight(true);
    try {
      await logWeight(val);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      setWeightInput('');
      setWeightModalVisible(false);
      queryClient.invalidateQueries({ queryKey: ['weight-30d'] });
      queryClient.invalidateQueries({ queryKey: ['weight-7d'] });
    } catch {
      Alert.alert('Error', 'Failed to log weight. Check server connection.');
    } finally {
      setSubmittingWeight(false);
    }
  };

  const latestWeight = weightLogs[0]?.weight_kg ?? null;

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      {/* Header */}
      <View style={styles.header}>
        <View>
          <Text style={[styles.screenTitle, { color: colors.textPrimary }]}>Progress</Text>
          <Text style={[styles.screenSubtitle, { color: colors.textSecondary }]}>
            Activity & body trends
          </Text>
        </View>

        {/* Segmented Control */}
        <View style={[styles.segmentedWrap, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <Pressable
            style={[
              styles.segmentBtn,
              activeTab === 'activity' && [styles.segmentBtnActive, { backgroundColor: colors.primary }],
            ]}
            onPress={() => {
              Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
              setActiveTab('activity');
            }}
          >
            <Text
              style={[
                styles.segmentText,
                { color: activeTab === 'activity' ? '#fff' : colors.textSecondary },
              ]}
            >
              Activity
            </Text>
          </Pressable>

          <Pressable
            style={[
              styles.segmentBtn,
              activeTab === 'body' && [styles.segmentBtnActive, { backgroundColor: colors.primary }],
            ]}
            onPress={() => {
              Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
              setActiveTab('body');
            }}
          >
            <Text
              style={[
                styles.segmentText,
                { color: activeTab === 'body' ? '#fff' : colors.textSecondary },
              ]}
            >
              Body & Fasting
            </Text>
          </Pressable>
        </View>
      </View>

      <ScrollView
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} tintColor={colors.primary} />
        }
      >
        {activeTab === 'activity' ? (
          <View style={styles.tabContent}>
            {/* Water Tracking */}
            <View style={styles.widgetWrapper}>
              <WaterWidget
                consumed={totalWaterMl}
                target={2500}
                onLog={handleLogWater}
              />
            </View>

            {/* Steps Tracking */}
            <View style={styles.widgetWrapper}>
              <StepsWidget
                steps={stepsData?.steps || 0}
                goal={10000}
                source={stepsData?.source || 'Manual'}
                onLog={handleLogSteps}
              />
            </View>

            {/* Exercise Tracking */}
            <View style={styles.widgetWrapper}>
              <ExerciseWidget
                exercises={exercises}
                onLog={handleLogExercise}
              />
            </View>
          </View>
        ) : (
          <View style={styles.tabContent}>
            {/* Weight Section Card */}
            <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              <View style={styles.cardHeaderRow}>
                <View style={styles.cardHeaderLeft}>
                  <View style={[styles.iconCircle, { backgroundColor: '#38bdf820' }]}>
                    <Ionicons name="scale-outline" size={20} color="#38bdf8" />
                  </View>
                  <Text style={[styles.cardTitle, { color: colors.textPrimary }]}>Body Weight</Text>
                </View>
                <Pressable
                  style={[styles.smallLogBtn, { backgroundColor: colors.primary }]}
                  onPress={() => {
                    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                    setWeightModalVisible(true);
                  }}
                >
                  <Ionicons name="add" size={16} color="#fff" />
                  <Text style={styles.smallLogBtnText}>Log Weight</Text>
                </Pressable>
              </View>

              <View style={styles.weightValueRow}>
                <Text style={[styles.weightBigNumber, { color: colors.textPrimary }]}>
                  {latestWeight ? latestWeight.toFixed(1) : '--'}
                </Text>
                <Text style={[styles.weightUnit, { color: colors.textSecondary }]}>kg</Text>
              </View>

              <Text style={[styles.weightSubtext, { color: colors.textTertiary }]}>
                {weightLogs.length > 0
                  ? `${weightLogs.length} logs in the past 30 days`
                  : 'No weight logged recently. Tap Log Weight to begin tracking.'}
              </Text>

              {weightLogs.length > 0 && (
                <View style={styles.recentLogsList}>
                  {weightLogs.slice(0, 4).map((wl) => (
                    <View
                      key={wl.id}
                      style={[styles.recentLogRow, { borderTopColor: colors.border }]}
                    >
                      <Text style={[styles.recentLogDate, { color: colors.textSecondary }]}>
                        {new Date(wl.date).toLocaleDateString(undefined, {
                          month: 'short',
                          day: 'numeric',
                        })}
                      </Text>
                      <Text style={[styles.recentLogWeight, { color: colors.textPrimary }]}>
                        {wl.weight_kg.toFixed(1)} kg
                      </Text>
                    </View>
                  ))}
                </View>
              )}
            </View>

            {/* Intermittent Fasting Card */}
            <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              <View style={styles.cardHeaderRow}>
                <View style={styles.cardHeaderLeft}>
                  <View style={[styles.iconCircle, { backgroundColor: '#f59e0b20' }]}>
                    <Ionicons name="time-outline" size={20} color="#f59e0b" />
                  </View>
                  <Text style={[styles.cardTitle, { color: colors.textPrimary }]}>Intermittent Fasting</Text>
                </View>
                <View
                  style={[
                    styles.fastingPill,
                    {
                      backgroundColor: fasting?.is_fasting ? '#10b98120' : `${colors.primary}20`,
                    },
                  ]}
                >
                  <Text
                    style={[
                      styles.fastingPillText,
                      { color: fasting?.is_fasting ? '#10b981' : colors.primary },
                    ]}
                  >
                    {fasting?.is_fasting ? 'FASTING' : 'EATING WINDOW'}
                  </Text>
                </View>
              </View>

              <View style={styles.fastingStatsGrid}>
                <View style={styles.fastingStat}>
                  <Text style={[styles.fastingStatValue, { color: colors.textPrimary }]}>
                    {fasting?.fasting_streak || goals?.fasting_streak || 0}
                  </Text>
                  <Text style={[styles.fastingStatLabel, { color: colors.textSecondary }]}>
                    Day Streak
                  </Text>
                </View>
                <View style={[styles.fastingDivider, { backgroundColor: colors.border }]} />
                <View style={styles.fastingStat}>
                  <Text style={[styles.fastingStatValue, { color: colors.textPrimary }]}>
                    {fasting?.eating_window_hours ? `${fasting.eating_window_hours}h` : '8h'}
                  </Text>
                  <Text style={[styles.fastingStatLabel, { color: colors.textSecondary }]}>
                    Eating Window
                  </Text>
                </View>
                <View style={[styles.fastingDivider, { backgroundColor: colors.border }]} />
                <View style={styles.fastingStat}>
                  <Text style={[styles.fastingStatValue, { color: colors.textPrimary }]}>
                    {goals?.fasting_window || '16:8'}
                  </Text>
                  <Text style={[styles.fastingStatLabel, { color: colors.textSecondary }]}>
                    Protocol
                  </Text>
                </View>
              </View>
            </View>
          </View>
        )}
      </ScrollView>

      {/* Log Weight Modal */}
      <Modal
        visible={weightModalVisible}
        transparent
        animationType="fade"
        onRequestClose={() => setWeightModalVisible(false)}
      >
        <View style={styles.modalBackdrop}>
          <View style={[styles.modalCard, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>Log Today's Weight</Text>
            <Text style={[styles.modalSub, { color: colors.textSecondary }]}>
              Enter your morning weigh-in in kilograms:
            </Text>

            <TextInput
              style={[
                styles.modalInput,
                { backgroundColor: colors.background, color: colors.textPrimary, borderColor: colors.border },
              ]}
              value={weightInput}
              onChangeText={setWeightInput}
              placeholder="e.g. 75.4"
              placeholderTextColor={colors.textTertiary}
              keyboardType="decimal-pad"
              autoFocus
            />

            <View style={styles.modalBtnRow}>
              <Pressable
                style={[styles.modalBtnCancel, { borderColor: colors.border }]}
                onPress={() => setWeightModalVisible(false)}
              >
                <Text style={{ color: colors.textSecondary, fontWeight: '600' }}>Cancel</Text>
              </Pressable>

              <Pressable
                style={[styles.modalBtnSave, { backgroundColor: colors.primary }]}
                onPress={handleSaveWeight}
                disabled={submittingWeight}
              >
                {submittingWeight ? (
                  <ActivityIndicator size="small" color="#fff" />
                ) : (
                  <Text style={{ color: '#fff', fontWeight: '700' }}>Save Weight</Text>
                )}
              </Pressable>
            </View>
          </View>
        </View>
      </Modal>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
  },
  header: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: spacing.md,
    gap: spacing.md,
  },
  screenTitle: {
    fontSize: 28,
    fontWeight: '800',
    letterSpacing: -0.5,
  },
  screenSubtitle: {
    fontSize: 14,
    marginTop: 2,
  },
  segmentedWrap: {
    flexDirection: 'row',
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    padding: 3,
  },
  segmentBtn: {
    flex: 1,
    paddingVertical: 8,
    alignItems: 'center',
    borderRadius: borderRadius.md,
  },
  segmentBtnActive: {
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.15,
    shadowRadius: 3,
  },
  segmentText: {
    fontSize: 13,
    fontWeight: '700',
  },
  scrollContent: {
    paddingHorizontal: spacing.lg,
    paddingBottom: spacing['2xl'],
  },
  tabContent: {
    gap: spacing.lg,
    marginTop: spacing.xs,
  },
  widgetWrapper: {
    width: '100%',
  },
  card: {
    borderRadius: borderRadius.xl,
    borderWidth: 1,
    padding: spacing.lg,
  },
  cardHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  cardHeaderLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
  },
  iconCircle: {
    width: 34,
    height: 34,
    borderRadius: 17,
    alignItems: 'center',
    justifyContent: 'center',
  },
  cardTitle: {
    fontSize: 17,
    fontWeight: '700',
  },
  smallLogBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: borderRadius.full,
    gap: 4,
  },
  smallLogBtnText: {
    color: '#fff',
    fontSize: 12,
    fontWeight: '700',
  },
  weightValueRow: {
    flexDirection: 'row',
    alignItems: 'baseline',
    marginTop: spacing.md,
    gap: 4,
  },
  weightBigNumber: {
    fontSize: 40,
    fontWeight: '800',
    letterSpacing: -1,
  },
  weightUnit: {
    fontSize: 18,
    fontWeight: '600',
  },
  weightSubtext: {
    fontSize: 12,
    marginTop: 4,
  },
  recentLogsList: {
    marginTop: spacing.md,
  },
  recentLogRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 8,
    borderTopWidth: 1,
  },
  recentLogDate: {
    fontSize: 13,
  },
  recentLogWeight: {
    fontSize: 14,
    fontWeight: '700',
  },
  fastingPill: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: borderRadius.full,
  },
  fastingPillText: {
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 0.5,
  },
  fastingStatsGrid: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    alignItems: 'center',
    marginTop: spacing.lg,
    paddingTop: spacing.sm,
  },
  fastingStat: {
    alignItems: 'center',
    flex: 1,
  },
  fastingStatValue: {
    fontSize: 20,
    fontWeight: '800',
  },
  fastingStatLabel: {
    fontSize: 12,
    marginTop: 3,
  },
  fastingDivider: {
    width: 1,
    height: 32,
  },
  modalBackdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.65)',
    justifyContent: 'center',
    padding: spacing.lg,
  },
  modalCard: {
    borderRadius: borderRadius.xl,
    padding: spacing.xl,
    borderWidth: 1,
  },
  modalTitle: {
    fontSize: 19,
    fontWeight: '700',
    marginBottom: 4,
  },
  modalSub: {
    fontSize: 13,
    marginBottom: spacing.lg,
  },
  modalInput: {
    borderWidth: 1,
    borderRadius: borderRadius.lg,
    padding: 14,
    fontSize: 18,
    marginBottom: spacing.xl,
    fontWeight: '700',
  },
  modalBtnRow: {
    flexDirection: 'row',
    gap: spacing.md,
  },
  modalBtnCancel: {
    flex: 1,
    padding: 14,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    alignItems: 'center',
  },
  modalBtnSave: {
    flex: 1,
    padding: 14,
    borderRadius: borderRadius.lg,
    alignItems: 'center',
    justifyContent: 'center',
  },
});
