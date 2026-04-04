import { useState, useRef } from 'react';
import { View, Text, StyleSheet, Pressable, ActivityIndicator } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { CameraView, CameraType, useCameraPermissions } from 'expo-camera';
import { manipulateAsync, SaveFormat } from 'expo-image-manipulator';
import * as Haptics from 'expo-haptics';
import { useRouter } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { light, dark, oled, spacing, borderRadius } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { identifyMealFromPhoto } from '@joules/api-client';
import CoachAvatar from '@/components/CoachAvatar';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

export default function CameraScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);
  const router = useRouter();
  const [permission, requestPermission] = useCameraPermissions();
  const [facing, setFacing] = useState<CameraType>('back');
  const [photo, setPhoto] = useState<string | null>(null);
  const [processing, setProcessing] = useState(false);
  const cameraRef = useRef<CameraView>(null);

  const handleCapture = async () => {
    if (!cameraRef.current) return;

    await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);

    const captured = await cameraRef.current.takePictureAsync({
      quality: 0.8,
      base64: false,
      skipProcessing: true,
    });

    if (!captured) return;

    setProcessing(true);

    try {
      const manipulated = await manipulateAsync(
        captured.uri,
        [{ resize: { width: 1200 } }],
        { compress: 0.7, format: SaveFormat.JPEG, base64: true }
      );

      const base64 = manipulated.base64;
      if (!base64) {
        setProcessing(false);
        return;
      }

      setPhoto(base64);

      const identifyResult = await identifyMealFromPhoto(base64);

      router.replace({
        pathname: '/log/confirm',
        params: {
          photo: encodeURIComponent(base64),
          results: JSON.stringify(identifyResult),
        },
      });
    } catch {
      setProcessing(false);
    }
  };

  const handleFlip = () => {
    setFacing((prev) => (prev === 'back' ? 'front' : 'back'));
  };

  const handleClose = () => {
    if (router.canGoBack()) {
      router.back();
    } else {
      router.replace('/(tabs)');
    }
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

  return (
    <SafeAreaView style={styles.container}>
      <CameraView ref={cameraRef} facing={facing} style={styles.camera}>
        <View style={styles.overlay}>
          <View style={styles.frameGuide}>
            <View style={[styles.corner, styles.topLeft, { borderColor: colors.primary }]} />
            <View style={[styles.corner, styles.topRight, { borderColor: colors.primary }]} />
            <View style={[styles.corner, styles.bottomLeft, { borderColor: colors.primary }]} />
            <View style={[styles.corner, styles.bottomRight, { borderColor: colors.primary }]} />
          </View>
        </View>

        <View style={styles.topBar}>
          <Pressable onPress={handleClose} style={styles.iconButton}>
            <Ionicons name="close" size={28} color="#fff" />
          </Pressable>
        </View>

        <View style={styles.bottomBar}>
          <Pressable onPress={handleFlip} style={styles.iconButton}>
            <Ionicons name="camera-reverse-outline" size={28} color="#fff" />
          </Pressable>

          <Pressable
            onPress={handleCapture}
            style={styles.captureButton}
            disabled={processing}
          >
            <View style={styles.captureInner} />
          </Pressable>

          <View style={styles.iconButtonPlaceholder} />
        </View>

        {processing && (
          <View style={styles.processingOverlay}>
            <CoachAvatar />
            <ActivityIndicator color="#fff" size="large" style={styles.spinner} />
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
  },
  corner: {
    position: 'absolute',
    width: CORNER_LENGTH,
    height: CORNER_LENGTH,
    borderWidth: CORNER_WIDTH,
    opacity: 0.3,
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
  iconButtonPlaceholder: {
    width: 44,
    height: 44,
  },
  bottomBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: spacing['3xl'],
    paddingBottom: spacing.xl,
  },
  captureButton: {
    width: 72,
    height: 72,
    borderRadius: 36,
    borderWidth: 4,
    borderColor: '#fff',
    justifyContent: 'center',
    alignItems: 'center',
  },
  captureInner: {
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: '#fff',
  },
  processingOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0, 0, 0, 0.7)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  spinner: {
    marginTop: spacing.lg,
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
