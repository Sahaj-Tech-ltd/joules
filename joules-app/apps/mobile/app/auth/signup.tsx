import { useState, useMemo } from 'react';
import {
  View,
  Text,
  TextInput,
  Pressable,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from 'react-native';
import { Link, router } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuthStore } from '@joules/api-client';
import { dark } from '@joules/ui';
import { signup as signupApi } from '@joules/api-client';

function getPasswordStrength(password: string): {
  label: string;
  color: string;
  fraction: number;
} {
  if (!password) return { label: '', color: 'transparent', fraction: 0 };
  let score = 0;
  if (password.length >= 8) score++;
  if (password.length >= 12) score++;
  if (/[A-Z]/.test(password)) score++;
  if (/[0-9]/.test(password)) score++;
  if (/[^A-Za-z0-9]/.test(password)) score++;

  if (score <= 1) return { label: 'Weak', color: dark.error, fraction: 0.25 };
  if (score <= 2) return { label: 'Fair', color: dark.warning, fraction: 0.5 };
  if (score <= 3) return { label: 'Good', color: '#3b82f6', fraction: 0.75 };
  return { label: 'Strong', color: dark.success, fraction: 1 };
}

export default function SignupScreen() {
  const insets = useSafeAreaInsets();
  const setToken = useAuthStore((s) => s.setToken);

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const strength = useMemo(() => getPasswordStrength(password), [password]);

  async function handleSignup() {
    if (!name.trim() || !email.trim() || !password) {
      setError('Please fill in all fields.');
      return;
    }
    if (password.length < 8) {
      setError('Password must be at least 8 characters.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const data = await signupApi(email.trim(), password, name.trim());
      setToken(data.access_token);
      router.replace('/(tabs)');
    } catch (err: any) {
      setError(err?.message || 'Signup failed. Please try again.');
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
          Create Account
        </Text>
        <Text
          style={{
            fontSize: 16,
            color: dark.textSecondary,
            marginBottom: 40,
          }}
        >
          Start tracking your nutrition today.
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
          placeholder="Name"
          placeholderTextColor={dark.textTertiary}
          value={name}
          onChangeText={setName}
          autoComplete="name"
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
          placeholder="Password (min 8 characters)"
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
            marginBottom: 8,
            borderWidth: 1,
            borderColor: dark.border,
          }}
        />

        {password.length > 0 && (
          <View style={{ marginBottom: 24 }}>
            <View
              style={{
                height: 4,
                borderRadius: 2,
                backgroundColor: dark.border,
                overflow: 'hidden',
              }}
            >
              <View
                style={{
                  height: 4,
                  borderRadius: 2,
                  backgroundColor: strength.color,
                  width: `${strength.fraction * 100}%`,
                }}
              />
            </View>
            <Text
              style={{
                color: strength.color,
                fontSize: 12,
                marginTop: 4,
                fontWeight: '500',
              }}
            >
              {strength.label}
            </Text>
          </View>
        )}

        <Pressable
          onPress={handleSignup}
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
              Create Account
            </Text>
          )}
        </Pressable>

        <Link href="/auth/login" asChild>
          <Pressable style={{ alignItems: 'center', padding: 8 }}>
            <Text style={{ color: dark.primary, fontSize: 14 }}>
              Already have an account?{' '}
              <Text style={{ fontWeight: '600' }}>Log In</Text>
            </Text>
          </Pressable>
        </Link>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
