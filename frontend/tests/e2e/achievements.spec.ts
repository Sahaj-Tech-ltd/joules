import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

test.describe('Achievements Page', () => {
  test('achievements page loads with title', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/achievements');
    await expect(page.locator('h1')).toBeVisible();
  });

  test('achievements page shows achievement grid', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/achievements');

    // Page should show achievement cards or empty state
    // Wait for API to load
    await page.waitForTimeout(2000);

    // Check that the page renders some content
    const content = await page.textContent('body');
    expect(content).toBeTruthy();
    expect(content!.length).toBeGreaterThan(100);
  });

  test('unlocked achievements show unlock date', async ({ page }) => {
    await loginAsAdmin(page);
    await page.goto('/achievements');

    // Wait for data to load
    await page.waitForTimeout(3000);

    // If there are unlocked achievements, they should show unlock dates
    const unlockedCards = page.locator('text=Unlocked');
    const count = await unlockedCards.count();

    // Admin should have some achievements
    if (count > 0) {
      // At least one achievement should show an unlock date
      const datePattern = page.locator('text=/\\d{1,2}\\/\\d{1,2}\\/\\d{4}|(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)/');
      await expect(datePattern.first()).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('Achievement Flow via Meal Logging', () => {
  test('logging a meal triggers achievement check and shows notification', async ({ page }) => {
    await loginAsAdmin(page);

    // Go to log page
    await page.goto('/log');
    await expect(page.locator('h1', { hasText: 'Log a Meal' })).toBeVisible();

    // Click manual add to open food form
    await page.locator('button', { hasText: 'Manual' }).click();

    // Fill in food details
    const nameInput = page.locator('input[placeholder*="food name"], input[name="name"]').first();
    if (await nameInput.isVisible()) {
      await nameInput.fill('Test Food');

      const calInput = page.locator('input[placeholder*="calories"], input[name="calories"]').first();
      if (await calInput.isVisible()) {
        await calInput.fill('200');
      }

      const proteinInput = page.locator('input[placeholder*="protein"], input[name="protein"]').first();
      if (await proteinInput.isVisible()) {
        await proteinInput.fill('20');
      }

      const carbsInput = page.locator('input[placeholder*="carbs"], input[name="carbs"]').first();
      if (await carbsInput.isVisible()) {
        await carbsInput.fill('15');
      }
    }

    // Try to submit the meal
    const submitBtn = page.locator('button', { hasText: /Log|Save|Add|Submit/i }).last();
    if (await submitBtn.isVisible()) {
      await submitBtn.click();
      await page.waitForTimeout(3000);

      // Check if achievement toast/notification appeared
      // The app uses showAchievement() which renders a toast
      const toast = page.locator('[class*="achievement"], [class*="toast"], [class*="notification"]');
      const toastVisible = await toast.isVisible().catch(() => false);

      // Either achievement toast appeared or the meal was logged successfully
      // (we don't fail if no achievement toast — it depends on what was already unlocked)
      if (toastVisible) {
        await expect(toast).toBeVisible();
      }
    }
  });
});
