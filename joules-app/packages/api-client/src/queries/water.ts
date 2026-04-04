import { api } from '../api';
import type { WaterLog } from '../types';

export function fetchWaterLogs(date?: string): Promise<WaterLog[]> {
  const query = date ? `?date=${encodeURIComponent(date)}` : '';
  return api.get<WaterLog[]>(`/water${query}`);
}

export function logWater(amountMl: number, date?: string): Promise<WaterLog> {
  return api.post<WaterLog>('/water/', {
    amount_ml: amountMl,
    ...(date ? { date } : {}),
  });
}
