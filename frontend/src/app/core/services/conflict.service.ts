import { Injectable } from '@angular/core';
import { Todo } from '../models/todo.model';

@Injectable({ providedIn: 'root' })
export class ConflictService {

  mergeTodos(local: Todo[], remote: Todo[]): Todo[] {
    const index = new Map<string, Todo>();

    for (const t of local) index.set(t.id, t);

    for (const remoteTodo of remote) {
      const localTodo = index.get(remoteTodo.id);
      if (!localTodo) {
        index.set(remoteTodo.id, remoteTodo); // new from another device
      } else {
        index.set(remoteTodo.id, this.mergeSingle(localTodo, remoteTodo));
      }
    }

    return Array.from(index.values());
  }

  private mergeSingle(local: Todo, remote: Todo): Todo {
    const localClock  = this.parseClock(local.vector_clock);
    const remoteClock = this.parseClock(remote.vector_clock);
    const rel = this.compareClocks(localClock, remoteClock);

    if (rel === 'local-newer')  return local;
    if (rel === 'remote-newer') return remote;

    // Concurrent conflict — merge field by field
    const mergedClock = this.mergeClock(localClock, remoteClock);
    return {
      ...local,
      // Most recently updated title/notes wins
      title:        remote.updated_at > local.updated_at ? remote.title : local.title,
      notes:        remote.updated_at > local.updated_at ? remote.notes : local.notes,
      // done=true wins across any device
      done:         local.done || remote.done,
      // deleted=true wins across any device
      deleted:      local.deleted || remote.deleted,
      updated_at:   Math.max(local.updated_at, remote.updated_at),
      version:      Math.max(local.version, remote.version) + 1,
      vector_clock: JSON.stringify(mergedClock),
      synced:       false
    };
  }

  private compareClocks(
    a: Record<string, number>,
    b: Record<string, number>
  ): 'local-newer' | 'remote-newer' | 'concurrent' {
    const keys = [...new Set([...Object.keys(a), ...Object.keys(b)])];
    let aGtB = false, bGtA = false;
    for (const k of keys) {
      if ((a[k] ?? 0) > (b[k] ?? 0)) aGtB = true;
      if ((b[k] ?? 0) > (a[k] ?? 0)) bGtA = true;
    }
    if (aGtB && !bGtA) return 'local-newer';
    if (bGtA && !aGtB) return 'remote-newer';
    return 'concurrent';
  }

  private mergeClock(
    a: Record<string, number>,
    b: Record<string, number>
  ): Record<string, number> {
    const result: Record<string, number> = { ...a };
    for (const [k, v] of Object.entries(b)) {
      result[k] = Math.max(result[k] ?? 0, v);
    }
    return result;
  }

  private parseClock(s: string): Record<string, number> {
    try { return JSON.parse(s); } catch { return {}; }
  }
}
