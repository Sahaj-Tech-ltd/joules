import { api } from '../api';

export interface Recipe {
  id: string;
  name: string;
  description: string;
  servings: number;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  foods: RecipeFood[];
  created_at: string;
}

export interface RecipeFood {
  name: string;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  serving_size: string;
  source: string;
}

export function fetchRecipes(): Promise<Recipe[]> {
  return api.get<Recipe[]>('/recipes/');
}

export function createRecipe(recipe: Partial<Recipe>): Promise<Recipe> {
  return api.post<Recipe>('/recipes/', recipe);
}

export function deleteRecipe(id: string): Promise<void> {
  return api.del(`/recipes/${id}`);
}

export function logFromRecipe(recipeId: string, mealType: string): Promise<unknown> {
  return api.post(`/meals/from-recipe/${recipeId}`, { meal_type: mealType });
}
