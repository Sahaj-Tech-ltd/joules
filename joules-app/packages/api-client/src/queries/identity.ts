import { api } from '../api';
import type { IdentityQuote } from '../types';

export function fetchIdentityQuote(): Promise<IdentityQuote> {
  return api.get<IdentityQuote>('/identity/quote');
}
