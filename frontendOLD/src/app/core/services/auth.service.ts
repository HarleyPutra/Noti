import { Injectable, signal } from '@angular/core';
import { PlatformService } from './platform.service';
import { User } from '../models/todo.model';

@Injectable({ providedIn: 'root' })
export class AuthService {
  currentUser = signal<User | null>(null);

  constructor(private platform: PlatformService) {}

  async init(): Promise<void> {
    const saved = localStorage.getItem('user');
    if (saved) this.currentUser.set(JSON.parse(saved));

    // On desktop, verify Go session is still valid
    if (this.platform.isDesktop()) {
      try {
        const user = await this.platform.callGo<User | null>('GetCurrentUser');
        if (user) {
          this.currentUser.set(user);
          localStorage.setItem('user', JSON.stringify(user));
        } else {
          this.currentUser.set(null);
          localStorage.removeItem('user');
        }
      } catch {
        this.currentUser.set(null);
      }
    }
  }

  async login(): Promise<void> {
    if (this.platform.isDesktop()) {
      const user = await this.platform.callGo<User>('Login');
      this.currentUser.set(user);
      localStorage.setItem('user', JSON.stringify(user));
    } else {
      // Capacitor Google Auth for mobile
      const { GoogleAuth } = await import('@codetrix-studio/capacitor-google-auth');
      const result = await GoogleAuth.signIn();
      const user: User = {
        id: result.id,
        email: result.email,
        name: result.name,
        picture: result.imageUrl
      };
      this.currentUser.set(user);
      localStorage.setItem('user', JSON.stringify(user));
    }
  }

  async logout(): Promise<void> {
    if (this.platform.isDesktop()) {
      await this.platform.callGo('Logout');
    } else {
      const { GoogleAuth } = await import('@codetrix-studio/capacitor-google-auth');
      await GoogleAuth.signOut();
    }
    this.currentUser.set(null);
    localStorage.removeItem('user');
  }

  isLoggedIn(): boolean {
    return this.currentUser() !== null;
  }
}
