import { test, expect } from '@playwright/test';

test.describe('API integration', () => {
  test('login API returns access_token', async ({ request }) => {
    const response = await request.post('/api/auth/login', {
      data: {
        email: 'admin@joules.local',
        password: 'asdfghjk2003@P',
      },
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body.data.access_token).toBeTruthy();
    expect(typeof body.data.access_token).toBe('string');
  });

  test('meals endpoint requires authentication', async ({ request }) => {
    const response = await request.get('/api/meals');
    expect(response.status()).toBe(401);
  });

  test('export CSV requires authentication', async ({ request }) => {
    const response = await request.get('/api/export/csv');
    expect(response.status()).toBe(401);
  });

  test('favorites CRUD operations', async ({ request }) => {
    const loginResponse = await request.post('/api/auth/login', {
      data: {
        email: 'admin@joules.local',
        password: 'asdfghjk2003@P',
      },
    });
    const loginBody = await loginResponse.json();
    const token = loginBody.data.access_token;

    const createResponse = await request.post('/api/favorites', {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        name: 'Test Favorite Food',
        calories: 250,
        protein_g: 10,
        carbs_g: 30,
        fat_g: 8,
        fiber_g: 2,
        serving_size: '1 serving',
        source: 'manual',
      },
    });
    expect(createResponse.ok()).toBeTruthy();
    const created = await createResponse.json();
    const favoriteId = created.data.id;
    expect(favoriteId).toBeTruthy();

    const listResponse = await request.get('/api/favorites', {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(listResponse.ok()).toBeTruthy();
    const listBody = await listResponse.json();
    const names = listBody.data.map((f: { name: string }) => f.name);
    expect(names).toContain('Test Favorite Food');

    const deleteResponse = await request.delete(`/api/favorites/${favoriteId}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(deleteResponse.ok()).toBeTruthy();
  });
});
