import { Component } from '@angular/core';
import { NoteService } from '../services/note.service';

import { Window } from '@wailsio/runtime';
import * as GoAuth from '@bindings/noti/auth';
import * as GoNoteService from '@bindings/noti/noteservice';

@Component({
  selector: 'app-login',
  standalone: true,
  template: `
    <div class="min-h-screen flex items-center justify-center"
         style="background:#1a0f0f">
      <div class="text-center p-10 rounded-2xl"
           style="background:#2d1515">
        <div class="text-5xl mb-4">📋</div>
        <h1 class="text-2xl font-semibold mb-2"
            style="color:#f5e6d3; font-family:'Georgia',serif">
          Noti
        </h1>
        <p class="text-sm mb-8" style="color:#c4a882">
          Your cozy floating notebook
        </p>
        <button
          (click)="login()"
          [disabled]="loading"
          class="flex items-center gap-3 mx-auto px-6 py-3 rounded-xl text-sm font-medium transition-all disabled:opacity-50"
          style="background:#6b3f3f; color:#f5e6d3"
        >
          <svg viewBox="0 0 24 24" width="18" height="18">
            <path fill="#f5e6d3" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
            <path fill="#f5e6d3" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
            <path fill="#f5e6d3" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"/>
            <path fill="#f5e6d3" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
          </svg>
          {{ loading ? 'Signing in...' : 'Sign in with Google' }}
        </button>
        @if (error) {
          <p class="mt-4 text-sm" style="color:#e07070">{{ error }}</p>
        }
      </div>
    </div>
  `
})
export class LoginComponent {
  loading = false;
  error = '';

  constructor(private noteService: NoteService) {}

  async login() {
    this.loading = true;
    this.error = '';

    try {
      const user = await GoNoteService.Login();

      if (user && user.id) {
        await GoNoteService.RestoreWindows(user.id);

        Window.Close();
      }
    } catch (e: any) {
      console.error(e);
      this.error = 'Login failed. Please try again.';
    } finally {
      this.loading = false;
    }
  }
}
