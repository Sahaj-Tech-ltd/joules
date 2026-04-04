import { api } from '../api';
import type { ExerciseEntry } from '../types';

export function fetchExercises(date?: string): Promise<ExerciseEntry[]> {
  const query = date ? `?date=${encodeURIComponent(date)}` : '';
  return api.get<ExerciseEntry[]>(`/exercise${query}`);
}

export function logExercise(exercise: {
  name: string;
  duration_min: number;
  calories_burned?: number;
  timestamp?: string;
}): Promise<ExerciseEntry> {
  return api.post<ExerciseEntry>('/exercise/', exercise);
}
