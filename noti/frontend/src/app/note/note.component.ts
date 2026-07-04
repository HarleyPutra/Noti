import { Component, OnInit, OnDestroy, signal, HostListener, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { interval, Subscription } from 'rxjs';
import { TiptapEditorDirective, TiptapFloatingMenuDirective } from 'ngx-tiptap';
import { Events } from '@wailsio/runtime';
import { Editor, Extension } from '@tiptap/core';
import StarterKit from '@tiptap/starter-kit';
import Suggestion from '@tiptap/suggestion';
import TaskList from '@tiptap/extension-task-list';
import TaskItem from '@tiptap/extension-task-item';
import Placeholder from '@tiptap/extension-placeholder';
import tippy, { Instance } from 'tippy.js';

import * as GoNoteService from '../../../bindings/noti/noteservice';

// === SLASH COMMANDS EXTENSION ===
const getCommandItems = ({ query }: { query: string }) => {
  return [
    {
      title: 'Heading 1',
      icon: 'H1',
      command: ({ editor, range }: any) => {
        editor.chain().focus().deleteRange(range).setNode('heading', { level: 1 }).run();
      },
    },
    {
      title: 'Heading 2',
      icon: 'H2',
      command: ({ editor, range }: any) => {
        editor.chain().focus().deleteRange(range).setNode('heading', { level: 2 }).run();
      },
    },
    {
      title: 'Bullet List',
      icon: '•',
      command: ({ editor, range }: any) => {
        editor.chain().focus().deleteRange(range).toggleBulletList().run();
      },
    },
    {
      title: 'Task List',
      icon: '☑',
      command: ({ editor, range }: any) => {
        editor.chain().focus().deleteRange(range).toggleTaskList().run();
      },
    },
  ].filter(item => item.title.toLowerCase().startsWith(query.toLowerCase())).slice(0, 5);
};

const CommandsExtension = Extension.create({
  name: 'slashCommands',
  addOptions() {
    return {
      suggestion: {
        char: '/',
        command: ({ editor, range, props }: any) => {
          props.command({ editor, range });
        },
        items: getCommandItems,
        render: () => {
          let component: HTMLDivElement;
          let popup: Instance[];
          let selectedIndex = 0;
          let currentItems: any[] = [];

          let executeCommand: any;

          const renderMenu = (items: any[], activeIndex: number) => {
            component.innerHTML = '';
            if (items.length === 0) {
              component.innerHTML = '<div class="px-2 py-1 text-xs opacity-50">No results</div>';
              return;
            }
            items.forEach((item, index) => {
              const btn = document.createElement('button');
              btn.className = `w-full text-left px-2 py-1.5 text-xs rounded transition-colors flex items-center gap-2 ${index === activeIndex ? 'bg-[#875c5c] text-[#f2ebe1]' : 'text-[#e8e0d5] hover:bg-[#4a3a3a]'}`;
              btn.innerHTML = `<span class="font-bold opacity-70 w-4">${item.icon}</span> ${item.title}`;
              btn.addEventListener('click', () => {
                if (executeCommand) executeCommand(item);
              });
              component.appendChild(btn);
            });
          };

          return {
            onStart: (props: any) => {
              currentItems = props.items;
              executeCommand = props.command;
              selectedIndex = 0;

              component = document.createElement('div');
              component.className = 'flex flex-col p-1 rounded-lg shadow-2xl border';
              component.style.backgroundColor = '#2b2323';
              component.style.borderColor = '#4a3a3a';
              component.style.minWidth = '160px';

              renderMenu(currentItems, selectedIndex);

              popup = tippy('body', {
                getReferenceClientRect: props.clientRect,
                appendTo: () => document.body,
                content: component,
                showOnCreate: true,
                interactive: true,
                trigger: 'manual',
                placement: 'bottom-start',
              });
            },
            onUpdate: (props: any) => {
              currentItems = props.items;
              executeCommand = props.command;
              selectedIndex = 0;
              renderMenu(currentItems, selectedIndex);
              popup[0].setProps({ getReferenceClientRect: props.clientRect });
            },
            onKeyDown: (props: any) => {
              if (props.event.key === 'Escape') {
                popup[0].hide();
                return true;
              }
              if (props.event.key === 'ArrowDown') {
                props.event.preventDefault();
                selectedIndex = (selectedIndex + 1) % currentItems.length;
                renderMenu(currentItems, selectedIndex);
                return true;
              }
              if (props.event.key === 'ArrowUp') {
                props.event.preventDefault();
                selectedIndex = (selectedIndex + currentItems.length - 1) % currentItems.length;
                renderMenu(currentItems, selectedIndex);
                return true;
              }
              if (props.event.key === 'Enter') {
                props.event.preventDefault();

                const item = currentItems[selectedIndex];
                if (item && executeCommand) {
                  executeCommand(item);
                }
                return true;
              }
              return false;
            },
            onExit: () => {
              popup[0].destroy();
            },
          };
        },
      },
    };
  },
  addProseMirrorPlugins() {
    return [Suggestion({ editor: this.editor, ...this.options.suggestion })];
  },
});

@Component({
  selector: 'app-note',
  standalone: true,
  imports: [CommonModule, FormsModule, TiptapEditorDirective],
  templateUrl: './note.component.html'
})
export class NoteComponent implements OnInit, OnDestroy {
  Math = Math;
  String = String;

  note = signal<any | null>(null);

  menuState = signal<'none' | 'main' | 'customize'>('none');
  menuX = 0;
  menuY = 0;
  showTimer = signal(false);
  isSaving = false;

  timerTotal = 0;

  // Hours & Minutes inputs! Defaults to 1 hr 00 mins
  inputHr = "01";
  inputMin = "00";

  timerSub?: Subscription;

  private wakeLockAudio = new Audio('data:audio/wav;base64,UklGRigAAABXQVZFZm10IBIAAAABAAEARKwAAIhYAQACABAAAABkYXRhAgAAAAEA');

  editor = new Editor({
    extensions: [
      StarterKit,
      TaskList,
      TaskItem.configure({ nested: true }),
      Placeholder.configure({
        placeholder: "Type '/' for commands...",
      }),
      CommandsExtension,
    ],
    editorProps: {
      attributes: {
        class: 'focus:outline-none min-h-[300px] w-full text-[15px] leading-relaxed',
      },
    },
    onUpdate: () => {
      this.saveContent();
    },
  });

  constructor(
    private route: ActivatedRoute,
    private ngZone: NgZone,
    private cdr: ChangeDetectorRef
  ) {}

  async ngOnInit() {
    Events.On("sync-complete", () => {
      this.ngZone.run(() => {
        this.isSaving = false;
        this.cdr.detectChanges();
      });
    });

    const noteId = this.route.snapshot.queryParamMap.get('noteId');
    if (noteId) {
      try {
        const n = await GoNoteService.GetNote(noteId);
        if (n) {
          this.note.set(n);
          this.checkExistingTimer();

          if (n.content) {
            try {
              const parsed = JSON.parse(n.content);
              this.editor.commands.setContent(parsed);
            } catch {
              this.editor.commands.setContent(n.content);
            }
          }
        }
      } catch (err) {
        console.error("Failed to load note data:", err);
      }
    }

    Events.On("menu-action", (e: any) => {
      this.ngZone.run(() => {
        const type = e.data[0];
        const value = e.data[1];
        const targetId = e.data[2];

        if (targetId !== this.note().id) return;

        if (type === 'mode') this.setMode(value);
        if (type === 'color') this.setColor(value);
        if (type === 'accent') this.setAccent(value); // NEW LINE!
        if (type === 'new') this.newNote();
        if (type === 'delete') this.deleteNote();
        if (type === 'timer') this.showTimer.set(true);
        if (type === 'debug-prune') {
          GoNoteService.PruneArtifactNotes().then((deletedCount: number) => {
            console.log(`Garbage Collector ran! Purged ${deletedCount} artifact notes.`);
          });
        }
      });
    });
  }

  ngOnDestroy() {
    this.editor.destroy();
    this.wakeLockAudio.pause();

    if (this.timerSub) {
      this.timerSub.unsubscribe();
    }
  }

  @HostListener('window:beforeunload')
  flushSaveOnClose() {
    const n = this.note();
    if (n) {
      const updated = {
        ...n,
        content: JSON.stringify(this.editor.getJSON())
      };
      GoNoteService.UpdateNote(updated);
    }
  }

  async saveContent() {
    const n = this.note();
    if (!n) return;

    this.isSaving = true;
    this.cdr.detectChanges();

    const updated = {
      ...n,
      content: JSON.stringify(this.editor.getJSON())
    };

    try {
      await GoNoteService.UpdateNote(updated);
    } finally {
      window.setTimeout(() => {
        this.ngZone.run(() => {
          this.isSaving = false;
          this.cdr.detectChanges();
        });
      }, 400);
    }
  }

  async forceSync() {
    this.isSaving = true;
    this.cdr.detectChanges();

    try {
      const user = await GoNoteService.GetCurrentUser();
      if (user && (user as any).id) {
        await GoNoteService.SyncNow((user as any).id);
      }
    } catch (err) {
      this.ngZone.run(() => {
        this.isSaving = false;
        this.cdr.detectChanges();
      });
    }
  }

  setMode(newMode: string) {
    this.note.update(n => n ? { ...n, mode: newMode } : n);
    this.saveContent();
  }

  setColor(event: any) {
    const newColor = event.target ? event.target.value : event;
    this.note.update(n => n ? { ...n, color: newColor } : n);
    this.saveContent();
  }

  setAccent(newAccent: string) {
    this.note.update(n => n ? { ...n, accentColor: newAccent } : n);
    this.saveContent();
  }

  setBgColor(event: any) {
    const newBg = event.target ? event.target.value : event;
    this.note.update(n => n ? { ...n, bgColor: newBg } : n);
    this.saveContent();
  }

  async newNote() {
    const user = await GoNoteService.GetCurrentUser();
    if(user) await GoNoteService.CreateNote((user as any).id);
    this.closeAll();
  }

  async deleteNote() {
    const n = this.note();
    if (!n) return;
    await GoNoteService.DeleteNote(n.id);
    GoNoteService.HideWindow(n.id);
  }

  closeNote() {
    const n = this.note();
    if (n && n.id) {
      GoNoteService.HideWindow(n.id);
    }
  }

  async togglePin() {
    const n = this.note();
    if (!n) return;

    const updated = { ...n, pinned: !n.pinned };
    this.note.set(updated);
    this.isSaving = true;
    this.cdr.detectChanges();

    await GoNoteService.SetAlwaysOnTop(n.id, updated.pinned);
    await GoNoteService.UpdateNote(updated);

    window.setTimeout(() => {
      this.ngZone.run(() => {
        this.isSaving = false;
        this.cdr.detectChanges();
      });
    }, 400);
  }

  toggleHeading(level: 1 | 2 | 3) {
    this.editor.chain().focus().toggleHeading({ level }).run();
  }

  toggleTaskList() {
    this.editor.chain().focus().toggleTaskList().run();
  }

  toggleBulletList() {
    this.editor.chain().focus().toggleBulletList().run();
  }

  // === UI INPUT FORMATTERS ===

  formatInput(type: 'hr' | 'min') {
    if (type === 'hr') {
      let val = parseInt(this.inputHr, 10);
      if (isNaN(val)) val = 0;
      this.inputHr = String(val).padStart(2, '0');
    } else {
      let val = parseInt(this.inputMin, 10);
      if (isNaN(val)) val = 0;
      if (val > 59) val = 59; // Cap minutes at 59
      this.inputMin = String(val).padStart(2, '0');
    }
  }

  onHrInput(event: any, nextElement: HTMLInputElement) {
    if (this.inputHr.length >= 2) {
      nextElement.focus();
      nextElement.select();
    }
  }

  // === STATELESS TIMER LOGIC ===

  get timerDisplay(): string {
    const h = Math.floor(this.timerTotal / 3600);
    const m = Math.floor((this.timerTotal % 3600) / 60);
    const s = this.timerTotal % 60;

    // Show HH:MM:SS if over an hour, otherwise just MM:SS
    if (h > 0) {
        return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
    }
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  }

  checkExistingTimer() {
    const n = this.note();
    if (n && n.timerDeadline && n.timerDeadline > Date.now()) {
      this.startVisualTimer(n.timerDeadline);
    }
  }

  async startCountdown() {
    this.showTimer.set(false);

    // Math is now Hours + Minutes!
    const hr = parseInt(this.inputHr, 10) || 0;
    const min = parseInt(this.inputMin, 10) || 0;
    const totalMs = ((hr * 3600) + (min * 60)) * 1000;

    // Prevent starting a 0:00 timer
    if (totalMs <= 0) return;

    const deadline = Date.now() + totalMs;

    const n = this.note();
    if (n) {
      const updated = { ...n, timerDeadline: deadline };
      this.note.set(updated);
      await GoNoteService.UpdateNote(updated);
    }

    this.startVisualTimer(deadline);
  }

  private startVisualTimer(deadline: number) {
    this.wakeLockAudio.loop = true;
    this.wakeLockAudio.play().catch(e => console.log('Wake lock suppressed:', e));

    if (this.timerSub) {
      this.timerSub.unsubscribe();
    }

    this.timerSub = interval(1000).subscribe(() => {
      const now = Date.now();
      const remainingSeconds = Math.max(0, Math.ceil((deadline - now) / 1000));

      this.timerTotal = remainingSeconds;

      if (remainingSeconds <= 0) {
        this.timerSub?.unsubscribe();
        this.wakeLockAudio.pause();

        const n = this.note();
        if (n) {
          const updated = { ...n, timerDeadline: 0 };
          this.note.set(updated);
          GoNoteService.UpdateNote(updated);
        }
      }

      this.cdr.detectChanges();
    });
  }

  onRightClick(e: MouseEvent) {
    e.preventDefault();
    GoNoteService.ShowContextMenu(e.screenX, e.screenY, this.note().id);
  }

closeAll() {
    this.menuState.set('none');
    this.showTimer.set(false);

    GoNoteService.HideContextMenu();
  }

  // Quick helper for the sub-menu back button
  backToMainMenu(e: MouseEvent) {
    e.stopPropagation();
    this.menuState.set('main');
  }
}
