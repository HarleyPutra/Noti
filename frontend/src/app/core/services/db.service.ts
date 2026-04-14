import { Injectable } from '@angular/core';
import { PlatformService } from './platform.service';
import { Todo } from '../models/todo.model';

@Injectable({ providedIn: 'root' })
export class DbService {
  private sqlite: any = null;
  private db: any = null;

  constructor(private platform: PlatformService) {}

  async init(): Promise<void> {
    if (this.platform.isDesktop()) return; // Go handles DB on desktop

    const { CapacitorSQLite, SQLiteConnection } =
      await import('@capacitor-community/sqlite');

    this.sqlite = new SQLiteConnection(CapacitorSQLite);
    this.db = await this.sqlite.createConnection(
      'todos_db', false, 'no-encryption', 1, false
    );
    await this.db.open();
    await this.createTables();
  }

  private async createTables(): Promise<void> {
    await this.db.execute(`
      CREATE TABLE IF NOT EXISTS todos (
        id           TEXT PRIMARY KEY,
        user_id      TEXT NOT NULL,
        title        TEXT NOT NULL,
        notes        TEXT DEFAULT '',
        done         INTEGER DEFAULT 0,
        created_at   INTEGER NOT NULL,
        updated_at   INTEGER NOT NULL,
        deleted      INTEGER DEFAULT 0,
        synced       INTEGER DEFAULT 0,
        version      INTEGER DEFAULT 1,
        vector_clock TEXT DEFAULT '{}'
      );
      CREATE TABLE IF NOT EXISTS sync_meta (
        key   TEXT PRIMARY KEY,
        value TEXT
      );
    `);
  }

  async getTodos(userId: string): Promise<Todo[]> {
    if (!this.db) return [];
    const result = await this.db.query(
      `SELECT * FROM todos WHERE user_id = ? AND deleted = 0
       ORDER BY created_at DESC`,
      [userId]
    );
    return (result.values ?? []).map(this.mapRow);
  }

  async upsertTodo(todo: Todo): Promise<void> {
    if (!this.db) return;
    await this.db.run(`
      INSERT INTO todos
        (id,user_id,title,notes,done,created_at,updated_at,
         deleted,synced,version,vector_clock)
      VALUES (?,?,?,?,?,?,?,?,?,?,?)
      ON CONFLICT(id) DO UPDATE SET
        title=excluded.title, notes=excluded.notes,
        done=excluded.done, updated_at=excluded.updated_at,
        deleted=excluded.deleted, synced=excluded.synced,
        version=excluded.version, vector_clock=excluded.vector_clock
    `, [
      todo.id, todo.user_id, todo.title, todo.notes,
      todo.done ? 1 : 0, todo.created_at, todo.updated_at,
      todo.deleted ? 1 : 0, todo.synced ? 1 : 0,
      todo.version, todo.vector_clock
    ]);
  }

  async getUnsynced(userId: string): Promise<Todo[]> {
    if (!this.db) return [];
    const result = await this.db.query(
      `SELECT * FROM todos WHERE user_id = ? AND synced = 0`, [userId]
    );
    return (result.values ?? []).map(this.mapRow);
  }

  async markSynced(id: string): Promise<void> {
    if (!this.db) return;
    await this.db.run(`UPDATE todos SET synced = 1 WHERE id = ?`, [id]);
  }

  async getLastSyncTime(): Promise<number> {
    if (!this.db) return 0;
    const result = await this.db.query(
      `SELECT value FROM sync_meta WHERE key = 'last_sync'`
    );
    return result.values?.[0]?.value ?? 0;
  }

  async setLastSyncTime(time: number): Promise<void> {
    if (!this.db) return;
    await this.db.run(`
      INSERT INTO sync_meta (key,value) VALUES ('last_sync',?)
      ON CONFLICT(key) DO UPDATE SET value=excluded.value
    `, [time]);
  }

  private mapRow(row: any): Todo {
    return {
      ...row,
      done:    row.done === 1,
      deleted: row.deleted === 1,
      synced:  row.synced === 1,
    };
  }
}
