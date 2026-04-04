import { api } from '../api';
import type { Meal, MealIdentifyResponse, FoodItem } from '../types';

export function fetchMeals(date?: string): Promise<Meal[]> {
  const query = date ? `?date=${encodeURIComponent(date)}` : '';
  return api.get<Meal[]>(`/meals${query}`);
}

export function fetchRecentMeals(limit?: number): Promise<Meal[]> {
  const query = limit ? `?limit=${limit}` : '';
  return api.get<Meal[]>(`/meals/recent${query}`);
}

export function identifyMeal(photo: FormData): Promise<MealIdentifyResponse> {
  return api.upload<MealIdentifyResponse>('/meals/identify', photo);
}

export function createMeal(meal: {
  meal_type: string;
  foods: Partial<FoodItem>[];
  note?: string;
  timestamp?: string;
}): Promise<Meal> {
  return api.post<Meal>('/meals/', meal);
}

export function deleteMeal(id: string): Promise<void> {
  return api.del(`/meals/${id}`);
}

export function updateFoodItem(
  mealId: string,
  foodId: string,
  updates: Partial<FoodItem>
): Promise<FoodItem> {
  return api.put<FoodItem>(`/meals/${mealId}/foods/${foodId}`, updates);
}

export function deleteFoodItem(mealId: string, foodId: string): Promise<void> {
  return api.del(`/meals/${mealId}/foods/${foodId}`);
}

export function carryForward(foods: Partial<FoodItem>[], mealType: string): Promise<Meal> {
  return api.post<Meal>('/meals/carry-forward', { foods, meal_type: mealType });
}
