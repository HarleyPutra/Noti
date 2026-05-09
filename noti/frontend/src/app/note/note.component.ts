import { Component, OnInit, OnDestroy, signal, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { TiptapEditorDirective, TiptapFloatingMenuDirective } from 'ngx-tiptap';
import { Editor } from '@tiptap/core';
import StarterKit from '@tiptap/starter-kit';
import TaskList from '@tiptap/extension-task-list';
import TaskItem from '@tiptap/extension-task-item';
import Placeholder from '@tiptap/extension-placeholder';

import * as GoNoteService from '../../../bindings/noti/noteservice';

@Component({
  selector: 'app-note',
  standalone: true,
  imports: [CommonModule, FormsModule, TiptapEditorDirective, TiptapFloatingMenuDirective],
  templateUrl: './note.component.html'
})
export class NoteComponent implements OnInit, OnDestroy {
  Math = Math;
  String = String;

  note = signal<any | null>(null);

  // -- UI State --
  showMenu = signal(false);
  showTimer = signal(false);
  isSaving = false;

  // -- Timer State --
  timerTotal = 0;
  timerProgress = 0;
  timerInputMin = 25;
  timerInterval: any;

  // -- Tiptap Editor --
  editor = new Editor({
    extensions: [
      StarterKit,
      TaskList,
      TaskItem.configure({ nested: true }),
      Placeholder.configure({
        placeholder: "Type or select a command...",
      })
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

  constructor(private route: ActivatedRoute) {}

  async ngOnInit() {
    const noteId = this.route.snapshot.queryParamMap.get('noteId');
    if (noteId) {
      try {
        const n = await GoNoteService.GetNote(noteId);
        if (n) {
          this.note.set(n);
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
  }

  ngOnDestroy() {
    this.editor.destroy();
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

    const updated = {
      ...n,
      content: JSON.stringify(this.editor.getJSON())
    };

    try {
      await GoNoteService.UpdateNote(updated);
    } finally {
      setTimeout(() => this.isSaving = false, 400);
    }
  }

  async forceSync() {
    this.isSaving = true;
    try {
      const user = await GoNoteService.GetCurrentUser();
      if (user && (user as any).id) {
        await GoNoteService.SyncNow((user as any).id);
        console.log("Sync complete!");
      }
    } catch (err) {
      console.error("Sync failed:", err);
    } finally {
      setTimeout(() => this.isSaving = false, 500);
    }
  }

  // === UI Customization ===
  setMode(newMode: string) {
    this.note.update(n => n ? { ...n, mode: newMode } : n);
    this.showMenu.set(false);
    this.saveContent();
  }

  setColor(newColor: string) {
    this.note.update(n => n ? { ...n, color: newColor } : n);
    this.showMenu.set(false);
    this.saveContent();
  }

  // === Note Window Management ===
  async newNote() {
    const user = await GoNoteService.GetCurrentUser();
    if(user) await GoNoteService.CreateNote((user as any).id);
    this.closeAll();
  }

  async deleteNote() {
    const n = this.note();
    if (!n) return;
    await GoNoteService.DeleteNote(n.id);
    // Hide it instead of destroying it to play nice with our System Tray architecture
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
    await GoNoteService.SetAlwaysOnTop(n.id, updated.pinned);

    this.isSaving = true;
    await GoNoteService.UpdateNote(updated);
    setTimeout(() => this.isSaving = false, 400);
  }

  // === Tiptap Commands ===
  toggleHeading(level: 1 | 2 | 3) {
    this.editor.chain().focus().toggleHeading({ level }).run();
  }

  toggleTaskList() {
    this.editor.chain().focus().toggleTaskList().run();
  }

  toggleBulletList() {
    this.editor.chain().focus().toggleBulletList().run();
  }

  // === Timer Logic ===
  get timerDisplay(): string {
    const m = Math.floor(this.timerTotal / 60);
    const s = this.timerTotal % 60;
    return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  }

  startCountdown() {
    this.timerTotal = this.timerInputMin * 60;
    this.showTimer.set(false);
    this.runTimer();
  }

  private runTimer() {
    const initial = this.timerTotal;
    if (this.timerInterval) clearInterval(this.timerInterval);
    this.timerInterval = setInterval(() => {
        if (this.timerTotal > 0) {
            this.timerTotal--;
            this.timerProgress = ((initial - this.timerTotal) / initial) * 100;
        } else {
            clearInterval(this.timerInterval);
            this.timerInterval = null;
        }
    }, 1000);
  }

  onRightClick(e: MouseEvent) {
    e.preventDefault();
    this.showMenu.set(true);
  }

  closeAll() {
    this.showMenu.set(false);
  }
}
