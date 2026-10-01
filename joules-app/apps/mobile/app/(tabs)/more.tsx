import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Pressable,
  Alert,
  TextInput,
  Modal,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  light,
  dark,
  oled,
  spacing,
  borderRadius,
  fontSizes,
} from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import {
  useAuthStore,
  clearToken,
  fetchCurrentUser,
  fetchProfile,
  fetchGoals,
  fetchRecipes,
  fetchFavorites,
} from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

export default function MoreScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();
  const queryClient = useQueryClient();

  const baseUrl = useAuthStore((s) => s.baseUrl);
  const setBaseUrl = useAuthStore((s) => s.setBaseUrl);
  const token = useAuthStore((s) => s.token);

  const [serverModalVisible, setServerModalVisible] = useState(false);
  const [newServerUrl, setNewServerUrl] = useState(baseUrl);
  const [testingServer, setTestingServer] = useState(false);

  const { data: user } = useQuery({
    queryKey: ['currentUser'],
    queryFn: fetchCurrentUser,
    enabled: !!token,
    staleTime: 1000 * 60 * 10,
  });

  const { data: profile } = useQuery({
    queryKey: ['userProfile'],
    queryFn: fetchProfile,
    enabled: !!token,
    staleTime: 1000 * 60 * 10,
  });

  const { data: goals } = useQuery({
    queryKey: ['userGoals'],
    queryFn: fetchGoals,
    enabled: !!token,
    staleTime: 1000 * 60 * 10,
  });

  const { data: recipes } = useQuery({
    queryKey: ['recipes-count'],
    queryFn: fetchRecipes,
    enabled: !!token,
    staleTime: 1000 * 60 * 5,
  });

  const { data: favorites } = useQuery({
    queryKey: ['favorites-count'],
    queryFn: fetchFavorites,
    enabled: !!token,
    staleTime: 1000 * 60 * 5,
  });

  const handleLogout = () => {
    Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
    Alert.alert('Sign Out', 'Are you sure you want to sign out?', [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Sign Out',
        style: 'destructive',
        onPress: () => {
          queryClient.clear();
          clearToken();
          router.replace('/auth/login');
        },
      },
    ]);
  };

  const handleSaveServer = async () => {
    if (!newServerUrl.trim()) return;
    setTestingServer(true);
    try {
      const clean = newServerUrl.trim().replace(/\/+$/, '');
      const testUrl = clean.endsWith('/api') ? `${clean}/banners` : `${clean}/api/banners`;
      const res = await fetch(testUrl);
      if (res.ok) {
        setBaseUrl(clean);
        setServerModalVisible(false);
        Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
        Alert.alert('Server Updated', `Connected to: ${useAuthStore.getState().baseUrl}`);
        queryClient.invalidateQueries();
      } else {
        Alert.alert('Error', `Server returned HTTP status ${res.status}`);
      }
    } catch {
      Alert.alert('Connection Failed', 'Could not reach server at this address.');
    } finally {
      setTestingServer(false);
    }
  };

  return (
    <SafeAreaView style={[styles.safe, { backgroundColor: colors.background }]} edges={['top']}>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}
      >
        {/* Header */}
        <View style={styles.header}>
          <Text style={[styles.screenTitle, { color: colors.textPrimary }]}>Profile & Settings</Text>
        </View>

        {/* User Card */}
        <View style={[styles.userCard, { backgroundColor: colors.surface, borderColor: colors.border }]}>
          <View style={[styles.avatarWrap, { backgroundColor: `${colors.primary}20` }]}>
            <Ionicons name="person" size={32} color={colors.primary} />
          </View>
          <View style={styles.userInfo}>
            <Text style={[styles.userName, { color: colors.textPrimary }]}>
              {profile?.name || user?.email?.split('@')[0] || 'Member'}
            </Text>
            <Text style={[styles.userEmail, { color: colors.textSecondary }]}>
              {user?.email || 'Logged In'}
            </Text>
            <View style={styles.planBadgeRow}>
              <View style={[styles.planBadge, { backgroundColor: `${colors.primary}25` }]}>
                <Ionicons name="sparkles" size={12} color={colors.primary} style={{ marginRight: 4 }} />
                <Text style={[styles.planBadgeText, { color: colors.primary }]}>
                  {user?.plan ? user.plan.toUpperCase() : 'FREE'} TIER
                </Text>
              </View>
            </View>
          </View>
        </View>

        {/* Nutrition Tools Section */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.textSecondary }]}>NUTRITION TOOLS</Text>
          <View style={[styles.menuGroup, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Pressable
              style={({ pressed }) => [
                styles.menuItem,
                { borderBottomColor: colors.border, opacity: pressed ? 0.7 : 1 },
              ]}
              onPress={() => {
                Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                router.push('/more/recipes');
              }}
            >
              <View style={[styles.menuIconWrap, { backgroundColor: '#38bdf820' }]}>
                <Ionicons name="restaurant" size={20} color="#38bdf8" />
              </View>
              <View style={styles.menuTextWrap}>
                <Text style={[styles.menuTitle, { color: colors.textPrimary }]}>My Recipes</Text>
                <Text style={[styles.menuSub, { color: colors.textTertiary }]}>
                  {recipes ? `${recipes.length} saved recipes` : 'Custom meals & prep'}
                </Text>
              </View>
              <Ionicons name="chevron-forward" size={18} color={colors.textTertiary} />
            </Pressable>

            <Pressable
              style={({ pressed }) => [
                styles.menuItem,
                { opacity: pressed ? 0.7 : 1 },
              ]}
              onPress={() => {
                Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                router.push('/more/favorites');
              }}
            >
              <View style={[styles.menuIconWrap, { backgroundColor: '#f59e0b20' }]}>
                <Ionicons name="star" size={20} color="#f59e0b" />
              </View>
              <View style={styles.menuTextWrap}>
                <Text style={[styles.menuTitle, { color: colors.textPrimary }]}>Favorite Foods</Text>
                <Text style={[styles.menuSub, { color: colors.textTertiary }]}>
                  {favorites ? `${favorites.length} starred items` : 'One-tap quick add items'}
                </Text>
              </View>
              <Ionicons name="chevron-forward" size={18} color={colors.textTertiary} />
            </Pressable>
          </View>
        </View>

        {/* Daily Goals Summary */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.textSecondary }]}>DAILY TARGETS</Text>
          <View style={[styles.goalsGrid, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <View style={styles.goalItem}>
              <Text style={[styles.goalVal, { color: colors.primary }]}>
                {goals?.daily_calorie_target ? `${goals.daily_calorie_target}` : '2,000'}
              </Text>
              <Text style={[styles.goalLbl, { color: colors.textSecondary }]}>Calories</Text>
            </View>
            <View style={[styles.goalDivider, { backgroundColor: colors.border }]} />
            <View style={styles.goalItem}>
              <Text style={[styles.goalVal, { color: '#38bdf8' }]}>
                {goals?.daily_protein_g ? `${goals.daily_protein_g}g` : '150g'}
              </Text>
              <Text style={[styles.goalLbl, { color: colors.textSecondary }]}>Protein</Text>
            </View>
            <View style={[styles.goalDivider, { backgroundColor: colors.border }]} />
            <View style={styles.goalItem}>
              <Text style={[styles.goalVal, { color: '#fb923c' }]}>
                {goals?.daily_carbs_g ? `${goals.daily_carbs_g}g` : '200g'}
              </Text>
              <Text style={[styles.goalLbl, { color: colors.textSecondary }]}>Carbs</Text>
            </View>
            <View style={[styles.goalDivider, { backgroundColor: colors.border }]} />
            <View style={styles.goalItem}>
              <Text style={[styles.goalVal, { color: '#a78bfa' }]}>
                {goals?.daily_fat_g ? `${goals.daily_fat_g}g` : '65g'}
              </Text>
              <Text style={[styles.goalLbl, { color: colors.textSecondary }]}>Fat</Text>
            </View>
          </View>
        </View>

        {/* Server & Connectivity */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, { color: colors.textSecondary }]}>SERVER CONNECTION</Text>
          <View style={[styles.menuGroup, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Pressable
              style={({ pressed }) => [
                styles.menuItem,
                { opacity: pressed ? 0.7 : 1 },
              ]}
              onPress={() => {
                setNewServerUrl(baseUrl.replace(/\/api$/, ''));
                setServerModalVisible(true);
              }}
            >
              <View style={[styles.menuIconWrap, { backgroundColor: '#10b98120' }]}>
                <Ionicons name="server-outline" size={20} color="#10b981" />
              </View>
              <View style={styles.menuTextWrap}>
                <Text style={[styles.menuTitle, { color: colors.textPrimary }]}>Backend Server</Text>
                <Text style={[styles.menuSub, { color: colors.textTertiary }]} numberOfLines={1}>
                  {baseUrl}
                </Text>
              </View>
              <View style={styles.statusPill}>
                <View style={[styles.statusDot, { backgroundColor: '#10b981' }]} />
                <Text style={[styles.statusText, { color: '#10b981' }]}>Active</Text>
              </View>
            </Pressable>
          </View>
        </View>

        {/* Sign Out Button */}
        <View style={[styles.section, { marginTop: spacing.xl, marginBottom: 40 }]}>
          <Pressable
            style={({ pressed }) => [
              styles.logoutButton,
              { backgroundColor: `${colors.error}15`, borderColor: `${colors.error}30`, opacity: pressed ? 0.7 : 1 },
            ]}
            onPress={handleLogout}
          >
            <Ionicons name="log-out-outline" size={20} color={colors.error} style={{ marginRight: 8 }} />
            <Text style={[styles.logoutText, { color: colors.error }]}>Sign Out</Text>
          </Pressable>
        </View>
      </ScrollView>

      {/* Edit Server URL Modal */}
      <Modal
        visible={serverModalVisible}
        animationType="slide"
        transparent
        onRequestClose={() => setServerModalVisible(false)}
      >
        <View style={styles.modalBackdrop}>
          <View style={[styles.modalCard, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Text style={[styles.modalTitle, { color: colors.textPrimary }]}>Configure Server URL</Text>
            <Text style={[styles.modalDesc, { color: colors.textSecondary }]}>
              Enter your self-hosted Joule backend URL:
            </Text>

            <TextInput
              style={[
                styles.modalInput,
                { backgroundColor: colors.background, color: colors.textPrimary, borderColor: colors.border },
              ]}
              value={newServerUrl}
              onChangeText={setNewServerUrl}
              placeholder="http://192.168.1.100:3000"
              placeholderTextColor={colors.textTertiary}
              autoCapitalize="none"
              autoCorrect={false}
            />

            <View style={styles.modalBtnRow}>
              <Pressable
                style={[styles.modalBtnCancel, { borderColor: colors.border }]}
                onPress={() => setServerModalVisible(false)}
              >
                <Text style={{ color: colors.textSecondary, fontWeight: '600' }}>Cancel</Text>
              </Pressable>

              <Pressable
                style={[styles.modalBtnSave, { backgroundColor: colors.primary }]}
                onPress={handleSaveServer}
                disabled={testingServer}
              >
                {testingServer ? (
                  <ActivityIndicator size="small" color="#fff" />
                ) : (
                  <Text style={{ color: '#fff', fontWeight: '700' }}>Connect & Save</Text>
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
  scrollContent: {
    paddingHorizontal: spacing.lg,
    paddingBottom: spacing['2xl'],
  },
  header: {
    paddingVertical: spacing.md,
  },
  screenTitle: {
    fontSize: 28,
    fontWeight: '800',
    letterSpacing: -0.5,
  },
  userCard: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: spacing.lg,
    borderRadius: borderRadius.xl,
    borderWidth: 1,
    marginTop: spacing.sm,
    gap: spacing.md,
  },
  avatarWrap: {
    width: 60,
    height: 60,
    borderRadius: 30,
    alignItems: 'center',
    justifyContent: 'center',
  },
  userInfo: {
    flex: 1,
  },
  userName: {
    fontSize: 19,
    fontWeight: '700',
  },
  userEmail: {
    fontSize: 13,
    marginTop: 2,
  },
  planBadgeRow: {
    flexDirection: 'row',
    marginTop: 6,
  },
  planBadge: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: borderRadius.full,
  },
  planBadgeText: {
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 0.5,
  },
  section: {
    marginTop: spacing.xl,
  },
  sectionTitle: {
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 0.8,
    marginBottom: spacing.sm,
    paddingLeft: spacing.xs,
  },
  menuGroup: {
    borderRadius: borderRadius.xl,
    borderWidth: 1,
    overflow: 'hidden',
  },
  menuItem: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: 'transparent',
    gap: spacing.md,
  },
  menuIconWrap: {
    width: 38,
    height: 38,
    borderRadius: borderRadius.md,
    alignItems: 'center',
    justifyContent: 'center',
  },
  menuTextWrap: {
    flex: 1,
  },
  menuTitle: {
    fontSize: 16,
    fontWeight: '600',
  },
  menuSub: {
    fontSize: 12,
    marginTop: 2,
  },
  statusPill: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 5,
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  statusDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
  },
  statusText: {
    fontSize: 12,
    fontWeight: '600',
  },
  goalsGrid: {
    flexDirection: 'row',
    borderRadius: borderRadius.xl,
    borderWidth: 1,
    paddingVertical: spacing.md,
    justifyContent: 'space-around',
    alignItems: 'center',
  },
  goalItem: {
    alignItems: 'center',
    flex: 1,
  },
  goalVal: {
    fontSize: 17,
    fontWeight: '800',
  },
  goalLbl: {
    fontSize: 11,
    marginTop: 2,
    fontWeight: '500',
  },
  goalDivider: {
    width: 1,
    height: 28,
  },
  logoutButton: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 16,
    borderRadius: borderRadius.xl,
    borderWidth: 1,
  },
  logoutText: {
    fontSize: 16,
    fontWeight: '700',
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
    fontSize: 20,
    fontWeight: '700',
    marginBottom: 6,
  },
  modalDesc: {
    fontSize: 14,
    marginBottom: spacing.lg,
  },
  modalInput: {
    borderWidth: 1,
    borderRadius: borderRadius.lg,
    padding: 14,
    fontSize: 15,
    marginBottom: spacing.xl,
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
