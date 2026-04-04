import React, { useEffect, useRef } from 'react';
import { View, Text, StyleSheet } from 'react-native';
import Animated, { useSharedValue, useAnimatedStyle, withRepeat, withTiming, withSequence } from 'react-native-reanimated';
import { Ionicons } from '@expo/vector-icons';
import { light, dark, oled, spacing, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

const oneLiners = [
  'Analyzing your meal...',
  'Almost there...',
  'Looking delicious!',
  'Counting those macros...',
  'Making sure I get this right...',
];

interface CoachAvatarProps {
  message?: string;
}

export default function CoachAvatar({ message }: CoachAvatarProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  const scale = useSharedValue(1);
  const textOpacity = useSharedValue(1);
  const currentIndex = useRef(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [displayMessage, setDisplayMessage] = React.useState(message ?? oneLiners[0]);

  useEffect(() => {
    scale.value = withRepeat(
      withSequence(
        withTiming(1.05, { duration: 1200 }),
        withTiming(1.0, { duration: 1200 })
      ),
      -1,
      false
    );
  }, []);

  useEffect(() => {
    if (message) {
      setDisplayMessage(message);
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      return;
    }

    intervalRef.current = setInterval(() => {
      textOpacity.value = withTiming(0, { duration: 250 }, (finished) => {
        if (finished) {
          currentIndex.current = (currentIndex.current + 1) % oneLiners.length;
          setDisplayMessage(oneLiners[currentIndex.current]);
          textOpacity.value = withTiming(1, { duration: 250 });
        }
      });
    }, 2500);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [message]);

  const pulseStyle = useAnimatedStyle(() => ({
    transform: [{ scale: scale.value }],
  }));

  const textStyle = useAnimatedStyle(() => ({
    opacity: textOpacity.value,
  }));

  return (
    <View style={styles.container}>
      <Animated.View
        style={[
          styles.avatar,
          {
            backgroundColor: `${colors.primary}33`,
            borderColor: colors.primary,
          },
          pulseStyle,
        ]}
      >
        <Ionicons name="nutrition" size={28} color={colors.primary} />
      </Animated.View>
      <Animated.Text
        style={[
          styles.oneLiner,
          { color: colors.textSecondary },
          textStyle,
        ]}
        numberOfLines={1}
      >
        {displayMessage}
      </Animated.Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
  },
  avatar: {
    width: 64,
    height: 64,
    borderRadius: 32,
    borderWidth: 2,
    alignItems: 'center',
    justifyContent: 'center',
  },
  oneLiner: {
    fontSize: fontSizes.sm,
    marginTop: spacing.xs,
    fontWeight: '500',
  },
});
