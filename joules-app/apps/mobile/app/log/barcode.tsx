import { useState, useRef, useEffect } from 'react';
import { View, Text, StyleSheet, Pressable, ActivityIndicator, Animated } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { CameraView, useCameraPermissions } from 'expo-camera';
import * as Haptics from 'expo-haptics';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { lookupBarcode } from '@joules/api-client';
import type { FoodSearchResult, FoodItem } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

function mapResultToFoodItem(result: FoodSearchResult): FoodItem {
  return {
    id: String(Date.now()),
    name: result.brand ? `${result.brand} ${result.name}` : result.name,
    calories: result.calories,
    protein_g: result.protein_g,
    carbs_g: result.carbs_g,
    fat_g: result.fat_g,
    fiber_g: result.fiber_g,
    serving_size: result.serving_size,
    source: result.source,
  };
}

export default function BarcodeScannerScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();
  const [permission, requestPermission] = useCameraPermissions();
  const [scanned, setScanned] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const [torchEnabled, setTorchEnabled] = useState(false);
  const scanLineY = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    const FRAME_INNER = FRAME_SIZE - CORNER_LENGTH * 2;
    const animation = Animated.loop(
      Animated.sequence([
        Animated.timing(scanLineY, {
          toValue: FRAME_INNER,
          duration: 2000,
          useNativeDriver: true,
        }),
        Animated.timing(scanLineY, {
          toValue: 0,
          duration: 2000,
          useNativeDriver: true,
        }),
      ])
    );
    animation.start();
    return () => animation.stop();
  }, [scanLineY]);

  const handleBarCodeScanned = async ({ data }: { type: string; data: string }) => {
    if (scanned) return;

    setScanned(true);
    await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
    setLoading(true);

    try {
      const foodResult = await lookupBarcode(data);
      const foodItem = mapResultToFoodItem(foodResult);

      router.replace({
        pathname: '/log/confirm',
        params: {
          results: JSON.stringify({ foods: [foodItem], confidence: 'high' }),
        },
      });
    } catch {
      setLoading(false);
      setError(true);
    }
  };

  const handleClose = () => {
    if (router.canGoBack()) {
      router.back();
    } else {
      router.replace('/(tabs)');
    }
  };

  const handleTryAgain = () => {
    setError(false);
    setScanned(false);
  };

  const handleSearchManually = () => {
    router.push('/log/search');
  };

  if (!permission) {
    return null;
  }

  if (!permission.granted) {
    return (
      <SafeAreaView style={[styles.container, { backgroundColor: '#000' }]}>
        <View style={styles.permissionContainer}>
          <Ionicons name="camera-outline" size={64} color="#fff" />
          <Pressable
            onPress={requestPermission}
            style={[styles.permissionButton, { backgroundColor: colors.primary }]}
          >
            <Text style={styles.permissionButtonText}>Grant Permission</Text>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  const FRAME_INNER = FRAME_SIZE - CORNER_LENGTH * 2;

  return (
    <SafeAreaView style={styles.container}>
      <CameraView
        style={styles.camera}
        onBarcodeScanned={scanned ? undefined : handleBarCodeScanned}
        enableTorch={torchEnabled}
      >
        <View style={styles.overlay}>
          <View style={styles.frameGuide}>
            <View style={[styles.corner, styles.topLeft, { borderColor: colors.primary }]} />
            <View style={[styles.corner, styles.topRight, { borderColor: colors.primary }]} />
            <View style={[styles.corner, styles.bottomLeft, { borderColor: colors.primary }]} />
            <View style={[styles.corner, styles.bottomRight, { borderColor: colors.primary }]} />
            <Animated.View
              style={[
                styles.scanLine,
                {
                  transform: [{ translateY: scanLineY }],
                  backgroundColor: colors.primary,
                },
              ]}
            />
          </View>
        </View>

        <View style={styles.topBar}>
          <Pressable onPress={handleClose} style={styles.iconButton}>
            <Ionicons name="close" size={28} color="#fff" />
          </Pressable>
        </View>

        <View style={styles.bottomBar}>
          <Text style={styles.instructionText}>Point your camera at a barcode</Text>
          <Pressable
            onPress={() => setTorchEnabled((prev) => !prev)}
            style={styles.iconButton}
          >
            <Ionicons
              name={torchEnabled ? 'flashlight' : 'flashlight-outline'}
              size={28}
              color="#fff"
            />
          </Pressable>
        </View>

        {loading && (
          <View style={styles.processingOverlay}>
            <ActivityIndicator color="#fff" size="large" />
            <Text style={styles.processingText}>Looking up...</Text>
          </View>
        )}

        {error && (
          <View style={styles.processingOverlay}>
            <Ionicons name="barcode-outline" size={48} color="#fff" />
            <Text style={styles.errorTitle}>Not Found</Text>
            <Text style={styles.errorSubtitle}>
              Could not find nutrition info for this barcode
            </Text>
            <View style={styles.errorButtons}>
              <Pressable
                onPress={handleSearchManually}
                style={[styles.errorButton, { backgroundColor: colors.primary }]}
              >
                <Text style={styles.errorButtonText}>Search Manually</Text>
              </Pressable>
              <Pressable
                onPress={handleTryAgain}
                style={[styles.errorButton, styles.errorButtonSecondary]}
              >
                <Text style={styles.errorButtonTextSecondary}>Try Again</Text>
              </Pressable>
            </View>
          </View>
        )}
      </CameraView>
    </SafeAreaView>
  );
}

const FRAME_SIZE = 260;
const CORNER_LENGTH = 40;
const CORNER_WIDTH = 3;

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000',
  },
  camera: {
    flex: 1,
  },
  overlay: {
    ...StyleSheet.absoluteFillObject,
    justifyContent: 'center',
    alignItems: 'center',
  },
  frameGuide: {
    width: FRAME_SIZE + 40,
    height: FRAME_SIZE + 40,
    overflow: 'hidden',
  },
  corner: {
    position: 'absolute',
    width: CORNER_LENGTH,
    height: CORNER_LENGTH,
    borderWidth: CORNER_WIDTH,
    opacity: 0.8,
  },
  topLeft: {
    top: 0,
    left: 0,
    borderBottomWidth: 0,
    borderRightWidth: 0,
    borderTopLeftRadius: borderRadius.lg,
  },
  topRight: {
    top: 0,
    right: 0,
    borderBottomWidth: 0,
    borderLeftWidth: 0,
    borderTopRightRadius: borderRadius.lg,
  },
  bottomLeft: {
    bottom: 0,
    left: 0,
    borderTopWidth: 0,
    borderRightWidth: 0,
    borderBottomLeftRadius: borderRadius.lg,
  },
  bottomRight: {
    bottom: 0,
    right: 0,
    borderTopWidth: 0,
    borderLeftWidth: 0,
    borderBottomRightRadius: borderRadius.lg,
  },
  scanLine: {
    position: 'absolute',
    left: CORNER_LENGTH,
    right: CORNER_LENGTH,
    top: CORNER_LENGTH,
    height: 2,
    opacity: 0.8,
  },
  topBar: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    justifyContent: 'flex-start',
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
  },
  iconButton: {
    width: 44,
    height: 44,
    justifyContent: 'center',
    alignItems: 'center',
  },
  bottomBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: spacing.lg,
    paddingBottom: spacing.xl,
  },
  instructionText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '500',
    opacity: 0.8,
    flex: 1,
  },
  processingOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0, 0, 0, 0.8)',
    justifyContent: 'center',
    alignItems: 'center',
    gap: spacing.md,
  },
  processingText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '600',
    marginTop: spacing.sm,
  },
  errorTitle: {
    color: '#fff',
    fontSize: fontSizes.xl,
    fontWeight: '700',
  },
  errorSubtitle: {
    color: 'rgba(255, 255, 255, 0.7)',
    fontSize: fontSizes.sm,
    textAlign: 'center',
    paddingHorizontal: spacing['2xl'],
  },
  errorButtons: {
    flexDirection: 'row',
    gap: spacing.md,
    marginTop: spacing.md,
  },
  errorButton: {
    paddingHorizontal: spacing.xl,
    paddingVertical: spacing.md,
    borderRadius: borderRadius.lg,
  },
  errorButtonSecondary: {
    backgroundColor: 'rgba(255, 255, 255, 0.15)',
  },
  errorButtonText: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '600',
  },
  errorButtonTextSecondary: {
    color: '#fff',
    fontSize: fontSizes.md,
    fontWeight: '600',
  },
  permissionContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    gap: spacing.xl,
  },
  permissionButton: {
    paddingHorizontal: spacing['2xl'],
    paddingVertical: spacing.md,
    borderRadius: borderRadius.lg,
  },
  permissionButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
