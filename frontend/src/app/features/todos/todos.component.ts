import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TodoService } from '../../core/services/todo.service';
import { AuthService } from '../../core/services/auth.service';
import { Router } from '@angular/router';
import { Todo } from '../../core/models/todo.model';

@Component({
  selector: 'app-todos',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './todos.component.html'
})
export class TodosComponent implements OnInit {
  newTitle = '';
  editingId = signal<string | null>(null);
  editingTitle = '';
  loading = true;

  constructor(
    public todoService: TodoService,
    public auth: AuthService,
    private router: Router
  ) {}

  async ngOnInit() {
    await this.todoService.loadTodos();
    this.loading = false;
  }

  async addTodo() {
    const title = this.newTitle.trim();
    if (!title) return;
    await this.todoService.createTodo(title);
    this.newTitle = '';
  }

  async toggle(id: string) {
    await this.todoService.toggleDone(id);
  }

  startEdit(todo: Todo) {
    this.editingId.set(todo.id);
    this.editingTitle = todo.title;
  }

  async saveEdit(id: string) {
    const title = this.editingTitle.trim();
    if (title) await this.todoService.updateTodo(id, { title });
    this.editingId.set(null);
  }

  cancelEdit() { this.editingId.set(null); }

  async deleteTodo(id: string) {
    await this.todoService.deleteTodo(id);
  }

  async logout() {
    await this.auth.logout();
    this.router.navigate(['/login']);
  }

  trackById(_: number, t: Todo) { return t.id; }
}
