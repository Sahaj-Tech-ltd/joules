import { api } from '../api';

export interface Group {
  id: string;
  name: string;
  description: string;
  member_count: number;
  is_member: boolean;
  created_at: string;
}

export interface GroupLeaderboardEntry {
  user_id: string;
  user_name: string;
  score: number;
  rank: number;
}

export interface GroupChallenge {
  id: string;
  name: string;
  description: string;
  start_date: string;
  end_date: string;
  participants: number;
}

export function fetchGroups(): Promise<Group[]> {
  return api.get<Group[]>('/groups/');
}

export function discoverGroups(): Promise<Group[]> {
  return api.get<Group[]>('/groups/discover');
}

export function createGroup(group: { name: string; description: string }): Promise<Group> {
  return api.post<Group>('/groups/', group);
}

export function joinGroup(code: string): Promise<Group> {
  return api.post<Group>('/groups/join', { code });
}

export function fetchGroup(id: string): Promise<Group> {
  return api.get<Group>(`/groups/${id}`);
}

export function leaveGroup(id: string): Promise<void> {
  return api.post(`/groups/${id}/leave`);
}

export function fetchGroupLeaderboard(id: string): Promise<GroupLeaderboardEntry[]> {
  return api.get<GroupLeaderboardEntry[]>(`/groups/${id}/leaderboard`);
}

export function fetchGroupChallenges(id: string): Promise<GroupChallenge[]> {
  return api.get<GroupChallenge[]>(`/groups/${id}/challenges`);
}

export function createGroupChallenge(
  id: string,
  challenge: { name: string; description: string; end_date: string }
): Promise<GroupChallenge> {
  return api.post<GroupChallenge>(`/groups/${id}/challenges`, challenge);
}
