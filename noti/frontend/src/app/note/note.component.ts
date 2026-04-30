import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { NoteService, Note } from '../services/note.service';

export interface Block {
  id: string;
  type: 'paragraph' | 'h1' | 'h2' | 'h3' | 'bullet' | 'numbered' | 'checklist' | 'image' | 'divider';
  content: string;
  checked?: boolean;
  imageUrl?: string;
}

interface Command {
  type: string;
  label: string;
  icon: string;
}

@Component({
  selector: 'app-note',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './note.component.html'
})
export class NoteComponent implements OnInit {
  // Expose JS globals to the Angular template
  Math = Math;
  String = String;

  note = signal<Note | null>(null);

  // -- Editor State --
  blocks = signal<Block[]>([]);
  showMenu = signal(false);
  showSlash = signal(false);
  showToolbar = signal(false);
  slashFilter = signal<Command[]>([]);

  commands: Command[] = [
    { type: 'paragraph', label: 'Text', icon: '¶' },
    { type: 'h1', label: 'Heading 1', icon: 'H1' },
    { type: 'h2', label: 'Heading 2', icon: 'H2' },
    { type: 'h3', label: 'Heading 3', icon: 'H3' },
    { type: 'bullet', label: 'Bullet List', icon: '•' },
    { type: 'numbered', label: 'Numbered List', icon: '1.' },
    { type: 'checklist', label: 'To-do List', icon: '☑' },
    { type: 'image', label: 'Image', icon: '🖼' },
    { type: 'divider', label: 'Divider', icon: '―' }
  ];

  // -- Timer State --
  showTimer = signal(false);
  timerTotal = 0;
  timerProgress = 0;
  timerInputMin = 25;
  timerInterval: any;

  constructor(
    private route: ActivatedRoute,
    private noteService: NoteService
  ) {}

  async ngOnInit() {
    const noteId = this.route.snapshot.queryParamMap.get('noteId');
    if (noteId) {
      const n = await this.noteService.getNoteById(noteId);
      if (n) {
        this.note.set(n);
        this.parseContent(n.content);
      }
    }
  }

  parseContent(content: string) {
    try {
      const parsed = JSON.parse(content);
      if (Array.isArray(parsed)) {
        this.blocks.set(parsed);
        return;
      }
    } catch {}
    this.blocks.set([{ id: Date.now().toString(), type: 'paragraph', content: '' }]);
  }

  async save() {
    const n = this.note();
    if (!n) return;
    const updated = {
      ...n,
      content: JSON.stringify(this.blocks())
    };
    await this.noteService.updateNote(updated);
    this.note.set(updated);
  }

  async newNote() {
    await this.noteService.createNote();
    this.closeAll();
  }

  async deleteNote() {
    const n = this.note();
    if (!n) return;
    await this.noteService.deleteNote(n.id);
    window.close();
  }

  async togglePin() {
    const n = this.note();
    if (!n) return;
    const updated = { ...n, pinned: !n.pinned };
    this.note.set(updated);
    await this.noteService.setAlwaysOnTop(n.id, updated.pinned);
    await this.noteService.updateNote(updated);
  }

  // === Block Editor Logic Stub ===
  onKeydown(event: Event, block: Block) {}

  onInput(event: Event, block: Block) {
    const target = event.target as HTMLElement;
    this.updateBlock(block.id, { content: target.textContent || '' });
  }

  updateBlock(id: string, data: Partial<Block>) {
    this.blocks.update(blocks => blocks.map(b => b.id === id ? { ...b, ...data } : b));
  }

  focusBlock(id: string) {}

  showToolbarFor(id: string) {
    this.showToolbar.set(true);
  }

  pickSlashCommand(cmd: Command) {
    this.showSlash.set(false);
  }

  pickToolbarCommand(cmd: Command) {
    this.showToolbar.set(false);
  }

  onImageDrop(event: Event, block: Block) {}
  onImageSelect(event: Event, block: Block) {}

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

  toggleCountdown() {
    if (this.timerInterval) {
        clearInterval(this.timerInterval);
        this.timerInterval = null;
    } else if (this.timerTotal > 0) {
        this.runTimer();
    }
  }

  resetCountdown() {
    clearInterval(this.timerInterval);
    this.timerInterval = null;
    this.timerTotal = 0;
    this.timerProgress = 0;
  }

  private runTimer() {
    const initial = this.timerTotal;
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

  // === General UI ===
  onRightClick(e: MouseEvent) {
    e.preventDefault();
    this.showMenu.set(true);
  }

  closeAll() {
    this.showMenu.set(false);
    this.showToolbar.set(false);
    this.showSlash.set(false);
  }

  async setMode(mode: 'list' | 'lined' | 'squares' | 'dots' | 'browse') {
    const n = this.note();
    if (!n) return;
    const updated = { ...n, mode };
    this.note.set(updated);
    this.showMenu.set(false); // Close menu after clicking
    await this.noteService.updateNote(updated);
  }
}
