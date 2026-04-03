import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test('login page loads with form elements', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toContainText('Sign in');
  });

  test('login with invalid credentials stays on login page', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#email', 'nonexistent@test.com');
    await page.fill('#password', 'wrongpassword123');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);
    expect(page.url()).toContain('/login');
    await expect(page.locator('#email')).toBeVisible();
  });

  test('login with valid admin credentials redirects to dashboard', async ({ page }) => {
    await page.goto('/login');
    await page.fill('#email', 'admin@joules.local');
    await page.fill('#password', 'asdfghjk2003@P');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard', { timeout: 10000 });
    const token = await page.evaluate(() => localStorage.getItem('auth_token'));
    expect(token).toBeTruthy();
  });

  test('unauthenticated access to dashboard redirects to login', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForURL('**/login', { timeout: 10000 });
    expect(page.url()).toContain('/login');
  });

  test('signup page loads with form elements', async ({ page }) => {
    await page.goto('/signup');
    await expect(page.locator('#signup-email')).toBeVisible();
    await expect(page.locator('#signup-password')).toBeVisible();
    await expect(page.locator('#signup-confirm')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toContainText('Create account');
  });
});
