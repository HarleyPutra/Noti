import { Injectable, signal } from '@angular/core';

import * as GoNoteService from '../../../bindings/noti/noteservice';

export interface Note {
  id: string;
  user_id: string;
  title: string;
  content: string;
  mode: 'list' | 'lined' | 'squares' | 'dots' | 'browse';
  color: string;
  bgColor: string;
  pinned: boolean;
  width: number;
  height: number;
  pos_x: number;
  pos_y: number;
  created_at: number;
  updated_at: number;
  deleted: boolean;
  synced: boolean;
  version: number;
  vector_clock: string;
  timerDeadline: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
  picture: string;
}

@Injectable({ providedIn: 'root' })
export class NoteService {
  currentNote = signal<Note | null>(null);
  currentUser = signal<User | null>(null);

  async login(): Promise<User> {
    const user = await GoNoteService.Login() as unknown as User;
    this.currentUser.set(user);
    return user;
  }

  async getCurrentUser(): Promise<User | null> {
    const user = await GoNoteService.GetCurrentUser() as unknown as User;
    this.currentUser.set(user);
    return user;
  }

  async getNoteById(id: string): Promise<Note | null> {
    const notes = await GoNoteService.GetNotes(this.currentUser()?.id ?? '') as unknown as Note[];
    return notes.find(n => n.id === id) ?? null;
  }

  async updateNote(note: Note): Promise<void> {
    await GoNoteService.UpdateNote(note);
    this.currentNote.set(note);
  }

  async createNote(): Promise<void> {
    const user = this.currentUser();
    if (!user) return;
    await GoNoteService.CreateNote(user.id);
  }

  async deleteNote(id: string): Promise<void> {
    await GoNoteService.DeleteNote(id);
  }

  async setAlwaysOnTop(noteId: string, pinned: boolean): Promise<void> {
    await GoNoteService.SetAlwaysOnTop(noteId, pinned);
  }

  async syncNow(): Promise<void> {
    const user = this.currentUser();
    if (user) await GoNoteService.SyncNow(user.id);
  }
}
