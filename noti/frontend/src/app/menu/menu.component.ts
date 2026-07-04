import { Component, HostListener, OnInit, signal } from '@angular/core';
import { Events } from '@wailsio/runtime';
import * as GoNoteService from '../../../bindings/noti/noteservice';

@Component({
  selector: 'app-menu',
  standalone: true,
  template: `
    <div class="w-screen h-screen flex flex-col text-[13px] border border-[#4a3a3a] overflow-hidden rounded-md shadow-2xl font-medium"
         style="background-color: #2b2323; color: #e8e0d5; user-select: none;">

      @if (menuState() === 'main') {
        <div class="flex flex-col p-1.5 gap-0.5">
          <button class="w-full text-left px-2 py-1.5 hover:bg-[#875c5c] rounded transition-colors flex justify-between items-center group"
                  (click)="setCustomize()">
            Style & Layout
            <span class="opacity-50 group-hover:opacity-100 font-bold">›</span>
          </button>
          <button class="w-full text-left px-2 py-1.5 hover:bg-[#875c5c] rounded transition-colors" (click)="emitAction('timer', '')">Set Timer</button>

          <div class="border-t border-[#4a3a3a] my-1 mx-1"></div>

          <button class="w-full text-left px-2 py-1.5 hover:bg-[#875c5c] rounded transition-colors" (click)="emitAction('new', '')">+ New Note</button>
          <button class="w-full text-left px-2 py-1.5 hover:bg-red-500/80 hover:text-white text-red-400 rounded transition-colors" (click)="emitAction('delete', '')">Delete Note</button>

          <div class="border-t border-[#4a3a3a] my-1 mx-1"></div>

          <button class="w-full text-left px-2 py-1.5 hover:bg-yellow-600/20 text-yellow-500 rounded transition-colors flex justify-between items-center group"
                  (click)="setDebug()">
            Debug Tools
            <span class="opacity-50 group-hover:opacity-100 font-bold">›</span>
          </button>

          <div class="border-t border-[#4a3a3a] my-1 mx-1"></div>
          <button class="w-full text-left px-2 py-1.5 hover:bg-red-900/40 hover:text-red-300 text-red-400 rounded transition-colors"
                  (click)="onSignOut()">
            Log Out
          </button>
        </div>
      }

      @if (menuState() === 'customize') {
        <div class="p-1.5 flex flex-col h-full">
          <div class="pb-1.5 border-b border-[#4a3a3a] mb-2 flex items-center">
            <button class="hover:bg-[#4a3a3a] rounded px-1.5 py-1 text-xs text-[#a38c8c] flex items-center gap-1 transition-colors"
                    (click)="setMain()">
              <span class="font-bold">‹</span> Back
            </button>
            <span class="mx-auto pr-6 text-[10px] font-bold tracking-widest text-[#a38c8c] uppercase">Style</span>
          </div>

          <div class="flex flex-col gap-1.5 mb-3 px-1">
            <div class="flex gap-1.5">
              <button (click)="emitAction('mode', 'list')" class="flex-1 border border-[#4a3a3a] rounded py-1 hover:bg-[#875c5c] transition-colors">List</button>
              <button (click)="emitAction('mode', 'lined')" class="flex-1 border border-[#4a3a3a] rounded py-1 hover:bg-[#875c5c] transition-colors">Lined</button>
            </div>
            <div class="flex gap-1.5">
              <button (click)="emitAction('mode', 'squares')" class="flex-1 border border-[#4a3a3a] rounded py-1 hover:bg-[#875c5c] transition-colors">Squares</button>
              <button (click)="emitAction('mode', 'dots')" class="flex-1 border border-[#4a3a3a] rounded py-1 hover:bg-[#875c5c] transition-colors">Dots</button>
            </div>
          </div>

          <div class="flex flex-col px-1 mb-3">
            <span class="text-[10px] font-bold text-[#a38c8c] uppercase tracking-widest mb-1.5">Note Color</span>
            <div class="flex items-center justify-between">
              <button (click)="emitAction('color', '#f2ebe1')" class="w-[22px] h-[22px] rounded-full border border-[#4a3a3a] hover:scale-110 transition-transform" style="background-color: #f2ebe1;" title="Latte"></button>
              <button (click)="emitAction('color', '#e6eedd')" class="w-[22px] h-[22px] rounded-full border border-[#4a3a3a] hover:scale-110 transition-transform" style="background-color: #e6eedd;" title="Matcha"></button>
              <button (click)="emitAction('color', '#f4e1e1')" class="w-[22px] h-[22px] rounded-full border border-[#4a3a3a] hover:scale-110 transition-transform" style="background-color: #f4e1e1;" title="Sakura"></button>
              <button (click)="emitAction('color', '#e0e7ec')" class="w-[22px] h-[22px] rounded-full border border-[#4a3a3a] hover:scale-110 transition-transform" style="background-color: #e0e7ec;" title="Sky"></button>
              <div class="w-[1px] h-3 bg-[#4a3a3a] mx-0.5"></div>
              <div class="relative w-[22px] h-[22px] rounded-full shadow-inner cursor-pointer hover:scale-110 transition-transform" style="background: conic-gradient(red, yellow, lime, cyan, blue, magenta, red);" title="Custom Note Color">
                <input type="color" (mousedown)="openColorPicker()" (change)="onColorPicked('color', $event)" class="absolute inset-0 w-full h-full opacity-0 cursor-pointer">
                <div class="absolute inset-[3px] bg-[#2b2323] rounded-full pointer-events-none"></div>
              </div>
            </div>
          </div>

          <div class="flex flex-col px-1 mb-2">
            <span class="text-[10px] font-bold text-[#a38c8c] uppercase tracking-widest mb-1.5">Accent Color</span>
            <div class="flex items-center justify-between">
              <button (click)="emitAction('accent', '#4a3a3a')" class="w-[22px] h-[22px] rounded-full border border-[#2b2323] hover:scale-110 transition-transform" style="background-color: #4a3a3a;" title="Espresso"></button>
              <button (click)="emitAction('accent', '#875c5c')" class="w-[22px] h-[22px] rounded-full border border-[#2b2323] hover:scale-110 transition-transform" style="background-color: #875c5c;" title="Cocoa"></button>
              <button (click)="emitAction('accent', '#4a5c4a')" class="w-[22px] h-[22px] rounded-full border border-[#2b2323] hover:scale-110 transition-transform" style="background-color: #4a5c4a;" title="Forest"></button>
              <button (click)="emitAction('accent', '#4a5c6c')" class="w-[22px] h-[22px] rounded-full border border-[#2b2323] hover:scale-110 transition-transform" style="background-color: #4a5c6c;" title="Ocean"></button>
              <div class="w-[1px] h-3 bg-[#4a3a3a] mx-0.5"></div>
              <div class="relative w-[22px] h-[22px] rounded-full shadow-inner cursor-pointer hover:scale-110 transition-transform" style="background: conic-gradient(red, yellow, lime, cyan, blue, magenta, red);" title="Custom Accent Color">
                <input type="color" (mousedown)="openColorPicker()" (change)="onColorPicked('accent', $event)" class="absolute inset-0 w-full h-full opacity-0 cursor-pointer">
                <div class="absolute inset-[3px] bg-[#2b2323] rounded-full pointer-events-none"></div>
              </div>
            </div>
          </div>

        </div>
      }

      @if (menuState() === 'debug') {
        <div class="p-1.5 flex flex-col h-full">
          <div class="pb-1.5 border-b border-[#4a3a3a] mb-2 flex items-center">
            <button class="hover:bg-[#4a3a3a] rounded px-1.5 py-1 text-xs text-[#a38c8c] flex items-center gap-1 transition-colors"
                    (click)="setMain()">
              <span class="font-bold">‹</span> Back
            </button>
            <span class="mx-auto pr-6 text-[10px] font-bold text-yellow-500 tracking-widest uppercase">Debug</span>
          </div>

          <div class="flex flex-col gap-1.5 px-1">
            <button class="w-full text-left px-2 py-1.5 bg-yellow-600/10 hover:bg-yellow-600/20 text-yellow-500 border border-yellow-600/30 rounded transition-colors"
                    (click)="emitAction('debug-prune', '')">
              Run Garbage Collector
            </button>
          </div>
        </div>
      }
    </div>
  `
})
export class MenuComponent implements OnInit {
  menuState = signal<'main' | 'customize' | 'debug'>('main');
  canClose = false;
  isPickingColor = false;

  ngOnInit() {
    Events.On("menu-opened", () => {
      this.setMain();
      this.isPickingColor = false;
    });
  }

  setDebug() {
    this.menuState.set('debug');
    GoNoteService.ResizeContextMenu(200, 110);
  }

  setMain() {
    this.menuState.set('main');
    GoNoteService.ResizeContextMenu(200, 250); // Height of context menu
  }

  setCustomize() {
    this.menuState.set('customize');
    GoNoteService.ResizeContextMenu(220, 310);
  }

  @HostListener('window:blur')
  onWindowBlur() {
    if (!this.isPickingColor) {
      GoNoteService.HideContextMenu();
    }
  }

  openColorPicker() {
    this.isPickingColor = true;
  }

  // Updated to accept the type (color vs accent)
  onColorPicked(type: string, event: any) {
    this.emitAction(type, event.target.value);
  }

  emitAction(type: string, value: string) {
    GoNoteService.TriggerMenuAction(type, value);
  }

  async onSignOut() {
    try {
      // wipe the token, kill the notes, and spawn the login window
      await GoNoteService.Logout();

      // Hide the context menu window
      GoNoteService.HideContextMenu();
    } catch (error) {
      console.error("Failed to sign out:", error);
    }
  }
}
