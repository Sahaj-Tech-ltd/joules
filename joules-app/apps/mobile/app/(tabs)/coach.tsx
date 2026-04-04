import React, { useState, useCallback, useEffect, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TextInput,
  Pressable,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  ActivityIndicator,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { Ionicons } from '@expo/vector-icons';
import * as Haptics from 'expo-haptics';
import { light, dark, oled, spacing, borderRadius, fontSizes } from '@joules/ui';
import { useColorScheme } from '@/hooks/useColorScheme';
import { fetchCoachMessages, sendCoachMessage } from '@joules/api-client';
import type { CoachMessage } from '@joules/api-client';

function getColors(scheme: string) {
  if (scheme === 'dark') return dark;
  if (scheme === 'oled') return oled;
  return light;
}

const DAILY_FREE_LIMIT = 5;

function formatTime(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
}

function isToday(dateStr: string): boolean {
  const d = new Date(dateStr);
  const now = new Date();
  return (
    d.getDate() === now.getDate() &&
    d.getMonth() === now.getMonth() &&
    d.getFullYear() === now.getFullYear()
  );
}

function renderMarkdownText(text: string, color: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  const lines = text.split('\n');
  let key = 0;

  for (const line of lines) {
    if (line.startsWith('**') && line.endsWith('**')) {
      parts.push(
        <Text key={key++} style={{ fontWeight: '700', color, fontSize: fontSizes.md }}>
          {line.replace(/\*\*/g, '')}
        </Text>
      );
    } else if (line.startsWith('- ') || line.startsWith('• ')) {
      parts.push(
        <Text key={key++} style={{ color, fontSize: fontSizes.md, paddingLeft: spacing.md }}>
          {'• ' + line.replace(/^[-•]\s*/, '')}
        </Text>
      );
    } else if (/^\d+\.\s/.test(line)) {
      parts.push(
        <Text key={key++} style={{ color, fontSize: fontSizes.md, paddingLeft: spacing.md }}>
          {line}
        </Text>
      );
    } else if (line.trim() === '') {
      parts.push(<Text key={key++}>{'\n'}</Text>);
    } else {
      const boldParts = line.split(/(\*\*[^*]+\*\*)/g);
      const rendered = boldParts.map((part, i) => {
        if (part.startsWith('**') && part.endsWith('**')) {
          return (
            <Text key={i} style={{ fontWeight: '700' }}>
              {part.replace(/\*\*/g, '')}
            </Text>
          );
        }
        return <Text key={i}>{part}</Text>;
      });
      parts.push(
        <Text key={key++} style={{ color, fontSize: fontSizes.md }}>
          {rendered}
        </Text>
      );
    }
  }

  return parts;
}

export default function CoachScreen() {
  const colorScheme = useColorScheme() ?? 'dark';
  const colors = getColors(colorScheme);

  const [messages, setMessages] = useState<CoachMessage[]>([]);
  const [inputText, setInputText] = useState('');
  const [sending, setSending] = useState(false);
  const [loading, setLoading] = useState(true);
  const flatListRef = useRef<FlatList>(null);
  const inputRef = useRef<TextInput>(null);

  const todayUserMessages = messages.filter(
    (m) => m.role === 'user' && isToday(m.created_at)
  );
  const freeRemaining = Math.max(0, DAILY_FREE_LIMIT - todayUserMessages.length);

  const loadMessages = useCallback(async () => {
    try {
      const data = await fetchCoachMessages(50);
      setMessages(data.reverse());
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadMessages();
  }, [loadMessages]);

  const scrollToBottom = useCallback(() => {
    setTimeout(() => {
      flatListRef.current?.scrollToEnd({ animated: true });
    }, 100);
  }, []);

  useEffect(() => {
    if (messages.length > 0) {
      scrollToBottom();
    }
  }, [messages.length, scrollToBottom]);

  const handleSend = useCallback(async () => {
    const text = inputText.trim();
    if (!text || sending) return;

    setInputText('');
    setSending(true);

    const optimisticUser: CoachMessage = {
      id: `temp-${Date.now()}`,
      role: 'user',
      content: text,
      created_at: new Date().toISOString(),
    };

    setMessages((prev) => [...prev, optimisticUser]);
    scrollToBottom();

    try {
      const response = await sendCoachMessage(text);
      setMessages((prev) => [...prev.filter((m) => m.id !== optimisticUser.id), optimisticUser, response]);
      scrollToBottom();
    } catch {
      setMessages((prev) => prev.filter((m) => m.id !== optimisticUser.id));
      setInputText(text);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    } finally {
      setSending(false);
    }
  }, [inputText, sending, scrollToBottom]);

  const renderItem = useCallback(
    ({ item }: { item: CoachMessage }) => {
      const isUser = item.role === 'user';

      if (isUser) {
        return (
          <View style={styles.userMessageWrap}>
            <View style={[styles.userBubble, { backgroundColor: colors.primary }]}>
              <Text style={styles.userBubbleText}>{item.content}</Text>
            </View>
            <Text style={[styles.messageTime, { color: colors.textTertiary }]}>
              {formatTime(item.created_at)}
            </Text>
          </View>
        );
      }

      return (
        <View style={styles.coachMessageWrap}>
          <View style={[styles.coachAvatar, { backgroundColor: `${colors.primary}20` }]}>
            <Ionicons name="nutrition" size={16} color={colors.primary} />
          </View>
          <View style={{ flex: 1 }}>
            <View style={[styles.coachBubble, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              {renderMarkdownText(item.content, colors.textPrimary)}
            </View>
            <Text style={[styles.messageTime, { color: colors.textTertiary }]}>
              {formatTime(item.created_at)}
            </Text>
          </View>
        </View>
      );
    },
    [colors]
  );

  if (loading) {
    return (
      <SafeAreaView style={[styles.container, { backgroundColor: colors.background }]} edges={['bottom']}>
        <View style={styles.header}>
          <View style={[styles.headerAvatar, { backgroundColor: `${colors.primary}20` }]}>
            <Ionicons name="nutrition" size={20} color={colors.primary} />
          </View>
          <View>
            <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Joule Coach</Text>
            <Text style={[styles.headerSub, { color: colors.textSecondary }]}>Your nutrition guide</Text>
          </View>
        </View>
        <View style={styles.loaderWrap}>
          <ActivityIndicator size="large" color={colors.primary} />
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={[styles.container, { backgroundColor: colors.background }]} edges={['bottom']}>
      <KeyboardAvoidingView
        style={styles.container}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        keyboardVerticalOffset={0}
      >
        <View style={[styles.header, { borderBottomColor: colors.border }]}>
          <View style={[styles.headerAvatar, { backgroundColor: `${colors.primary}20` }]}>
            <Ionicons name="nutrition" size={20} color={colors.primary} />
          </View>
          <View style={styles.headerInfo}>
            <Text style={[styles.headerTitle, { color: colors.textPrimary }]}>Joule Coach</Text>
            <Text style={[styles.headerSub, { color: colors.textSecondary }]}>Your nutrition guide</Text>
          </View>
          {freeRemaining <= 2 && (
            <View style={[styles.limitBadge, { backgroundColor: `${colors.warning}20` }]}>
              <Text style={[styles.limitText, { color: colors.warning }]}>
                {freeRemaining} of {DAILY_FREE_LIMIT}
              </Text>
            </View>
          )}
        </View>

        <FlatList
          ref={flatListRef}
          data={messages}
          keyExtractor={(item) => item.id}
          renderItem={renderItem}
          contentContainerStyle={styles.messageList}
          showsVerticalScrollIndicator={false}
          onContentSizeChange={scrollToBottom}
          ListEmptyComponent={
            <View style={styles.emptyWrap}>
              <View style={[styles.emptyAvatar, { backgroundColor: `${colors.primary}15` }]}>
                <Ionicons name="nutrition" size={40} color={colors.primary} />
              </View>
              <Text style={[styles.emptyTitle, { color: colors.textPrimary }]}>
                Hi there! I'm your Joule Coach
              </Text>
              <Text style={[styles.emptySub, { color: colors.textSecondary }]}>
                Ask me anything about nutrition, meal planning, or your progress.
              </Text>
            </View>
          }
        />

        {sending && (
          <View style={[styles.typingBar, { borderTopColor: colors.border }]}>
            <View style={[styles.typingDots, { backgroundColor: colors.surface }]}>
              <ActivityIndicator size="small" color={colors.primary} />
              <Text style={[styles.typingText, { color: colors.textSecondary }]}>Coach is thinking...</Text>
            </View>
          </View>
        )}

        <View style={[styles.inputBar, { borderTopColor: colors.border, backgroundColor: colors.background }]}>
          <TextInput
            ref={inputRef}
            style={[
              styles.textInput,
              {
                color: colors.textPrimary,
                backgroundColor: colors.surfaceElevated,
                borderColor: colors.border,
              },
            ]}
            placeholder="Ask your coach..."
            placeholderTextColor={colors.textTertiary}
            value={inputText}
            onChangeText={setInputText}
            multiline
            maxLength={500}
            editable={!sending}
          />
          <Pressable
            onPress={handleSend}
            disabled={!inputText.trim() || sending}
            style={({ pressed }) => [
              styles.sendButton,
              {
                backgroundColor:
                  inputText.trim() && !sending ? colors.primary : colors.border,
                transform: [{ scale: pressed ? 0.92 : 1 }],
              },
            ]}
          >
            <Ionicons name="send" size={18} color="#fff" />
          </Pressable>
        </View>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: 1,
    gap: spacing.md,
  },
  headerAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerInfo: {
    flex: 1,
  },
  headerTitle: {
    fontSize: fontSizes.md,
    fontWeight: '700',
  },
  headerSub: {
    fontSize: fontSizes.xs,
    marginTop: 1,
  },
  limitBadge: {
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderRadius: borderRadius.full,
  },
  limitText: {
    fontSize: 11,
    fontWeight: '700',
  },
  loaderWrap: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  messageList: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md,
    paddingBottom: spacing.md,
  },
  emptyWrap: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    paddingTop: spacing['4xl'],
    gap: spacing.md,
  },
  emptyAvatar: {
    width: 80,
    height: 80,
    borderRadius: 40,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: spacing.sm,
  },
  emptyTitle: {
    fontSize: fontSizes.lg,
    fontWeight: '700',
    textAlign: 'center',
  },
  emptySub: {
    fontSize: fontSizes.sm,
    textAlign: 'center',
    lineHeight: 20,
    paddingHorizontal: spacing.xl,
  },
  userMessageWrap: {
    alignItems: 'flex-end',
    marginBottom: spacing.md,
  },
  userBubble: {
    maxWidth: '80%',
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.lg,
    borderBottomRightRadius: 4,
  },
  userBubbleText: {
    color: '#fff',
    fontSize: fontSizes.md,
    lineHeight: 20,
  },
  coachMessageWrap: {
    flexDirection: 'row',
    marginBottom: spacing.md,
    gap: spacing.sm,
    alignItems: 'flex-start',
  },
  coachAvatar: {
    width: 28,
    height: 28,
    borderRadius: 14,
    alignItems: 'center',
    justifyContent: 'center',
    marginTop: 2,
  },
  coachBubble: {
    maxWidth: '85%',
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.lg,
    borderBottomLeftRadius: 4,
    borderWidth: 1,
    gap: 2,
  },
  messageTime: {
    fontSize: 10,
    marginTop: 2,
    marginHorizontal: 4,
  },
  typingBar: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
    borderTopWidth: 1,
  },
  typingDots: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: borderRadius.lg,
    alignSelf: 'flex-start',
  },
  typingText: {
    fontSize: fontSizes.sm,
    fontWeight: '500',
  },
  inputBar: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: spacing.md,
    borderTopWidth: 1,
    gap: spacing.sm,
  },
  textInput: {
    flex: 1,
    minHeight: 40,
    maxHeight: 100,
    borderWidth: 1,
    borderRadius: borderRadius.xl,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    fontSize: fontSizes.md,
    lineHeight: 20,
    paddingTop: Platform.OS === 'ios' ? 10 : spacing.sm,
  },
  sendButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 2,
  },
});
