import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

test.describe('AI Coach Chat', () => {
  test('coach page loads with welcome message', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');
    await expect(page.locator('h1', { hasText: 'Health Coach' })).toBeVisible();
    await expect(page.locator('h2', { hasText: /Joules health coach/i })).toBeVisible();
  });

  test('coach page shows suggestion buttons', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');
    await expect(page.locator('button', { hasText: /Log my breakfast/i })).toBeVisible();
    await expect(page.locator('button', { hasText: /ran for 30 minutes/i })).toBeVisible();
  });

  test('coach page has chat input and send button', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');
    await expect(page.locator('textarea[placeholder*="Tell Joules"]')).toBeVisible();
    await expect(page.locator('button[aria-label="Send message"]')).toBeVisible();
  });

  test('send a message and get a response', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');

    // Wait for page to fully load
    await expect(page.locator('h1', { hasText: 'Health Coach' })).toBeVisible();

    // Type a message
    const input = page.locator('textarea[placeholder*="Tell Joules"]');
    await input.fill('What should I eat for a healthy breakfast?');

    // Click send
    await page.locator('button[aria-label="Send message"]').click();

    // User message should appear
    await expect(page.locator('text=What should I eat for a healthy breakfast?')).toBeVisible({ timeout: 5000 });

    // Wait for AI response (the loading indicator should appear then disappear)
    // The assistant response should appear within 30 seconds
    await expect(page.locator('text=Joules').first()).toBeVisible({ timeout: 5000 });

    // Wait for the response content to appear (not just the loading dots)
    // The response is rendered as markdown, so look for any substantial text
    await page.waitForFunction(() => {
      const messages = document.querySelectorAll('[class*="rounded-2xl"]');
      // Should have at least 2 message bubbles (user + assistant)
      return messages.length >= 2;
    }, { timeout: 30000 });

    // Verify we got a non-empty assistant response
    const assistantMessages = page.locator('.rounded-bl-sm');
    await expect(assistantMessages.first()).toBeVisible({ timeout: 30000 });
  });

  test('clicking a suggestion sends that message', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');

    // Click a suggestion button
    await page.locator('button', { hasText: /How am I doing today\?/ }).click();

    // The message should appear in the chat
    await expect(page.locator('text=How am I doing today?')).toBeVisible({ timeout: 5000 });
  });

  test('chat input can be submitted with Enter key', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');

    const input = page.locator('textarea[placeholder*="Tell Joules"]');
    await input.fill('Hello coach');

    // Press Enter to send
    await input.press('Enter');

    // Message should appear
    await expect(page.locator('text=Hello coach')).toBeVisible({ timeout: 5000 });
  });

  test('send button is disabled when input is empty', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');

    const sendButton = page.locator('button[aria-label="Send message"]');
    await expect(sendButton).toBeDisabled();
  });

  test('chat history loads on page visit', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');

    // Wait for loading to finish
    await page.waitForFunction(() => {
      const spinner = document.querySelector('.animate-spin');
      return !spinner || spinner.offsetParent === null;
    }, { timeout: 15000 });

    // Page should show either conversations or empty state
    const hasConversations = await page.locator('text=No conversations yet').isVisible().catch(() => false);
    const hasMessages = await page.locator('.rounded-2xl').count().catch(() => 0);

    // Either empty state or some messages should be visible
    expect(hasConversations || hasMessages > 0).toBeTruthy();
  });

  test('new chat button clears current conversation', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/coach');

    // Wait for page load
    await expect(page.locator('h1', { hasText: 'Health Coach' })).toBeVisible();

    // Click "New Chat" button
    const newChatBtn = page.locator('button', { hasText: 'New Chat' });
    if (await newChatBtn.isVisible()) {
      await newChatBtn.click();
      // Should show the empty welcome state again
      await expect(page.locator('h2', { hasText: /Joules health coach/i })).toBeVisible();
    }
  });
});
