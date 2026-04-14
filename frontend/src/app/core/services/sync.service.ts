import { Injectable } from '@angular/core';
import { PlatformService } from './platform.service';
import { DbService } from './db.service';
import { ConflictService } from './conflict.service';
import { AuthService } from './auth.service';
import { Todo } from '../models/todo.model';
import { Network } from '@capacitor/network';

@Injectable({ providedIn: 'root' })
export class SyncService {
  private syncing = false;

  constructor(
    private platform: PlatformService,
    private db: DbService,
    private conflict: ConflictService,
    private auth: AuthService
  ) {}

  // Call this after every local write — non-blocking
  syncInBackground(): void {
    if (this.syncing) return;
    setTimeout(() => this.runSync(), 0);
  }

  private async runSync(): Promise<void> {
    if (this.syncing) return;
    this.syncing = true;

    try {
      if (this.platform.isDesktop()) {
        // Go handles sync entirely on desktop
        const userId = this.auth.currentUser()?.id;
        if (userId) await this.platform.callGo('SyncNow', userId);
      } else {
        await this.mobilSync();
      }
    } catch (e) {
      // Offline or error — silently ignore, retry next time
      console.warn('Sync skipped:', e);
    } finally {
      this.syncing = false;
    }
  }

  private async mobilSync(): Promise<void> {
    const status = await Network.getStatus();
    if (!status.connected) return;

    const userId = this.auth.currentUser()?.id;
    if (!userId) return;

    // Pull from Drive
    const remote = await this.pullFromDrive();
    if (remote.length > 0) {
      const local = await this.db.getTodos(userId);
      const merged = this.conflict.mergeTodos(local, remote);
      for (const t of merged) await this.db.upsertTodo(t);
    }

    // Push unsynced local todos
    const unsynced = await this.db.getUnsynced(userId);
    if (unsynced.length > 0) {
      await this.pushToDrive(unsynced);
      for (const t of unsynced) await this.db.markSynced(t.id);
    }

    await this.db.setLastSyncTime(Date.now());
  }

  // Google Drive REST calls for mobile
  // These use the OAuth token stored after Google Sign-In
  private async pullFromDrive(): Promise<Todo[]> {
    try {
      const token = await this.getAccessToken();
      // Search for our sync file in appDataFolder
      const searchRes = await fetch(
        `https://www.googleapis.com/drive/v3/files?spaces=appDataFolder&q=name='todos_sync.json'`,
        { headers: { Authorization: `Bearer ${token}` } }
      );
      const { files } = await searchRes.json();
      if (!files?.length) return [];

      const fileRes = await fetch(
        `https://www.googleapis.com/drive/v3/files/${files[0].id}?alt=media`,
        { headers: { Authorization: `Bearer ${token}` } }
      );
      return await fileRes.json();
    } catch { return []; }
  }

  private async pushToDrive(todos: Todo[]): Promise<void> {
    const token = await this.getAccessToken();
    const body = JSON.stringify(todos);

    // Check if file exists
    const searchRes = await fetch(
      `https://www.googleapis.com/drive/v3/files?spaces=appDataFolder&q=name='todos_sync.json'`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    const { files } = await searchRes.json();

    if (files?.length) {
      // Update existing file
      await fetch(
        `https://www.googleapis.com/upload/drive/v3/files/${files[0].id}?uploadType=media`,
        {
          method: 'PATCH',
          headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json'
          },
          body
        }
      );
    } else {
      // Create new file in appDataFolder
      const meta = JSON.stringify({
        name: 'todos_sync.json',
        parents: ['appDataFolder']
      });
      const form = new FormData();
      form.append('metadata', new Blob([meta], { type: 'application/json' }));
      form.append('file', new Blob([body], { type: 'application/json' }));
      await fetch(
        'https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart',
        {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
          body: form
        }
      );
    }
  }

  private async getAccessToken(): Promise<string> {
    const { GoogleAuth } = await import('@codetrix-studio/capacitor-google-auth');
    const user = await GoogleAuth.refresh();
    return user.accessToken;
  }
}
