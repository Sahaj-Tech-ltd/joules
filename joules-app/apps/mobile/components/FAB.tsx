import { View, StyleSheet, Pressable } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { useRouter } from 'expo-router';
import { light, dark, oled } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { spacing, borderRadius } from '@joules/ui';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

export default function FAB() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();

  const handlePress = () => {
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    router.push('/log/camera');
  };

  return (
    <Pressable
      onPress={handlePress}
      style={({ pressed }) => [
        styles.button,
        {
          backgroundColor: colors.primary,
          transform: [{ scale: pressed ? 0.92 : 1 }],
        },
      ]}
      android_ripple={{ color: 'rgba(255,255,255,0.2)', borderless: true }}
    >
      <Ionicons name="camera" size={28} color="#fff" />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: {
    width: 60,
    height: 60,
    borderRadius: borderRadius.full,
    justifyContent: 'center' as const,
    alignItems: 'center' as const,
    position: 'absolute' as const,
    bottom: spacing.xl,
    elevation: 6,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 3 },
    shadowOpacity: 0.3,
    shadowRadius: 6,
  },
});
