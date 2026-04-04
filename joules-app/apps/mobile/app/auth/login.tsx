import { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  Pressable,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  Alert,
} from 'react-native';
import { Link, router } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuthStore } from '@joules/api-client';
import { dark } from '@joules/ui';
import { login as loginApi } from '@joules/api-client';

export default function LoginScreen() {
  const insets = useSafeAreaInsets();
  const setToken = useAuthStore((s) => s.setToken);
  const setBaseUrl = useAuthStore((s) => s.setBaseUrl);
  const baseUrl = useAuthStore((s) => s.baseUrl);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [showAdvanced, setShowAdvanced] = useState(false);
  const [serverUrl, setServerUrl] = useState('');
  const [connecting, setConnecting] = useState(false);

  async function handleConnect() {
    if (!serverUrl.trim()) return;
    setConnecting(true);
    try {
      const res = await fetch(`${serverUrl.replace(/\/+$/, '')}/api/banners`);
      if (res.ok) {
        setBaseUrl(serverUrl.replace(/\/+$/, ''));
        Alert.alert('Connected', 'Server URL saved.');
      } else {
        Alert.alert('Error', 'Server did not respond correctly.');
      }
    } catch {
      Alert.alert('Error', 'Could not reach server.');
    } finally {
      setConnecting(false);
    }
  }

  async function handleLogin() {
    if (!email.trim() || !password) {
      setError('Please enter your email and password.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await loginApi(email.trim(), password);
      setToken(data.access_token);
      router.replace('/(tabs)');
    } catch (err: any) {
      setError(err?.message || 'Login failed. Please try again.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      style={{ flex: 1, backgroundColor: dark.background }}
    >
      <ScrollView
        contentContainerStyle={{
          flexGrow: 1,
          paddingTop: insets.top + 60,
          paddingBottom: insets.bottom + 24,
          paddingHorizontal: 24,
        }}
        keyboardShouldPersistTaps="handled"
      >
        <Text
          style={{
            fontSize: 36,
            fontWeight: '700',
            color: dark.textPrimary,
            marginBottom: 8,
          }}
        >
          Joule
        </Text>
        <Text
          style={{
            fontSize: 16,
            color: dark.textSecondary,
            marginBottom: 40,
          }}
        >
          Welcome back. Log in to continue.
        </Text>

        {error && (
          <View
            style={{
              backgroundColor: dark.error + '1a',
              borderRadius: 10,
              padding: 12,
              marginBottom: 16,
              borderLeftWidth: 3,
              borderLeftColor: dark.error,
            }}
          >
            <Text style={{ color: dark.error, fontSize: 14 }}>{error}</Text>
          </View>
        )}

        <TextInput
          placeholder="Email"
          placeholderTextColor={dark.textTertiary}
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
          style={{
            backgroundColor: dark.surface,
            borderRadius: 12,
            padding: 16,
            fontSize: 16,
            color: dark.textPrimary,
            marginBottom: 12,
            borderWidth: 1,
            borderColor: dark.border,
          }}
        />

        <TextInput
          placeholder="Password"
          placeholderTextColor={dark.textTertiary}
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          style={{
            backgroundColor: dark.surface,
            borderRadius: 12,
            padding: 16,
            fontSize: 16,
            color: dark.textPrimary,
            marginBottom: 24,
            borderWidth: 1,
            borderColor: dark.border,
          }}
        />

        <Pressable
          onPress={handleLogin}
          disabled={loading}
          style={{
            backgroundColor: loading ? dark.primary + '80' : dark.primary,
            borderRadius: 12,
            padding: 16,
            alignItems: 'center',
            marginBottom: 16,
          }}
        >
          {loading ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={{ color: '#fff', fontSize: 16, fontWeight: '600' }}>
              Log In
            </Text>
          )}
        </Pressable>

        <Link href="/auth/signup" asChild>
          <Pressable style={{ alignItems: 'center', padding: 8 }}>
            <Text style={{ color: dark.primary, fontSize: 14 }}>
              Don't have an account?{' '}
              <Text style={{ fontWeight: '600' }}>Sign Up</Text>
            </Text>
          </Pressable>
        </Link>

        <Pressable
          onPress={() => setShowAdvanced((v) => !v)}
          style={{ marginTop: 32, alignItems: 'center' }}
        >
          <Text style={{ color: dark.textTertiary, fontSize: 13 }}>
            {showAdvanced ? 'Hide' : 'Advanced'} — Self-hosted server
          </Text>
        </Pressable>

        {showAdvanced && (
          <View style={{ marginTop: 12 }}>
            <TextInput
              placeholder="https://your-server.com"
              placeholderTextColor={dark.textTertiary}
              value={serverUrl}
              onChangeText={setServerUrl}
              autoCapitalize="none"
              keyboardType="url"
              autoComplete="off"
              style={{
                backgroundColor: dark.surface,
                borderRadius: 12,
                padding: 14,
                fontSize: 14,
                color: dark.textPrimary,
                marginBottom: 10,
                borderWidth: 1,
                borderColor: dark.border,
              }}
            />
            <Pressable
              onPress={handleConnect}
              disabled={connecting}
              style={{
                backgroundColor: dark.surfaceElevated,
                borderRadius: 10,
                padding: 12,
                alignItems: 'center',
              }}
            >
              {connecting ? (
                <ActivityIndicator color={dark.textPrimary} size="small" />
              ) : (
                <Text style={{ color: dark.textPrimary, fontSize: 14, fontWeight: '500' }}>
                  Connect
                </Text>
              )}
            </Pressable>
            {baseUrl !== 'http://localhost:3000/api' && (
              <Text
                style={{
                  color: dark.textTertiary,
                  fontSize: 12,
                  textAlign: 'center',
                  marginTop: 8,
                }}
              >
                Connected to: {baseUrl}
              </Text>
            )}
          </View>
        )}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
