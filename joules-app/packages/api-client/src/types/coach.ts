export interface CoachMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

export interface CoachMemory {
  id: string;
  category: string;
  content: string;
  source: string;
  created_at: string;
}

export interface CoachReminder {
  id: string;
  type: string;
  message: string;
  reminder_time: string;
  enabled: boolean;
}

export interface IdentityQuote {
  id: string;
  quote: string;
  date: string;
  context_type: string;
}
