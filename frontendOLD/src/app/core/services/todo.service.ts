import { Injectable, signal } from '@angular/core';
import { PlatformService } from './platform.service';
import { DbService } from './db.service';
import { SyncService } from './sync.service';
import { AuthService } from './auth.service';
import { Todo } from '../models/todo.model';
import { v4 as uuidv4 } from 'uuid';

@Injectable({ providedIn: 'root' })
export class TodoService {
  todos = signal<Todo[]>([]);

  private get deviceId(): string {
    let id = localStorage.getItem('device_id');
    if (!id) { id = uuidv4(); localStorage.setItem('device_id', id); }
    return id;
  }

  constructor(
    private platform: PlatformService,
    private db: DbService,
    private sync: SyncService,
    private auth: AuthService
  ) {}

  async loadTodos(): Promise<void> {
    const userId = this.auth.currentUser()!.id;
    let todos: Todo[];

    if (this.platform.isDesktop()) {
      todos = await this.platform.callGo<Todo[]>('GetTodos', userId);
    } else {
      todos = await this.db.getTodos(userId);
    }

    this.todos.set(todos);
  }

  async createTodo(title: string, notes = ''): Promise<void> {
    const user = this.auth.currentUser()!;
    const now = Date.now();
    const todo: Todo = {
      id: uuidv4(),
      user_id: user.id,
      title, notes,
      done: false,
      created_at: now,
      updated_at: now,
      deleted: false,
      synced: false,
      version: 1,
      vector_clock: JSON.stringify({ [this.deviceId]: 1 })
    };

    await this.upsert(todo);
    this.todos.update(prev => [todo, ...prev]);
    this.sync.syncInBackground();
  }

  async updateTodo(id: string, changes: Partial<Pick<Todo, 'title' | 'notes'>>): Promise<void> {
    const current = this.todos().find(t => t.id === id)!;
    const updated = this.bumpClock({ ...current, ...changes, updated_at: Date.now(), synced: false });
    await this.upsert(updated);
    this.todos.update(prev => prev.map(t => t.id === id ? updated : t));
    this.sync.syncInBackground();
  }

  async toggleDone(id: string): Promise<void> {
    const current = this.todos().find(t => t.id === id)!;
    const updated = this.bumpClock({
      ...current, done: !current.done,
      updated_at: Date.now(), synced: false
    });
    await this.upsert(updated);
    this.todos.update(prev => prev.map(t => t.id === id ? updated : t));
    this.sync.syncInBackground();
  }

  async deleteTodo(id: string): Promise<void> {
    const current = this.todos().find(t => t.id === id)!;
    const updated = this.bumpClock({
      ...current, deleted: true,
      updated_at: Date.now(), synced: false
    });
    await this.upsert(updated);
    this.todos.update(prev => prev.filter(t => t.id !== id));
    this.sync.syncInBackground();
  }

  private bumpClock(todo: Todo): Todo {
    const clock = JSON.parse(todo.vector_clock ?? '{}');
    return {
      ...todo,
      version: todo.version + 1,
      vector_clock: JSON.stringify({
        ...clock,
        [this.deviceId]: (clock[this.deviceId] ?? 0) + 1
      })
    };
  }

  private async upsert(todo: Todo): Promise<void> {
    if (this.platform.isDesktop()) {
      await this.platform.callGo('UpsertTodo', todo);
    } else {
      await this.db.upsertTodo(todo);
    }
  }
}
