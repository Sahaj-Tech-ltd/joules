import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers/auth';

test.describe('Dashboard', () => {
  test('dashboard loads after login with greeting', async ({ page }) => {
    await loginAsAdmin(page);
    await expect(page.locator('h1')).toContainText(/Good/);
  });

  test('dashboard shows macro rings section', async ({ page }) => {
    await loginAsAdmin(page);
    await expect(page.locator('p', { hasText: /^Calories$/ })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('p', { hasText: /^Protein$/ })).toBeVisible();
  });

  test('sign out clears token and redirects to login', async ({ page }) => {
    await loginAsAdmin(page);
    const signOutButton = page.locator('button', { hasText: 'Sign out' });
    await expect(signOutButton).toBeVisible();
    await signOutButton.click();
    await page.waitForURL('**/login', { timeout: 10000 });
    const token = await page.evaluate(() => localStorage.getItem('auth_token'));
    expect(token).toBeNull();
  });
});
