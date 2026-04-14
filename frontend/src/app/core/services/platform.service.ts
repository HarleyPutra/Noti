import { Injectable } from '@angular/core';

declare const window: any;

@Injectable({ providedIn: 'root' })
export class PlatformService {

  isDesktop(): boolean {
    return typeof window.__wails !== 'undefined' ||
           typeof (window as any).go !== 'undefined';
  }

  isMobile(): boolean {
    return typeof window.Capacitor !== 'undefined' &&
           window.Capacitor.isNativePlatform();
  }

  isWeb(): boolean {
    return !this.isDesktop() && !this.isMobile();
  }

  async callGo<T>(fn: string, ...args: any[]): Promise<T> {
    if (!this.isDesktop()) throw new Error('Not running on desktop');
    return await (window as any).go.main.App[fn](...args);
  }
}
