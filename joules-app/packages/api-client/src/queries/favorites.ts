import { api } from '../api';
import type { FoodFavorite } from '../types';

export function fetchFavorites(): Promise<FoodFavorite[]> {
  return api.get<FoodFavorite[]>('/favorites/');
}

export function fetchTopFavorites(limit?: number): Promise<FoodFavorite[]> {
  const query = limit ? `?limit=${limit}` : '';
  return api.get<FoodFavorite[]>(`/favorites/top${query}`);
}

export function createFavorite(favorite: Partial<FoodFavorite>): Promise<FoodFavorite> {
  return api.post<FoodFavorite>('/favorites/', favorite);
}

export function useFavorite(id: string): Promise<FoodFavorite> {
  return api.post<FoodFavorite>(`/favorites/${id}/use`);
}

export function deleteFavorite(id: string): Promise<void> {
  return api.del(`/favorites/${id}`);
}
