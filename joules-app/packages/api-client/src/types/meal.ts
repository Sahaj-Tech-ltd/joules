export interface FoodItem {
  id: string;
  name: string;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  serving_size: string;
  source: string;
}

export interface Meal {
  id: string;
  timestamp: string;
  meal_type: string;
  photo_path: string | null;
  note: string;
  foods: FoodItem[];
}

export interface MealIdentifyResponse {
  foods: FoodItem[];
  confidence: 'high' | 'medium' | 'low';
  suggestions?: string[];
}

export interface FoodMemoryEntry {
  id: string;
  food_name: string;
  canonical_name: string | null;
  calories: number;
  protein: number;
  carbs: number;
  fat: number;
  fiber: number;
  serving_size: number | null;
  serving_unit: string | null;
  correction_count: number;
  source: string;
}

export interface FoodSearchResult {
  id?: number;
  barcode?: string;
  name: string;
  brand?: string;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  serving_size: string;
  ingredients?: string;
  source: string;
}

export interface FoodFavorite {
  id: string;
  name: string;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  serving_size: string;
  source: string;
  use_count: number;
}
