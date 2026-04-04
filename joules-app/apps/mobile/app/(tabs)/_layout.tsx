import React from 'react';
import { View, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { Tabs } from 'expo-router';
import { light, dark, oled, spacing } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import FAB from '@/components/FAB';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

function TabIcon({ name, color, focused }: { name: keyof typeof Ionicons.glyphMap; color: string; focused: boolean }) {
  return (
    <View style={[styles.iconWrap, focused && styles.iconWrapActive]}>
      <Ionicons name={focused ? name : `${name}-outline` as any} size={24} color={color} />
    </View>
  );
}

export default function TabLayout() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  return (
    <View style={styles.wrapper}>
      <Tabs
        screenOptions={{
          tabBarActiveTintColor: colors.primary,
          tabBarInactiveTintColor: colors.textTertiary,
          tabBarStyle: {
            backgroundColor: colors.surface,
            borderTopColor: colors.border,
            borderTopWidth: 1,
            height: 80,
            paddingBottom: 20,
            paddingTop: spacing.sm,
          },
          tabBarLabelStyle: {
            fontSize: 11,
            fontWeight: '600' as const,
            marginTop: 2,
          },
          headerStyle: {
            backgroundColor: colors.background,
          },
          headerTintColor: colors.textPrimary,
          headerTitleStyle: {
            fontWeight: '700' as const,
          },
        }}
        tabBar={(props) => {
          const { state, descriptors, navigation } = props;
          return (
            <View style={[styles.tabBar, { backgroundColor: colors.surface, borderTopColor: colors.border }]}>
              {state.routes.map((route, index) => {
                const { options } = descriptors[route.key];
                const isFocused = state.index === index;
                const label = options.tabBarLabel ?? options.title ?? route.name;

                if (route.name === 'log') {
                  return (
                    <View key={route.name} style={styles.logTabWrap}>
                      <FAB />
                    </View>
                  );
                }

                const iconMap: Record<string, keyof typeof Ionicons.glyphMap> = {
                  index: 'home',
                  log: 'camera',
                  coach: 'chatbubbles',
                  progress: 'stats-chart',
                  more: 'ellipsis-horizontal',
                };

                const onPress = () => {
                  const event = navigation.emit({
                    type: 'tabPress',
                    target: route.key,
                    canPreventDefault: true,
                  });
                  if (!isFocused && !event.defaultPrevented) {
                    navigation.navigate(route.name);
                  }
                };

                return (
                  <View key={route.name} style={styles.tabItem}>
                    <TabIcon
                      name={iconMap[route.name] ?? 'ellipse'}
                      color={isFocused ? colors.primary : colors.textTertiary}
                      focused={isFocused}
                    />
                  </View>
                );
              })}
            </View>
          );
        }}
      >
        <Tabs.Screen
          name="index"
          options={{ title: 'Home', headerShown: false }}
        />
        <Tabs.Screen
          name="log"
          options={{ title: '', headerShown: false }}
        />
        <Tabs.Screen
          name="coach"
          options={{ title: 'Coach', headerShown: true }}
        />
        <Tabs.Screen
          name="progress"
          options={{ title: 'Progress', headerShown: true }}
        />
        <Tabs.Screen
          name="more"
          options={{ title: 'More', headerShown: true }}
        />
      </Tabs>
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: {
    flex: 1,
  },
  tabBar: {
    flexDirection: 'row' as const,
    borderTopWidth: 1,
    height: 80,
    paddingBottom: 20,
    paddingTop: spacing.sm,
    alignItems: 'center' as const,
    justifyContent: 'space-around' as const,
  },
  tabItem: {
    flex: 1,
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
  },
  logTabWrap: {
    flex: 1,
    alignItems: 'center' as const,
    justifyContent: 'flex-end' as const,
    paddingBottom: 0,
  },
  iconWrap: {
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    paddingVertical: 4,
  },
  iconWrapActive: {
    opacity: 1,
  },
});
