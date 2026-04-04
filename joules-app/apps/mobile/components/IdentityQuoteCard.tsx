import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { useQuery } from '@tanstack/react-query';
import { fetchIdentityQuote } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

export default function IdentityQuoteCard() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  const { data } = useQuery({
    queryKey: ['identity-quote'],
    queryFn: fetchIdentityQuote,
    staleTime: 1000 * 60 * 60,
  });

  if (!data?.quote) return null;

  return (
    <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
      <Text style={[styles.quote, { color: colors.textPrimary }]}>"{data.quote}"</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    padding: spacing.lg,
    borderRadius: borderRadius.lg,
    borderWidth: 1,
  },
  quote: {
    fontSize: 15,
    fontStyle: 'italic',
    lineHeight: 22,
  },
});
