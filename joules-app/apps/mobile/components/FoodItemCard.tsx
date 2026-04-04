import React, { useState } from 'react';
import { View, Text, TextInput, StyleSheet, TouchableOpacity } from 'react-native';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import type { FoodItem } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

interface FoodItemCardProps {
  food: FoodItem;
  onChange?: (updated: FoodItem) => void;
}

export default function FoodItemCard({ food, onChange }: FoodItemCardProps) {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const [expanded, setExpanded] = useState(false);
  const [localFood, setLocalFood] = useState<FoodItem>({ ...food });

  const updateField = <K extends keyof FoodItem>(key: K, value: FoodItem[K]) => {
    const updated = { ...localFood, [key]: value };
    setLocalFood(updated);
    onChange?.(updated);
  };

  return (
    <TouchableOpacity
      activeOpacity={0.7}
      onPress={() => setExpanded(!expanded)}
      style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}
    >
      <View style={styles.collapsedRow}>
        <View style={styles.nameCol}>
          <Text style={[styles.foodName, { color: colors.textPrimary }]} numberOfLines={1}>
            {localFood.name}
          </Text>
        </View>
        <View style={styles.rightCol}>
          <Text style={[styles.calories, { color: colors.textPrimary }]}>
            {Math.round(localFood.calories)} cal
          </Text>
          {localFood.source === 'food_memory' && (
            <View style={[styles.badge, { backgroundColor: `${colors.primary}20` }]}>
              <Text style={[styles.badgeText, { color: colors.primary }]}>✓ Learned</Text>
            </View>
          )}
        </View>
      </View>

      {expanded && (
        <View style={styles.expandedContent}>
          <View style={styles.fieldGroup}>
            <Text style={[styles.fieldLabel, { color: colors.textSecondary }]}>Food name</Text>
            <TextInput
              style={[styles.input, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.surfaceElevated }]}
              value={localFood.name}
              onChangeText={(text) => updateField('name', text)}
              returnKeyType="done"
            />
          </View>

          <View style={styles.fieldGroup}>
            <Text style={[styles.fieldLabel, { color: colors.textSecondary }]}>Calories</Text>
            <View style={styles.inputRow}>
              <TextInput
                style={[styles.input, styles.numericInput, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.surfaceElevated }]}
                value={String(localFood.calories)}
                onChangeText={(text) => updateField('calories', Number(text) || 0)}
                keyboardType="numeric"
                returnKeyType="done"
              />
              <Text style={[styles.unitSuffix, { color: colors.textTertiary }]}>cal</Text>
            </View>
          </View>

          <View style={styles.macroRow}>
            <View style={[styles.macroField, { marginRight: spacing.xs }]}>
              <Text style={[styles.fieldLabel, { color: colors.textSecondary }]}>Protein</Text>
              <View style={styles.inputRow}>
                <TextInput
                  style={[styles.input, styles.numericInput, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.surfaceElevated }]}
                  value={String(localFood.protein_g)}
                  onChangeText={(text) => updateField('protein_g', Number(text) || 0)}
                  keyboardType="numeric"
                  returnKeyType="done"
                />
                <Text style={[styles.unitSuffix, { color: colors.textTertiary }]}>g</Text>
              </View>
            </View>
            <View style={[styles.macroField, { marginRight: spacing.xs }]}>
              <Text style={[styles.fieldLabel, { color: colors.textSecondary }]}>Carbs</Text>
              <View style={styles.inputRow}>
                <TextInput
                  style={[styles.input, styles.numericInput, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.surfaceElevated }]}
                  value={String(localFood.carbs_g)}
                  onChangeText={(text) => updateField('carbs_g', Number(text) || 0)}
                  keyboardType="numeric"
                  returnKeyType="done"
                />
                <Text style={[styles.unitSuffix, { color: colors.textTertiary }]}>g</Text>
              </View>
            </View>
            <View style={styles.macroField}>
              <Text style={[styles.fieldLabel, { color: colors.textSecondary }]}>Fat</Text>
              <View style={styles.inputRow}>
                <TextInput
                  style={[styles.input, styles.numericInput, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.surfaceElevated }]}
                  value={String(localFood.fat_g)}
                  onChangeText={(text) => updateField('fat_g', Number(text) || 0)}
                  keyboardType="numeric"
                  returnKeyType="done"
                />
                <Text style={[styles.unitSuffix, { color: colors.textTertiary }]}>g</Text>
              </View>
            </View>
          </View>

          <View style={styles.fieldGroup}>
            <Text style={[styles.fieldLabel, { color: colors.textSecondary }]}>Serving size</Text>
            <TextInput
              style={[styles.input, { color: colors.textPrimary, borderColor: colors.border, backgroundColor: colors.surfaceElevated }]}
              value={localFood.serving_size}
              onChangeText={(text) => updateField('serving_size', text)}
              returnKeyType="done"
            />
          </View>
        </View>
      )}
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  card: {
    borderRadius: borderRadius.lg,
    borderWidth: 1,
    padding: spacing.lg,
    marginBottom: spacing.sm,
  },
  collapsedRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  nameCol: {
    flex: 1,
    marginRight: spacing.sm,
  },
  rightCol: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.xs,
  },
  foodName: {
    fontSize: fontSizes.md,
    fontWeight: '600',
  },
  calories: {
    fontSize: fontSizes.sm,
    fontWeight: '600',
  },
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: borderRadius.full,
  },
  badgeText: {
    fontSize: 10,
    fontWeight: '700',
  },
  expandedContent: {
    marginTop: spacing.md,
    paddingTop: spacing.md,
    borderTopWidth: 1,
    borderTopColor: 'rgba(150,150,150,0.2)',
  },
  fieldGroup: {
    marginBottom: spacing.md,
  },
  fieldLabel: {
    fontSize: 12,
    fontWeight: '600',
    marginBottom: 4,
  },
  input: {
    borderWidth: 1,
    borderRadius: borderRadius.md,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs + 2,
    fontSize: fontSizes.sm,
  },
  numericInput: {
    flex: 1,
    minWidth: 0,
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  unitSuffix: {
    fontSize: fontSizes.sm,
    fontWeight: '500',
  },
  macroRow: {
    flexDirection: 'row',
    marginBottom: spacing.md,
  },
  macroField: {
    flex: 1,
  },
});
