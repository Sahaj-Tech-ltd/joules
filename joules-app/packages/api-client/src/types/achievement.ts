export interface Achievement {
  id: string;
  type: string;
  title: string;
  description: string;
  category: string;
  progress_current: number;
  progress_target: number;
  unlocked_at: string;
}
