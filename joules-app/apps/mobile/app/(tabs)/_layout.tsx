import React from 'react';
import { View, Text, StyleSheet, Pressable } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { Tabs } from 'expo-router';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface TabDef {
  name: string;
  label: string;
  icon: keyof typeof Ionicons.glyphMap;
  outlineIcon: keyof typeof Ionicons.glyphMap;
}

const TABS: TabDef[] = [
  { name: 'index', label: 'Today', icon: 'flame', outlineIcon: 'flame-outline' },
  { name: 'log', label: 'Log', icon: 'receipt', outlineIcon: 'receipt-outline' },
  { name: 'coach', label: 'Coach', icon: 'sparkles', outlineIcon: 'sparkles-outline' },
  { name: 'progress', label: 'Progress', icon: 'stats-chart', outlineIcon: 'stats-chart-outline' },
  { name: 'more', label: 'More', icon: 'person-circle', outlineIcon: 'person-circle-outline' },
];

export default function TabLayout() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  return (
    <View style={styles.wrapper}>
      <Tabs
        screenOptions={{
          headerShown: false,
        }}
        tabBar={({ state, navigation }) => {
          return (
            <View style={[styles.tabBar, { backgroundColor: colors.surface, borderTopColor: colors.border }]}>
              {TABS.map((tab, index) => {
                const isFocused = state.index === index;
                const activeColor = colors.primary;
                const inactiveColor = colors.textTertiary;

                const onPress = () => {
                  Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
                  const event = navigation.emit({
                    type: 'tabPress',
                    target: state.routes[index]?.key || tab.name,
                    canPreventDefault: true,
                  });
                  if (!isFocused && !event.defaultPrevented) {
                    navigation.navigate(tab.name);
                  }
                };

                return (
                  <Pressable
                    key={tab.name}
                    onPress={onPress}
                    style={styles.tabItem}
                    hitSlop={8}
                  >
                    <Ionicons
                      name={isFocused ? tab.icon : tab.outlineIcon}
                      size={24}
                      color={isFocused ? activeColor : inactiveColor}
                    />
                    <Text
                      style={[
                        styles.tabLabel,
                        { color: isFocused ? activeColor : inactiveColor, fontWeight: isFocused ? '700' : '500' },
                      ]}
                    >
                      {tab.label}
                    </Text>
                    {isFocused && (
                      <View style={[styles.activeDot, { backgroundColor: activeColor }]} />
                    )}
                  </Pressable>
                );
              })}
            </View>
          );
        }}
      >
        <Tabs.Screen name="index" options={{ title: 'Today' }} />
        <Tabs.Screen name="log" options={{ title: 'Log' }} />
        <Tabs.Screen name="coach" options={{ title: 'Coach' }} />
        <Tabs.Screen name="progress" options={{ title: 'Progress' }} />
        <Tabs.Screen name="more" options={{ title: 'More' }} />
      </Tabs>
    </View>
  );
}

const styles = StyleSheet.create({
  wrapper: {
    flex: 1,
  },
  tabBar: {
    flexDirection: 'row',
    borderTopWidth: 1,
    height: 84,
    paddingBottom: 24,
    paddingTop: 8,
    alignItems: 'center',
    justifyContent: 'space-around',
  },
  tabItem: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 2,
    position: 'relative',
  },
  tabLabel: {
    fontSize: 10,
    marginTop: 3,
    letterSpacing: 0.2,
  },
  activeDot: {
    width: 4,
    height: 4,
    borderRadius: 2,
    marginTop: 2,
  },
});
