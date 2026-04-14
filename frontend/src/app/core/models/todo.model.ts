export interface Todo {
  id: string;
  user_id: string;
  title: string;
  notes: string;
  done: boolean;
  created_at: number;
  updated_at: number;
  deleted: boolean;
  synced: boolean;
  version: number;
  vector_clock: string;
}

export interface User {
  id: string;
  email: string;
  name: string;
  picture: string;
}
